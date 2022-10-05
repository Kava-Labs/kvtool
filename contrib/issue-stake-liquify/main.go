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
	delegationGas = int64(550_000)
)

func main() {
	app.SetSDKConfig()

	// load config from env
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %s", err)
	}

	// get allocations input
	allocations := config.ReadAllocationsInput()

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

	// create a signer for each account and determine total delegation
	// accounts are generated from the same mnemonic, using different address indices in the hd path
	signerByIdx := make(map[int]*signing.Signer, len(allocations.Delegations))
	totalByIdx := make(map[int]sdk.Int, len(allocations.Delegations))
	for addressIdx, delegation := range allocations.Delegations {
		// create signer for delegator
		signerByIdx[addressIdx] = makeSigner(cfg.DelegatorAccountsMnemonic, addressIdx)
		// process distributions
		total, err := delegation.Process(allocations.Validators)
		if err != nil {
			log.Fatalf("failed to process delegation for account %d: %s", addressIdx, err)
		}
		// include gas monies in issuance
		totalByIdx[addressIdx] = total.AddRaw(delegationGas)
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
	go ReportOnResults(wg, devWalletResponses, "issuing KAVA from dev wallet")

	// issue kava to all accounts
	for idx, acc := range signerByIdx {
		wg.Add(1)
		issueTokensMsg := issuancetypes.NewMsgIssueTokens(
			devWalletSigner.Address().String(),
			sdk.NewCoin("ukava", totalByIdx[idx]),
			acc.Address().String(),
		)

		devWalletRequests <- signing.MsgRequest{
			Msgs:      []sdk.Msg{issueTokensMsg},
			GasLimit:  200000,
			FeeAmount: sdk.NewCoins(sdk.NewCoin("ukava", sdk.NewInt(55000))),
			Memo:      "happy delegating!",
			Data:      acc.Address().String(),
		}
	}

	log.Println("waiting for all accounts to be issued funds.")
	wg.Wait()
	log.Println("all accounts funded with newly issued tokens.")

	for idx, delegation := range allocations.Delegations {
		// start the signer for the account
		signer := signerByIdx[idx]
		accRequests := make(chan signing.MsgRequest, 100)
		accResponses, err := signer.Run(accRequests)
		if err != nil {
			log.Fatalf("failed to start signer for account %d: %s", idx, err)
		}

		// watch and report on
		go ReportOnResults(wg, accResponses, fmt.Sprintf("delegation, mint, & deposit from account %d", idx))

		// baseAmount was validated during Process()
		baseAmount, _ := sdk.NewIntFromString(delegation.BaseAmount)

		for i, validator := range allocations.Validators {
			wg.Add(1)
			amount := baseAmount.MulRaw(delegation.Weights[i])
			stakingDelegation := stakingtypes.NewMsgDelegate(
				signer.Address(),
				validator.OperatorAddress,
				sdk.NewCoin("ukava", amount),
			)
			liquidMinting := liquidtypes.NewMsgMintDerivative(
				signer.Address(),
				validator.OperatorAddress,
				sdk.NewCoin("ukava", amount),
			)
			earnDeposit := earntypes.NewMsgDeposit(
				signer.Address().String(),
				sdk.NewCoin(
					liquidtypes.GetLiquidStakingTokenDenom("bkava", validator.OperatorAddress),
					amount,
				),
				earntypes.STRATEGY_TYPE_SAVINGS,
			)
			accRequests <- signing.MsgRequest{
				Msgs:      []sdk.Msg{stakingDelegation, &liquidMinting, earnDeposit},
				GasLimit:  uint64(delegationGas),
				FeeAmount: sdk.NewCoins(sdk.NewCoin("ukava", sdk.NewInt(10000))),
				Memo:      "staking my kava!",
				Data:      validator.OperatorAddress.String(),
			}
		}
	}

	wg.Wait()

	return nil
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
// Assumes the `Data` of the tx is a string of a relevant address
func ReportOnResults(
	wg *sync.WaitGroup,
	responses <-chan signing.MsgResponse,
	msg string,
) {
	for {
		res := <-responses
		if res.Err != nil {
			log.Fatalf("%s failed: %#v", msg, res)
		}
		log.Printf(
			"successful broadcast of %s to %s (%s)\n",
			msg,
			res.Request.Data.(string),
			res.Result.TxHash,
		)
		wg.Done()
	}
}
