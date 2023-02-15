package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/kava-labs/kvtool/kavaclient"
)

func InflationRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inflation [sub-command]",
		Short: "Various utilities for checking realized inflation",
	}

	var kavaGrpcUrl string
	cmd.PersistentFlags().StringVar(&kavaGrpcUrl, "node", "https://grpc.data.kava.io:443", "kava GRPC url to run queries against")

	client, err := kavaclient.NewClient(kavaGrpcUrl)
	if err != nil {
		panic(fmt.Sprintf("failed to create kava grpc client: %s", err))
	}

	cmd.AddCommand(InflationSpotCheckAPY(client))

	return cmd
}

func InflationSpotCheckAPY(k *kavaclient.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "realized-apy [start-height] [end-height]",
		Short: "Calculate the real inflation over a block range as an APY",
		Long: `Looks at the number of coins minted over a range of blocks & extrapolates it to a yearly APY.
End height is optional, defaults to latest block. If start height is negative, it will subtract from end.`,
		Args: cobra.MatchAll(cobra.MinimumNArgs(1), cobra.MaximumNArgs(2)),
		Example: `calculate inflation over a block range:
$ kvtool inflation realized-apy 2000000 2500000

calculate inflation from block 2M to present:
$ kvtool inflation realized-apy 2000000

calculate inflation from last 10 blocks ("--" is necessary to interpret as an argument):
$ kvtool inflation realized-apy -- -10

calculate inflation over the 1000 blocks before height 3000000:
$ kvtool inflation realized-apy -- -1000 3000000
`,
		RunE: func(_ *cobra.Command, args []string) error {
			var end int64
			var err error
			// default to latest block if no end provided
			if len(args) == 1 {
				latest, err := k.LatestBlock(5)
				if err != nil {
					return fmt.Errorf("failed to fetch latest block: %s", err)
				}
				end = latest.Header.Height
			} else {
				end, err = strconv.ParseInt(args[1], 10, 64)
				if err != nil {
					return fmt.Errorf("failed to parse end block: %s", err)
				}
			}

			start, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse start block: %s", err)
			}
			if start == 0 {
				return fmt.Errorf("start block cannot equal 0")
			}
			// interpret negative start values as a diff from end block.
			if start < 0 {
				start = end + start
			}

			result, err := k.InflationApyOverBlocks(start, end)
			if err != nil {
				return err
			}

			fmt.Printf(`inflation extrapolated as an APY
start block: %d
end block: %d
total seconds passed: %f
inflation apy (%d block avg): %s
`, result.Start, result.End, result.SecondsPassed, result.End-result.Start, result.Inflation)

			return nil
		},
	}
	return cmd
}
