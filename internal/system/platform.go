// Package system provides platform, distro, architecture and capability
// detection. All detection is read-only and capability-based per spec section 33.
package system

import (
	"os"
	"runtime"
	"strings"
	"sync"
)

// Platform describes the host in a capability-oriented way.
type Platform struct {
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	KernelVersion string `json:"kernel_version"`
	DistroID      string `json:"distro_id,omitempty"`
	DistroName    string `json:"distro_name,omitempty"`
	DistroVersion string `json:"distro_version,omitempty"`
	IsContainer   bool   `json:"is_container"`
	IsVM          bool   `json:"is_vm"`
	InitSystem    string `json:"init_system"` // systemd|openrc|runit|s6|sysvinit|unknown
}

var (
	platformOnce sync.Once
	cachedPlat   Platform
)

// DetectPlatform gathers host facts. Safe to call repeatedly; result cached.
func DetectPlatform() Platform {
	platformOnce.Do(func() {
		p := Platform{
			OS:            runtime.GOOS,
			Arch:          runtime.GOARCH,
			KernelVersion: kernelVersion(),
			IsContainer:   inContainer(),
			IsVM:          inVM(),
			InitSystem:    detectInit(),
		}
		p.DistroID, p.DistroName, p.DistroVersion = readOSRelease()
		cachedPlat = p
	})
	return cachedPlat
}

func kernelVersion() string {
	b, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(b))
}

func inContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	b, err := os.ReadFile("/proc/1/cgroup")
	if err != nil {
		return false
	}
	return strings.Contains(string(b), "/docker/") ||
		strings.Contains(string(b), "/lxc/") ||
		strings.Contains(string(b), "containerd")
}

func inVM() bool {
	cpu, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return false
	}
	s := string(cpu)
	// Common hypervisor flags on x86; harmless elsewhere.
	return strings.Contains(s, "hypervisor") ||
		strings.Contains(s, "QEMU") ||
		strings.Contains(s, "vmware") ||
		strings.Contains(s, "VirtualBox")
}

func detectInit() string {
	if _, err := os.Stat("/run/systemd/system"); err == nil {
		return "systemd"
	}
	if _, err := os.Stat("/sbin/openrc"); err == nil {
		return "openrc"
	}
	if _, err := os.Stat("/etc/runit/stopit"); err == nil {
		return "runit"
	}
	if _, err := os.Stat("/run/s6/container_environment"); err == nil {
		return "s6"
	}
	if pid1, err := os.Readlink("/proc/1/exe"); err == nil {
		base := pid1
		if i := strings.LastIndexByte(pid1, '/'); i >= 0 {
			base = pid1[i+1:]
		}
		switch {
		case strings.HasPrefix(base, "systemd"), strings.HasPrefix(base, "init"):
			return "systemd"
		case strings.Contains(base, "openrc"):
			return "openrc"
		case strings.HasPrefix(base, "runit"):
			return "runit"
		case strings.HasPrefix(base, "s6"):
			return "s6"
		}
	}
	return "unknown"
}

// readOSRelease parses /etc/os-release (or /usr/lib/os-release) for the ID,
// PRETTY_NAME and VERSION_ID fields. Never assumes a specific distro.
func readOSRelease() (id, name, version string) {
	for _, path := range []string{"/etc/os-release", "/usr/lib/os-release"} {
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		fields := map[string]string{}
		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			k, v, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			v = strings.Trim(v, `"'`)
			fields[k] = v
		}
		return fields["ID"], fields["PRETTY_NAME"], fields["VERSION_ID"]
	}
	return "unknown", "unknown", ""
}
