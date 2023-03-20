package testnet

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/kava-labs/kvtool/config/generate"
	"github.com/spf13/cobra"
)

func BootstrapCmd() *cobra.Command {
	bootstrapCmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "A convenience command that creates a kava testnet with the input configTemplate (defaults to master)",
		Example: `Run kava node with particular template:
$ kvtool testnet bootstrap --kava.configTemplate v0.12

Run kava & another chain with open IBC channel & relayer:
$ kvtool testnet bootstrap --ibc

Run kava & an ethereum node:
$ kvtool testnet bootstrap --geth

The master template supports dynamic override of the Kava node's container image:
$ KAVA_TAG=v0.21 kvtool testnet bootstrap
`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			cmd := exec.Command("docker-compose", "--file", generatedPath("docker-compose.yaml"), "down")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			// check that dockerfile exists before calling 'docker-compose down down'
			if _, err := os.Stat(generatedPath("docker-compose.yaml")); err == nil {
				if err2 := cmd.Run(); err2 != nil {
					return err2
				}
			}
			if err := os.RemoveAll(generatedConfigDir); err != nil {
				return fmt.Errorf("could not clear old generated config: %v", err)
			}
			if err := generate.GenerateKavaConfig(kavaConfigTemplate, generatedConfigDir); err != nil {
				return err
			}
			if ibcFlag {
				if err := generate.GenerateIbcChainConfig(generatedConfigDir); err != nil {
					return err
				}
				if err := generate.GenerateHermesRelayerConfig(generatedConfigDir); err != nil {
					return err
				}
				if err := generate.GenerateGoRelayerConfig(generatedConfigDir); err != nil {
					return err
				}
			}
			if gethFlag {
				if err := generate.GenerateGethConfig(generatedConfigDir); err != nil {
					return err
				}
			}

			cmd = exec.Command("docker-compose", "--file", generatedPath("docker-compose.yaml"), "pull")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Println(err.Error())
			}

			upCmd := exec.Command("docker-compose", "--file", generatedPath("docker-compose.yaml"), "up", "-d")
			upCmd.Stdout = os.Stdout
			upCmd.Stderr = os.Stderr
			if err := upCmd.Run(); err != nil {
				fmt.Println(err.Error())
			}
			if ibcFlag {
				fmt.Printf("Starting ibc connection between chains...\n")
				time.Sleep(time.Second * 7)
				setupIbcPathCmd := exec.Command("docker", "run", "-v", fmt.Sprintf("%s:%s", generatedPath("relayer"), "/home/relayer/.relayer"), "--network", "generated_default", relayerImageTag, "rly", "paths", "new", kavaChainId, ibcChainId, "transfer")
				setupIbcPathCmd.Stdout = os.Stdout
				setupIbcPathCmd.Stderr = os.Stderr
				if err := setupIbcPathCmd.Run(); err != nil {
					fmt.Println(err.Error())
					return fmt.Errorf("[hermes] failed to setup ibc path")
				}
				openConnectionCmd := exec.Command("docker", "run", "-v", fmt.Sprintf("%s:%s", generatedPath("relayer"), "/home/relayer/.relayer"), "--network", "generated_default", relayerImageTag, "rly", "transact", "link", "transfer")
				openConnectionCmd.Stdout = os.Stdout
				openConnectionCmd.Stderr = os.Stderr
				if err := openConnectionCmd.Run(); err != nil {
					fmt.Println(err.Error())
					return fmt.Errorf("[relayer] failed to open ibc connection")
				}
				fmt.Printf("IBC connection complete, starting relayer process...\n")
				time.Sleep(time.Second * 5)
				err := generate.AddHermesRelayerToNetwork(generatedConfigDir)
				if err != nil {
					return err
				}
				restartCmd := exec.Command("docker-compose", "--file", generatedPath("docker-compose.yaml"), "up", "-d", "hermes-relayer")
				restartCmd.Stdout = os.Stdout
				restartCmd.Stderr = os.Stderr
				err = restartCmd.Run()
				if err != nil {
					return err
				}
				pruneCmd := exec.Command("docker", "container", "prune", "-f")
				pruneCmd.Stdout = os.Stdout
				pruneCmd.Stderr = os.Stderr
				err = pruneCmd.Run()
				if err != nil {
					return err
				}
				fmt.Printf("IBC relayer ready!\n")
			}
			return nil
		},
	}

	bootstrapCmd.Flags().StringVar(&kavaConfigTemplate, "kava.configTemplate", "master", "the directory name of the template used to generating the kava config")
	bootstrapCmd.Flags().BoolVar(&ibcFlag, "ibc", false, "flag for if ibc is enabled")
	bootstrapCmd.Flags().BoolVar(&gethFlag, "geth", false, "flag for if geth is enabled")

	return bootstrapCmd
}
