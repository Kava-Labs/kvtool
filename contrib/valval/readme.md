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

Though originally configured with 2 validators, the repo has been updated to run many validators (enough to get consensus on mainnet).

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
./gen.sh

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
**NOTE: the docker image you use must be setup to run rocksdb.**

To use a local version, first build & tag the kava image:
```
# wherever the Kava-Labs/kava git repo is
cd ~/kava
docker build -f Dockerfile-rocksdb -t kava/kava:local .
cd -
```

Then run this with the new tag:
```sh
KAVA_IMAGE_TAG=local docker-compose up --force-recreate
```

Note that `--force-recreate` is necessary if run previously. It will force the image tag from the environment to be picked up even if the containers have already been created.

## how to add a validator
The intention of this code is to use it to run enough validators to get consensus after replacing
the top validators in mainnet data to do things like test upgrade migrations on a mirror of mainnet.
That means that if more validators are needed to run with >66.7% of consensus power, more validators
may need to be added here. This is how:

1. update the number stored in [NUM_VALIDATORS](./NUM_VALIDATORS). this is used by the scripts to iterate through the home directories.

2. run `./gen.sh`. this generates the correct number of data directories and updates all the persistent_peers of each one to include every other validator as a peer.

3. add another node to the docker compose (replace `11` in the name and `volumes` below with the new node index):
```yaml
  kava-11:
    image: "kava/kava:${KAVA_IMAGE_TAG:-master}"
    volumes:
      - "./kava-11:/root/.kava"
    # start the blockchain, and set rpc to listen to connections from outside the container
    command:
      - "sh"
      - "-c"
      - "/root/.kava/config/init-data-directory.sh && kava start --rpc.laddr=tcp://0.0.0.0:26657"
```

4. resume your regularly scheduled meganode running
