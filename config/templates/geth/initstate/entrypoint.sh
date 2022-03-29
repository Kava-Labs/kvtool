#!/bin/sh

set -e

echo initializing genesis...
geth init /root/genesis.json

echo importing state...
geth import /root/exported_state

echo starting geth...
geth --unlock 21e360e198cde35740e88572b59f2cade421e6b1 \
     --password /root/eth-password \
     --mine \
     --allow-insecure-unlock \
     --http \
     --http.addr 0.0.0.0 \
     --http.corsdomain '*' \
     --ws \
     --ws.addr 0.0.0.0 \
     --ws.origins '*'
