This directory contains a setup for running two validators simultaneously.
This allows you to do things like start networks from nonzero height exports.

The problem with nonzero height exports is that even if your validator has a controlling share of the power, it will not produce blocks until it is caught up, but it won't consider itself caught up until it can contact at least one other peer in the network.

Maybe someday this is built into kvtool or better automated. For now, this is how to do it manually.

# UPDATE & DISCLAIMER

The goal of this was to create a mirrornet by replacing the top two validators of a non-zero height
mainnet export and giving them a controlling share of the power.

Unfortunately it's not that simple. At the end of a block being committed, power is reassessed based
on the underlying delegations to the validators. This means that the two validators will only have
enough power to create a block once (if given a controlling share with `--min-power`), and then they
will revert back to their original power.

Though originally configured with 2 validators, the repo has been updated to run 10 validators.
This is the number required for a controlling power share of mainnet data.

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
Replace top ten validators with our nodes:
```sh
# setup keys file for update-genesis-validators
mkdir keys
./copy-keys.sh

# replace validators in original genesis
update-genesis-validators example-genesis.json --chain-id kavamirror_2221-10

# copy updated genesis to all validator config directories
./copy-gen.sh
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
