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

/*
localnet => start just kava v0.10?
localnet --binance => start just binance
localnet --kava kava/kava:latest --binance --deputy --oracle => starts all components

kvtool testnet gen-config
kvtool testnet up
kvtool testnet down
kvtool testnet defaults to config --kava v0.10, and up


specifying master kava isn't easy - which config should be paired with it? v0.10 won't work right now. Could specify genesis file additionally.
But doesn't handle other changes: kvd flags (--pruning, unsafe cors), kvcli data

Only going to be two versions of kava at any one time

Support features piece by piece, always the manual edit fallback.
specify kava version - if folder not present error
person working on new kava branch can create new kvtool branch and put updated genesis file in master template

--kava --binance --deputy --kava.configTemplate=dm-update-querier --deputy.configTemplate=d8c3e51
*/

func TestnetCmd() *cobra.Command {

	var generatedConfigDir string

	rootCmd := &cobra.Command{
		Use:   "testnet",
		Short: "Start a local kava testnet",
		Args:  cobra.NoArgs,
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
		Use:       "gen-config [services...]",
		Short:     "Generate a complete docker-compose configuration for a new testnet.",
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
