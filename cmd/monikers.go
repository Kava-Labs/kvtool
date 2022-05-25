package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	staking "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func getAddressMonikersFromStaking(cdc codec.Codec, rawGenState json.RawMessage) (map[string]string, error) {
	result := make(map[string]string)

	var stakingGenState staking.GenesisState
	err := cdc.UnmarshalJSON(rawGenState, &stakingGenState)
	if err != nil {
		return result, err
	}

	for _, val := range stakingGenState.Validators {
		cons, err := val.GetConsAddr()
		if err != nil {
			return result, err
		}
		hexAddr := fmt.Sprintf("%X", cons.Bytes())
		result[hexAddr] = val.Description.Moniker
	}
	return result, nil
}
