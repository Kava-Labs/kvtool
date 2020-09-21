#! /bin/sh
set -e

# This script creates a data directory if one doesn't exist.
# It's designed to run before the chain starts to properly initialize the data directory in case `bnbchaind init` was not run.
# This behaviour should probably live in tendermint.

configDir=$HOME/.bnbchaind/config
dataDir=$HOME/.bnbchaind/data
valStateFile=$dataDir/priv_validator_state.json

if ! test -f "$valStateFile"; then
    echo "$valStateFile doesn't exist, creating it"
    mkdir -p $dataDir
    cp $configDir/priv_validator_state.json.example "$valStateFile"
fi
