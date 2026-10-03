package cli

import (
	"fmt"

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
	)
	return c
}

// NewVersionCmd prints the build version.
func NewVersionCmd(d Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "vpn %s\n", d.Version)
			return nil
		},
	}
}
