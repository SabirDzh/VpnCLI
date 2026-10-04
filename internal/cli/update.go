package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/SabirDzh/VpnCLI/internal/core/singbox"
)

// singboxVersion resolves the core binary and reports its version.
func singboxVersion(explicit string) (string, error) {
	bin, err := singbox.FindBinary(explicit)
	if err != nil {
		return "", err
	}
	return singbox.BinaryVersion(bin)
}

// NewUpdateCmd checks for (and installs) CLI updates:
// vpn update --check | vpn update [--version vX.Y.Z].
func NewUpdateCmd(d Deps) *cobra.Command {
	var checkOnly bool
	var want string
	c := &cobra.Command{
		Use:   "update",
		Short: "Check for and install CLI updates",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			res, err := d.Update.Check(cmd.Context())
			if err != nil {
				return err
			}
			cur := res.Current
			if cur == "" {
				cur = "dev"
			}
			if cur == "dev" {
				fmt.Fprintf(out, "dev build (latest release %s)\n", res.Latest)
				return nil
			}
			if !res.Available {
				fmt.Fprintf(out, "vpn %s is up to date (latest %s)\n", cur, res.Latest)
				return nil
			}
			fmt.Fprintf(out, "update available: %s -> %s\n", cur, res.Latest)
			if checkOnly {
				return nil
			}
			tag := res.Latest
			if want != "" {
				tag = want
			}
			if err := d.Update.Update(cmd.Context(), tag); err != nil {
				return err
			}
			fmt.Fprintf(out, "updated to %s\n", tag)
			return nil
		},
	}
	c.Flags().BoolVar(&checkOnly, "check", false, "only check, do not install")
	c.Flags().StringVar(&want, "version", "", "install a specific tag (e.g. v0.1.0)")
	return c
}
