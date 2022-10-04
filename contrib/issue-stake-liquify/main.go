package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
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

func main() {
	app.SetSDKConfig()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %s", err)
	}

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
	var allocations config.Allocations
	if err := json.Unmarshal(jsonContent, &allocations); err != nil {
		log.Fatalf("failed to unmarshal json: %s", err)
	}

	// no delegation distributions falls back to default - DefaultBaseAmount to all validators
	if len(allocations.Delegations) == 0 {
		log.Printf("no delegations specified. defaulting to equal distribution of %s ukava\n", config.DefaultBaseAmount)
		allocations.Delegations = []*config.DelegationDistribution{
			config.DefaultDistribution(),
		}
	}

	makeSigner := SignerFactory(cfg.ChainID, cfg.KavaGrpcEndpoint)

	delegationGas := int64(550_000)

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
	devWalletRequests := make(chan signing.MsgRequest)
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

	wg.Wait()

	for idx, delegation := range allocations.Delegations {
		wg.Add(1)
		// start the signer for the account
		signer := signerByIdx[idx]
		accRequests := make(chan signing.MsgRequest)
		accResponses, err := signer.Run(accRequests)
		if err != nil {
			log.Fatalf("failed to start signer for account %d: %s", idx, err)
		}

		// watch and report on
		go ReportOnResults(wg, accResponses, fmt.Sprintf("delegation, mint, & deposit from account %d", idx))

		// baseAmount was validated during Process()
		baseAmount, _ := sdk.NewIntFromString(delegation.BaseAmount)

		for i, validator := range allocations.Validators {
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

	fmt.Println("success")
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
		fmt.Printf(
			"successful broadcast of %s to %s (%s)\n",
			msg,
			res.Request.Data.(string),
			res.Result.TxHash,
		)
		wg.Done()
	}
}
