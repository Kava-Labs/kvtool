This directory contains a setup for running two validators simultaneously.
This allows you to do things like start networks from nonzero height exports.

The problem with nonzero height exports is that even if your validator has a controlling share of the power, it will not produce blocks until it is caught up, but it won't consider itself caught up until it can contact at least one other peer in the network.

Maybe someday this is built into kvtool or better automated. For now, this is how to do it manually.

# UPDATE & DISCLAIMER

The goal of this was to create a mirrornet by replacing the top two validators of a non-zero height
mainnet export and giving them a controlling share of the power.

Unfortunately it's not that simple. At the end of a block being committed, power is reassessed based
on the underlying delegations to the validators. This means that the two validators will only have
enough power to create a block once, and then they will revert back to their original power.

It's worth keeping here because you can still run two validators in a docker container. However,
the current capabilities of `update-genesis-validators` is insufficient for the mirrornet use case.
If it were updated to also update the underlying token delegations for the replaced validators it
would work better. For now, if the original top two validators don't have a controlling network share,
the `--min-power` flag will only allow it to create one block before the power is reassessed.

# Instructions

## prereqs
Have a non-zero height genesis file. The genesis file must have at least two validators.

Install `update-genesis-validators` script:
```sh
cd ../update-genesis-validators
go install ./...
```

Have `docker` & `docker-compose` installed.

This example uses the following, but the instructions should work for any genesis:
* starting genesis file: `./example-genesis.json`
* new chain id: `kavamirror_2221-1`

## configure genesis to use our validators
Replace top two validators with our nodes. Collectively give them at least 90% of the network power.
```sh
mkdir keys
cp kava-1/config/priv_validator_key.json keys/priv_validator_key_0.json
cp kava-2/config/priv_validator_key.json keys/priv_validator_key_1.json

update-genesis-validators example-genesis.json --chain-id kavamirror_2221-1 --min-power .9

cp updated-genesis.json kava-1/config/genesis.json
cp updated-genesis.json kava-2/config/genesis.json
```

## run the networks
```sh
docker-compose up
```

## change the kava version
By default, this uses the `master` tag of the kava docker image.
You can override the tag with the `KAVA_IMAGE_TAG` env variable.

To use a local version, first build & tag the kava image:
```
# wherever the Kava-Labs/kava git repo is
cd ~/kava
docker build -t kava/kava:local .
cd -
```

Then run this with the new tag:
```sh
KAVA_IMAGE_TAG=local docker-compose up
```
