package cmd

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kava-labs/kava/app"
	"github.com/spf13/cobra"

	"github.com/kava-labs/kvtool/cmd/testnet"
)

var kavaGrpcUrl string

var rootCmd = &cobra.Command{
	Use:   "kvtool",
	Short: "Dev tools for working with the kava blockchain.",
}

// Execute runs the root command.
func Execute() error {
	config := sdk.GetConfig()
	app.SetBech32AddressPrefixes(config)
	app.SetBip44CoinType(config)
	config.Seal()

	var cdc *codec.LegacyAmino = app.MakeEncodingConfig().Amino

	rootCmd.AddCommand(EstimateBlockHeightCmd())
	rootCmd.AddCommand(InflationRootCmd())
	rootCmd.AddCommand(MaccAddrCmd())
	rootCmd.AddCommand(NodeKeysCmd(cdc))
	rootCmd.AddCommand(SwapIDCmd(cdc))
	rootCmd.AddCommand(testnet.Cmd())

	// testnet aliases
	rootCmd.AddCommand(testnet.KavaCmd())
	rootCmd.AddCommand(testnet.IbcCmd())

	return rootCmd.Execute()
}
