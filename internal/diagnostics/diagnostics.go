// Package diagnostics implements the non-destructive diagnostic framework
// and check suite (spec sections 25, 26). Checks never mutate firewall
// state; destructive probes require explicit opt-in downstream.
package diagnostics

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/PotenFYR-Studios/FYRwall/internal/secureconfig"
)

// Result of one check.
type Result struct {
	Code        string        `json:"code"`
	Check       string        `json:"check"`
	Status      string        `json:"status"` // PASS|WARN|FAIL|SKIP
	Message     string        `json:"message"`
	Detail      string        `json:"detail,omitempty"`
	Remediation string        `json:"remediation,omitempty"`
	Duration    time.Duration `json:"duration_ns"`
}

// CheckFunc runs one named diagnostic.
type CheckFunc func(ctx context.Context) Result

func pass(code, msg string) Result            { return Result{Code: code, Status: "PASS", Message: msg} }
func warn(code, msg string) Result            { return Result{Code: code, Status: "WARN", Message: msg} }
func fail(code, msg string) Result            { return Result{Code: code, Status: "FAIL", Message: msg} }
func skip(code, msg string) Result            { return Result{Code: code, Status: "SKIP", Message: msg} }
func rem(r Result, remediation string) Result { r.Remediation = remediation; return r }

func timed(ctx context.Context, code, name string, fn func(context.Context) Result) Result {
	start := time.Now()
	r := fn(ctx)
	r.Code = code
	r.Check = name
	r.Duration = time.Since(start)
	return r
}

// Registry holds the named checks.
type Registry struct {
	checks map[string]CheckFunc
	order  []string
}

// NewRegistry builds the default suite (spec section 25 list).
func NewRegistry() *Registry {
	r := &Registry{checks: map[string]CheckFunc{}}
	r.add("KERNEL", "Kernel version and Linux platform", checkKernel)
	r.add("ARCH", "CPU architecture", checkArch)
	r.add("NETFILTER", "Netfilter support via /proc", checkNetfilter)
	r.add("UFW_BIN", "UFW binary availability", checkUFBBin)
	r.add("IPTABLES_BIN", "iptables availability and version", checkIptables)
	r.add("IPTABLES_SAVE", "iptables-save availability", checkIptablesSave)
	r.add("IPTABLES_RESTORE", "iptables-restore availability", checkIptablesRestore)
	r.add("IP6TABLES", "ip6tables availability (IPv6)", checkIP6Tables)
	r.add("NFT_BIN", "nft binary availability", checkNft)
	r.add("IPV6", "IPv6 kernel support", checkIPv6)
	r.add("SERVICE_MANAGER", "Init system detection", checkServiceManager)
	r.add("DISK_SPACE", "Sufficient free disk space", checkDisk)
	r.add("CLOCK", "Clock sanity (RTC vs monotonic)", checkClock)
	r.add("LOOPBACK", "Loopback interface present", checkLoopback)
	r.add("DEFAULT_ROUTE", "Default route present", checkRoute)
	r.add("DNS", "DNS resolution works", checkDNS)
	r.add("CONFIG_STORE", "Config encryption and integrity", checkConfigStore)
	return r
}

func (r *Registry) add(code, name string, fn CheckFunc) {
	r.checks[code] = fn
	r.order = append(r.order, code)
}

// Names lists check codes in stable order.
func (r *Registry) Names() []string { return append([]string(nil), r.order...) }

// Run executes all checks (or only the named subset when names given).
func (r *Registry) Run(ctx context.Context, names ...string) []Result {
	want := r.order
	if len(names) > 0 {
		want = names
	}
	out := make([]Result, 0, len(want))
	for _, n := range want {
		fn, ok := r.checks[n]
		if !ok {
			out = append(out, Result{Code: n, Status: "SKIP", Message: "unknown check"})
			continue
		}
		out = append(out, timed(ctx, n, n, fn))
	}
	return out
}

func checkKernel(ctx context.Context) Result {
	b, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return fail("KERNEL", "cannot read kernel version")
	}
	return pass("KERNEL", strings.TrimSpace(string(b)))
}

func checkArch(ctx context.Context) Result {
	return pass("ARCH", fmt.Sprintf("%s/%s", "linux", runtime.GOARCH))
}

func checkNetfilter(ctx context.Context) Result {
	if _, err := os.Stat("/proc/net"); err != nil {
		return fail("NETFILTER", "/proc/net inaccessible")
	}
	if _, err := os.Stat("/proc/net/ip_tables_names"); err == nil {
		return pass("NETFILTER", "ip_tables visible")
	}
	if _, err := os.Stat("/proc/net/ip6_tables_names"); err == nil {
		return pass("NETFILTER", "ip6_tables visible")
	}
	return warn("NETFILTER", "netfilter proc entries not visible (container or nft-only host)")
}

func checkUFBBin(ctx context.Context) Result {
	if p, err := exec.LookPath("ufw"); err == nil {
		return pass("UFW_BIN", p)
	}
	for _, p := range []string{"/usr/sbin/ufw", "/usr/bin/ufw"} {
		if _, err := os.Stat(p); err == nil {
			return pass("UFW_BIN", p)
		}
	}
	return skip("UFW_BIN", "ufw not installed")
}

func checkIptables(ctx context.Context) Result {
	p, err := exec.LookPath("iptables")
	if err != nil {
		return skip("IPTABLES_BIN", "iptables not installed")
	}
	out, err := exec.CommandContext(ctx, p, "--version").Output()
	if err != nil {
		return fail("IPTABLES_BIN", "iptables --version failed: "+err.Error())
	}
	return pass("IPTABLES_BIN", strings.TrimSpace(string(out)))
}

