package cmd

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/crypto/keys"
	"github.com/cosmos/cosmos-sdk/crypto/keys/hd"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/kava-labs/go-tools/signer"
	kava "github.com/kava-labs/kava/app"
	"github.com/kava-labs/kvtool/config"
	"github.com/kava-labs/kvtool/config/generate"

	rpchttpclient "github.com/tendermint/tendermint/rpc/client/http"
)

var (
	defaultNodeRPCURL     string = "http://localhost:26657"
	defaultRosettaNetwork        = "kava-localnet"
	defaultFee                   = authtypes.StdFee{
		Amount: sdk.Coins{sdk.Coin{Denom: "ukava", Amount: sdk.NewInt(15000)}},
		Gas:    300000,
	}
)

func TxCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "tx",
		Short: "Transaction commands for kvtool",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			return nil
		},
	}
	var rpcURL string
	rosettaCmd := &cobra.Command{
		Use:     "spam-rosetta",
		Short:   "Automatically generates signed transactions and broadcasts them to running kvtool instance",
		Example: "spam-rosetta --node http://localhost:26657",
		Args:    cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			// load mnemonics
			data, err := config.LoadKavaWallets(filepath.Join(filepath.Dir(generate.ConfigTemplatesDir), "common", "kava-addresses.json"))
			if err != nil {
				return err
			}

			// create http client
			http, err := rpchttpclient.New(rpcURL, "/websocket")
			if err != nil {
				return err
			}

			// ensure that node is alive and network is kava-localnet
			err = checkNetwork(http)
			if err != nil {
				return err
			}

			//
			// create codec for messages
			//
			cdc := kava.MakeCodec()

			//
			// client for broadcasting txs
			//
			broadcastClient := signer.NewRpcBroadcastClient(http, cdc)

			sendMsgRequests, signerMnemonic, err := buildSendMessages(data)
			if err != nil {
				return err
			}

			err = signAndBroadcastMessages(sendMsgRequests, broadcastClient, signerMnemonic)

			if err != nil {
				return err
			}

			return nil
		},
	}
	rosettaCmd.Flags().StringVar(&rpcURL, "node", defaultNodeRPCURL, "Tendermint RPC URL of kava node")
	rootCmd.AddCommand(rosettaCmd)
	return rootCmd
}

func createSignerFromMnemonic(mnemonic string, bc *signer.RpcBroadcastClient, channels uint64) (*signer.Signer, error) {
	// derive private key and signer
	params := *hd.NewFundraiserParams(0, 459, 0)
	hdPath := params.String()
	derivedPriv, err := keys.StdDeriveKey(mnemonic, "", hdPath, keys.Secp256k1)
	if err != nil {
		return nil, err
	}
	privKey, err := keys.StdPrivKeyGen(derivedPriv, keys.Secp256k1)
	if err != nil {
		return nil, err
	}

	return signer.NewSigner(bc, privKey, channels), nil
}

func checkNetwork(http *rpchttpclient.HTTP) error {
	result, err := http.Status()
	if err != nil {
		return err
	}

	network := result.NodeInfo.Network
	if network != defaultRosettaNetwork {
		return fmt.Errorf("Connected to wrong network: expected %s, got %s\n", defaultRosettaNetwork, network)
	}

	fmt.Printf("Connected to %s\n", network)
	return nil
}

func buildSendMessages(data config.NamedWallets) ([]signer.MsgRequest, string, error) {
	senderWallet, err := data.GetWalletByName("user_1")
	if err != nil {
		return []signer.MsgRequest{}, "", err
	}
	receiverWallet, err := data.GetWalletByName("user_2")
	if err != nil {
		return []signer.MsgRequest{}, "", err
	}
	msgs := []sdk.Msg{
		banktypes.NewMsgSend(mustKavaAccAddressFromBech32(senderWallet.Address), mustKavaAccAddressFromBech32(receiverWallet.Address), sdk.NewCoins(sdk.NewCoin("ukava", sdk.NewInt(100000000)))),
		banktypes.NewMsgSend(mustKavaAccAddressFromBech32(senderWallet.Address), mustKavaAccAddressFromBech32(receiverWallet.Address), sdk.NewCoins(sdk.NewCoin("ukava", sdk.NewInt(100000000)), sdk.NewCoin("hard", sdk.NewInt(100000000)))),
		banktypes.NewMsgSend(mustKavaAccAddressFromBech32(senderWallet.Address), mustKavaAccAddressFromBech32(receiverWallet.Address), sdk.NewCoins(sdk.NewCoin("ukava", sdk.NewInt(100000000)), sdk.NewCoin("hard", sdk.NewInt(100000000)), sdk.NewCoin("usdx", sdk.NewInt(100000000)))),
		banktypes.NewMsgSend(mustKavaAccAddressFromBech32(senderWallet.Address), mustKavaAccAddressFromBech32(receiverWallet.Address), sdk.NewCoins(sdk.NewCoin("ukava", sdk.NewInt(100000000)), sdk.NewCoin("hard", sdk.NewInt(100000000)), sdk.NewCoin("usdx", sdk.NewInt(100000000)), sdk.NewCoin("busd", sdk.NewInt(10000000000)))),
	}

	var requests []signer.MsgRequest
	for _, msg := range msgs {
		msgReq := signer.MsgRequest{
			Msgs: []sdk.Msg{msg},
			Fee:  defaultFee,
			Memo: "",
		}
		requests = append(requests, msgReq)
	}
	return requests, senderWallet.Mnemonic, nil
}

func signAndBroadcastMessages(msgReqs []signer.MsgRequest, bc *signer.RpcBroadcastClient, mnemonic string) error {

	// creates a new signer.Signer with the input mnemonic
	signerService, err := createSignerFromMnemonic(mnemonic, bc, uint64(len(msgReqs)))
	if err != nil {
		return err
	}

	// channels to communicate with signer
	requests := make(chan signer.MsgRequest, len(msgReqs))

	// signer starts it's own go routines and returns
	responses, err := signerService.Run(requests)

	if err != nil {
		return err
	}

	var requestGroup sync.WaitGroup
	go func() {
		for {
			// response is not returned until the msg is committed to a block
			response := <-responses
			requestGroup.Done()
			// error will be set if response is not Code 0 (success) or Code 19 (already in mempool)
			if response.Err != nil {
				fmt.Printf("response code: %d error %s\n", response.Result.Code, response.Err)
				continue
			}
			// code and result are from broadcast, not deliver tx
			// it is up to the caller/requester to check the deliver tx code and deal with failure
			fmt.Printf("response code: %d, hash %s\n", response.Result.Code, response.Result.Hash)
		}
	}()
	requestGroup.Add(len(msgReqs))
	for _, msgReq := range msgReqs {
		requests <- msgReq
	}
	requestGroup.Wait()
	return nil
}
