# kvtool

Assorted dev tools for working with the kava blockchain.

To get started with running a local kava network, check out our docs on [Getting Started](https://docs.kava.io/docs/cosmos/getting-started).

## Installation

```bash
make install
```

## Initialization: kvtool testnet

Note that the most accurate documentation lives in the CLI itself. It's recommended you read through `kvtool testnet bootstrap --help`.

Option 1:

The `kvtool testnet bootstrap` command starts a local Kava blockchain as a
background docker container called `generated-kavanode-1`. The bootstrap command
only starts the Kava blockchain and Kava REST server services.

```bash
# Start new testnet
kvtool testnet bootstrap --kava.configTemplate master
```

The endpoints are exposed to localhost:

* RPC: http://localhost:26657
* REST: http://localhost:1317
* GRPC: http://localhost:9090
* GRPC Websocket: http://localhost:9091
* EVM JSON-RPC: http://localhost:8545
* EVM Websocket: http://localhost:8546

Option 2:

To generate a testnet for kava, binance chain, and a deputy that relays swaps between them:

```bash
# Generate a new kvtool configuration based off template files
kvtool testnet gen-config kava binance deputy --kava.configTemplate master

# Pull latest docker images. Docker must be running.
cd ./full_configs/generated && docker-compose pull

# start the testnet
kvtool testnet up

# When finished with usage, shut down the processes
kvtool testnet down
```

### Flags

Additional flags can be added when initializing a testnet to add additional
services:

`--ibc`: Run Kava testnet with an additional IBC chain. The IBC chain runs in the container named `ibcnode`. It has primary denom `ukava`.

Example:

```bash
# Run Kava testnet with an additional IBC chain
kvtool testnet bootstrap --kava.configTemplate master --ibc
```

`--geth`: Run a go-ethereum node alongside the Kava testnet. The geth node is
initialized with the Kava Bridge contract and test ERC20 tokens. The Kava EVM
also includes Multicall contracts deployed. The contract addresses can be found
on the [Kava-Labs/kava-bridge](https://github.com/Kava-Labs/kava-bridge#development)
README.

Example:

```bash
# Run the testnet with a geth node in parallel
kvtool testnet bootstrap --kava.configTemplate master --geth
```

Geth node ports are **not** default, as the Kava EVM will use default JSON-RPC
ports:

Kava EVM RPC Ports:

* HTTP JSON-RPC: `8545`
* WS-RPC port: `8546`

Geth RPC Ports:

* HTTP JSON-RPC: `8555`
* WS-RPC port: `8556`

To connect to the associated Ethereum wallet with Metamask, setup a new network with the following parameters:
* New RPC URL: `http://localhost:8555`
* Chain ID: `88881` (configured from the [genesis](config/templates/geth/initstate/genesis.json#L3))
* Currency Symbol: `ETH`

Finally, connect the mining account by importing the JSON config in [this directory](config/templates/geth/initstate/.geth/keystore)
with [this password](config/templates/geth/initstate/eth-password).

## Automated Chain Upgrade

Kvtool supports running upgrades on a chain. To do this requires the kava final docker image to have a registered upgrade handler.
The upgrade will start a chain with the docker container tag from `--upgrade-base-image-tag`. Once it reaches height `--upgrade-height`, it halts the chain for an upgrade named `--upgrade-name`. At that point, the container is restated with the desired container: `KAVA_TAG` if defined, of if not defined, the default tag for the config template.

**Example**:
Test a chain upgrade from v0.19.2 -> v0.21.0 at height 15.

Using an overridden docker image tag:
```
$ KAVA_TAG=v0.21.0 kvtool testnet bootstrap --upgrade-name v0.21.0 --upgrade-height 15 --upgrade-base-image-tag v0.19.2
```

Using a config template:
```
Test a chain upgrade from v0.19.2 -> v0.21.0:
$ kvtool testnet bootstrap --kava.configTemplate v0.21 --upgrade-name v0.21.0 --upgrade-height 15 --upgrade-base-image-tag v0.19.2
```

## Usage: kvtool testnet

REST APIs for both blockchains are exposed on localhost:

- Kava: http://localhost:1317
- Binance Chain: http://localhost:8080

You can also interact with the blockchain using the `kava` command line. In a
new terminal window, set up an alias to `kava` on the dockerized kava node and
use it to send a query.

```bash
# Add an alias to the dockerized kava cli
alias dkava='docker exec -it generated_kavanode_1 kava'

# Confirm that the alias has been added
alias kava

# For versions before v0.16.x
alias dkvcli='docker exec -it generated_kavanode_1 kvcli'
```

Note that for some architectures or docker versions, the containers are generated with hyphens (`-`) instead of underscores (`_`).

You can test the set up and alias by executing a sample query:

```bash
dkava status
dkava q cdp params
```

To send transactions you'll need to recover a user account in the dockerized environment. Valid mnemonics for the blockchains be found in the `config/common/addresses.json` file.

```bash
# Recover user account
dkava keys add user --recover
# Enter mnemonic
arrive guide way exit polar print kitchen hair series custom siege afraid shrug crew fashion mind script divorce pattern trust project regular robust safe
```

Test transaction sending by transferring some coins to yourself.

```bash
# Query the recovered account's address
dkava keys show user -a
# Send yourself some coins by creating a send transaction with your address as both sender and receiver
dkava tx bank send [user-address] [user-address] 1000000ukava --from user
# Enter 'y' to confirm the transaction
confirm transaction before signing and broadcasting [y/N]:

# Check transaction result by tx hash
dkava q tx [tx-hash]
```

## Shut down: kvtool testnet

When you're done make sure to shut down the kvtool testnet. Always shut down the kvtool testnets before pulling the latest image from docker, otherwise you may experience errors.

```bash
kvtool testnet down
```

# Updating kava genesis

When new versions of kava are released, they often involve changes to genesis.
The kava `master` template includes a genesis.json that is generated from a pure state:
* Ensure the desired version of `kava` is in your path as `kava`
* Run `make generate-kava-genesis`
* The script will create a genesis with desired accounts & validator
* Updates to the genesis should be made in [`update-kava-genesis.sh`](./config/generate/genesis/generate-kava-genesis.sh)
