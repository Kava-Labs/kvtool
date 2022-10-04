package config

import (
	"log"

	"github.com/caarlos0/env/v6"
	"github.com/joho/godotenv"
)

type Config struct {
	ChainID                   string `env:"CHAIN_ID"`
	KavaGrpcEndpoint          string `env:"KAVA_GRPC_ENDPOINT"`
	DelegatorAccountsMnemonic string `env:"DELEGATOR_ACCOUNTS_MNEMONIC"`
	DevWalletMnemonic         string `env:"DEV_WALLET_MNEMONIC"`
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

	return cfg, nil
}
