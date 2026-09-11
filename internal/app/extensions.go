package app

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PotenFYR-Studios/FYRwall/internal/extension"
)

// extensionRegistry returns the system extension registry.
func extensionRegistry() *extension.Registry {
	root := os.Getenv("FYRWALL_EXTENSIONS_DIR")
	if root == "" {
		root = "/var/lib/fyrwall/extensions"
	}
	return extension.NewRegistry(root)
}

// promptCapabilities validates the manifest, shows the requested
// capabilities and asks for an explicit y/N grant per capability.
func promptCapabilities(ctx context.Context, dir string) ([]extension.Capability, error) {
	m, err := readManifest(dir)
	if err != nil {
		return nil, err
	}
	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("manifest rejected: %w", err)
	}
	required := m.RequiredCapabilities()
	fmt.Printf("Extension %q v%s by %s\\n", m.Name, m.Version, m.Author)
	fmt.Printf("Capabilities requested (denied unless granted):\\n")
	in := bufio.NewReader(os.Stdin)
	var granted []extension.Capability
	for _, c := range required {
		fmt.Printf("  grant %-22s [y/N]: ", c)
		line, _ := in.ReadString('\n')
		ans := strings.ToLower(strings.TrimSpace(line))
		if ans == "y" || ans == "yes" {
			granted = append(granted, c)
		}
	}
	return granted, nil
}

func readManifest(dir string) (*extension.Manifest, error) {
	// Delegate manifest parsing to the registry via a dry-run: reuse
	// InstallFromDir against a throwaway registry root so parsing and
	// validation rules stay in one place.
	tmp := filepath.Join(os.TempDir(), "fyrwall-ext-dryrun")
	_ = os.MkdirAll(tmp, 0o750)
	defer os.RemoveAll(tmp)
	reg := extension.NewRegistry(tmp)
	inst, err := reg.InstallFromDir(dir, nil)
	if err != nil {
		return nil, err
	}
	return &inst.Manifest, nil
}
