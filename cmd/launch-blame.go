package cmd

import (
	"fmt"
	"regexp"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/consensus/types"
	"github.com/tendermint/tendermint/rpc/client/http"
)

const fingerPrintLength = 12

// LaunchBlameCmd fetches the consesnus state from a rpc node. It outputs the monikers of the validators that have not signed.
// It's useful for running on stalled chain launches to see which validators are not online yet.
func LaunchBlameCmd(cdc *codec.Codec) *cobra.Command {
	var nodeAddress string

	cmd := &cobra.Command{
		Use:   "launch-blame",
		Short: "Find non signing validators in a launching network",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {

			c, err := http.New(nodeAddress, "/websocket")
			if err != nil {
				return fmt.Errorf("can't connect to node: %w", err)
			}
			// 1) Load the validator monikers and consensus addresses
			vals, err := fetchValidators(cdc, c)
			if err != nil {
				return fmt.Errorf("could not fetch validator monikers: %w", err)
			}

			// 3) Fetch signing validators from the node

			state, err := c.ConsensusState()
			if err != nil {
				return fmt.Errorf("can't get consensus state from node: %w", err)
			}
			var roundState types.RoundStateSimple
			err = cdc.UnmarshalJSON(state.RoundState, &roundState)
			if err != nil {
				return fmt.Errorf("can't unmarshal response from node: %w", err)
			}
			var votes []roundVotes
			err = cdc.UnmarshalJSON(roundState.Votes, &votes)
			if err != nil {
				return fmt.Errorf("can't unmarshal response from node: %w", err)
			}

			// The votes state is returned in a string format that cannot be unmarshalled neatly into structs.
			// This parsed out the first 6 bytes of the addresses of signing validators.
			valAddrFingerprints := []string{}
			for _, v := range votes[0].Prevotes {
				if v == "nil-Vote" {
					continue
				}
				r := regexp.MustCompile(`^Vote{\d+:([0-9a-fA-F]+) \d+`)
				matches := r.FindStringSubmatch(v)
				if len(matches) < 2 { // FindStringSubmatch returns a slice of submatches where the first is the whole string match
					continue
				}
				valAddrFingerprints = append(valAddrFingerprints, matches[1])
			}

			// 4) return non-signing validator monikers

			output := displayData{}
			for _, val := range vals {
				found := false
				for _, fp := range valAddrFingerprints {
					if val.MatchesFingerPrint(fp) {
						found = true
						break
					}
				}
				if found {
					output.Online = append(output.Online, val)
				} else {
					output.Offline = append(output.Offline, val)
				}
			}

			bz, err := cdc.MarshalJSONIndent(output, "", "  ")
			if err != nil {
				panic(err)
			}
			fmt.Println(string(bz))

			return nil
		},
	}

	cmd.Flags().StringVar(&nodeAddress, "node", "http://localhost:26657", "rpc node address")

	return cmd
}

type roundVotes struct {
	Round              int      `json:"round"`
	Prevotes           []string `json:"prevotes"`
	PrevotesBitArray   string   `json:"prevotes_bit_array"`
	Precommits         []string `json:"precommits"`
	PrecommitsBitArray string   `json:"precommits_bit_array"`
}

type displayData struct {
	Offline validators `json:"offline" yaml:"offline"`
	Online  validators `json:"online" yaml:"online"`
}

type validators []validator

func (vs validators) TotalPower() int64 {
	var totalPower int64
	for _, v := range vs {
		totalPower += v.VotingPower
	}
	return totalPower
}

type validator struct {
	Moniker     string
	ConsAddress string
	VotingPower int64
}

func (v validator) MatchesFingerPrint(print string) bool {
	if len(v.ConsAddress) < fingerPrintLength {
		panic("address too short")
	}
	return print == v.ConsAddress[:fingerPrintLength]
}

func fetchValidators(cdc *codec.Codec, client *http.HTTP) (validators, error) {
	genAppState, err := fetchGenesisState(cdc, client)
	if err != nil {
		return nil, err
	}
	addrMonikers, err := extractAddressMonikersFromGenesis(cdc, genAppState)
	if err != nil {
		return nil, err
	}
	var startHeight int64 = 1 // endpoint requires height > 0
	validatorsResult, err := client.Validators(&startHeight, 0, 500)
	if err != nil {
		return nil, err
	}
	// ensure all validator were fetched
	if validatorsResult.Count != validatorsResult.Total {
		panic("did not fetch all validators")
	}
	var vals validators
	for _, v := range validatorsResult.Validators {
		vals = append(
			vals,
			validator{
				Moniker:     addrMonikers[v.Address.String()],
				ConsAddress: v.Address.String(),
				VotingPower: v.VotingPower,
			},
		)
	}
	// add any extra validators
	return vals, nil
}

func fetchGenesisState(cdc *codec.Codec, client *http.HTTP) (genutil.AppMap, error) {
	genResponse, err := client.Genesis()
	if err != nil {
		return nil, fmt.Errorf("can't get genesis file from node: %w", err)
	}
	var genAppState genutil.AppMap
	if err := cdc.UnmarshalJSON(genResponse.Genesis.AppState, &genAppState); err != nil {
		return nil, fmt.Errorf("can't unmarshal genesis app state: %w", err)
	}
	return genAppState, nil
}

func extractAddressMonikersFromGenesis(cdc *codec.Codec, genAppState genutil.AppMap) (map[string]string, error) {
	// Use genutil.GenTxs, unless it's empty then use staking.Validators
	validators, err := getAddressMonikersFromGenTxs(cdc, genAppState["genutil"])
	if err != nil {
		validators, err = getAddressMonikersFromStaking(cdc, genAppState["staking"])
		if err != nil {
			return nil, err
		}
	}
	return validators, nil
}
