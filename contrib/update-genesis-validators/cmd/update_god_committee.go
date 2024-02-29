package cmd

import (
	"fmt"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	committeetypes "github.com/kava-labs/kava/x/committee/types"
)

const (
	GOD_COMMITTEE_DESCRIPTION           = "Kava God Committee (testing only)"
	GOD_COMMITTEE_MEMBER_BALANCE_AMOUNT = int64(100000000000000)
	GOD_COMMITTEE_MEMBER_BALANCE_DENOM  = "ukava"
)

func InjectGodCommitteeMember(codec codec.Codec, appState genutiltypes.AppMap, memberAddress string) (genutiltypes.AppMap, error) {
	// unmarshal auth module state
	authState := authTypes.GenesisState{}
	if err := codec.UnmarshalJSON(appState[authTypes.ModuleName], &authState); err != nil {
		return nil, fmt.Errorf("failed to unmarshal app_state.auth: %s", err)
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
			if account.Address == memberAddress {
				memberAccountExist = true
				break
			}
		}
	}

	// if not create the members account
	if !memberAccountExist {
		fmt.Printf("member's %s account not found, creating it\n", memberAddress)

		account := &authTypes.BaseAccount{
			Address: memberAddress,
		}

		// convert back to any so we can store it in genesis state
		memberAccount := authTypes.GenesisAccount(account)

		anyAccounts, err := authTypes.PackAccounts([]authTypes.GenesisAccount{memberAccount})

		if err != nil {
			return nil, fmt.Errorf("failed to pack god committee: %s", err)
		}

		authState.Accounts = append(authState.Accounts, anyAccounts...)

		// re-marshal updated state
		appState[authTypes.ModuleName], err = codec.MarshalJSON(&authState)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal updated auth state: %s", err)
		}

		// and fund the account
		// unmarshal bank module state
		bankState := bankTypes.GenesisState{}
		if err := codec.UnmarshalJSON(appState[bankTypes.ModuleName], &bankState); err != nil {
			return nil, fmt.Errorf("failed to unmarshal app_state.bank: %s", err)
		}

		balance := bankTypes.Balance{
			Address: memberAddress,
			Coins: []sdk.Coin{
				sdk.Coin{
					Denom:  GOD_COMMITTEE_MEMBER_BALANCE_DENOM,
					Amount: math.NewInt(GOD_COMMITTEE_MEMBER_BALANCE_AMOUNT),
				},
			},
		}

		bankState.Balances = append(bankState.Balances, balance)

		// update the total supply with the funded amount
		var denomSupplyDefined bool
		for idx, denom := range bankState.Supply {
			if denom.Denom == GOD_COMMITTEE_MEMBER_BALANCE_DENOM {
				denomSupplyDefined = true

				updatedAmount := denom.Amount.Add(math.NewInt(GOD_COMMITTEE_MEMBER_BALANCE_AMOUNT))
				bankState.Supply[idx].Amount = updatedAmount

				fmt.Printf("updated total supply for %s from %d to %d\n", denom.Denom, denom.Amount.Int64(), updatedAmount.Int64())
				break
			}
		}

		if !denomSupplyDefined {
			supply := sdk.Coin{
				Denom:  GOD_COMMITTEE_MEMBER_BALANCE_DENOM,
				Amount: math.NewInt(GOD_COMMITTEE_MEMBER_BALANCE_AMOUNT),
			}

			bankState.Supply = append(bankState.Supply, supply)

			fmt.Printf("added %+v to supply", supply)
		}

		// re-marshal updated state
		appState[bankTypes.ModuleName], err = codec.MarshalJSON(&bankState)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal updated bank state: %s", err)
		}

	}

	// unmarshal gov module state
	committeeState := committeetypes.GenesisState{}
	if err := codec.UnmarshalJSON(appState[committeetypes.ModuleName], &committeeState); err != nil {
		return nil, fmt.Errorf("failed to unmarshal app_state.committee: %s", err)
	}

	// inject god committee
	nextCommitteeId := uint64(len(committeeState.Committees) + 1)
	godCommittee := committeetypes.MustNewMemberCommittee(
		nextCommitteeId,
		GOD_COMMITTEE_DESCRIPTION,
		[]sdk.AccAddress{sdk.MustAccAddressFromBech32(memberAddress)},
		[]committeetypes.Permission{&committeetypes.GodPermission{}},
		sdk.MustNewDecFromStr("0.667000000000000000"),
		604800*time.Second,
		committeetypes.TALLY_OPTION_FIRST_PAST_THE_POST,
	)

	// massage member committee to proto.Any for inclusion in genesis state
	genesisCommittee := committeetypes.Committee(godCommittee)
	if err := genesisCommittee.UnpackInterfaces(codec); err != nil {
		return nil, fmt.Errorf("failed to unpack committee interface: %s", err)
	}
	anyCommittee, err := committeetypes.PackCommittee(genesisCommittee)
	if err != nil {
		return nil, fmt.Errorf("failed to pack god committee: %s", err)
	}

	committeeState.Committees = append(committeeState.Committees, anyCommittee)

	// re-marshal updated state
	appState[committeetypes.ModuleName], err = codec.MarshalJSON(&committeeState)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal updated committee state: %s", err)
	}

	fmt.Printf("added god committee with member %s\n", ugvGodCommitteeMember)

	return appState, nil
}
