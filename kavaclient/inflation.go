package kavaclient

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	SecondsPerYear = 365 * 24 * 3600
)

type InflationResult struct {
	Start         int64
	End           int64
	Inflation     *sdk.Dec
	SecondsPassed float64
}

func (c *Client) InflationApyOverBlocks(start, end int64) (InflationResult, error) {
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
	result.Inflation, err = calculateInflationApy(supplyBefore.Amount, supplyAfter.Amount, result.SecondsPassed)
	if err != nil {
		return result, err
	}

	return result, nil
}

func calculateInflationApy(beforeAmount, afterAmount sdk.Int, secondsPassed float64) (*sdk.Dec, error) {
	// extrapolate kava produced in last block to an APY inflation rate
	diff := afterAmount.Sub(beforeAmount).ToDec()
	avg := afterAmount.Add(beforeAmount).ToDec().QuoInt64(2)
	// doing the math this way to reduce error from a per-second approach
	pow := sdk.NewDec(SecondsPerYear).Quo(sdk.MustNewDecFromStr(fmt.Sprintf("%f", secondsPassed))).RoundInt()
	inflation := diff.Quo(avg).Add(sdk.OneDec()).Power(pow.Uint64()).Sub(sdk.OneDec())
	return &inflation, nil
}
