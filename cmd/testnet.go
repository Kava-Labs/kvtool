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

			// 2) prepare new config
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

func generateFullConfig(generatedConfigDst string) error {
	// copy templates into generated config folder
	err := copy.Copy("./config_templates/kava/v0.10", filepath.Join(generatedConfigDst, "kava"))
	if err != nil {
		return err
	}
	// put together final compose file
	composeFileName := filepath.Join(generatedConfigDst, "docker-compose.yaml")
	err = copy.Copy("./config_templates/kava/v0.10/docker-compose.yaml", composeFileName)
	if err != nil {
		return err
	}

	bz, err := ioutil.ReadFile(composeFileName)
	if err != nil {
		return err
	}
	var composeConfig map[string]interface{}
	err = yaml.Unmarshal(bz, &composeConfig)
	if err != nil {
		return err
	}

	serviceConfig := composeConfig["services"].(map[interface{}]interface{})
	kavaConfig := serviceConfig["kavanode"].(map[interface{}]interface{})
	volumeConfig := kavaConfig["volumes"].([]interface{})
	volume := volumeConfig[0].(string)

	volumePaths := strings.Split(volume, ":")
	volumePaths[0] = strings.Join([]string{".", "kava", volumePaths[0]}, "/") // can't use filepath.Join as docker-compose requires the leading "./"

	volume = strings.Join(volumePaths, ":")
	volumeConfig[0] = volume
	kavaConfig["volumes"] = volumeConfig
	serviceConfig["kavanode"] = kavaConfig
	composeConfig["services"] = serviceConfig

	bz, err = yaml.Marshal(composeConfig)
	if err != nil {
		return err
	}

	fmt.Println("output: \n", string(bz))
	if err := ioutil.WriteFile(composeFileName, bz, 0644); err != nil {
		return err
	}
	return nil
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
