package testnet

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var runDetachedFlag bool

func UpCmd() *cobra.Command {
	upCmd := &cobra.Command{
		Use:   "up",
		Short: "A convenience command that runs `docker compose up` on the generated config.",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			cmd := []string{"docker", "compose", "--file", generatedPath("docker-compose.yaml"), "up"}
			if runDetachedFlag {
				cmd = append(cmd, "-d")
			}
			fmt.Println("running:", strings.Join(cmd, " "))
			if err := replaceCurrentProcess(cmd...); err != nil {
				return fmt.Errorf("could not run command: %v", err)
			}
			return nil
		},
	}

	upCmd.Flags().BoolVarP(&runDetachedFlag, "detach", "d", false, "Detached mode: Run containers in the background.")

	return upCmd
}
