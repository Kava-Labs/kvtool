package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"sort"
	"time"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/kava-labs/kava/app"
	"github.com/spf13/cobra"
	tmjson "github.com/tendermint/tendermint/libs/json"
	pvtypes "github.com/tendermint/tendermint/privval"
	tmtypes "github.com/tendermint/tendermint/types"
)

var ugvKeysDir string
var ugvKeyPrefix string
var ugvOutFile string
var ugvChainID string

var updateGenesisValidatorsCmd = &cobra.Command{
	Use:   "update-genesis-validators",
	Short: "Updates the validators of a non-zero height genesis.json",
	Long: `Takes a non-zero height genesis.json and a directory of indexed priv_validator_keys
and replaces the validators of the genesis file with ones corresponding to the keys`,
	Args: cobra.MinimumNArgs(1),
	RunE: updateGenesisValidators,
}

func Execute() {
	if err := updateGenesisValidatorsCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	updateGenesisValidatorsCmd.Flags().StringVar(
		&ugvChainID,
		"chain-id", "updated-genesis.json",
		"Chain id of modified output genesis file.",
	)
	updateGenesisValidatorsCmd.Flags().StringVarP(
		&ugvKeysDir,
		"keys-dir", "d", "keys/",
		"Path to directory containing new validator keys.",
	)
	updateGenesisValidatorsCmd.Flags().StringVarP(
		&ugvKeyPrefix,
		"key-prefix", "p", "priv_validator_key_",
		`The validator private key file prefix. Keys should be named '${prefix}${idx}.json'.
idx is 0-indexed. Example: priv_validator_key_0.json, priv_validator_key_1.json
Validators are replaced by in order of highest voting power.`,
	)
	updateGenesisValidatorsCmd.Flags().StringVarP(
		&ugvOutFile,
		"out", "o", "updated-genesis.json",
		"Name of output json file for updated genesis with replaced validators.",
	)
}

func updateGenesisValidators(cmd *cobra.Command, args []string) error {
	filename := args[0]
	doc, err := loadGenesisFile(filename)
	if err != nil {
		return err
	}

	// ensure it's not 0-height
	if doc.InitialHeight <= 1 {
		return fmt.Errorf("expected genesis file to be for height > 1. found %d", doc.InitialHeight)
	}

	// get all priv_validator_keys
	valKeys, err := loadValidatorKeys(ugvKeysDir, ugvKeyPrefix)
	if err != nil {
		return fmt.Errorf("failed to load validator keys: %s", err)
	}

	log.Printf("found %d validator keys\n", len(valKeys))

	// update chain id or warn
	if ugvChainID == "" || ugvChainID == doc.ChainID {
		log.Printf("%s%s\n",
			"WARNING: the output will have the same chain id.",
			" This can put the new chain at risk of replay attacks.",
		)
		log.Println("Consider changing the chain id with the --chain-id flag.")
	} else {
		doc.ChainID = ugvChainID
	}

	// perform necessary updates
	if err = UpdateGenesisFileWithNewValidators(doc, valKeys); err != nil {
		return err
	}

	// save to output
	if err = doc.SaveAs(ugvOutFile); err != nil {
		return fmt.Errorf("failed to save output file: %s", err)
	}

	log.Printf("successfully updated validators. file saved to %s\n", ugvOutFile)

	return nil
}

