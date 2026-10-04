package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/SabirDzh/VpnCLI/internal/domain"
	"github.com/SabirDzh/VpnCLI/internal/platform"
)

// NewProfileCmd groups profile management commands.
func NewProfileCmd(d Deps) *cobra.Command {
	c := &cobra.Command{Use: "profile", Short: "Manage profiles", Args: cobra.NoArgs}
	c.AddCommand(
		&cobra.Command{
			Use:   "add <uri|file>",
			Short: "Add a profile from URI, file or clipboard",
			Args:  cobra.MaximumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				isFile, _ := cmd.Flags().GetBool("file")
				useClip, _ := cmd.Flags().GetBool("clipboard")
				proto, _ := cmd.Flags().GetString("protocol")
				switch {
				case useClip:
					data, err := platform.ReadClipboard()
					if err != nil {
						return err
					}
					p, err := addPayload(d, "clipboard", proto, []byte(strings.TrimSpace(data)))
					if err != nil {
						return err
					}
					fmt.Fprintf(cmd.OutOrStdout(), "added %s (%s %s)\n", p.ID, p.Protocol, p.Name)
					return nil
				case isFile:
					if len(args) != 1 {
						return fmt.Errorf("add --file requires a path")
					}
					data, err := os.ReadFile(args[0])
					if err != nil {
						return err
					}
					p, err := addPayload(d, filepath.Base(args[0]), proto, data)
					if err != nil {
						return err
					}
					fmt.Fprintf(cmd.OutOrStdout(), "added %s (%s %s)\n", p.ID, p.Protocol, p.Name)
					return nil
				default:
					if len(args) != 1 {
						return fmt.Errorf("add requires a uri argument")
					}
					p, err := d.Profile.AddFromURI(args[0])
					if err != nil {
						return err
					}
					fmt.Fprintf(cmd.OutOrStdout(), "added %s (%s %s)\n", p.ID, p.Protocol, p.Name)
					return nil
				}
			},
		},
		&cobra.Command{
			Use:   "list",
			Short: "List profiles",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				list, err := d.Profile.List()
				if err != nil {
					return err
				}
				out := cmd.OutOrStdout()
				active, _ := d.Store.ActiveProfile()
				for _, p := range list {
					mark := " "
					if p.ID == active.ID {
						mark = "*"
					}
					fmt.Fprintf(out, "%s %s %-10s %-21s %s [%s]\n",
						mark, p.ID, p.Protocol, p.Endpoint.Host+":"+portStr(p.Endpoint.Port), p.Name, p.Source)
				}
				return nil
			},
		},
		&cobra.Command{
			Use:   "use <id|name>",
			Short: "Select the active profile",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				p, err := d.Profile.Use(args[0])
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "active: %s (%s)\n", p.Name, p.ID)
				return nil
			},
		},
		&cobra.Command{
			Use:   "remove <id|name>",
			Short: "Delete a profile",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				if err := d.Profile.Remove(args[0]); err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), "removed")
				return nil
			},
		},
	)
	c.Commands()[0].Flags().Bool("file", false, "import profile or native core config from file")
	c.Commands()[0].Flags().Bool("clipboard", false, "import profile or native core config from clipboard")
	c.Commands()[0].Flags().String("protocol", string(domain.ProtocolVLESS), "protocol hint for raw --file import")
	return c
}

// addPayload imports a URI payload or, when the data does not look like a
// URI, a native core config as-is.
func addPayload(d Deps, name string, proto string, data []byte) (domain.Profile, error) {
	if looksLikeURI(data) {
		return d.Profile.AddFromURI(strings.TrimSpace(string(data)))
	}
	return d.Profile.AddRaw(name, domain.Protocol(proto), data)
}

func looksLikeURI(data []byte) bool {
	i := strings.Index(string(data), "://")
	return i > 0
}

func portStr(p uint16) string {
	return fmt.Sprintf("%d", p)
}
