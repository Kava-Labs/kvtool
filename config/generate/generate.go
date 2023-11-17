package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/otiai10/copy"
)

var (
	// ConfigTemplatesDir is the absolute path to the config templates directory.
	// It's set at build time using an -X flag. eg -ldflags "-X github.com/kava-labs/kvtool/config/generate.ConfigTemplatesDir=/home/user1/kvtool/config/templates"
	ConfigTemplatesDir string
)

func GenerateDefaultConfig(generatedConfigDir string) error {
	if err := GenerateKavaConfig("v0.10", generatedConfigDir); err != nil {
		return err
	}
	if err := GenerateBnbConfig(generatedConfigDir); err != nil {
		return err
	}
	if err := GenerateDeputyConfig(generatedConfigDir); err != nil {
		return err
	}
	return nil
}

func GenerateKavaConfig(kavaConfigTemplate, generatedConfigDir string) error {
	// copy templates into generated config folder
	err := copy.Copy(filepath.Join(ConfigTemplatesDir, "kava", kavaConfigTemplate), filepath.Join(generatedConfigDir, "kava"))
	if err != nil {
		return err
	}

	// put together final compose file
	err = overwriteMergeYAML(
		filepath.Join(ConfigTemplatesDir, "kava", kavaConfigTemplate, "docker-compose.yaml"),
		filepath.Join(generatedConfigDir, "docker-compose.yaml"),
	)
	return err
}

func GenerateBnbConfig(generatedConfigDir string) error {
	// copy templates into generated config folder
	err := copy.Copy(filepath.Join(ConfigTemplatesDir, "binance/v0.8"), filepath.Join(generatedConfigDir, "binance"))
	if err != nil {
		return err
	}

	// put together final compose file
	err = overwriteMergeYAML(
		filepath.Join(ConfigTemplatesDir, "binance/v0.8/docker-compose.yaml"),
		filepath.Join(generatedConfigDir, "docker-compose.yaml"),
	)
	return err
}

func GenerateDeputyConfig(generatedConfigDir string) error {
	// copy templates into generated config folder
	err := copy.Copy(filepath.Join(ConfigTemplatesDir, "deputy"), filepath.Join(generatedConfigDir, "deputy"))
	if err != nil {
		return err
	}

	// put together final compose file
	err = overwriteMergeYAML(
		filepath.Join(ConfigTemplatesDir, "deputy/docker-compose.yaml"),
		filepath.Join(generatedConfigDir, "docker-compose.yaml"),
	)
	return err
}

func GenerateGethConfig(generatedConfigDir string) error {
	// copy templates into generated config folder
	err := copy.Copy(filepath.Join(ConfigTemplatesDir, "geth"), filepath.Join(generatedConfigDir, "geth"))
	if err != nil {
		return err
	}

	// put together final compose file
	err = overwriteMergeYAML(
		filepath.Join(ConfigTemplatesDir, "geth", "docker-compose.yaml"),
		filepath.Join(generatedConfigDir, "docker-compose.yaml"),
	)
	return err
}

// GenerateIbcConfigs calls all necessary generation funcs for setting up the ibcchain & relayer
func GenerateIbcConfigs(generatedConfigDir string) error {
	if err := GenerateIbcChainConfig(generatedConfigDir); err != nil {
		return err
	}
	return GenerateRelayerConfig(generatedConfigDir)
}

func GenerateIbcChainConfig(generatedConfigDir string) error {
	// copy templates into generated config folder
	err := copy.Copy(filepath.Join(ConfigTemplatesDir, "ibcchain", "master"), filepath.Join(generatedConfigDir, "ibcchain"))
	if err != nil {
		return err
	}

	// put together final compose file
	err = overwriteMergeYAML(
		filepath.Join(ConfigTemplatesDir, "ibcchain", "master", "docker-compose.yaml"),
		filepath.Join(generatedConfigDir, "docker-compose.yaml"),
	)
	return err
}

func AddRelayerToNetwork(generatedConfigDir string) error {
	return overwriteMergeYAML(
		filepath.Join(ConfigTemplatesDir, "relayer", "docker-compose.yaml"),
		filepath.Join(generatedConfigDir, "docker-compose.yaml"),
	)
}

func GenerateRelayerConfig(generatedConfigDir string) error {
	err := copy.Copy(
		filepath.Join(ConfigTemplatesDir, "relayer"),
		filepath.Join(generatedConfigDir, "relayer"),
		copy.Options{AddPermission: 0666},
	)
	return err
}

func GenerateKavaPruningConfig(kavaConfigTemplate, generatedConfigDir string) error {
	pruningTemplateDir := filepath.Join(ConfigTemplatesDir, "kava", "pruning-node")
	serviceDir := filepath.Join(generatedConfigDir, "kava-pruning")
	// copy configuration files
	if err := copy.Copy(
		filepath.Join(pruningTemplateDir, "shared"),
		filepath.Join(serviceDir, "shared"),
	); err != nil {
		return err
	}

	// copy genesis file from kava template
	if err := copy.Copy(
		filepath.Join(ConfigTemplatesDir, "kava", kavaConfigTemplate, "initstate", ".kava", "config", "genesis.json"),
		filepath.Join(serviceDir, "shared", "genesis.json"),
	); err != nil {
		return err
	}

	// get kava template's image tag
	image, err := extractDockerComposeImage(
		filepath.Join(ConfigTemplatesDir, "kava", kavaConfigTemplate, "docker-compose.yaml"),
	)
	if err != nil {
		return err
	}

	// read template's docker-compose file
	content, err := os.ReadFile(filepath.Join(pruningTemplateDir, "docker-compose.yaml"))
	if err != nil {
		return err
	}

	// replace image tag in template's docker-compose
	updatedDockerCompose := strings.ReplaceAll(string(content), "KAVA_IMAGE_TAG_REPLACED_BY_KVTOOL_HERE", image)

	// save docker-compose
	if err := os.WriteFile(filepath.Join(serviceDir, "docker-compose.yaml"),
		[]byte(updatedDockerCompose), 0644); err != nil {
		return err
	}

	// put together final compose file
	err = overwriteMergeYAML(
		filepath.Join(serviceDir, "docker-compose.yaml"),
		filepath.Join(generatedConfigDir, "docker-compose.yaml"),
	)
	return err
}

func extractDockerComposeImage(dockerComposeFilePath string) (string, error) {
	// read docker-compose contents
	content, err := os.ReadFile(dockerComposeFilePath)
	if err != nil {
		return "", fmt.Errorf("error opening docker-compose file (%s): %s", dockerComposeFilePath, err)
	}

	// extract image & tag
	matches := regexp.MustCompile(`image:\s*"(.*)"`).FindStringSubmatch(string(content))
	if len(matches) != 2 {
		return "", fmt.Errorf("expected exactly one line with 'image:' in %s, found %+v", dockerComposeFilePath, matches)
	}

	dockerImage := strings.TrimSpace(matches[1])
	if dockerImage == "" {
		return "", fmt.Errorf("empty image found in %s", dockerComposeFilePath)
	}

	return dockerImage, nil
}
