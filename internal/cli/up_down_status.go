package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewUpCmd starts the VPN: vpn up [profile].
func NewUpCmd(d Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "up [profile]",
		Short: "Connect the VPN",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ref := ""
			if len(args) > 0 {
				ref = args[0]
			}
			info, err := d.Conn.Up(cmd.Context(), ref)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "vpn up: pid %d (%s)\n", info.PID, info.Core)
			return nil
		},
	}
}

// NewDownCmd stops the VPN.
func NewDownCmd(d Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Disconnect the VPN",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := d.Conn.Down(cmd.Context()); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "vpn down")
			return nil
		},
	}
}

// NewStatusCmd prints connection status.
func NewStatusCmd(d Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show connection status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			st, err := d.Conn.Status(cmd.Context())
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if !st.Running {
				if st.StalePID != 0 {
					fmt.Fprintf(out, "stopped (stale pid %d cleaned)\n", st.StalePID)
				} else {
					fmt.Fprintln(out, "stopped")
				}
				return nil
			}
			fmt.Fprintf(out, "running: %s via %s (pid %d) since %s\n",
				st.ProfileName, st.Core, st.PID, st.Since.Format("15:04:05"))
			if st.Endpoint != "" {
				fmt.Fprintf(out, "endpoint: %s\n", st.Endpoint)
			}
			return nil
		},
	}
}
