package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// NewSubCmd groups subscription commands.
func NewSubCmd(d Deps) *cobra.Command {
	c := &cobra.Command{Use: "sub", Short: "Manage subscriptions", Args: cobra.NoArgs}
	c.AddCommand(
		&cobra.Command{
			Use:   "add <name> <url>",
			Short: "Add a subscription",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				sub, err := d.Sub.Add(args[0], args[1])
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "added sub %s (%s)\n", sub.ID, sub.Name)
				return nil
			},
		},
		&cobra.Command{
			Use:   "list",
			Short: "List subscriptions",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				list, err := d.Sub.List()
				if err != nil {
					return err
				}
				for _, s := range list {
					fmt.Fprintf(cmd.OutOrStdout(), "%s %s %s\n", s.ID, s.Name, s.URL)
				}
				return nil
			},
		},
		&cobra.Command{
			Use:   "update [id|name]",
			Short: "Fetch all (or one) subscriptions",
			Args:  cobra.MaximumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				if len(args) == 1 {
					n, err := d.Sub.Update(args[0])
					if err != nil {
						return err
					}
					fmt.Fprintf(cmd.OutOrStdout(), "updated: %d profiles\n", n)
					return nil
				}
				list, err := d.Sub.List()
				if err != nil {
					return err
				}
				total := 0
				for _, s := range list {
					n, err := d.Sub.Update(s.ID)
					if err != nil {
						return err
					}
					total += n
				}
				fmt.Fprintf(cmd.OutOrStdout(), "updated: %d profiles\n", total)
				return nil
			},
		},
		&cobra.Command{
			Use:   "remove <id|name>",
			Short: "Delete a subscription",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				if err := d.Sub.Remove(args[0]); err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), "removed")
				return nil
			},
		},
		&cobra.Command{
			Use:   "export",
			Short: "Export subscription URLs (share)",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				outPath, _ := cmd.Flags().GetString("out")
				list, err := d.Sub.List()
				if err != nil {
					return err
				}
				var lines []string
				for _, s := range list {
					lines = append(lines, fmt.Sprintf("%s %s", s.Name, s.URL))
				}
				return exportOutput(cmd, outPath, lines)
			},
		},
	)
	for _, sub := range c.Commands() {
		if sub.Name() == "export" {
			sub.Flags().String("out", "", "write to file instead of stdout")
		}
	}
	return c
}

// exportOutput writes the given lines to stdout, a file or the clipboard.
func exportOutput(cmd *cobra.Command, outPath string, lines []string) error {
	body := strings.Join(lines, "\n")
	if body != "" {
		body += "\n"
	}
	if outPath != "" {
		return os.WriteFile(outPath, []byte(body), 0o600)
	}
	fmt.Fprint(cmd.OutOrStdout(), body)
	return nil
}

// NewVersionCmd prints CLI and core versions.
func NewVersionCmd(d Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "vpn %s\n", d.Version)
			if v, err := singboxVersion(d.Config.Core.SingBox.Path); err == nil {
				fmt.Fprintf(out, "sing-box %s (min %s)\n", v, d.Config.Core.SingBox.MinVers)
			} else {
				fmt.Fprintf(out, "sing-box not found (min %s)\n", d.Config.Core.SingBox.MinVers)
			}
			return nil
		},
	}
}
