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
			return nil
		},
	}
	genConfigCmd.Flags().StringVar(&kavaConfigTemplate, "kava.configTemplate", "master", "the directory name of the template used to generating the kava config")
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
