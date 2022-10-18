package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
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

// flag options for the command
var (
	// chain id for the modified genesis file's network
	ugvChainID string
	// minimum %age of power held by the included validators
	ugvMinimumPowerPercent float64
	// path to directory containing all private validator keys to use for replacement
	ugvKeysDir string
	// the prefix of the private validator key files
	ugvKeyPrefix string
	// path & name of file to save updated genesis to
	ugvOutFile string
)

var updateGenesisValidatorsCmd = &cobra.Command{
	Use:   "update-genesis-validators",
	Short: "Updates the validators of a non-zero height genesis.json",
	Long: `Takes a non-zero height genesis.json and a directory of indexed priv_validator_keys and replaces the validators of the genesis file with ones corresponding to the keys.

By default, the new validators replace the old ones from highest power to lowest. Any other validators are left as is.`,
	Args: cobra.MinimumNArgs(1),
	RunE: updateGenesisValidators,
}

func Execute() {
	if err := updateGenesisValidatorsCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	updateGenesisValidatorsCmd.Flags().StringVar(
		&ugvChainID,
		"chain-id", "",
		"Chain id of modified output genesis file.",
	)
	updateGenesisValidatorsCmd.Flags().Float64Var(
		&ugvMinimumPowerPercent,
		"min-power", 0,
		`Optional, minimum percentage of total power given to replaced validators.
Note that any power adjustments will be reverted after the first block is committed.`,
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
	if ugvMinimumPowerPercent >= 1 || ugvMinimumPowerPercent < 0 {
		return fmt.Errorf("minimum power is a percent. out of range: 0 <= power < 1")
	}

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

	fmt.Printf("found %d validator keys\n", len(valKeys))

	// update chain id or warn
	if ugvChainID == "" || ugvChainID == doc.ChainID {
		fmt.Printf("%s%s\n",
			"WARNING: the output will have the same chain id.",
			" This can put the new chain at risk of replay attacks.",
		)
		fmt.Println("Consider changing the chain id with the --chain-id flag.")
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

	fmt.Printf("successfully updated validators. file saved to %s\n", ugvOutFile)

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
		fmt.Printf("WARN: no json files with prefix %s found in directory %s. continuing anyway.\n", ugvKeyPrefix, ugvKeysDir)
		return valKeys, nil
	}

	return valKeys, nil
}

func UpdateGenesisFileWithNewValidators(
	doc *tmtypes.GenesisDoc,
	valKeys []pvtypes.FilePVKey,
) error {
	// warn that we aren't using all val keys
	if len(doc.Validators) < len(valKeys) {
		fmt.Println("warning: more validator keys provided than original validators.")
		fmt.Printf(
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

	// preprocess and replace `.validators`
	var (
		// total initial power of replaced validators
		initialValPower = sdk.NewInt(0)
		// a map of original valcons address -> new valcons private key
		replacements = make(map[string]pvtypes.FilePVKey, numReplace)
	)
	for i := 0; i < numReplace; i++ {
		// build old -> new valcons mapping
		orig := sdk.ConsAddress(doc.Validators[i].Address)
		replacements[orig.String()] = valKeys[i]

		fmt.Printf("replacing \"%s\"\n  %s -> %s\n", doc.Validators[i].Name, orig, sdk.ConsAddress(valKeys[i].Address))

		// replace pub key in `.validators` array
		doc.Validators[i].PubKey = valKeys[i].PubKey
		doc.Validators[i].Address = valKeys[i].Address

		// add validator power to total
		initialValPower = initialValPower.AddRaw(doc.Validators[i].Power)
	}

	// update `genesis_time`
	// we need to do this to ensure there's no consensus error
	// see https://github.com/tendermint/tendermint/issues/8773
	doc.GenesisTime = time.Now()
	fmt.Println("updated genesis_time to now")

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

	// determine current total power percentage
	totalPowerDelta := calcPowerDelta(
		stakingState.LastTotalPower,
		initialValPower,
		ugvMinimumPowerPercent,
	)
	powerDelta := sdk.NewInt(0)
	if !totalPowerDelta.Equal(powerDelta) {
		powerDelta = totalPowerDelta.QuoRaw(int64(numReplace))
	}

	fmt.Printf("total power delta: %s; per validator: %s\n", totalPowerDelta.String(), powerDelta)

	// replace the `consensus_pubkey.value` of `.app_state.staking.validators`
	// flag the valoper addresses we'll update power of
	valoperAddresses := make(map[string]struct{}, numReplace)
	for i, val := range stakingState.Validators {
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

			// flag the valoper address for power adjustments
			valoperAddresses[val.OperatorAddress] = struct{}{}
		}
	}

	// update `last_validator_powers` to increase validator power
	for i, valPow := range stakingState.LastValidatorPowers {
		if _, replaced := valoperAddresses[valPow.Address]; replaced {
			fmt.Printf("updating power for %s: %d -> ", valPow.Address, valPow.Power)
			stakingState.LastValidatorPowers[i].Power += powerDelta.Int64()
			fmt.Println(stakingState.LastValidatorPowers[i].Power)
		}
	}

	// update `last_total_power` to new total
	fmt.Printf("increasing total power: %s -> ", stakingState.LastTotalPower.String())
	stakingState.LastTotalPower = stakingState.LastTotalPower.Add(
		powerDelta.MulRaw(int64(numReplace)),
	)
	fmt.Println(stakingState.LastTotalPower.String())

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
		fmt.Printf("updating previous proposer: %s -> ", distributionState.PreviousProposer)
		distributionState.PreviousProposer = sdk.ConsAddress(replacement.Address.Bytes()).String()
		fmt.Println(distributionState.PreviousProposer)
	}

	// remarshal updated state
	appState[distributiontypes.ModuleName], err = codec.MarshalJSON(&distributionState)
	if err != nil {
		return fmt.Errorf("failed to marshal updated distribution state: %s", err)
	}

	//----------------------
	// FINALIZE APP STATE
	//----------------------
	// update the power of `validators` (required staking state read first)
	// doc.validators are still sorted so its safe to iterate by number
	for i := 0; i < numReplace; i++ {
		doc.Validators[i].Power += powerDelta.Int64()
	}

	// finalize the app_state updates!
	doc.AppState, err = json.Marshal(appState)
	if err != nil {
		return fmt.Errorf("error marshalling app state updates: %s", err)
	}

	return nil
}

// calcPowerDelta calculates the necessary total power increase that, when given to the replaced
// validators, adjusts the total power such that the replaced validators control at least the
// desired percentage
func calcPowerDelta(
	initialTotalPower sdk.Int,
	initialValPower sdk.Int,
	desiredPercent float64,
) sdk.Int {
	iTotalPower := new(big.Float).SetInt(initialTotalPower.BigInt())
	iValPower := new(big.Float).SetInt(initialValPower.BigInt())
	initialPercent := new(big.Float).Quo(iValPower, iTotalPower)
	fmt.Printf("initial power = %s / %s = %s\n", initialValPower.String(), initialTotalPower.String(), initialPercent.String())

	percentAfter := big.NewFloat(desiredPercent)
	// if we already have enough power, no change is necessary
	if initialPercent.Cmp(percentAfter) >= 0 {
		return sdk.NewInt(0)
	}

	// a = (P + Δ) / (T + Δ) => Δ = (a*T - P) / (1 - a)
	// Δ - total change in power given to replaced validators
	// a - desired percentage of total power
	// P - initial power of replaced validators
	// T - initial total power of all validators
	num := new(big.Float).Sub(
		new(big.Float).Mul(percentAfter, iTotalPower),
		iValPower,
	)
	den := new(big.Float).Sub(big.NewFloat(1), percentAfter)
	delta := new(big.Float).Quo(num, den)

	// convert the delta to a big int
	roundedDelta := new(big.Int)
	delta.Int(roundedDelta)
	// add 1 to ensure any rounding is in our validators' favor
	return sdk.NewIntFromBigInt(roundedDelta).AddRaw(1)
}
