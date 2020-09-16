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

	"github.com/Jeffail/gabs/v2"
	"github.com/otiai10/copy"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	kavaServiceName    = "kava"
	binanceServiceName = "binance"
	deputyServiceName  = "deputy"
)

var (
	// ProjectDir is the root directory of the kvtool repo.
	// It's set at build time using an -X flag. eg -ldflags "-X github.com/kava-labs/kvtool/cmd.ProjectDir=/home/user1/kvtool"
	ProjectDir string

	configTemplatesDir        string = filepath.Join(ProjectDir, "config_templates")
	defaultGeneratedConfigDir string = filepath.Join(ProjectDir, "full_configs", "generated")

	supportedServices = []string{kavaServiceName, binanceServiceName, deputyServiceName}
)

func TestnetCmd() *cobra.Command {

	var generatedConfigDir string

	rootCmd := &cobra.Command{
		Use:   "testnet",
		Short: "Start a default kava and binance local testnet with a deputy. Stop with Ctrl-C and remove with 'testnet down'. Use sub commands for more options.",
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
			if err := generateDefaultConfig(configTemplatesDir, generatedConfigDir); err != nil {
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
				if err := generateKavaConfig(configTemplatesDir, kavaConfigTemplate, generatedConfigDir); err != nil {
					return err
				}
			}
			if stringSlice(args).contains(binanceServiceName) {
				if err := generateBnbConfig(configTemplatesDir, generatedConfigDir); err != nil {
					return err
				}
			}
			if stringSlice(args).contains(deputyServiceName) {
				if err := generateDeputyConfig(configTemplatesDir, generatedConfigDir); err != nil {
					return err
				}
			}
			return nil
		},
	}
	genConfigCmd.Flags().StringVar(&kavaConfigTemplate, "kava.configTemplate", "v0.10", "the directory name of the template used to generating the kava config")
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

	return rootCmd
}

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

func generateDefaultConfig(configTemplatesDir, generatedConfigDir string) error {
	if err := generateKavaConfig(configTemplatesDir, "v0.10", generatedConfigDir); err != nil {
		return err
	}
	if err := generateBnbConfig(configTemplatesDir, generatedConfigDir); err != nil {
		return err
	}
	if err := generateDeputyConfig(configTemplatesDir, generatedConfigDir); err != nil {
		return err
	}
	return nil
}

func generateKavaConfig(configTemplatesDir, kavaConfigTemplate, generatedConfigDir string) error {
	// copy templates into generated config folder
	err := copy.Copy(filepath.Join(configTemplatesDir, "kava", kavaConfigTemplate), filepath.Join(generatedConfigDir, "kava"))
	if err != nil {
		return err
	}

	// put together final compose file
	err = overwriteMergeYAML(
		filepath.Join(configTemplatesDir, "kava", kavaConfigTemplate, "docker-compose.yaml"),
		filepath.Join(generatedConfigDir, "docker-compose.yaml"),
	)
	return err
}

func generateBnbConfig(configTemplatesDir, generatedConfigDir string) error {
	// copy templates into generated config folder
	err := copy.Copy(filepath.Join(configTemplatesDir, "binance/v0.6"), filepath.Join(generatedConfigDir, "binance"))
	if err != nil {
		return err
	}

	// put together final compose file
	err = overwriteMergeYAML(
		filepath.Join(configTemplatesDir, "binance/v0.6/docker-compose.yaml"),
		filepath.Join(generatedConfigDir, "docker-compose.yaml"),
	)
	return err
}

func generateDeputyConfig(configTemplatesDir, generatedConfigDir string) error {
	// copy templates into generated config folder
	err := copy.Copy(filepath.Join(configTemplatesDir, "deputy"), filepath.Join(generatedConfigDir, "deputy"))
	if err != nil {
		return err
	}

	// put together final compose file
	err = overwriteMergeYAML(
		filepath.Join(configTemplatesDir, "deputy/docker-compose.yaml"),
		filepath.Join(generatedConfigDir, "docker-compose.yaml"),
	)
	return err
}

func overwriteMergeYAML(sourceFileName, destinationFileName string) error {
	source, err := importYAML(sourceFileName)
	if err != nil {
		return err
	}
	destination, err := importYAML(destinationFileName)
	if err != nil {
		if os.IsNotExist(err) {
			destination = gabs.New()
		} else {
			return err
		}
	}
	err = destination.MergeFn(source, func(destination, source interface{}) interface{} {
		// overwrite any non-object values with the source's version
		return source
	})
	if err != nil {
		return err
	}
	if err := exportYAML(destinationFileName, destination); err != nil {
		return err
	}
	return nil
}

func importYAML(filename string) (*gabs.Container, error) {
	bz, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	unmarshalStructure := map[string]interface{}{}
	err = yaml.Unmarshal(bz, &unmarshalStructure)
	if err != nil {
		return nil, err
	}
	return gabs.Wrap(unmarshalStructure), nil
}

func exportYAML(filename string, data *gabs.Container) error {
	bz, err := yaml.Marshal(data.Data())
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(filename, bz, 0644); err != nil {
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
