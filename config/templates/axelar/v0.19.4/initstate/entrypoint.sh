#!/bin/bash

# init tofnd
echo initalizing tofnd database
tofnd -m create --no-password
rm /root/.tofnd/export

# start axelar in background (must be running for vald)
echo starting axelar daemon
axelard start 2>&1 > axelard.log &

# wait for API: https://stackoverflow.com/a/21189440/5852777
echo -n 'waiting for API'
until $(curl --output /dev/null --silent --head --fail http://localhost:26657/status); do
    printf '.'
    sleep 5
done
echo success!

# start companion services
echo starting tofnd
tofnd -m existing --no-password > tofnd.log 2>&1 &

echo starting vald
axelard vald-start \
  --keyring-backend test \
  --validator-addr $VALIDATOR_OPERATOR_ADDRESS \
  --log_level debug \
  --chain-id $CHAIN_ID \
  --from broadcaster > vald.log 2>&1 &

# tail the logs for docker output
tail -f axelard.log
