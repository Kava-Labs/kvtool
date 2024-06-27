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

	relayerImageTag = "kava/relayer:v2.4.2"
)

var (
	ibcFlag            bool
	gethFlag           bool
	includePruningFlag bool
	kavaConfigTemplate string

	kavaDbBackend string

	chainUpgradeName         string
	chainUpgradeHeight       int64
	chainUpgradeBaseImageTag string

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

	Processes are run via docker compose. This command generates a docker-compose.yaml and other necessary config files that are synchronized with each so the services all work together.

	By default this command will generate configuration for a kvd node and rest server, a binance node and rest server, and a deputy. And then run 'docker compose up'.
	This is the equivalent of running 'testnet gen-config kava binance deputy' then 'testnet up'.

	Docker compose files are (by default) written to %s`, defaultGeneratedConfigDir),
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {

			// 1) clear out generated config folder
			if err := os.RemoveAll(generatedConfigDir); err != nil {
				return fmt.Errorf("could not clear old generated config: %v", err)
			}

			// 2) generate a complete docker-compose config
			if err := generate.GenerateDefaultConfig(generatedConfigDir, kavaDbBackend); err != nil {
				return fmt.Errorf("could not generate config: %v", err)
			}

			// 3) run docker-compose up
			cmd := []string{"docker", "compose", "--file", generatedPath("docker-compose.yaml"), "up"}
			fmt.Println("running:", strings.Join(cmd, " "))
			if err := replaceCurrentProcess(cmd...); err != nil {
				return fmt.Errorf("could not run command: %v", err)
			}
			return nil
		},
	}

	testnetCmd.PersistentFlags().StringVar(&generatedConfigDir, "generated-dir", defaultGeneratedConfigDir, "output directory for the generated config")
	testnetCmd.PersistentFlags().StringVar(&kavaDbBackend, "kava.db", "goleveldb", "update the db_backend of kava. KAVA_TAG must be compatible with db choice.")

	testnetCmd.AddCommand(GenConfigCmd())
	testnetCmd.AddCommand(BootstrapCmd())
	testnetCmd.AddCommand(ExportCmd())
	testnetCmd.AddCommand(DcCmd())

	// kept for convenience/legacy reasons.
	testnetCmd.AddCommand(UpCmd())
	testnetCmd.AddCommand(DownCmd())

	return testnetCmd
}

func KavaCmd() *cobra.Command {
	kavaCmd := &cobra.Command{
		Use:     "kava -- [kava commands & args]",
		Short:   "Run `kava` inside the kavanode container",
		Long:    "An alias for `kvtool testnet dc exec kavanode -- kava <...all arguments>`.",
		Example: `TODO`,
		Args:    cobra.ArbitraryArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			cmd := []string{
				"docker", "compose", "--file", generatedPath("docker-compose.yaml"),
				"exec", "kavanode", "kava",
			}
			cmd = append(cmd, args...)
			if err := replaceCurrentProcess(cmd...); err != nil {
				return fmt.Errorf("could not run command: %v", err)
			}
			return nil
		},
	}
	return kavaCmd
}

func IbcCmd() *cobra.Command {
	kavaCmd := &cobra.Command{
		Use:     "ibc -- [kava commands & args]",
		Short:   "Run `kava` inside the ibcnode container (the IBC node is another kava chain, but with denom `uatom`)",
		Long:    "An alias for `kvtool testnet dc exec ibcnode -- kava <...all arguments>`.",
		Example: `TODO`,
		Args:    cobra.ArbitraryArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			cmd := []string{
				"docker", "compose", "--file", generatedPath("docker-compose.yaml"),
				"exec", "ibcnode", "kava",
			}
			cmd = append(cmd, args...)
			if err := replaceCurrentProcess(cmd...); err != nil {
				return fmt.Errorf("could not run command: %v", err)
			}
			return nil
		},
	}
	return kavaCmd
}
