package kavaclient

import (
	"context"
	"strconv"

	"google.golang.org/grpc/metadata"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	sdk "github.com/cosmos/cosmos-sdk/types"
	grpctypes "github.com/cosmos/cosmos-sdk/types/grpc"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	kavagrpc "github.com/kava-labs/go-tools/grpc"
)

type Client struct {
	bankClient banktypes.QueryClient
	tmService  tmservice.ServiceClient
}

func NewClient(grpcUrl string) (*Client, error) {
	conn, err := kavagrpc.NewGrpcConnection(grpcUrl)
	if err != nil {
		return &Client{}, err
	}

	return &Client{
		bankClient: banktypes.NewQueryClient(conn),
		tmService:  tmservice.NewServiceClient(conn),
	}, nil
}

func (c Client) GetBalance(address string, denom string, maxRetries int) (*sdk.Coin, error) {
	res, err := c.bankClient.Balance(context.Background(), &banktypes.QueryBalanceRequest{
		Address: address,
		Denom:   denom,
	})
	if err != nil {
		if maxRetries != 0 {
			return c.GetBalance(address, denom, maxRetries-1)
		}
		return nil, err
	}
	return res.Balance, nil
}

func (c Client) Block(height int64, maxRetries int) (*tmservice.Block, error) {
	res, err := c.tmService.GetBlockByHeight(context.Background(), &tmservice.GetBlockByHeightRequest{
		Height: height,
	})
	if err != nil {
		if maxRetries != 0 {
			return c.Block(height, maxRetries-1)
		}
		return nil, err
	}
	return res.SdkBlock, nil
}

func (c Client) LatestBlock(maxRetries int) (*tmservice.Block, error) {
	res, err := c.tmService.GetLatestBlock(context.Background(), &tmservice.GetLatestBlockRequest{})
	if err != nil {
		if maxRetries != 0 {
			return c.LatestBlock(maxRetries - 1)
		}
		return nil, err
	}
	return res.SdkBlock, nil
}

func (c Client) Supply(height int64, maxRetries int) (sdk.Coin, error) {
	res, err := c.bankClient.SupplyOf(ctxAtHeight(height), &banktypes.QuerySupplyOfRequest{
		Denom: "ukava",
	})
	if err != nil {
		if maxRetries != 0 {
			return c.Supply(height, maxRetries-1)
		}
		return sdk.Coin{}, err
	}
	return res.Amount, nil
}

func ctxAtHeight(height int64) context.Context {
	heightStr := strconv.FormatInt(height, 10)
	return metadata.AppendToOutgoingContext(context.Background(), grpctypes.GRPCBlockHeightHeader, heightStr)
}
