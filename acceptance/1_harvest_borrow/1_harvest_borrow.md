# Harvest Borrow

Acceptance instructions for basic borrowing functionality on Harvest.

## Kvtool Testnet Setup

First, start a kvtool testnet that contains the latest kava-labs/kava master branch changes and the relevant feature genesis.json file.

```bash
# Copy feature genesis.json file to kvtool's kava genesis template
cp acceptance/1_harvest_borrow/genesis.json config/templates/kava/master/initstate/.kvd/config/genesis.json

# Generate a new kvtool configuration based off template files
kvtool testnet gen-config kava binance deputy --kava.configTemplate master

# Pull the latest version Kava docker image containing the feature changing
cd ./full_configs/generated && docker-compose pull

# Start testnet
cd ../.. && kvtool testnet up
```

You can test the set up by querying:

```bash
docker exec -it generated_kavanode_1 kvcli q harvest params
```

The output of the query should include a `money_markets` section at the bottom of the result.

## Chain State Setup

In order for USDX to be deposited and KAVA borrowed from HARD protocol the chain must have a valid state. First, recover the specified user account and load tokens into the Harvest lending module account by depositing some KAVA.

```bash
# Recover user account
docker exec -it generated_kavanode_1 kvcli keys add user --recover
# Enter mnemonic
arrive guide way exit polar print kitchen hair series custom siege afraid shrug crew fashion mind script divorce pattern trust project regular robust safe

# Load KAVA into Harvest module account
docker exec -it generated_kavanode_1 kvcli tx harvest deposit 10000000ukava lp --from user
```

## Feature Testing

Now, we can succesfully deposit and borrow funds from the money market.
```bash
# Deposit USDX and borrow KAVA
docker exec -it generated_kavanode_1 kvcli tx harvest deposit 10000000usdx lp --from user
docker exec -it generated_kavanode_1 kvcli tx harvest borrow 1000000ukava --from user

# Check results of borrow transaction
docker exec -it generated_kavanode_1 kvcli q tx [tx-hash]
```

## Shut down kvtool

When you're done testing, make sure to shut down the kvtool testnet:

```bash
kvtool testnet down
```
