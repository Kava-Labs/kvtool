package testnet

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/kava-labs/kvtool/config/generate"
	"github.com/spf13/cobra"
)

// this env variable is used in supported kava templates to allow override of the image tag
// automated chain upgrades make use of it to switch between binary versions.
const kavaTagEnv = "KAVA_TAG"

var dockerComposeConfig string

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

			dockerComposeConfig = generatedPath("docker-compose.yaml")
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
			// when doing automated chain upgrade, ensure the node starts with the desired image tag
			// if this is empty, the docker-compose should default to intended image tag
			if chainUpgradeBaseImageTag != "" {
				upCmd.Env = os.Environ()
				upCmd.Env = append(upCmd.Env, fmt.Sprintf("%s=%s", kavaTagEnv, chainUpgradeBaseImageTag))
				fmt.Printf("starting chain with image tag %s\n", chainUpgradeBaseImageTag)
			}
			if err := upCmd.Run(); err != nil {
				fmt.Println(err.Error())
			}

			if ibcFlag {
				if err := setupIbcChannelAndRelayer(); err != nil {
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

	// optional data for running an automated chain upgrade
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

func setupIbcChannelAndRelayer() error {
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

	// write upgrade proposal to json file
	upgradeJson, err := writeUpgradeProposal()
	if err != nil {
		return err
	}

	// wait for chain to start producing blocks
	if err := waitForBlock(1, 5*time.Second); err != nil {
		return err
	}

	// submit upgrade proposal via God Committee (committee 3)
	fmt.Println("submitting upgrade proposal")
	cmd := fmt.Sprintf("tx committee submit-proposal 3 %s --gas auto --gas-adjustment 1.2 --gas-prices 0.05ukava --from committee -y",
		upgradeJson,
	)
	if err := runKavaCli(dockerComposeConfig, strings.Split(cmd, " ")...); err != nil {
		return err
	}

	// vote on the committee proposal
	cmd = "tx committee vote 1 yes --from committee --gas auto --gas-adjustment 1.2 --gas-prices 0.01ukava -y"
	if err := runKavaCli(dockerComposeConfig, strings.Split(cmd, " ")...); err != nil {
		return err
	}

	// wait for chain halt at upgrade height
	if err := waitForBlock(chainUpgradeHeight, time.Duration(chainUpgradeHeight)*4*time.Second); err != nil {
		return err
	}
	fmt.Printf("chain has reached upgrade height (%d) and halted!\n", chainUpgradeHeight)

	return nil
}

// writeUpgradeProposal writes a proposal json to a file in the kavanode container and returns the path
func writeUpgradeProposal() (string, error) {
	content := fmt.Sprintf(`{
		"@type": "/cosmos.upgrade.v1beta1.SoftwareUpgradeProposal",
		"title": "Automated Chain Upgrade",
		"description": "An auto-magical chain upgrade performed by kvtool.",
		"plan": { "name": "%s", "height": "%d" }
	}`, chainUpgradeName, chainUpgradeHeight)
	// write the file to a location inside the container
	return "/root/.kava/config/upgrade-proposal.json", os.WriteFile(
		generatedPath("kava", "initstate", ".kava", "config", "upgrade-proposal.json"),
		[]byte(content),
		0644,
	)
}

func waitForBlock(n int64, timeout time.Duration) error {
	b := backoff.NewExponentialBackOff()
	b.MaxInterval = 2 * time.Second
	b.MaxElapsedTime = timeout
	return backoff.Retry(blockGTE(n), b)
}

// blockGTE is a backoff operation that uses kava's CLI to query the chain for the current block number
// the operation fails in the following cases:
// 1. the chain cannot be reached, 2. result cannot be parsed, 3. current height is less than desired height `n`
func blockGTE(n int64) backoff.Operation {
	return func() error {
		cmd := "kava status | jq -r .sync_info.latest_block_height"
		out, err := exec.Command("docker-compose", "-f", dockerComposeConfig, "exec", "-T", "kavanode", "bash", "-c", cmd).Output()
		if err != nil {
			return err
		}
		height, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
		if err != nil {
			return err
		}
		if height < n {
			return fmt.Errorf("waiting for height %d, found %d", n, height)
		}
		return nil
	}
}

// runKavaCli execs into the kava container and runs `kava args...`
func runKavaCli(dockerComposeConfig string, args ...string) error {
	pieces := make([]string, 4, len(args)+4)
	pieces[0] = "exec"
	pieces[1] = "-T"
	pieces[2] = "kavanode"
	pieces[3] = "kava"
	pieces = append(pieces, args...)
	return runDockerCompose(dockerComposeConfig, pieces...)
}

func runDockerCompose(dockerComposeConfig string, args ...string) error {
	pieces := make([]string, 2, len(args)+2)
	pieces[0] = "--file"
	pieces[1] = dockerComposeConfig
	pieces = append(pieces, args...)
	cmd := exec.Command("docker-compose", pieces...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
