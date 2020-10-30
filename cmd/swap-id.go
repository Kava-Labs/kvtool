package cmd

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kava-labs/kava/x/bep3/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/kava-labs/kvtool/binance"
)

// SwapIDCmd returns a command to calculate a bep3 swap ID for binance and kava chains.
func SwapIDCmd(cdc *codec.Codec) *cobra.Command {

	var deputyAddrStr string

	mainnetKavaDeputy, err := sdk.AccAddressFromBech32("kava1r4v2zdhdalfj2ydazallqvrus9fkphmglhn6u6")
	if err != nil {
		panic(err.Error())
	}
	mainnetBnbDeputy, err := binance.AccAddressFromBech32("bnb1jh7uv2rm6339yue8k4mj9406k3509kr4wt5nxn")
	if err != nil {
		panic(err.Error())
	}

	cmd := &cobra.Command{
		Use:   "swap-id random_number_hash original_sender_address",
		Short: "Calculate binance and kava swap IDs given swap details.",
		Long: `A swap's ID is: hash(swap.RandomNumberHash, swap.Sender, swap.SenderOtherChain)
One of the senders is always the deputy's address, the other is the user who initiated the first swap (the original sender).
Corresponding swaps on each chain have the same addresses but switched order.
		
By default kava-3 mainnet deputy addresses are used, but can be overridden with a flag.
The original sender and deputy address cannot be from the same chain.
`,
		Example: "swap-id 464105c245199d02a4289475b8b231f3f73918b6f0fdad898825186950d46f36 bnb10rr5f8m73rxgnz9afvnfn7fn9pwhfskem5kn0x --deputy kava1agcvt07tcw0tglu0hmwdecsnuxp2yd45f3avgm",
		Args:    cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {

			// get deputy addresses

			randomNumberHash, err := hex.DecodeString(args[0])
			if err != nil {
				return err
			}

			// try and decode the bech32 address as either kava or bnb
			addressKava, errKava := sdk.AccAddressFromBech32(args[1])
			addressBnb, errBnb := binance.AccAddressFromBech32(args[1])

			// fail if both decoding failed
			isKavaAddress := errKava == nil && errBnb != nil
			isBnbAddress := errKava != nil && errBnb == nil
			if !isKavaAddress && !isBnbAddress {
				return fmt.Errorf("can't unmarshal original sender address as either kava or bnb: (%s) (%s)", errKava.Error(), errBnb.Error())
			}

			// calculate swap IDs
			var swapIDKava, swapIDBnb []byte
			if isKavaAddress {
				if addressKava.Equals(mainnetKavaDeputy) {
					return fmt.Errorf("original sender address cannot be mainnnet deputy address: %s", mainnetKavaDeputy)
				}
				var bnbDeputy binance.AccAddress
				if deputyAddrStr == "" {
					bnbDeputy = mainnetBnbDeputy
				} else {
					bnbDeputy, err = binance.AccAddressFromBech32(deputyAddrStr)
					if err != nil {
						return fmt.Errorf("can't unmarshal deputy address as bnb address (%s)", err)
					}
				}
				swapIDKava = types.CalculateSwapID(randomNumberHash, addressKava, bnbDeputy.String())
				swapIDBnb = binance.CalculateSwapID(randomNumberHash, bnbDeputy, addressKava.String())
			} else {
				if bytes.Equal(addressBnb, mainnetBnbDeputy) {
					return fmt.Errorf("original sender address cannot be mainnet deputy address %s", mainnetBnbDeputy)
				}
				var kavaDeputy sdk.AccAddress
				if deputyAddrStr == "" {
					kavaDeputy = mainnetKavaDeputy
				} else {
					kavaDeputy, err = sdk.AccAddressFromBech32(deputyAddrStr)
					if err != nil {
						return fmt.Errorf("can't unmarshal deputy address as kava address (%s)", err)
					}
				}
				swapIDBnb = binance.CalculateSwapID(randomNumberHash, addressBnb, kavaDeputy.String())
				swapIDKava = types.CalculateSwapID(randomNumberHash, kavaDeputy, addressBnb.String())
			}

			// print out result
			result := struct {
				KavaSwapID string
				BnbSwapID  string
			}{
				KavaSwapID: hex.EncodeToString(swapIDKava),
				BnbSwapID:  hex.EncodeToString(swapIDBnb),
			}
			bz, err := yaml.Marshal(result)
			if err != nil {
				return err
			}
			fmt.Println(string(bz))
			return nil
		},
	}
	cmd.Flags().StringVarP(&deputyAddrStr, "deputy-address", "d", "", "the deputy address on the receiving chain")

	return cmd
}
