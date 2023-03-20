package testnet

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func ExportCmd() *cobra.Command {
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

	return exportCmd
}
