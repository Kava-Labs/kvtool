# kvtool

Assorted dev tools for working with the kava blockchain.

## Installation

```bash
make install
```

## Initialization: kvtool testnet

Note: The current mainnet version of kava is `v0.16.0`. To start a local testnet with the current mainnet version use `--kava.configTemplate v0.16`. To start a local testnet with the latest v44 version, use `--kava configTemplate master`

Option 1:

The `kvtool testnet bootstrap` command starts a local Kava blockchain as a background docker container called `generated_kavanode_1`. The bootstrap command only starts the Kava blockchain and Kava REST server services.

```bash

# Start new testnet
kvtool testnet bootstrap --kava.configTemplate master
```

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

`--ibc`: Run Kava testnet with an additional IBC chain

Example:

```bash
# Run Kava testnet with an additional IBC chain
kvtool testnet bootstrap --kava.configTemplate master --ibc
```

`--geth`: Run a go-ethereum node alongside the Kava testnet. The geth node is
initialized with the Kava Bridge contract and test ERC20 tokens.

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

You can test the set up and alias by executing a sample query:

```bash
dkava status
dkava q cdp params
```

To send transactions you'll need to recover a user account in the dockerized environment. Valid mnemonics for the blockchains be found in the `config/common/addresses.yaml` file.

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
