package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	"github.com/cosmos/cosmos-sdk/x/staking"
)

// MonikersMapCmd returns a command to print out a map of validator hex addresses to their monikers.
func MonikersCmd(cdc *codec.Codec) *cobra.Command {
	var genesisFile string

	cmd := &cobra.Command{
		Use:   "monikers",
		Short: "Print a JSON map of the validator's addresses to their monikers.",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {

			// 1) Load the genesis file.
			genAppState, _, err := genutil.GenesisStateFromGenFile(cdc, genesisFile)
			if err != nil {
				return fmt.Errorf("failed to unmarshal genesis state: %w", err)
			}

			// 2) Create the map from hex addresses to monikers
			// Use genutil.GenTxs if it isn't empty, otherwise use staking.Validators
			addrMonikers, err := getAddressMonikersFromGenTxs(cdc, genAppState["genutil"])
			if err != nil {
				addrMonikers, err = getAddressMonikersFromStaking(cdc, genAppState["staking"])
			}
			if err != nil {
				return err
			}

			// 3) Print results
			bz, err := cdc.MarshalJSONIndent(addrMonikers, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(bz))
			return nil
		},
	}

	homeDir, err := os.UserHomeDir()
	defaultGenesisFile := filepath.Join(homeDir, ".kvd/config/genesis.json")
	if err != nil {
		defaultGenesisFile = ""
	}
	cmd.Flags().StringVar(&genesisFile, "genesis", defaultGenesisFile, "genesis file location")

	return cmd
}

func getAddressMonikersFromGenTxs(cdc *codec.Codec, rawGenState json.RawMessage) (map[string]string, error) {
	result := make(map[string]string)

	var genutilGenState genutil.GenesisState
	err := cdc.UnmarshalJSON(rawGenState, &genutilGenState)
	if err != nil {
		return result, err
	}

	if len(genutilGenState.GenTxs) < 1 {
		return result, errors.New("no gentxs in genesis state")
	}

	for _, rawGenTx := range genutilGenState.GenTxs {
		var genTx auth.StdTx
		cdc.MustUnmarshalJSON(rawGenTx, &genTx)
		msg := genTx.Msgs[0].(staking.MsgCreateValidator)

		hexConsAddr := msg.PubKey.Address().String()
		result[hexConsAddr] = msg.Description.Moniker
	}
	return result, nil
}

func getAddressMonikersFromStaking(cdc *codec.Codec, rawGenState json.RawMessage) (map[string]string, error) {
	result := make(map[string]string)

	var stakingGenState staking.GenesisState
	err := cdc.UnmarshalJSON(rawGenState, &stakingGenState)
	if err != nil {
		return result, err
	}

	for _, val := range stakingGenState.Validators {
		hexAddr := fmt.Sprintf("%X", val.ConsAddress().Bytes())
		result[hexAddr] = val.Description.Moniker
	}
	return result, nil
}
