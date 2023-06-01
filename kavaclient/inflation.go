package kavaclient

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	SecondsPerYear = 365 * 24 * 3600
)

type InflationResult struct {
	Start         int64
	End           int64
	InflationApr  *sdk.Dec
	InflationApy  *sdk.Dec
	SecondsPassed float64
}

func (ir InflationResult) String() string {
	blockDiff := ir.End - ir.Start
	avgBlockTime := ir.SecondsPassed / float64(blockDiff)
	return fmt.Sprintf(`realized & average inflation in APR & APY
start block: %d
end block: %d
total seconds passed: %fs
avg block time: %fs
inflation apr (%d block avg): %s
inflation apy (%d block avg): %s`,
		ir.Start, ir.End, ir.SecondsPassed, avgBlockTime, blockDiff, ir.InflationApr, blockDiff, ir.InflationApy)
}

// InflationOverBlocks calculates average inflation by taking the inflation over a block range and
// extrapolating it to a rate.
func (c *Client) InflationOverBlocks(start, end int64) (InflationResult, error) {
	result := InflationResult{
		Start: start,
		End:   end,
	}
	retries := 5

	// fetch start & end blocks for block time
	startBlock, err := c.Block(start, retries)
	if err != nil {
		return result, fmt.Errorf("failed to fetch start block (height=%d): %s", start, err)
	}
	endBlock, err := c.Block(end, retries)
	if err != nil {
		return result, fmt.Errorf("failed to fetch end block (height=%d): %s", end, err)
	}

	// get total seconds passed
	result.SecondsPassed = endBlock.Header.Time.Sub(startBlock.Header.Time).Seconds()

	// get total supply @ start & end
	supplyBefore, err := c.Supply(start, retries)
	if err != nil {
		return result, fmt.Errorf("failed to fetch total supply (start) at height %d: %s", start, err)
	}
	supplyAfter, err := c.Supply(end, retries)
	if err != nil {
		return result, fmt.Errorf("failed to fetch total supply (end) at height %d: %s", end, err)
	}

	// calculate inflation
	result.InflationApr = calculateInflationApr(supplyBefore.Amount, supplyAfter.Amount, result.SecondsPassed)
	result.InflationApy, err = calculateInflationApy(supplyBefore.Amount, supplyAfter.Amount, result.SecondsPassed)

	return result, err
}

func calculateInflationApy(beforeAmount, afterAmount sdkmath.Int, secondsPassed float64) (*sdk.Dec, error) {
	// extrapolate kava produced in last block to an APY inflation rate
	diff := sdk.NewDecFromInt(afterAmount.Sub(beforeAmount))
	avg := sdk.NewDecFromInt(afterAmount.Add(beforeAmount)).QuoInt64(2)
	// doing the math this way to reduce error from a per-second approach
	pow := sdk.NewDec(SecondsPerYear).Quo(sdk.MustNewDecFromStr(fmt.Sprintf("%f", secondsPassed))).RoundInt()
	inflation := diff.Quo(avg).Add(sdk.OneDec()).Power(pow.Uint64()).Sub(sdk.OneDec())
	return &inflation, nil
}

func calculateInflationApr(beforeAmount, afterAmount sdkmath.Int, secondsPassed float64) *sdk.Dec {
	diff := sdk.NewDecFromInt(afterAmount.Sub(beforeAmount))
	inflation := diff.Quo(sdk.MustNewDecFromStr(fmt.Sprintf("%f", secondsPassed))).QuoInt(beforeAmount).MulInt64(SecondsPerYear)
	return &inflation
}
