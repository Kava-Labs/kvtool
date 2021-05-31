package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	"github.com/cosmos/cosmos-sdk/x/genutil"

	"github.com/tendermint/tendermint/crypto/secp256k1"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/kava-labs/kava/app"
	"github.com/kava-labs/kvtool/config/generate"
)

const (
	kavaServiceName    = "kava"
	binanceServiceName = "binance"
	deputyServiceName  = "deputy"
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
	const flagVesting = "add-vesting-accounts"

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
				boolValue := viper.GetBool(flagVesting)
				fmt.Printf("Vesting boolean: %t\n", boolValue)
				if boolValue {
					err := addVestingAccountsToGenesis(kavaConfigTemplate, generatedConfigDir, 500)
					if err != nil {
						return err
					}
				}
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
			return nil
		},
	}
	genConfigCmd.Flags().StringVar(&kavaConfigTemplate, "kava.configTemplate", "master", "the directory name of the template used to generating the kava config")
	genConfigCmd.Flags().Bool(flagVesting, false, "generates 500 additional vesting accounts as part of genesis file")
	err := viper.BindPFlag(flagVesting, genConfigCmd.Flags().Lookup(flagVesting))
	if err != nil {
		panic(fmt.Sprintf("failed to bind flag: %s", err))
	}
	rootCmd.AddCommand(genConfigCmd)

	upCmd := &cobra.Command{
		Use:   "up",
		Short: "A convenience command that runs `docker-compose up` on the generated config.",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			cmd := []string{"docker-compose", "--file", filepath.Join(generatedConfigDir, "docker-compose.yaml"), "up"}
			fmt.Println("running:", strings.Join(cmd, " "))
			if err := replaceCurrentProcess(cmd...); err != nil {
				return fmt.Errorf("could not run command: %v", err)
			}
			return nil
		},
	}
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
			if err := generate.GenerateKavaConfig(kavaConfigTemplate, generatedConfigDir); err != nil {
				return err
			}
			cmd = exec.Command("docker-compose", "--file", filepath.Join(generatedConfigDir, "docker-compose.yaml"), "pull")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Println(err.Error())
			}

			upCmd := []string{"docker-compose", "--file", filepath.Join(generatedConfigDir, "docker-compose.yaml"), "up", "-d"}
			fmt.Println("running:", strings.Join(upCmd, " "))
			if err := replaceCurrentProcess(upCmd...); err != nil {
				return fmt.Errorf("could not run command: %v", err)
			}
			return nil
		},
	}
	bootstrapCmd.Flags().StringVar(&kavaConfigTemplate, "kava.configTemplate", "master", "the directory name of the template used to generating the kava config")
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
			containerIDCmd := exec.Command("docker", "ps", "-aqf", "name=generated_kavanode_1")
			container, err := containerIDCmd.Output()
			if err != nil {
				return err
			}

			makeNewImageCmd := exec.Command("docker", "commit", strings.TrimSpace(string(container)), "kava-export-temp")

			imageOutput, err := makeNewImageCmd.Output()
			if err != nil {
				return err
			}
			localKvdMountPath := filepath.Join(generatedConfigDir, "kava", "initstate", ".kvd", "config")
			localKvcliMountPath := filepath.Join(generatedConfigDir, "kava", "initstate", ".kvcli")
			exportCmd := exec.Command(
				"docker", "run",
				"-v", strings.TrimSpace(fmt.Sprintf("%s:/root/.kvd/config", localKvdMountPath)),
				"-v", strings.TrimSpace(fmt.Sprintf("%s:/root/.kvcli", localKvcliMountPath)),
				"kava-export-temp",
				"kvd", "export")
			exportJSON, err := exportCmd.Output()
			if err != nil {
				return err
			}

			filename := fmt.Sprintf("export-%d.json", time.Now().Unix())

			fmt.Printf("Created export %s\nCleaning up...", filename)

			err = ioutil.WriteFile(filename, exportJSON, 0644)
			if err != nil {
				return err
			}

			// docker ps -aqf "name=containername"
			tempContainerIDCmd := exec.Command("docker", "ps", "-aqf", "ancestor=kava-export-temp")
			tempContainer, err := tempContainerIDCmd.Output()

			if err != nil {
				return err
			}

			deleteContainerCmd := exec.Command("docker", "rm", strings.TrimSpace(string(tempContainer)))
			err = deleteContainerCmd.Run()
			if err != nil {
				return err
			}

			deleteImageCdm := exec.Command("docker", "rmi", strings.TrimSpace(string(imageOutput)))
			err = deleteImageCdm.Run()
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

