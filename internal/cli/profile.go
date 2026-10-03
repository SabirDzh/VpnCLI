package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/SabirDzh/VpnCLI/internal/domain"
)

// NewProfileCmd groups profile management commands.
func NewProfileCmd(d Deps) *cobra.Command {
	c := &cobra.Command{Use: "profile", Short: "Manage profiles", Args: cobra.NoArgs}
	c.AddCommand(
		&cobra.Command{
			Use:   "add <uri|file>",
			Short: "Add a profile from URI or native config file",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				isFile, _ := cmd.Flags().GetBool("file")
				if isFile {
					data, err := os.ReadFile(args[0])
					if err != nil {
						return err
					}
					proto, _ := cmd.Flags().GetString("protocol")
					p, err := d.Profile.AddRaw(filepath.Base(args[0]), domain.Protocol(proto), data)
					if err != nil {
						return err
					}
					fmt.Fprintf(cmd.OutOrStdout(), "added raw profile %s (%s)\n", p.ID, p.Name)
					return nil
				}
				p, err := d.Profile.AddFromURI(args[0])
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "added %s (%s %s)\n", p.ID, p.Protocol, p.Name)
				return nil
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
	c.Commands()[0].Flags().Bool("file", false, "import native core config file as-is")
	c.Commands()[0].Flags().String("protocol", string(domain.ProtocolVLESS), "protocol hint for --file import")
	return c
}

func portStr(p uint16) string {
	return fmt.Sprintf("%d", p)
}
