// Package main is the fyrwall binary entrypoint (spec section 29).
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/PotenFYR-Studios/FYRwall/internal/app"
	"github.com/PotenFYR-Studios/FYRwall/internal/version"
)

func main() {
	root := &cobra.Command{
		Use:   "fyrwall",
		Short: "FYRwall: safe, focused Linux firewall administration",
	}
	root.AddCommand(serverCmd(), agentCmd(), statusCmd(), doctorCmd(),
		preflightCmd(), versionCmd(), configValidateCmd(), migrateCmd(),
		restoreCmd(), serviceCmd(),
		setupCmd(), extensionCmd(), updateCmd(), uninstallSelfCmd(), trayCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var flagConfig string

func addConfigFlag(c *cobra.Command) {
	c.Flags().StringVar(&flagConfig, "config", "/etc/fyrwall/config.yaml", "config file path")
}

func serverCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "server",
		Short: "Run the FYRwall web/API server (unprivileged)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunServer(cmd.Context(), flagConfig)
		},
	}
	addConfigFlag(c)
	return c
}

func agentCmd() *cobra.Command {
	var serverURL string
	c := &cobra.Command{
		Use:   "agent",
		Short: "Run the local firewall agent (Unix socket, unprivileged)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if serverURL != "" {
				if err := os.Setenv("FYRWALL_AGENT_SERVER_URL", serverURL); err != nil {
					return err
				}
			}
			return app.RunAgent(cmd.Context(), flagConfig)
		},
	}
	addConfigFlag(c)
	c.Flags().StringVar(&serverURL, "server", "", "central server URL for outbound fleet synchronization")
	return c
}

func statusCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "status",
		Short: "Show firewall and service status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.PrintStatus(cmd.Context(), flagConfig)
		},
	}
	addConfigFlag(c)
	return c
}

func doctorCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "doctor",
		Short: "Run diagnostics without changing firewall state",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunDoctor(cmd.Context())
		},
	}
	return c
}

func preflightCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "preflight",
		Short: "Run non-destructive preflight checks",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunPreflight(cmd.Context(), flagConfig)
		},
	}
	addConfigFlag(c)
	return c
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version.String())
		},
	}
}

func configValidateCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "config validate",
		Short: "Validate the configuration file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.ValidateConfig(flagConfig)
		},
	}
	addConfigFlag(c)
	return c
}

func migrateCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "db migrate",
		Short: "Apply pending database migrations",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunMigrations(flagConfig)
		},
	}
	addConfigFlag(c)
	return c
}

func restoreCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "restore", Short: "Manage restore points"}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List restore points",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			return app.RestoreList(flagConfig)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "create",
		Short: "Create a manual restore point",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			return app.RestoreCreate(cmd.Context(), flagConfig)
		},
	})
	return cmd
}

func setupCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "setup",
		Short: "Interactive first-time setup wizard (bind, TLS, domain, autostart, admin)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunSetup(cmd.Context(), flagConfig)
		},
	}
	addConfigFlag(c)
	c.Flags().Bool("non-interactive", false, "accept defaults without prompting")
	c.Flags().String("set", "", "set a key=value directly (repeatable, comma-separated)")
	return c
}

func extensionCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "extension", Short: "Manage extensions"}
	cmd.AddCommand(&cobra.Command{
		Use:   "install <dir>",
		Short: "Install an extension from a directory (asks for capability grants)",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return app.ExtensionInstall(cmd.Context(), args[0])
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List installed extensions",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			return app.ExtensionList()
		},
	})
	return cmd
}

func updateCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "update", Short: "Update FYRwall safely"}
	cmd.AddCommand(&cobra.Command{
		Use:   "check",
		Short: "Check for a newer release (uses the configured manifest)",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			return app.UpdateCheck(cmd.Context(), flagConfig)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "apply",
		Short: "Apply an update: backup, restore point, download, verify, migrate, restart",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			return app.UpdateApply(cmd.Context(), flagConfig)
		},
	})
	return cmd
}

func uninstallSelfCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Remove FYRwall from this machine (asks what to keep; never touches firewall rules)",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			return app.SelfUninstall()
		},
	}
}

func serviceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "service status",
		Short: "Show FYRwall service status via the detected init system",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.ServiceStatus(cmd.Context())
		},
	}
}

func trayCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "tray",
		Short: "Run the system tray icon (Open Web UI, Restart, Stop, Quit)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunTray(cmd.Context(), flagConfig)
		},
	}
	addConfigFlag(c)
	return c
}
