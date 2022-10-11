package main

import (
	"fmt"
	"log"
	"sync"

	"github.com/kava-labs/kvtool/contrib/issue-stake-liquify/config"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/kava-labs/go-tools/grpc"
	"github.com/kava-labs/go-tools/signing"

	"github.com/kava-labs/kava/app"
	earntypes "github.com/kava-labs/kava/x/earn/types"
	issuancetypes "github.com/kava-labs/kava/x/issuance/types"
	liquidtypes "github.com/kava-labs/kava/x/liquid/types"
)

const (
	devWalletBatchSize = 50
	gasPrice           = 500000
)

func main() {
	app.SetSDKConfig()

	// load config from env
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %s", err)
	}

	// get allocations input
	allocations := config.ReadAllocationsInput(cfg)

	err = ProcessDelegationAllocations(cfg, allocations)
	if err != nil {
		log.Fatalf("failed to process delegations: %s", err)
	}

	fmt.Println("success")
}

// ProcessDelegationAllocations performs the following actions:
// - fund each delegator account with the required amount of Kava (via dev-wallet issuing)
// - stake the kava by designated weights to validators
// - mint the bkava derivative token for all delegations
// - deposit the liquid bonded kava into the earn module
func ProcessDelegationAllocations(cfg config.Config, allocations config.Allocations) error {
	// create factory for generating account signers
	makeSigner := SignerFactory(cfg.ChainID, cfg.KavaGrpcEndpoint)

	numAccounts := allocations.GetNumAccounts()

	// create a signer for each account and determine total delegation
	// accounts are generated from the same mnemonic, using different address indices in the hd path
	signerByIdx := make(map[int]*signing.Signer, len(allocations.Delegations))
	totalByIdx := make(map[int]sdk.Int, len(allocations.Delegations))
	for addressIdx := 0; addressIdx < numAccounts; addressIdx++ {
		// create signer for delegator
		signerByIdx[addressIdx] = makeSigner(cfg.DelegatorAccountsMnemonic, addressIdx)
		// process distributions
		total, err := allocations.GetTotalForAccount(addressIdx)
		if err != nil {
			log.Fatalf("failed to process delegation for account %d: %s", addressIdx, err)
		}
		// total to issue the account, not including gas money!
		totalByIdx[addressIdx] = total
	}

	// make dev wallet signer to issue tokens to each address
	devWalletSigner := makeSigner(cfg.DevWalletMnemonic, 0)

	wg := &sync.WaitGroup{}
	devWalletRequests := make(chan signing.MsgRequest, 100)
	devWalletResponses, err := devWalletSigner.Run(devWalletRequests)
	if err != nil {
		log.Fatalf("failed to start signer for dev wallet: %s", err)
	}

	// watch for all responses from dev wallet txs
	// after being issued tokens, we want to perform the delegations from the newly-funded account
	go func() {
		for {
			res := <-devWalletResponses
			idxs := res.Request.Data.([]int)
			if res.Err != nil {
				fmt.Printf("dev wallet token issuance to account(s) %v failed: %#v", idxs, res)
				continue
			}

			log.Printf("successfully broadcast issuance to account(s) %v (%s)", idxs, res.Result.TxHash)
			for _, idx := range idxs {

				signer := signerByIdx[idx]
				wg.Add(1)

				if allocations.SpamDelegations != nil {
					go DelegateBySpamParams(wg, cfg, signer, idx, allocations.Validators, totalByIdx[idx])
				} else {
					go DelegateByWeightedDistribution(wg, cfg, signer, idx, allocations.Validators, allocations.Delegations[idx])
				}
			}

			wg.Done()
		}
	}()

	// issue kava to all accounts. response will manage further txs from funded account.
	msgs := make([]sdk.Msg, 0, devWalletBatchSize)
	idxs := make([]int, 0, devWalletBatchSize)
	for idx := 0; idx < numAccounts; idx++ {
		total := sdk.NewIntFromBigInt(totalByIdx[idx].BigInt()).AddRaw(gasPrice)
		issueTokensMsg := issuancetypes.NewMsgIssueTokens(
			devWalletSigner.Address().String(),
			sdk.NewCoin("ukava", total),
			signerByIdx[idx].Address().String(),
		)
		msgs = append(msgs, issueTokensMsg)
		idxs = append(idxs, idx)

		// send tx batch when size is reached, or when on last account
		if len(msgs) == devWalletBatchSize || idx == (numAccounts-1) {
			wg.Add(1)
			fmt.Printf("issuing tokens to account(s) %v\n", idxs)
			devWalletRequests <- signing.MsgRequest{
				Msgs:      msgs,
				GasLimit:  2000000,
				FeeAmount: sdk.NewCoins(sdk.NewCoin("ukava", sdk.NewInt(gasPrice))),
				Memo:      "happy delegating!",
				Data:      idxs,
			}
			msgs = make([]sdk.Msg, 0, devWalletBatchSize)
			idxs = make([]int, 0, devWalletBatchSize)
		}
	}

	wg.Wait()

	return nil
}

type Data struct {
	Address    string
	AddressIdx int
}