func loadGenesisFile(filename string) (*tmtypes.GenesisDoc, error) {
	doc, err := tmtypes.GenesisDocFromFile(filename)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

func loadValidatorKeys(dir string, prefix string) ([]pvtypes.FilePVKey, error) {
	valKeys := []pvtypes.FilePVKey{}
	for idx := 0; true; idx++ {
		filename := fmt.Sprintf("%s%d.json", prefix, idx)
		keyJSONBytes, err := ioutil.ReadFile(filepath.Join(dir, filename))
		if err != nil {
			// file doesn't exist or is malformed
			break
		}
		pvKey := pvtypes.FilePVKey{}
		// NOTE: must use tendermint's json!
		err = tmjson.Unmarshal(keyJSONBytes, &pvKey)
		if err != nil {
			return valKeys, fmt.Errorf("error parsing %s to json: %s", filename, err)
		}
		valKeys = append(valKeys, pvKey)
	}

	if len(valKeys) == 0 {
		return valKeys, fmt.Errorf("no json files with prefix %s found in directory %s", prefix, dir)
	}

	return valKeys, nil
}

func UpdateGenesisFileWithNewValidators(
	doc *tmtypes.GenesisDoc,
	valKeys []pvtypes.FilePVKey,
) error {
	// warn that we aren't using all val keys
	if len(doc.Validators) < len(valKeys) {
		log.Println("warning: more validator keys provided than original validators.")
		log.Printf(
			"  continuing with replacement but will only inject %d new validators\n",
			len(doc.Validators),
		)
	}

	// determine number of validators we'll replace
	numReplace := len(doc.Validators)
	if len(valKeys) < numReplace {
		numReplace = len(valKeys)
	}

	// sort the validators from highest power to lowest
	sort.Slice(doc.Validators, func(i, j int) bool {
		return doc.Validators[i].Power > doc.Validators[j].Power
	})

	// build map of old valcons -> new valcons
	replacements := make(map[string]pvtypes.FilePVKey, numReplace)
	for i := 0; i < numReplace; i++ {
		orig := sdk.ConsAddress(doc.Validators[i].Address)
		replacements[orig.String()] = valKeys[i]

		// replace pub key in `.validators` array
		doc.Validators[i].PubKey = valKeys[i].PubKey
		doc.Validators[i].Address = valKeys[i].Address
	}

	// update `genesis_time`
	// we need to do this to ensure there's no consensus error
	// see https://github.com/tendermint/tendermint/issues/8773
	doc.GenesisTime = time.Now()
	log.Println("updated genesis_time to now")

	// unmarshal app state so we can unmarshal state of relevant modules
	encodingConfig := app.MakeEncodingConfig()
	codec := encodingConfig.Marshaler
	var appState genutiltypes.AppMap
	err := json.Unmarshal(doc.AppState, &appState)
	if err != nil {
		return fmt.Errorf("error unmarshalling app state: %s", err)
	}

	//----------------------
	// STAKING STATE
	//----------------------
	// unmarshal staking module state
	stakingState := stakingtypes.GenesisState{}
	if err = codec.UnmarshalJSON(appState[stakingtypes.ModuleName], &stakingState); err != nil {
		return fmt.Errorf("failed to unmarshal app_state.staking: %s", err)
	}

	// replace the `consensus_pubkey.value` of `.app_state.staking.validators`
	for i, val := range stakingState.Validators {
		// orig := sdk.ValAddress(val.ConsensusPubkey.Value)
		var orig cryptotypes.PubKey
		err = codec.UnpackAny(val.ConsensusPubkey, &orig)
		if err != nil {
			return err
		}
		origAddress := sdk.ConsAddress(orig.Address()).String()
		replacement, found := replacements[origAddress]
		if found {
			// convert tendermint pub key -> cosmos pub key -> protobuf Any
			pk := &ed25519.PubKey{Key: replacement.PubKey.Bytes()}
			pubKey, err := codectypes.NewAnyWithValue(pk)
			if err != nil {
				return err
			}
			// set new pubkey!
			stakingState.Validators[i].ConsensusPubkey = pubKey
		}
	}

	// remarshal updated state
	appState[stakingtypes.ModuleName], err = codec.MarshalJSON(&stakingState)
	if err != nil {
		return fmt.Errorf("failed to marshal updated staking state: %s", err)
	}

	//----------------------
	// SLASHING STATE
	//----------------------

	// unmarshal slashing state
	slashingState := slashingtypes.GenesisState{}
	if err = codec.UnmarshalJSON(appState[slashingtypes.ModuleName], &slashingState); err != nil {
		return fmt.Errorf("failed to unmarshal app_state.slashing: %s", err)
	}

	// update valcons addresses in slashing state:
	// the `missed_blocks` address for validators
	for i, mb := range slashingState.MissedBlocks {
		replacement, found := replacements[mb.Address]
		if found {
			slashingState.MissedBlocks[i].Address = sdk.ConsAddress(replacement.Address.Bytes()).String()
		}
	}

	// both addresses in `signing_infos`
	for i, si := range slashingState.SigningInfos {
		replacement, found := replacements[si.Address]
		if found {
			newValAddr := sdk.ConsAddress(replacement.Address.Bytes()).String()
			slashingState.SigningInfos[i].Address = newValAddr
			slashingState.SigningInfos[i].ValidatorSigningInfo.Address = newValAddr
		}
	}

	// remarshal updated state
	appState[slashingtypes.ModuleName], err = codec.MarshalJSON(&slashingState)
	if err != nil {
		return fmt.Errorf("failed to marshal updated staking state: %s", err)
	}

	//----------------------
	// DISTRIBUTION STATE
	//----------------------

	// unmarshal distribution module state
	distributionState := distributiontypes.GenesisState{}
	if err = codec.UnmarshalJSON(appState[distributiontypes.ModuleName], &distributionState); err != nil {
		return fmt.Errorf("failed to unmarshal app_state.distribution: %s", err)
	}

	// update valcons address in previous `.app_state.distribution.previous_proposer`
	replacement, found := replacements[distributionState.PreviousProposer]
	if found {
		distributionState.PreviousProposer = sdk.ConsAddress(replacement.Address.Bytes()).String()
	}

	// remarshal updated state
	appState[distributiontypes.ModuleName], err = codec.MarshalJSON(&distributionState)
	if err != nil {
		return fmt.Errorf("failed to marshal updated distribution state: %s", err)
	}

	//----------------------
	// FINALIZE APP STATE
	//----------------------
	// finalize the app_state updates!
	doc.AppState, err = json.Marshal(appState)
	if err != nil {
		return fmt.Errorf("error marshalling app state updates: %s", err)
	}

	return nil
}
