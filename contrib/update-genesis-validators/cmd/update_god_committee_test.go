package cmd

import (
	"encoding/json"
	"fmt"
	"testing"

	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/types"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	committeetypes "github.com/kava-labs/kava/x/committee/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kava-labs/kava/app"
)

var (
	TEST_GENESIS_FILEPATH             = "./test/test_genesis.json"
	TEST_GOD_COMMITTEE_MEMBER_ADDRESS = "kava1fy5zeuutmxzwcx5hncu5q83ug3zcqmxcpwrjsn"
	genesisState                      *tmtypes.GenesisDoc
)

func init() {
	// Configure cosmos-sdk with kava overrides
	app.SetSDKConfig()

	// load genesis state once and re-use across tests
	var err error
	genesisState, err = loadGenesisFile(TEST_GENESIS_FILEPATH)
	if err != nil {
		panic(fmt.Sprintf("error %s loading genesis state from %s", err, TEST_GENESIS_FILEPATH))
	}

}

func TestAddsGodCommitteeWithSpecifiedFundedMemberToAppState(t *testing.T) {
	// SETUP

	// unmarshal app state so we can unmarshal state of relevant modules
	encodingConfig := app.MakeEncodingConfig()
	codec := encodingConfig.Marshaler

	var appState genutiltypes.AppMap

	err := json.Unmarshal(genesisState.AppState, &appState)

	if err != nil {
		t.Fatalf("error unmarshalling app state: %s", err)
	}

	committeeState := committeetypes.GenesisState{}
	if err := codec.UnmarshalJSON(appState[committeetypes.ModuleName], &committeeState); err != nil {
		t.Fatalf("failed to unmarshal app_state.committee: %s", err)
	}

	existingNumberOfCommittees := len(committeeState.Committees)

	// get current supply for denom god committee member will be funded with
	bankState := bankTypes.GenesisState{}
	if err := codec.UnmarshalJSON(appState[bankTypes.ModuleName], &bankState); err != nil {
		t.Fatalf("failed to unmarshal app_state.bank: %s", err)
	}

	var initialGodCommitteeMemberFundedDenomTotalSupply int64

	for _, denom := range bankState.Supply {
		if denom.Denom == GOD_COMMITTEE_MEMBER_BALANCE_DENOM {
			initialGodCommitteeMemberFundedDenomTotalSupply = denom.Amount.Int64()
			break
		}
	}

	// EXECUTE
	updatedAppState, err := InjectGodCommitteeMember(codec, appState, TEST_GOD_COMMITTEE_MEMBER_ADDRESS)
	if err != nil {
		t.Fatal(err)
	}

	// ASSERT

	// genesis state for committee module should still be decode-able
	committeeState = committeetypes.GenesisState{}
	if err := codec.UnmarshalJSON(updatedAppState[committeetypes.ModuleName], &committeeState); err != nil {
		t.Fatalf("failed to unmarshal app_state.committee: %s", err)
	}

	// there should be 1 more committee then existed before
	assert.Equal(t, existingNumberOfCommittees+1, len(committeeState.Committees), "expected number of committees to be 1 more then started with")

	// one of the committee's should be a god committee (permission check)
	var godCommittee *committeetypes.MemberCommittee
	var godCommitteeExists bool

	for idx, rawCommittee := range committeeState.Committees {
		committeeValue := rawCommittee.GetCachedValue()

		var committee *committeetypes.MemberCommittee

		switch committeeValue.(type) {
		case *committeetypes.MemberCommittee:
			committee = committeeValue.(*committeetypes.MemberCommittee)
			if committee.Description == GOD_COMMITTEE_DESCRIPTION {
				godCommitteeExists = true
				godCommittee = committeeState.Committees[idx].GetCachedValue().(*committeetypes.MemberCommittee)
				break
			}
		}

	}

	assert.True(t, godCommitteeExists, "expected god committee to be created")

	// with the specified user as a member
	assert.Contains(t, godCommittee.Members, types.MustAccAddressFromBech32(TEST_GOD_COMMITTEE_MEMBER_ADDRESS))

	// the commmittee member should have an account
	// unmarshal auth module state
	authState := authTypes.GenesisState{}
	if err := codec.UnmarshalJSON(appState[authTypes.ModuleName], &authState); err != nil {
		t.Fatal(fmt.Errorf("failed to unmarshal app_state.auth: %s", err))
	}

	// check to see if member's account exists already
	var memberAccountExist bool
	// assumes it is a simple base account
	for _, rawAccount := range authState.Accounts {
		accountValue := rawAccount.GetCachedValue()

		var account *authTypes.BaseAccount

		switch accountValue.(type) {
		case *authTypes.BaseAccount:
			account = accountValue.(*authTypes.BaseAccount)
			if account.Address == TEST_GOD_COMMITTEE_MEMBER_ADDRESS {
				memberAccountExist = true
				break
			}
		}
	}
	assert.True(t, memberAccountExist, "expected member's account to exist or be created")

	// with non-zero balance
	bankState = bankTypes.GenesisState{}
	if err := codec.UnmarshalJSON(appState[bankTypes.ModuleName], &bankState); err != nil {
		t.Fatal(fmt.Errorf("failed to unmarshal app_state.bank: %s", err))
	}

	var memberHasBalance bool
	var memberUkavabalance int64

	for _, balance := range bankState.Balances {
		if balance.Address == TEST_GOD_COMMITTEE_MEMBER_ADDRESS {
			memberHasBalance = true

			require.True(t, func() bool {
				var nonZeroBalance bool
				for _, coin := range balance.Coins {
					if coin.Amount.Int64() > 0 {
						nonZeroBalance = true
						if coin.Denom != GOD_COMMITTEE_MEMBER_BALANCE_DENOM {
							t.Fatalf("expected god committee member to only have %s balance, found %+v", GOD_COMMITTEE_MEMBER_BALANCE_DENOM, coin)
						}
						memberUkavabalance = coin.Amount.Int64()
						break
					}
				}
				return nonZeroBalance
			}(), "expected member's balance to have non zero funds")
		}
	}

	assert.True(t, memberHasBalance)
	assert.Equal(t, GOD_COMMITTEE_MEMBER_BALANCE_AMOUNT, memberUkavabalance)

	// genesis supply updated with amount funded to god committee member
	var finalGodCommitteeMemberFundedDenomTotalSupply int64
	for _, denom := range bankState.Supply {
		if denom.Denom == GOD_COMMITTEE_MEMBER_BALANCE_DENOM {
			finalGodCommitteeMemberFundedDenomTotalSupply = denom.Amount.Int64()
			break
		}
	}
	assert.True(t, func() bool {
		return finalGodCommitteeMemberFundedDenomTotalSupply-initialGodCommitteeMemberFundedDenomTotalSupply == GOD_COMMITTEE_MEMBER_BALANCE_AMOUNT
	}(), fmt.Sprintf("expected total supply for %s to be increased by %d, initial amount %d final amount %d", GOD_COMMITTEE_MEMBER_BALANCE_DENOM, GOD_COMMITTEE_MEMBER_BALANCE_AMOUNT, initialGodCommitteeMemberFundedDenomTotalSupply, finalGodCommitteeMemberFundedDenomTotalSupply))
}