func addVestingAccountsToGenesis(kavaConfigTemplate, generatedConfigDir string, numAccounts int) error {
	genesisTemplate := filepath.Join(generate.ConfigTemplatesDir, "kava", kavaConfigTemplate, "initstate", ".kvd", "config", "genesis.json")
	templateGenesisDoc, err := tmtypes.GenesisDocFromFile(genesisTemplate)
	if err != nil {
		return err
	}
	cdc := app.MakeCodec()
	var appStateMap genutil.AppMap
	if err := cdc.UnmarshalJSON(templateGenesisDoc.AppState, &appStateMap); err != nil {
		return err
	}

	var templateAuthGenesisState auth.GenesisState
	cdc.MustUnmarshalJSON(appStateMap[auth.ModuleName], &templateAuthGenesisState)

	authGenState := addAccountsToAuthGenesisState(cdc, templateAuthGenesisState, numAccounts)

	appStateMap[auth.ModuleName] = cdc.MustMarshalJSON(&authGenState)
	templateGenesisDoc.AppState = cdc.MustMarshalJSON(appStateMap)
	err = templateGenesisDoc.SaveAs(genesisTemplate)
	if err != nil {
		return err
	}

	return nil
}

func addAccountsToAuthGenesisState(cdc *codec.Codec, authGenState auth.GenesisState, numAccounts int) auth.GenesisState {
	for i := 0; i < numAccounts; i++ {
		vestingAccount := makeRandomVestingAccount()
		authGenState.Accounts = append(authGenState.Accounts, vestingAccount)
	}
	return authGenState
}

func makeRandomVestingAccount() *vesting.PeriodicVestingAccount {
	rand.Seed(time.Now().UnixNano())
	coins := sdk.NewCoins()
	numPeriods := rand.Intn(20-1) + 1
	startTime := rand.Intn(1617883200-1573218000) + 1573218000
	endTime := startTime
	periods := vesting.Periods{}
	for i := 0; i < numPeriods; i++ {
		periodCoins := sdk.NewCoins()
		hasHard := rand.Float32() < 0.5
		oneMonth := rand.Float32() < 0.5
		kavaAmount := rand.Intn(10000000000-10000000) + 10000000
		periodCoins = periodCoins.Add(sdk.NewInt64Coin("ukava", int64(kavaAmount)))
		if hasHard {
			hardAmount := rand.Intn(10000000000-10000000) + 10000000
			periodCoins = periodCoins.Add(sdk.NewInt64Coin("hard", int64(hardAmount)))
		}
		length := 86400 * 30
		if !oneMonth {
			length = 86400 * 365
		}
		period := vesting.Period{Length: int64(length), Amount: periodCoins}
		periods = append(periods, period)
		coins = coins.Add(periodCoins...)
		endTime += length
	}
	randomAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	bacc := auth.NewBaseAccountWithAddress(randomAddr)
	bacc.Coins = coins
	bva, err := vesting.NewBaseVestingAccount(&bacc, coins, int64(endTime))
	if err != nil {
		panic(err)
	}
	pva := vesting.NewPeriodicVestingAccountRaw(bva, int64(startTime), periods)
	if err := pva.Validate(); err != nil {
		panic(err)
	}
	return pva
}
