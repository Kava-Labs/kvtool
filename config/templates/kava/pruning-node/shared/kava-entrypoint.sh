#!/bin/bash

# log all commands to stdout and stop the script on the first error
set -ex

SHARED_DIR=/docker/shared

# exit early if geneis.json already exists
# which will happen if the kava docker container is stopped and later restarted
if test -f "/root/.kava/config/genesis.json"; then
  echo "genesis.json alredy exists, skipping chain init and validator initilization"
else
  # create default genesis and node config
  kava init test --chain-id=kavalocalnet_8888-1

  # copy over genesis file
  cp "$SHARED_DIR/genesis.json" /root/.kava/config/genesis.json
fi

# set configs for kava processes to use
cp /docker/shared/app.toml ~/.kava/config/app.toml
cp /docker/shared/config.toml ~/.kava/config/config.toml

# get node id of validator, with retries to wait for kavanode apis to start
MAX_RETRIES=10
DELAY_SECONDS=2

for ((i = 0; i < MAX_RETRIES; i++)); do
  VALIDATOR_NODE_ID=$(kava --node http://kavanode:26657 status 2>/dev/null | jq -r .node_info.id)
  if [ -n "$VALIDATOR_NODE_ID" ]; then
    break
  fi
  sleep $DELAY_SECONDS
done

if [ -z "$VALIDATOR_NODE_ID" ]; then
  echo "error: unable to get validator node id from kavanode after $((MAX_RETRIES * DELAY_SECONDS)) seconds"
  exit 1
fi
echo found validator: "$VALIDATOR_NODE_ID@kavanode:26656"

# start the kava process
kava start --p2p.persistent_peers "$VALIDATOR_NODE_ID@kavanode:26656"

# run forever (kava start is non-blocking)
tail -f /dev/null
