package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	EQUAL_DISTRIBUTION  = "equal"
	CUSTOM_DISTRIBUTION = "custom"

	DefaultBaseAmount = "1_000_000_000"
)

type Allocations struct {
	Validators  []Validator               `json:"validators"`
	Delegations []*DelegationDistribution `json:"delegations"`
}

type Validator struct {
	OperatorAddress sdk.ValAddress `json:"operator_address"`
}

type DelegationDistribution struct {
	Distribution string  `json:"distribution"`
	BaseAmount   string  `json:"base_amount"`
	Weights      []int64 `json:"weights"`
}

// Process sets the weights of equal distributions & returns the total delegation amount
func (d *DelegationDistribution) Process(validators []Validator) (sdk.Int, error) {
	var total sdk.Int
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
		if len(validators) != len(d.Weights) {
			err = fmt.Errorf("incorrect number of weights for validators")
		}
		for _, weight := range d.Weights {
			total = total.Add(baseAmount.MulRaw(weight))
		}

	default:
		err = fmt.Errorf("unknown distribution: %s", d.Distribution)
	}

	return total, err
}

func DefaultDistribution() *DelegationDistribution {
	return &DelegationDistribution{
		Distribution: EQUAL_DISTRIBUTION,
		BaseAmount:   DefaultBaseAmount,
	}
}

// ReadAllocationsInput reads in JSON from stdin for the delegation allocations desired
func ReadAllocationsInput() Allocations {
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

	// absence of distributions falls back to default - DefaultBaseAmount delegated to all validators
	if len(allocations.Delegations) == 0 {
		log.Printf("no delegations specified. defaulting to equal distribution of %s ukava\n", DefaultBaseAmount)
		allocations.Delegations = []*DelegationDistribution{
			DefaultDistribution(),
		}
	}

	return allocations
}
