package config

import (
	"log"

	"github.com/caarlos0/env/v6"
	"github.com/joho/godotenv"
)

const (
	delegationGasWithLiquid = int64(550_000)
	delegationGasNoLiquid   = int64(200_000)
)

type Config struct {
	ChainID                   string `env:"CHAIN_ID"`
	KavaGrpcEndpoint          string `env:"KAVA_GRPC_ENDPOINT"`
	DelegatorAccountsMnemonic string `env:"DELEGATOR_ACCOUNTS_MNEMONIC"`
	DevWalletMnemonic         string `env:"DEV_WALLET_MNEMONIC"`
	DefaultBaseAmount         string `env:"DEFAULT_BASE_AMOUNT" envDefault:"1_000_000_000"`
	SkipLiquify               bool   `env:"SKIP_LIQUIFY"`

	DelegationGas int64
}

func LoadConfig() (Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println(".env file not found, attempting to proceed with available env variables")
	} else {
		log.Println("loaded config from .env")
	}

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, err
	}

	if cfg.SkipLiquify {
		cfg.DelegationGas = delegationGasNoLiquid
	} else {
		cfg.DelegationGas = delegationGasWithLiquid
	}

	return cfg, nil
}
