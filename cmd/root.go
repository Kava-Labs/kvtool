package cmd

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kava-labs/kava/app"
	"github.com/spf13/cobra"
)

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

	cdc := app.MakeCodec()

	rootCmd.AddCommand(MonikersCmd(cdc))
	rootCmd.AddCommand(LaunchBlameCmd(cdc))

	return rootCmd.Execute()
}
