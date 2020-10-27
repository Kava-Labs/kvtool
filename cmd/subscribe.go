package cmd

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/rpc/client/http"
)

func SubscribeCmd(cdc *codec.Codec) *cobra.Command {
	var nodeAddress string
	var queryString string

	cmd := &cobra.Command{
		Use:   "subscribe",
		Short: "Listen for events on a node and print them out.",
		Long: `Subscribe to events produced by a node. By default listen to all new blocks.
Use --query "tm.event='Tx'" to listen for all transactions.`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {

			c, err := http.New(nodeAddress, "/websocket")
			if err != nil {
				return fmt.Errorf("can't connect to node: %w", err)
			}
			err = c.Start() // just call this undocumented function otherwise c.Subscribe panics with a cryptic error
			if err != nil {
				return fmt.Errorf("can't connect to node: %w", err)
			}

			ch, err := c.Subscribe(context.Background(), "subscriber", "tm.event='NewBlock'")
			if err != nil {
				return fmt.Errorf("can't subscribe to node: %w", err)
			}

			fmt.Println("listening...")
			for {
				event := <-ch

				bz, err := cdc.MarshalJSONIndent(event.Events, "", "  ")
				if err != nil {
					panic(err)
				}
				fmt.Println(string(bz))

				// TODO graceful shutdown
			}
		},
	}

	cmd.Flags().StringVar(&nodeAddress, "node", "http://localhost:26657", "rpc node address")
	cmd.Flags().StringVar(&queryString, "query", "tm.event='NewBlock'", "subscribe to events in the form {eventType}.{eventAttribute}={value}")

	return cmd
}