// SignerFactory returns a function of mnemonic & address index that creates a signer for that account
func SignerFactory(chainID, grpcEndpoint string) func(string, int) *signing.Signer {
	grpcConn, err := grpc.NewGrpcConnection(grpcEndpoint)
	if err != nil {
		log.Fatal(err)
	}

	encodingConfig := app.MakeEncodingConfig()
	authClient := authtypes.NewQueryClient(grpcConn)
	txClient := txtypes.NewServiceClient(grpcConn)

	return func(mnemonic string, addressIdx int) *signing.Signer {
		hdPath := hd.CreateHDPath(app.Bip44CoinType, 0, uint32(addressIdx))
		privKeyBytes, err := hd.Secp256k1.Derive()(mnemonic, "", hdPath.String())
		if err != nil {
			log.Fatalf("failed to generate mnemonic for account %d: %s", addressIdx, privKeyBytes)
		}
		privKey := &secp256k1.PrivKey{Key: privKeyBytes}

		return signing.NewSigner(
			chainID,
			encodingConfig,
			authClient,
			txClient,
			privKey,
			100,
		)
	}
}

// ReportOnResults pulls responses off the channel and reports on tx success.
func ReportOnResults(
	wg *sync.WaitGroup,
	responses <-chan signing.MsgResponse,
	msg string,
) {
	for {
		res := <-responses
		if res.Err != nil {
			fmt.Printf("ERROR! FAILED TX: %s failed: %#v", msg, res)
		} else {
			log.Printf(
				"successful broadcast of %s to %s (%s)\n",
				msg,
				res.Request.Data.(Data).Address,
				res.Result.TxHash,
			)
		}
		wg.Done()
	}
}

func DelegateByWeightedDistribution(
	wg *sync.WaitGroup,
	cfg config.Config,
	signer *signing.Signer,
	addressIdx int,
	validators []config.Validator,
	distribution *config.DelegationDistribution,
) {
	defer wg.Done()

	accWg := &sync.WaitGroup{}
	accRequests := make(chan signing.MsgRequest, 100)
	accResponses, err := signer.Run(accRequests)
	if err != nil {
		log.Fatalf("failed to start signer for account %d: %s", addressIdx, err)
	}

	// watch and report on responses
	go ReportOnResults(
		accWg, accResponses,
		fmt.Sprintf("delegation, mint, & deposit from account %d", addressIdx),
	)

	// baseAmount was validated during Process()
	baseAmount, _ := sdk.NewIntFromString(distribution.BaseAmount)

	for i, validator := range validators {
		// handle a smaller weights array than number of validators
		if len(distribution.Weights) < (i + 1) {
			log.Printf(
				"delegator %d has no weights for remaining validators (%d+), breaking distribution loop\n",
				addressIdx, i,
			)
			break
		}
		// skip sending 0 KAVA
		if distribution.Weights[i] == 0 {
			log.Printf("delegator %d has 0 weight for validator %d, skipping\n", addressIdx, i)
			continue
		}

		accWg.Add(1)
		amount := baseAmount.MulRaw(distribution.Weights[i])

		accRequests <- BuildDelegationRequest(cfg, amount, signer.Address(), validator.OperatorAddress, addressIdx)
	}

	accWg.Wait()
}

func DelegateBySpamParams(
	wg *sync.WaitGroup,
	cfg config.Config,
	signer *signing.Signer,
	addressIdx int,
	validators []config.Validator,
	amount sdk.Int,
) {
	defer wg.Done()

	accWg := &sync.WaitGroup{}
	accRequests := make(chan signing.MsgRequest, 1)
	accResponses, err := signer.Run(accRequests)
	if err != nil {
		log.Fatalf("failed to start signer for account %d: %s", addressIdx, err)
	}

	// watch and report on responses
	go ReportOnResults(
		accWg, accResponses,
		fmt.Sprintf("spam delegation of %s ukava from account %d", amount.String(), addressIdx),
	)

	accWg.Add(1)

	// choose validator by cycling through each one
	validator := validators[addressIdx%len(validators)]

	// get random amount to delegate
	accRequests <- BuildDelegationRequest(cfg, amount, signer.Address(), validator.OperatorAddress, addressIdx)

	accWg.Wait()
}

func BuildDelegationRequest(
	cfg config.Config,
	amount sdk.Int,
	signerAddress sdk.AccAddress,
	validatorAddress sdk.ValAddress,
	addressIdx int,
) signing.MsgRequest {
	msgs := make([]sdk.Msg, 0, 3)

	stakingDelegation := stakingtypes.NewMsgDelegate(
		signerAddress,
		validatorAddress,
		sdk.NewCoin("ukava", amount),
	)
	msgs = append(msgs, stakingDelegation)
	if !cfg.SkipLiquify {
		liquidMinting := liquidtypes.NewMsgMintDerivative(
			signerAddress,
			validatorAddress,
			sdk.NewCoin("ukava", amount),
		)
		msgs = append(msgs, &liquidMinting)
		earnDeposit := earntypes.NewMsgDeposit(
			signerAddress.String(),
			sdk.NewCoin(
				liquidtypes.GetLiquidStakingTokenDenom("bkava", validatorAddress),
				amount,
			),
			earntypes.STRATEGY_TYPE_SAVINGS,
		)
		msgs = append(msgs, earnDeposit)
	}

	return signing.MsgRequest{
		Msgs:      msgs,
		GasLimit:  uint64(cfg.DelegationGas),
		FeeAmount: sdk.NewCoins(sdk.NewCoin("ukava", sdk.NewInt(gasPrice))),
		Memo:      "staking my kava!",
		Data: Data{
			Address:    validatorAddress.String(),
			AddressIdx: addressIdx,
		},
	}
}
