package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/SabirDzh/VpnCLI/internal/subscription/uri"
)

// NewBenchCmd measures the URI parser throughput on a synthetic batch.
func NewBenchCmd(d Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "bench",
		Short: "Benchmark the profile URI parser",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			const n = 10000
			uris := uri.BenchURIs(n)
			start := time.Now()
			for _, u := range uris {
				if _, err := uri.Parse(u); err != nil {
					return err
				}
			}
			elapsed := time.Since(start)
			perConf := elapsed / n
			fmt.Fprintf(cmd.OutOrStdout(),
				"parsed %d configs in %s (%s per config, %.0f configs/sec)\n",
				n, elapsed.Round(time.Microsecond), perConf.Round(time.Nanosecond),
				float64(n)/elapsed.Seconds())
			return nil
		},
	}
}