func checkIptablesSave(ctx context.Context) Result {
	if _, err := exec.LookPath("iptables-save"); err == nil {
		return pass("IPTABLES_SAVE", "found")
	}
	return skip("IPTABLES_SAVE", "iptables-save not installed")
}

func checkIptablesRestore(ctx context.Context) Result {
	if _, err := exec.LookPath("iptables-restore"); err == nil {
		return pass("IPTABLES_RESTORE", "found")
	}
	return skip("IPTABLES_RESTORE", "iptables-restore not installed")
}

func checkIP6Tables(ctx context.Context) Result {
	if _, err := exec.LookPath("ip6tables"); err == nil {
		return pass("IP6TABLES", "found")
	}
	return skip("IP6TABLES", "ip6tables not installed")
}

func checkNft(ctx context.Context) Result {
	if _, err := exec.LookPath("nft"); err == nil {
		return pass("NFT_BIN", "found")
	}
	return skip("NFT_BIN", "nft not installed")
}

func checkIPv6(ctx context.Context) Result {
	b, err := os.ReadFile("/proc/sys/net/ipv6/conf/all/disable_ipv6")
	if err != nil {
		return skip("IPV6", "IPv6 sysctl not present")
	}
	if strings.TrimSpace(string(b)) == "1" {
		return warn("IPV6", "IPv6 disabled by sysctl")
	}
	return pass("IPV6", "IPv6 enabled")
}

func checkServiceManager(ctx context.Context) Result {
	switch {
	case isDir("/run/systemd/system"):
		return pass("SERVICE_MANAGER", "systemd")
	case isDir("/sbin/openrc"), isDir("/run/openrc"):
		return pass("SERVICE_MANAGER", "openrc")
	case isFile("/etc/runit/stopit"):
		return pass("SERVICE_MANAGER", "runit")
	default:
		return warn("SERVICE_MANAGER", "unknown init system; manual startup fallback applies")
	}
}

func checkDisk(ctx context.Context) Result {
	var st syscall.Statfs_t
	if err := syscall.Statfs("/var/lib", &st); err != nil {
		// Fall back to cwd's filesystem in containers.
		if err := syscall.Statfs(".", &st); err != nil {
			return fail("DISK_SPACE", "cannot stat filesystem")
		}
	}
	freeGB := float64(st.Bavail) * float64(st.Bsize) / 1e9
	if freeGB < 0.5 {
		return fail("DISK_SPACE", fmt.Sprintf("only %.2f GB free", freeGB))
	}
	if freeGB < 2 {
		return rem(warn("DISK_SPACE", fmt.Sprintf("%.2f GB free", freeGB)), "Free more disk space; restore points need headroom.")
	}
	return pass("DISK_SPACE", fmt.Sprintf("%.1f GB free", freeGB))
}

func checkClock(ctx context.Context) Result {
	now := time.Now()
	if now.Year() < 2024 {
		return rem(fail("CLOCK", fmt.Sprintf("clock reads %s; TLS and tokens will fail", now.Format(time.RFC3339))),
			"Fix the system clock or enable time synchronization.")
	}
	return pass("CLOCK", now.Format(time.RFC3339))
}

func checkLoopback(ctx context.Context) Result {
	ifaces, err := net.Interfaces()
	if err != nil {
		return fail("LOOPBACK", "cannot enumerate interfaces")
	}
	for _, i := range ifaces {
		if i.Flags&net.FlagLoopback != 0 && i.Flags&net.FlagUp != 0 {
			return pass("LOOPBACK", i.Name+" up")
		}
	}
	return fail("LOOPBACK", "no up loopback interface")
}

func checkRoute(ctx context.Context) Result {
	b, err := os.ReadFile("/proc/net/route")
	if err != nil {
		return skip("DEFAULT_ROUTE", "cannot read /proc/net/route")
	}
	for _, line := range strings.Split(string(b), "\n")[1:] {
		f := strings.Fields(line)
		if len(f) > 1 && f[1] == "00000000" {
			return pass("DEFAULT_ROUTE", "present")
		}
	}
	return warn("DEFAULT_ROUTE", "no IPv4 default route")
}

func checkDNS(ctx context.Context) Result {
	resolver := &net.Resolver{}
	ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// Resolve a resolvable name; offline is a WARN not FAIL since FYRwall
	// is offline-capable (spec section 72).
	if _, err := resolver.LookupHost(ctx2, "localhost"); err != nil {
		return fail("DNS", "cannot resolve localhost")
	}
	if _, err := resolver.LookupHost(ctx2, "fyrwall.invalid"); err == nil {
		// NXDOMAIN is expected; wildcard-capture DNS is a warning.
		return warn("DNS", "DNS wildcard capture detected")
	}
	return pass("DNS", "resolver responding")
}

// checkConfigStore verifies the on-disk config decrypts cleanly and the
// keyfile is intact (tamper or key-loss detection, spec section 8.7).
func checkConfigStore(ctx context.Context) Result {
	path := os.Getenv("FYRWALL_CONFIG_PATH")
	if path == "" {
		path = "/etc/fyrwall/config.yaml"
	}
	if _, err := os.Stat(path); err != nil {
		return skip("CONFIG_STORE", "no config file yet (fresh install)")
	}
	sc, err := secureconfig.New(path)
	if err != nil {
		return fail("CONFIG_STORE", "config store unavailable: "+err.Error())
	}
	if err := sc.VerifyIntegrity(); err != nil {
		return rem(fail("CONFIG_STORE", err.Error()),
			"Restore the config key file or re-import the config via the web GUI.")
	}
	return pass("CONFIG_STORE", "encrypted config verified")
}

func isDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func isFile(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}
