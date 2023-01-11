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

## how to add a validator
The intention of this code is to use it to run enough validators to get consensus after replacing
the top validators in mainnet data to do things like test upgrade migrations on a mirror of mainnet.
That means that if more validators are needed to run with >66.7% of consensus power, more validators
may need to be added here. This is how:

1. update all the shell scripts in this directory (`gen.sh`, `clean.sh`, `copy-gen.sh`) to iterate counting to the new number
example: changing from 10 -> 11 validators
```diff
--- gen.sh
+++ gen.sh
-for i in {1..10}
+for i in {1..11}
```

1. run `./gen.sh`. this generates the correct number of data directories

2. the above command outputs the list of all peer node ids and addresses. we need to update the new node to have all the other nodes as a peer and we need to add the new node as a peer to all existing nodes:
   * copy all the peers. remove the new node from the string. open `kava-<new_node_index>/config/config.yml` and set `persistent_peers` to all other nodes
   * add the new node id & address to the `persistent_peers` of all the already-existing node configs

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

1. resume your regularly scheduled meganode running
