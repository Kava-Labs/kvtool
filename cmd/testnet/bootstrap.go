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
			if err := validateBootstrapFlags(); err != nil {
				return err
			}

			dockerComposeConfig := generatedPath("docker-compose.yaml")
			// shutdown existing networks if a docker-compose.yaml already exists.
			if _, err := os.Stat(dockerComposeConfig); err == nil {
				downCmd := exec.Command("docker-compose", "--file", dockerComposeConfig, "down")
				downCmd.Stdout = os.Stdout
				downCmd.Stderr = os.Stderr
				if err2 := downCmd.Run(); err2 != nil {
					return err2
				}
			}

			// remove entire generated dir in order to start from scratch
			if err := os.RemoveAll(generatedConfigDir); err != nil {
				return fmt.Errorf("could not clear old generated config: %v", err)
			}

			// generate kava node configuration
			if err := generate.GenerateKavaConfig(kavaConfigTemplate, generatedConfigDir); err != nil {
				return err
			}
			// handle ibc configuration
			if ibcFlag {
				if err := generate.GenerateIbcConfigs(generatedConfigDir); err != nil {
					return err
				}
			}
			// handle geth configuration
			if gethFlag {
				if err := generate.GenerateGethConfig(generatedConfigDir); err != nil {
					return err
				}
			}

			pullContainersCmd := exec.Command("docker-compose", "--file", dockerComposeConfig, "pull")
			pullContainersCmd.Stdout = os.Stdout
			pullContainersCmd.Stderr = os.Stderr
			if err := pullContainersCmd.Run(); err != nil {
				fmt.Println(err.Error())
			}

			upCmd := exec.Command("docker-compose", "--file", dockerComposeConfig, "up", "-d")
			upCmd.Stdout = os.Stdout
			upCmd.Stderr = os.Stderr
			if err := upCmd.Run(); err != nil {
				fmt.Println(err.Error())
			}

			if ibcFlag {
				if err := setupIbcChannelAndRelayer(dockerComposeConfig); err != nil {
					return err
				}
			}

			// validation of all necessary data for an automated chain upgrade is performed in validateBootstrapFlags()
			if chainUpgradeName != "" {
				if err := runChainUpgrade(); err != nil {
					return err
				}
			}

			return nil
		},
	}

	bootstrapCmd.Flags().StringVar(&kavaConfigTemplate, "kava.configTemplate", "master", "the directory name of the template used to generating the kava config")
	bootstrapCmd.Flags().BoolVar(&ibcFlag, "ibc", false, "flag for if ibc is enabled")
	bootstrapCmd.Flags().BoolVar(&gethFlag, "geth", false, "flag for if geth is enabled")

	bootstrapCmd.Flags().StringVar(&chainUpgradeName, "upgrade-name", "", "name of automated chain upgrade to run, if desired. the upgrade must be defined in the running kava container.")
	bootstrapCmd.Flags().Int64Var(&chainUpgradeHeight, "upgrade-height", 0, "height of automated chain upgrade to run.")
	bootstrapCmd.Flags().StringVar(&chainUpgradeBaseImageTag, "upgrade-base-image-tag", "", "the kava docker image tag that will be upgraded. the chain is initialized from this tag and then upgraded to the new image.\nthe binary must be compatible with the kava.configTemplate genesis.")

	return bootstrapCmd
}

func validateBootstrapFlags() error {
	hasUpgradeName := chainUpgradeName != ""
	hasUpgradeBaseImageTag := chainUpgradeBaseImageTag != ""
	// the upgrade flags are all or nothing. both the upgrade name and the image tag are required for
	// an automated chain upgrade
	if (hasUpgradeName && !hasUpgradeBaseImageTag) || (hasUpgradeBaseImageTag && !hasUpgradeName) {
		return fmt.Errorf("automated chain upgrades require both --upgrade-name and --upgrade-base-image-tag to be defined")
	}
	// if running an automate chain upgrade, there must be a sufficiently high upgrade height.
	if hasUpgradeName && chainUpgradeHeight < 10 {
		// TODO: is 10 a sufficient height for an upgrade to occur with proposal & voting? probs not..
		return fmt.Errorf("upgrade height must be > 10, found %d", chainUpgradeHeight)
	}
	return nil
}

func setupIbcChannelAndRelayer(dockerComposeConfig string) error {
	fmt.Printf("Starting ibc connection between chains...\n")
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
	restartCmd := exec.Command("docker-compose", "--file", dockerComposeConfig, "up", "-d", "hermes-relayer")
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
	return nil
}

func runChainUpgrade() error {
	fmt.Println("would run chain upgrade!")
	fmt.Printf("upgrade name: %s\nupgrade height: %d\nstarting tag: %s\n", chainUpgradeName, chainUpgradeHeight, chainUpgradeBaseImageTag)
	return nil
}
