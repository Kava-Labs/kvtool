package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/otiai10/copy"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
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
	return nil
}

func generateKavaConfig(generatedConfigDst string) error {
	// copy templates into generated config folder
	err := copy.Copy("./config_templates/kava/v0.10", filepath.Join(generatedConfigDst, "kava"))
	if err != nil {
		return err
	}

	// put together final compose file
	bz, err := ioutil.ReadFile("./config_templates/kava/v0.10/docker-compose.yaml")
	if err != nil {
		return err
	}
	var composeConfig map[string]interface{}
	err = yaml.Unmarshal(bz, &composeConfig)
	if err != nil {
		return err
	}
	serviceConfig := composeConfig["services"].(map[interface{}]interface{})

	composeFileName := filepath.Join(generatedConfigDst, "docker-compose.yaml")
	bz, err = ioutil.ReadFile(composeFileName)
	if err != nil {
		if os.IsNotExist(err) {
			bz, err := yaml.Marshal(composeConfig)
			if err != nil {
				return err
			}
			if err := ioutil.WriteFile(composeFileName, bz, 0644); err != nil {
				return fmt.Errorf("could not create file: %w", err)
			}
			return nil // TODO create empty file here, and fix yaml merging below to work on empty files
		} else {
			return err
		}
	}
	var existingComposeConfig map[string]interface{}
	err = yaml.Unmarshal(bz, &existingComposeConfig)
	if err != nil {
		return err
	}
	existingServiceConfig := existingComposeConfig["services"].(map[interface{}]interface{})

	for k, v := range serviceConfig {
		existingServiceConfig[k] = v
	}
	existingComposeConfig["services"] = existingServiceConfig
	// TODO handle other compose file sections - volumes, networks

	bz, err = yaml.Marshal(existingComposeConfig)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(composeFileName, bz, 0644); err != nil {
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
	bz, err := ioutil.ReadFile("./config_templates/binance/v0.6/docker-compose.yaml")
	if err != nil {
		return err
	}
	var composeConfig map[string]interface{}
	err = yaml.Unmarshal(bz, &composeConfig)
	if err != nil {
		return err
	}
	serviceConfig := composeConfig["services"].(map[interface{}]interface{})

	composeFileName := filepath.Join(generatedConfigDst, "docker-compose.yaml")
	bz, err = ioutil.ReadFile(composeFileName)
	if err != nil {
		if os.IsNotExist(err) {
			bz, err := yaml.Marshal(composeConfig)
			if err != nil {
				return err
			}
			if err := ioutil.WriteFile(composeFileName, bz, 0644); err != nil {
				return fmt.Errorf("could not create file: %w", err)
			}
			return nil // TODO create empty file here, and fix yaml merging below to work on empty files
		} else {
			return err
		}
	}
	var existingComposeConfig map[string]interface{}
	err = yaml.Unmarshal(bz, &existingComposeConfig)
	if err != nil {
		return err
	}
	existingServiceConfig := existingComposeConfig["services"].(map[interface{}]interface{})

	for k, v := range serviceConfig {
		existingServiceConfig[k] = v
	}
	existingComposeConfig["services"] = existingServiceConfig
	// TODO handle other compose file sections - volumes, networks

	bz, err = yaml.Marshal(existingComposeConfig)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(composeFileName, bz, 0644); err != nil {
		return err
	}
	return nil
}

// Could also use github.com/docker/cli/cli/compose to parse a compose file into a known type.
// However the code has no go.mod, and a lot of dependencies (kept in a vendor directory, with exact listed in vendor.conf)
// so importing it is not straightforwards.

/*
	unmarshal yaml
	mapping, err := loader.ParseYAML(bz)

	preloadConfig := types.ConfigDetails{
		Version:    "", // ?
		WorkingDir: "", // ?
		ConfigFiles: []types.ConfigFile{{
			Filename: "docker-compose.yaml", // ?
			Config:   mapping,
		}},
		Environment: nil, // ?
	}
	composeConfig, err := loader.Load(preloadConfig)
	if err != nil {
		return err
	}
*/

// Other failed attempt used github.com/Jeffail/gabs to automatically set/get from the map[string]interface{} structure created by yaml.Unmarshal
// However it's designed for json, yaml actually uses an incompatible type map[interface{}]interface{}.
