package generate

import (
	"path/filepath"

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
