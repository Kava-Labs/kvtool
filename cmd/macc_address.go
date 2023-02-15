package cmd

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/crypto"
)

// MaccAddrCmd returns a command that gives the module account address of the passed in module name
func MaccAddrCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "macc-address [module-name]",
		Short: "Helper for getting the address of a module account.",
		Example: `$ kvtool macc-address kavadist
> kava1cj7njkw2g9fqx4e768zc75dp9sks8u9znxrf0w

$ kvtool macc-address liquid
> kava1gggszchqvw2l65my03mak6q5qfhz9cn2g0px29

$ kvtool macc-address hypothetical-module-name
> kava1s9z272h8cacjjj84yps2fk2rvwpruc3juqpn85`,
		Args: cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			moduleName := args[0]
			fmt.Println(sdk.AccAddress(crypto.AddressHash([]byte(moduleName))).String())
		},
	}

	return cmd
}
