package testnet

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kava-labs/kvtool/config/generate"
)

const (
	kavaServiceName    = "kava"
	binanceServiceName = "binance"
	deputyServiceName  = "deputy"

	hermesImageTag  = "kava/hermes:latest"
	relayerImageTag = "kava/relayer:v2.2.0"

	kavaChainId = "kavalocalnet_8888-1"
	ibcChainId  = "kavalocalnet_8889-2"
)

var (
	ibcFlag            bool
	gethFlag           bool
	kavaConfigTemplate string

	generatedConfigDir        string
	defaultGeneratedConfigDir string = filepath.Join(generate.ConfigTemplatesDir, "../..", "full_configs", "generated")

	supportedServices = []string{kavaServiceName, binanceServiceName, deputyServiceName}
)

// Cmd is the CLI command for starting kava testnets with docker
func Cmd() *cobra.Command {
	var testnetCmd = &cobra.Command{
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

	testnetCmd.PersistentFlags().StringVar(&generatedConfigDir, "generated-dir", defaultGeneratedConfigDir, "output directory for the generated config")

	testnetCmd.AddCommand(GenConfigCmd())
	testnetCmd.AddCommand(BootstrapCmd())
	testnetCmd.AddCommand(DcCmd())

	// kept for convenience/legacy reasons.
	testnetCmd.AddCommand(UpCmd())
	testnetCmd.AddCommand(DownCmd())

	return testnetCmd
}
