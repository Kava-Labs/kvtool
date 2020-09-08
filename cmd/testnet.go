package cmd

import (
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

func TestnetCmd() *cobra.Command {
	const defaultConfigOutput = "full_configs/generated/"
	var generatedConfigOutput string

	cmd := &cobra.Command{
		Use:   "testnet",
		Short: "Start a local kava testnet",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {

			// 1) clear out generated config folder
			if err := os.RemoveAll(generatedConfigOutput); err != nil {
				return fmt.Errorf("could not clear old generated config: %v", err)
			}

			// 2) generate a complete docker-compose config
			if err := generateFullConfig(generatedConfigOutput); err != nil {
				return fmt.Errorf("could not generate config: %v", err)
			}

			// 3) run docker-compose up
			cmd := []string{"docker-compose", "--file", filepath.Join(generatedConfigOutput, "docker-compose.yaml"), "up"}
			fmt.Println("running:", strings.Join(cmd, " "))
			if err := replaceCurrentProcess(cmd...); err != nil {
				return fmt.Errorf("could not run command: %v", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&generatedConfigOutput, "config-output", defaultConfigOutput, "output directory for the generated config")

	return cmd
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

func generateFullConfig(generatedConfigDst string) error {
	if err := generateKavaConfig(generatedConfigDst); err != nil {
		return err
	}
	if err := generateBnbConfig(generatedConfigDst); err != nil {
		return err
	}
	if err := generateDeputyConfig(generatedConfigDst); err != nil {
		return err
	}
	return nil
}

func generateKavaConfig(generatedConfigDst string) error {
	// copy templates into generated config folder
	err := copy.Copy("./config_templates/kava/v0.10", filepath.Join(generatedConfigDst, "kava"))
	if err != nil {
		return err
	}

	// put together final compose file
	composeFileName := filepath.Join(generatedConfigDst, "docker-compose.yaml")
	if err := overwriteMergeYAML("./config_templates/kava/v0.10/docker-compose.yaml", composeFileName); err != nil {
		return err
	}
	return nil
}

func generateBnbConfig(generatedConfigDst string) error {
	// copy templates into generated config folder
	err := copy.Copy("./config_templates/binance/v0.6", filepath.Join(generatedConfigDst, "binance"))
	if err != nil {
		return err
	}

	// put together final compose file
	composeFileName := filepath.Join(generatedConfigDst, "docker-compose.yaml")
	if err := overwriteMergeYAML("./config_templates/binance/v0.6/docker-compose.yaml", composeFileName); err != nil {
		return err
	}
	return nil
}

func generateDeputyConfig(generatedConfigDst string) error {
	// copy templates into generated config folder
	err := copy.Copy("./config_templates/deputy", filepath.Join(generatedConfigDst, "deputy"))
	if err != nil {
		return err
	}

	// put together final compose file
	composeFileName := filepath.Join(generatedConfigDst, "docker-compose.yaml")
	if err := overwriteMergeYAML("./config_templates/deputy/docker-compose.yaml", composeFileName); err != nil {
		return err
	}
	return nil
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
