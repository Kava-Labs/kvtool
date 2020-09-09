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

var (
	// ProjectDir is the root directory of the kvtool repo.
	// It's set at build time using an -X flag. eg -ldflags "-X github.com/kava-labs/kvtool/cmd.ProjectDir=/home/user1/kvtool"
	ProjectDir string

	configTemplatesDir        string = filepath.Join(ProjectDir, "config_templates")
	defaultGeneratedConfigDir string = filepath.Join(ProjectDir, "full_configs", "generated")
)

func TestnetCmd() *cobra.Command {

	var generatedConfigDir string

	cmd := &cobra.Command{
		Use:   "testnet",
		Short: "Start a local kava testnet",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {

			// 1) clear out generated config folder
			if err := os.RemoveAll(generatedConfigDir); err != nil {
				return fmt.Errorf("could not clear old generated config: %v", err)
			}

			// 2) generate a complete docker-compose config
			if err := generateFullConfig(configTemplatesDir, generatedConfigDir); err != nil {
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

	cmd.Flags().StringVar(&generatedConfigDir, "generated-dir", defaultGeneratedConfigDir, "output directory for the generated config")

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

func generateFullConfig(configTemplatesDir, generatedConfigDir string) error {
	if err := generateKavaConfig(configTemplatesDir, generatedConfigDir); err != nil {
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

func generateKavaConfig(configTemplatesDir, generatedConfigDir string) error {
	// copy templates into generated config folder
	err := copy.Copy(filepath.Join(configTemplatesDir, "kava/v0.10"), filepath.Join(generatedConfigDir, "kava"))
	if err != nil {
		return err
	}

	// put together final compose file
	err = overwriteMergeYAML(
		filepath.Join(configTemplatesDir, "kava/v0.10/docker-compose.yaml"),
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
