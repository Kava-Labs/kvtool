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

	cmd := &cobra.Command{
		Use:   "swap-id [random number hash] [non deputy address]",
		Short: "Calculate the binance and kava swap IDs given swap details.",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {

			// get deputy addresses
			kavaDeputy, err := sdk.AccAddressFromBech32("kava1r4v2zdhdalfj2ydazallqvrus9fkphmglhn6u6")
			if err != nil {
				panic(err.Error())
			}
			bnbDeputy, err := binance.AccAddressFromBech32("bnb1jh7uv2rm6339yue8k4mj9406k3509kr4wt5nxn")
			if err != nil {
				panic(err.Error())
			}

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
				return fmt.Errorf("can't unmarshal input address as either kava or bnb: (%s) (%s)", errKava.Error(), errBnb.Error())
			}

			// calculate swap IDs
			var swapIDKava, swapIDBnb []byte
			if isKavaAddress {
				if addressKava.Equals(kavaDeputy) {
					return fmt.Errorf("input address cannot be deputy address: %s", kavaDeputy)
				}
				swapIDKava = types.CalculateSwapID(randomNumberHash, addressKava, bnbDeputy.String())
				swapIDBnb = binance.CalculateSwapID(randomNumberHash, bnbDeputy, addressKava.String())
			} else {
				if bytes.Equal(addressBnb, bnbDeputy) {
					return fmt.Errorf("address cannot be deputy address %s", bnbDeputy)
				}
				swapIDBnb = binance.CalculateSwapID(randomNumberHash, addressBnb, kavaDeputy.String())
				swapIDKava = types.CalculateSwapID(randomNumberHash, kavaDeputy, addressBnb.String())
			}

			// print out result
			result := struct {
				KavaID string
				BnbID  string
			}{
				KavaID: hex.EncodeToString(swapIDKava),
				BnbID:  hex.EncodeToString(swapIDBnb),
			}
			bz, err := yaml.Marshal(result)
			if err != nil {
				return err
			}
			fmt.Println(string(bz))
			return nil
		},
	}

	return cmd
}
