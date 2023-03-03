package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/kava-labs/kvtool/config/generate"
)

const (
	kavaServiceName    = "kava"
	binanceServiceName = "binance"
	deputyServiceName  = "deputy"

	hermesImageTag  = "kava/hermes:latest"
	relayerImageTag = "kava/relayer:v2.2.0"

	kavaChainId = "kavalocalnet_8888-1"
	ibcChainId  = "kavalocalnet_8889-2"
)

var (
	defaultGeneratedConfigDir string = filepath.Join(generate.ConfigTemplatesDir, "../..", "full_configs", "generated")

	supportedServices = []string{kavaServiceName, binanceServiceName, deputyServiceName}
)

// TestnetCmd cli command for starting kava testnets with docker
func TestnetCmd() *cobra.Command {

	var generatedConfigDir string

	rootCmd := &cobra.Command{
		Use:     "testnet",
		Aliases: []string{"t"},
		Short:   "Start a default kava and binance local testnet with a deputy. Stop with Ctrl-C and remove with 'testnet down'. Use sub commands for more options.",
		Long: fmt.Sprintf(`This command helps run local kava testnets composed of various independent processes.

Processes are run via docker-compose. This command generates a docker-compose.yaml and other necessary config files that are synchronized with each so the services all work together.

By default this command will generate configuration for a kvd node and rest server, a binance node and rest server, and a deputy. And then 'run docker-compose up'.
This is the equivalent of running 'testnet gen-config kava binance deputy' then 'testnet up'.

Docker compose files are (by default) written to %s`, defaultGeneratedConfigDir),
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {

			// 1) clear out generated config folder
			if err := os.RemoveAll(generatedConfigDir); err != nil {
				return fmt.Errorf("could not clear old generated config: %v", err)
			}

			// 2) generate a complete docker-compose config
			if err := generate.GenerateDefaultConfig(generatedConfigDir); err != nil {
				return fmt.Errorf("could not generate config: %v", err)
			}

			// 3) run docker-compose up
			cmd := []string{"docker-compose", "--file", filepath.Join(generatedConfigDir, "docker-compose.yaml"), "up"}
			fmt.Println("running:", strings.Join(cmd, " "))
			if err := replaceCurrentProcess(cmd...); err != nil {
				return fmt.Errorf("could not run command: %v", err)
			}
			return nil
		},
	}
	rootCmd.PersistentFlags().StringVar(&generatedConfigDir, "generated-dir", defaultGeneratedConfigDir, "output directory for the generated config")

	var kavaConfigTemplate string
	var ibcFlag bool
	var gethFlag bool

	genConfigCmd := &cobra.Command{
		Use:   "gen-config services_to_include...",
		Short: "Generate a complete docker-compose configuration for a new testnet.",
		Long: fmt.Sprintf(`Generate a docker-compose.yaml file and any other necessary config files needed by services.

available services: %s
`, supportedServices),
		Example:   "gen-config kava binance deputy --kava.configTemplate v0.10",
		ValidArgs: supportedServices,
		Args:      Minimum1ValidArgs,
		RunE: func(_ *cobra.Command, args []string) error {

			// 1) clear out generated config folder
			if err := os.RemoveAll(generatedConfigDir); err != nil {
				return fmt.Errorf("could not clear old generated config: %v", err)
			}

			// 2) generate a complete docker-compose config
			if stringSlice(args).contains(kavaServiceName) {
				if err := generate.GenerateKavaConfig(kavaConfigTemplate, generatedConfigDir); err != nil {
					return err
				}
			}
			if stringSlice(args).contains(binanceServiceName) {
				if err := generate.GenerateBnbConfig(generatedConfigDir); err != nil {
					return err
				}
			}
			if stringSlice(args).contains(deputyServiceName) {
				if err := generate.GenerateDeputyConfig(generatedConfigDir); err != nil {
					return err
				}
			}
			if ibcFlag {
				if err := generate.GenerateIbcChainConfig(generatedConfigDir); err != nil {
					return err
				}
			}
			if gethFlag {
				if err := generate.GenerateGethConfig(generatedConfigDir); err != nil {
					return err
				}
			}

			return nil
		},
	}
	genConfigCmd.Flags().StringVar(&kavaConfigTemplate, "kava.configTemplate", "master", "the directory name of the template used to generating the kava config")
	genConfigCmd.Flags().BoolVar(&ibcFlag, "ibc", false, "flag for if ibc is enabled")
	genConfigCmd.Flags().BoolVar(&gethFlag, "geth", false, "flag for if geth node is enabled")
	rootCmd.AddCommand(genConfigCmd)

	var runDetachedFlag bool

	upCmd := &cobra.Command{
		Use:   "up",
		Short: "A convenience command that runs `docker-compose up` on the generated config.",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			cmd := []string{"docker-compose", "--file", filepath.Join(generatedConfigDir, "docker-compose.yaml"), "up"}
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
	rootCmd.AddCommand(upCmd)

	downCmd := &cobra.Command{
		Use:   "down",
		Short: "A convenience command that runs `docker-compose down` on the generated config.",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			cmd := []string{"docker-compose", "--file", filepath.Join(generatedConfigDir, "docker-compose.yaml"), "down"}
			fmt.Println("running:", strings.Join(cmd, " "))
			if err := replaceCurrentProcess(cmd...); err != nil {
				return fmt.Errorf("could not run command: %v", err)
			}
			return nil
		},
	}
	rootCmd.AddCommand(downCmd)

	bootstrapCmd := &cobra.Command{
		Use:     "bootstrap",
		Short:   "A convenience command that creates a kava testnet with the input configTemplate (defaults to master)",
		Example: "bootstrap --kava.configTemplate v0.12",
		Args:    cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			cmd := exec.Command("docker-compose", "--file", filepath.Join(generatedConfigDir, "docker-compose.yaml"), "down")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			// check that dockerfile exists before calling 'docker-compose down down'
			if _, err := os.Stat(filepath.Join(generatedConfigDir, "docker-compose.yaml")); err == nil {
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

			cmd = exec.Command("docker-compose", "--file", filepath.Join(generatedConfigDir, "docker-compose.yaml"), "pull")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Println(err.Error())
			}

			upCmd := exec.Command("docker-compose", "--file", filepath.Join(generatedConfigDir, "docker-compose.yaml"), "up", "-d")
			upCmd.Stdout = os.Stdout
			upCmd.Stderr = os.Stderr
			if err := upCmd.Run(); err != nil {
				fmt.Println(err.Error())
			}
			if ibcFlag {
				fmt.Printf("Starting ibc connection between chains...\n")
				time.Sleep(time.Second * 7)
				restoreKeys1Cmd := exec.Command("docker", "run", "-v", fmt.Sprintf("%s:%s", filepath.Join(generatedConfigDir, "hermes"), "/home/hermes/.hermes"), hermesImageTag, "keys", "restore", kavaChainId, "-n", "testkey", "-m", "very health column only surface project output absent outdoor siren reject era legend legal twelve setup roast lion rare tunnel devote style random food", "--hd-path", "m/44'/459'/0'/0/0")
				restoreKeys1Cmd.Stdout = os.Stdout
				restoreKeys1Cmd.Stderr = os.Stderr
				if err := restoreKeys1Cmd.Run(); err != nil {
					fmt.Println(err.Error())
					return fmt.Errorf("[hermes] failed to restore keys on main chain")
				}
				restoreKeys2Cmd := exec.Command("docker", "run", "-v", fmt.Sprintf("%s:%s", filepath.Join(generatedConfigDir, "hermes"), "/home/hermes/.hermes"), hermesImageTag, "keys", "restore", ibcChainId, "-n", "testkey", "-m", "very health column only surface project output absent outdoor siren reject era legend legal twelve setup roast lion rare tunnel devote style random food", "--hd-path", "m/44'/459'/0'/0/0")
				restoreKeys2Cmd.Stdout = os.Stdout
				restoreKeys2Cmd.Stderr = os.Stderr
				if err := restoreKeys2Cmd.Run(); err != nil {
					fmt.Println(err.Error())
					return fmt.Errorf("[hermes] failed to restore keys on ibc chain")
				}
				setupIbcPathCmd := exec.Command("docker", "run", "-v", fmt.Sprintf("%s:%s", filepath.Join(generatedConfigDir, "relayer"), "/home/relayer/.relayer"), "--network", "generated_default", relayerImageTag, "rly", "paths", "new", kavaChainId, ibcChainId, "transfer")
				setupIbcPathCmd.Stdout = os.Stdout
				setupIbcPathCmd.Stderr = os.Stderr
				if err := setupIbcPathCmd.Run(); err != nil {
					fmt.Println(err.Error())
					return fmt.Errorf("[hermes] failed to setup ibc path")
				}
				openConnectionCmd := exec.Command("docker", "run", "-v", fmt.Sprintf("%s:%s", filepath.Join(generatedConfigDir, "relayer"), "/home/relayer/.relayer"), "--network", "generated_default", relayerImageTag, "rly", "transact", "link", "transfer")
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
				restartCmd := exec.Command("docker-compose", "--file", filepath.Join(generatedConfigDir, "docker-compose.yaml"), "up", "-d", "hermes-relayer")
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
	rootCmd.AddCommand(bootstrapCmd)

	exportCmd := &cobra.Command{
		Use:     "export",
		Short:   "Pauses the current kava testnet, exports the current kava testnet state to a JSON file, then restarts the testnet.",
		Example: "export",
		Args:    cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			cmd := exec.Command("docker-compose", "--file", filepath.Join(generatedConfigDir, "docker-compose.yaml"), "stop")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			err := cmd.Run()
			if err != nil {
				return err
			}
			// docker ps -aqf "name=containername"
			kavaContainerIDCmd := exec.Command("docker", "ps", "-aqf", "name=generated_kavanode_1")
			kavaContainer, err := kavaContainerIDCmd.Output()
			if err != nil {
				return err
			}

			ibcChainContainerIDCmd := exec.Command("docker", "ps", "-aqf", "name=generated_ibcnode_1")
			ibcContainer, err := ibcChainContainerIDCmd.Output()
			if err != nil {
				return err
			}

			makeNewKavaImageCmd := exec.Command("docker", "commit", strings.TrimSpace(string(kavaContainer)), "kava-export-temp")

			kavaImageOutput, err := makeNewKavaImageCmd.Output()
			if err != nil {
				return err
			}

			makeNewIbcImageCmd := exec.Command("docker", "commit", strings.TrimSpace(string(ibcContainer)), "ibc-export-temp")
			ibcImageOutput, err := makeNewIbcImageCmd.Output()
			if err != nil {
				return err
			}

			localKavaMountPath := filepath.Join(generatedConfigDir, "kava", "initstate", ".kava", "config")
			localIbcMountPath := filepath.Join(generatedConfigDir, "ibcchain", "initstate", ".kava", "config")

			kavaExportCmd := exec.Command(
				"docker", "run",
				"-v", strings.TrimSpace(fmt.Sprintf("%s:/root/.kava/config", localKavaMountPath)),
				"kava-export-temp",
				"kava", "export")
			kavaExportJSON, err := kavaExportCmd.Output()
			if err != nil {
				return err
			}

			ibcExportCmd := exec.Command(
				"docker", "run",
				"-v", strings.TrimSpace(fmt.Sprintf("%s:/root/.kava/config", localIbcMountPath)),
				"ibc-export-temp",
				"kava", "export")
			ibcExportJSON, err := ibcExportCmd.Output()
			if err != nil {
				return err
			}
			ts := time.Now().Unix()
			kavaFilename := fmt.Sprintf("kava-export-%d.json", ts)
			ibcFilename := fmt.Sprintf("ibc-export-%d.json", ts)

			fmt.Printf("Created exports %s and %s\nCleaning up...", kavaFilename, ibcFilename)

			err = ioutil.WriteFile(kavaFilename, kavaExportJSON, 0644)
			if err != nil {
				return err
			}
			err = ioutil.WriteFile(ibcFilename, ibcExportJSON, 0644)
			if err != nil {
				return err
			}

			// docker ps -aqf "name=containername"
			tempKavaContainerIDCmd := exec.Command("docker", "ps", "-aqf", "ancestor=kava-export-temp")
			tempKavaContainer, err := tempKavaContainerIDCmd.Output()
			if err != nil {
				return err
			}
			tempIbcContainerIDCmd := exec.Command("docker", "ps", "-aqf", "ancestor=ibc-export-temp")
			tempIbcContainer, err := tempIbcContainerIDCmd.Output()
			if err != nil {
				return err
			}

			deleteKavaContainerCmd := exec.Command("docker", "rm", strings.TrimSpace(string(tempKavaContainer)))
			err = deleteKavaContainerCmd.Run()
			if err != nil {
				return err
			}
			deleteIbcContainerCmd := exec.Command("docker", "rm", strings.TrimSpace(string(tempIbcContainer)))
			err = deleteIbcContainerCmd.Run()
			if err != nil {
				return err
			}

			deleteKavaImageCmd := exec.Command("docker", "rmi", strings.TrimSpace(string(kavaImageOutput)))
			err = deleteKavaImageCmd.Run()
			if err != nil {
				return err
			}
			deleteIbcImageCmd := exec.Command("docker", "rmi", strings.TrimSpace(string(ibcImageOutput)))
			err = deleteIbcImageCmd.Run()
			if err != nil {
				return err
			}

			fmt.Printf("Restarting testnet...")
			restartCmd := exec.Command("docker-compose", "--file", filepath.Join(generatedConfigDir, "docker-compose.yaml"), "start")
			restartCmd.Stdout = os.Stdout
			restartCmd.Stderr = os.Stderr

			err = restartCmd.Run()
			if err != nil {
				return err
			}
			return nil
		},
	}
	rootCmd.AddCommand(exportCmd)

	return rootCmd
}

// Minimum1ValidArgs checks if the input command has valid args
func Minimum1ValidArgs(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return errors.New("must specify at least one argument")
	}
	return cobra.OnlyValidArgs(cmd, args)
}

func replaceCurrentProcess(command ...string) error {
	if len(command) < 1 {
		panic("must provide name of executable to run")
	}
	executable, err := exec.LookPath(command[0])
	if err != nil {
		return err
	}

	// pass the current environment variables to the command
	env := os.Environ()

	err = syscall.Exec(executable, command, env)
	if err != nil {
		return err
	}
	return nil
}

type stringSlice []string

func (strings stringSlice) contains(match string) bool {
	for _, s := range strings {
		if match == s {
			return true
		}
	}
	return false
}
