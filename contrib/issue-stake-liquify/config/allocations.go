package config

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	EQUAL_DISTRIBUTION  = "equal"
	CUSTOM_DISTRIBUTION = "custom"
)

type Allocations struct {
	Validators      []Validator               `json:"validators"`
	Delegations     []*DelegationDistribution `json:"delegations"`
	SpamDelegations *SpamParams               `json:"spam_delegations"`
}

type Validator struct {
	OperatorAddress sdk.ValAddress `json:"operator_address"`
}

// DelegationDistribution wraps data for an explicit delegation distribution from one account
// to a set of validators
type DelegationDistribution struct {
	Distribution string  `json:"distribution"`
	BaseAmount   string  `json:"base_amount"`
	Weights      []int64 `json:"weights"`
}

// SpamParams wraps data for generating many delegations from many accounts with a random value
// between min and max amounts
type SpamParams struct {
	Count     int    `json:"count"`      // number of delegations (and accounts)
	MinAmount string `json:"min_amount"` // min ukava amount
	MaxAmount string `json:"max_amount"` // max ukava amount

	min *big.Int
	max *big.Int
}

// GetNumAccounts returns the number of accounts delegations will be sent from
func (a Allocations) GetNumAccounts() int {
	if a.SpamDelegations != nil {
		return a.SpamDelegations.Count
	}
	return len(a.Delegations)
}

// GetTotalForAccount returns the total amount to be delegated by account with address index `idx`
func (a Allocations) GetTotalForAccount(idx int) (sdk.Int, error) {
	// explicit delegations are calculated by base_amount * weights
	if a.SpamDelegations == nil {
		return a.Delegations[idx].Process(a.Validators)
	}

	// spam delegations chooses a random amount between min & max
	// it only sends to one account so this is the total.
	return randomAmount(a.SpamDelegations.min, a.SpamDelegations.max)
}

func randomAmount(min *big.Int, max *big.Int) (sdk.Int, error) {
	// get the diff by subtracting min from max ((-1 * min) + max)
	diff := big.NewInt(0).Mul(big.NewInt(-1), min)
	diff = diff.Add(diff, max)
	// amount has range [0, max-min)
	amount, err := rand.Int(rand.Reader, diff)
	if err != nil {
		return sdk.Int{}, err
	}
	// adjust range back to [min, max)
	amount = amount.Add(amount, min)
	return sdk.NewIntFromBigInt(amount), nil
}

// Process sets the weights of equal distributions & returns the total delegation amount
func (d *DelegationDistribution) Process(validators []Validator) (sdk.Int, error) {
	total := sdk.NewInt(0)
	var err error
	baseAmount, ok := sdk.NewIntFromString(d.BaseAmount)
	if !ok {
		return total, fmt.Errorf("unable to parse base_amount: %s", d.BaseAmount)
	}

	switch d.Distribution {
	case EQUAL_DISTRIBUTION:
		for range validators {
			d.Weights = append(d.Weights, 1)
		}
		total = baseAmount.MulRaw(int64(len(validators)))
	case CUSTOM_DISTRIBUTION:
		if len(d.Weights) == 0 {
			err = fmt.Errorf("must include non-empty weights for custom distribution")
		}
		if len(d.Weights) > len(validators) {
			fmt.Printf("warning: more weights than validators provided. ignoring extra weights.")
		}
		for _, weight := range d.Weights {
			total = total.Add(baseAmount.MulRaw(weight))
		}

	default:
		err = fmt.Errorf("unknown distribution: %s", d.Distribution)
	}

	return total, err
}

func DefaultDistribution(amount string) *DelegationDistribution {
	return &DelegationDistribution{
		Distribution: EQUAL_DISTRIBUTION,
		BaseAmount:   amount,
	}
}

// ReadAllocationsInput reads in JSON from stdin for the delegation allocations desired
func ReadAllocationsInput(cfg Config) Allocations {
	// read stdin for json of validator allocation info
	var jsonContent []byte
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		jsonContent = append(jsonContent, scanner.Bytes()...)
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("error reading standard input: %s", err)
	}

	// parse the allocations
	var allocations Allocations
	if err := json.Unmarshal(jsonContent, &allocations); err != nil {
		log.Fatalf("failed to unmarshal json: %s", err)
	}

	// only valid to do either spam or explicit delegations
	if allocations.SpamDelegations != nil && len(allocations.Delegations) > 0 {
		log.Fatal("only one of `delegations` or `spam_delegations` may be defined")
	}

	// parse & validate min/max amounts
	if allocations.SpamDelegations != nil {
		min, ok := sdk.NewIntFromString(allocations.SpamDelegations.MinAmount)
		if !ok {
			log.Fatal("spam_delegations.min_amount can't be parsed to big int")
		}
		allocations.SpamDelegations.min = min.BigInt()
		max, ok := sdk.NewIntFromString(allocations.SpamDelegations.MaxAmount)
		if !ok {
			log.Fatal("spam_delegations.max_amount can't be parsed to big int")
		}
		allocations.SpamDelegations.max = max.BigInt()
	}

	// absence of distributions falls back to default - DefaultBaseAmount delegated to all validators
	if allocations.SpamDelegations == nil && len(allocations.Delegations) == 0 {
		log.Printf("no delegations specified. defaulting to equal distribution of %s ukava\n", cfg.DefaultBaseAmount)
		allocations.Delegations = []*DelegationDistribution{
			DefaultDistribution(cfg.DefaultBaseAmount),
		}
	}

	return allocations
}
