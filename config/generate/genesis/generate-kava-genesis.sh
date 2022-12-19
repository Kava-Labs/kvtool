#! /bin/bash
# This script builds the genesis.json for the master kava template
# It requires moreutils (for sponge) & jq
# NOTE: many values here were just copied directly. there are much better ways
#  that they could be abstracted and reduced down to simple to modify lists of values.

set -e

mkdir -p scratch

DATA=./scratch/.kava
DEST=./config/templates/kava/master/initstate/.kava
ADDRESSES=./config/common/addresses.json

BINARY="kava --home $DATA"

# set-app-state loads a json file in this directory and sets the app_state value matched by the filename.
# files may contains bash variables exported from this file.
# example: set-app-state bep3.params.asset_params
# this will set .app_state.bep3.params.asset_params to the contents of bep3.params.asset_params.json
# optionally, include a jq manipulation of the contents:
# example: set-app-state issuance.params.assets '[ .[] | { hello: . } ]'
function set-app-state {
  app_state_path=".app_state.$1"
  file="./config/generate/genesis/$1.json"
  manipulation="$2"

  # error if expected state file doesn't exist.
  if [ ! -e "$file" ]; then
    echo 'set-app-state error: '"$file"' does not exist.'
    exit 1
  fi

  # apply manipulation to contents if present, otherwise, use file contents.
  if [ -z "$manipulation" ]; then
    contents=$(cat $file)
  else
    contents=$(jq "$manipulation" "$file")
  fi
  # variable substitution for contents! allows use of $vars in the json files.
  # variables must be `export`ed before this func is called.
  contents=$(echo "$contents" | envsubst)

  jq "$app_state_path"' = '"$contents" $DATA/config/genesis.json | sponge $DATA/config/genesis.json
}

###########################
##### INIT CHAIN HOME #####
###########################
# remove any old state and config
rm -rf $DATA

# Create new data directory, overwriting any that alread existed
chainID="kavalocalnet_8888-1"
$BINARY init validator --chain-id $chainID

# Copy over original validator keys
cp $DEST/config/node_key.json $DATA/config/node_key.json
cp $DEST/config/priv_validator_key.json $DATA/config/priv_validator_key.json

####################
##### APP.TOML #####
####################
# hacky enable of rest api
sed -i '' 's/enable = false/enable = true/g' $DATA/config/app.toml

# Set evm tracer to json
sed -i '' 's/tracer = ""/tracer = "json"/g' $DATA/config/app.toml

# Enable full error trace to be returned on tx failure
sed -i '' '/iavl-cache-size/a\
trace = true' $DATA/config/app.toml

# Enable unsafe CORs
sed -i '' 's/enabled-unsafe-cors = false/enabled-unsafe-cors = true/g' $DATA/config/app.toml

#######################
##### CLIENT.TOML #####
#######################
# Set client chain id
sed -i '' 's/chain-id = ""/chain-id = "kavalocalnet_8888-1"/g' $DATA/config/client.toml

#######################
##### CONFIG.TOML #####
#######################
# lower default commit timeout
sed -i '' 's/timeout_commit = "5s"/timeout_commit = "1s"/g' $DATA/config/config.toml


#########################
##### CONFIGURATION #####
#########################
# avoid having to use password for keys
$BINARY config keyring-backend test

# set broadcast-mode to block
$BINARY config broadcast-mode block

###########################
##### SETUP ADDRESSES #####
###########################
# add-genesis-account initiates an account with funds in genesis.json
function add-genesis-account {
  account_name_or_addr=$1
  initial_funds=$2

  # NOTE: this successfully sets the account's initial funds.
  # however, the `auth.accounts` item added is always an EthAccount.
  # THIS SCRIPT OVERRIDES ALL `auth.accounts` AFTER ALL add-genesis-account calls are made
  $BINARY add-genesis-account $account_name_or_addr $initial_funds
}
# add-genesis-account-key initiates an account with funds & adds the user's mnemonic to the keyring
function add-genesis-account-key {
  account_name=$1
  mnemonic_path=$2
  initial_funds=$3

  mnemonic=$(jq -r "$mnemonic_path.mnemonic" $ADDRESSES)

  echo "$mnemonic" | $BINARY keys add $account_name --recover
  add-genesis-account $account_name $initial_funds
}
function get-address {
  path=$1
  echo $(jq -r "$path.address" $ADDRESSES)
}

# Setup Validator
export validator=$(get-address '.kava.validators[0]')
export valoper=$(jq -r '.kava.validators[0].val_address' $ADDRESSES)
add-genesis-account-key validator '.kava.validators[0]' 1000000000ukava

$BINARY gentx validator 1000000000ukava \
--chain-id="$chainID" \
--moniker="validator"

$BINARY collect-gentxs

# Bep3 Deputies
export bnb_cold=$(get-address '.kava.deputys.bnb.cold_wallet')
add-genesis-account-key deputy-bnb-cold '.kava.deputys.bnb.cold_wallet' 1000000000000ukava
export bnb_deputy=$(get-address '.kava.deputys.bnb.hot_wallet')
add-genesis-account-key deputy-bnb-hot '.kava.deputys.bnb.hot_wallet' 1000000000000ukava

export btcb_cold=$(get-address '.kava.deputys.btcb.cold_wallet')
add-genesis-account-key deputy-btcb-cold '.kava.deputys.btcb.cold_wallet' 1000000000000ukava
export btcb_deputy=$(get-address '.kava.deputys.btcb.hot_wallet')
add-genesis-account-key deputy-btcb-hot '.kava.deputys.btcb.hot_wallet' 1000000000000ukava

export xrpb_cold=$(get-address '.kava.deputys.xrpb.cold_wallet')
add-genesis-account-key deputy-xrpb-cold '.kava.deputys.xrpb.cold_wallet' 1000000000000ukava
export xrpb_deputy=$(get-address '.kava.deputys.xrpb.hot_wallet')
add-genesis-account-key deputy-xrpb-hot '.kava.deputys.xrpb.hot_wallet' 1000000000000ukava

export busd_cold=$(get-address '.kava.deputys.busd.cold_wallet')
add-genesis-account-key deputy-busd-cold '.kava.deputys.busd.cold_wallet' 1000000000000ukava
export busd_deputy=$(get-address '.kava.deputys.busd.hot_wallet')
add-genesis-account-key deputy-busd-hot '.kava.deputys.busd.hot_wallet' 1000000000000ukava

# Users
export generic_0=$(get-address .kava.users.generic_0)
add-genesis-account-key generic-0 '.kava.users.generic_0' 1000000000000ukava
export generic_1=$(get-address .kava.users.generic_1)
add-genesis-account-key generic-1 '.kava.users.generic_1' 1000000000000ukava
export generic_2=$(get-address .kava.users.generic_2)
add-genesis-account-key generic-2 '.kava.users.generic_2' 1000000000000ukava
export vesting_periodic=$(get-address .kava.users.vesting_periodic)
add-genesis-account-key vesting-periodic '.kava.users.vesting_periodic' 10000000000ukava

export whale=$(get-address '.kava.users.whale')
whalefunds=1000000000000ukava,10000000000000000bkava-"$valoper",10000000000000000bnb,10000000000000000btcb,10000000000000000busd,1000000000000000000hard,1000000000000000000swp,10000000000000000usdx,10000000000000000xrpb
add-genesis-account-key whale '.kava.users.whale' $whalefunds
# dev-wallet! key is in 1pass.
export devwallet=$(jq -r '.kava.users.dev_wallet.address' $ADDRESSES)
add-genesis-account $devwallet $whalefunds

# Misc
export oracle=$(get-address '.kava.oracles[0]')
add-genesis-account-key oracle '.kava.oracles[0]' 1000000000000ukava
export committee=$(get-address '.kava.committee_members[0]')
add-genesis-account-key committee '.kava.committee_members[0]' 1000000000000ukava

# Accounts without keys
# issuance module
add-genesis-account kava1cj7njkw2g9fqx4e768zc75dp9sks8u9znxrf0w 1000000000000ukava,1000000000000swp,1000000000000hard
# swap module
add-genesis-account kava1mfru9azs5nua2wxcd4sq64g5nt7nn4n8s2w8cu 5000000000ukava,200000000btcb,1000000000hard,5000000000swp,103000000000usdx

# ???
# idk what these accounts are, but they were funded in genesis before this was automated.
add-genesis-account kava123fxg0l602etulhhcdm0vt7l57qya5wjcrwhzz 100000000000bnb,1000000000ukava
add-genesis-account kava1ak4pa9z2aty94ze2cs06wsdnkg9hsvfkp40r02 100000000000bnb,2000000000ukava
add-genesis-account kava10wlnqzyss4accfqmyxwx5jy5x9nfkwh6qm7n4t 1000000000ukava
add-genesis-account kava15tmj37vh7ch504px9fcfglmvx6y9m70646ev8t 1000000000ukava

# override `auth.accounts` array.
# DO NOT CALL `add-genesis-account` AFTER HERE UNLESS IT IS AN EthAccount
# this uses all exported account variables.
account_data_dir='./config/generate/genesis/auth.accounts'
account_data=$(jq -s '
  [ .[0][] | {
      "@type": "/cosmos.auth.v1beta1.BaseAccount",
      "account_number": "0",
      "address": .,
      "pub_key": null,
      "sequence": "0"
    }
  ]
  + [.[1]]
  + .[2]
' $account_data_dir/base-accounts.json $account_data_dir/vesting-periodic.json $account_data_dir/eth-accounts.json |
  envsubst
)
jq ".app_state.auth.accounts"' = '"$account_data" $DATA/config/genesis.json | sponge $DATA/config/genesis.json


############################
##### MODULE APP STATE #####
############################

# Replace stake with ukava
sed -i '' 's/stake/ukava/g' $DATA/config/genesis.json
# Replace the default evm denom of aphoton with ukava
sed -i '' 's/aphoton/akava/g' $DATA/config/genesis.json

# Zero out the total supply so it gets recalculated during InitGenesis
jq '.app_state.bank.supply = []' $DATA/config/genesis.json|sponge $DATA/config/genesis.json

# x/bep3 assets
set-app-state bep3.params.asset_params

# x/cdp params
jq '.app_state.cdp.params.global_debt_limit.amount = "53000000000000"' $DATA/config/genesis.json | sponge $DATA/config/genesis.json
set-app-state cdp.params.collateral_params

# x/committee (uses $committee)
set-app-state committee.committees

# x/distribution: disable community tax
jq '.app_state.distribution.params.community_tax = "0.000000000000000000"' $DATA/config/genesis.json | sponge $DATA/config/genesis.json

# x/earn
set-app-state earn.params.allowed_vaults

# x/evm
# disable london fork
jq '.app_state.evm.params.chain_config.london_block = null' $DATA/config/genesis.json | sponge $DATA/config/genesis.json
jq '.app_state.evm.params.chain_config.arrow_glacier_block = null' $DATA/config/genesis.json | sponge $DATA/config/genesis.json
jq '.app_state.evm.params.chain_config.merge_fork_block = null' $DATA/config/genesis.json | sponge $DATA/config/genesis.json
# setup accounts
set-app-state evm.accounts

# x/evmutil: enable conversion pairs
jq '.app_state.evmutil.params.enabled_conversion_pairs = [
  {
    "kava_erc20_address": "0xeA7100edA2f805356291B0E55DaD448599a72C6d",
    "denom": "erc20/multichain/usdc"
  }
]' $DATA/config/genesis.json | sponge $DATA/config/genesis.json

# x/feemarket: Disable fee market
jq '.app_state.feemarket.params.no_base_fee = true' $DATA/config/genesis.json|sponge $DATA/config/genesis.json

# x/hard: money markets (Kava Lend)
set-app-state hard.params.money_markets

# x/incentive params
set-app-state incentive.params

# # TODO: are nonempty swap claims important?

# x/issuance assets
set-app-state issuance.params.assets '
  [.[] | {
    owner: "'$devwallet'",
    denom: .,
    blocked_addresses: [],
    paused: false,
    blockable: false,
    rate_limit: {
      active: false,
      limit: "0",
      time_period: "0s"
    }
}]'

# x/kavamint: set kavamint inflation params
jq '.app_state.kavamint.params.community_pool_inflation = "0.150000000000000000"' $DATA/config/genesis.json | sponge $DATA/config/genesis.json
jq '.app_state.kavamint.params.staking_rewards_apy = "0.200000000000000000"' $DATA/config/genesis.json | sponge $DATA/config/genesis.json

# x/pricefeed (uses $oracle)
set-app-state pricefeed

# x/savings supported denoms
jq '.app_state.savings.params.supported_denoms =
  [ "bkava-'$valoper'", "usdx", "ukava", "hard", "swp", "bkava", "erc20/multichain/usdc" ]' \
  $DATA/config/genesis.json | sponge $DATA/config/genesis.json

# x/swap (uses $whale)
set-app-state swap

############################
##### MOVE FILE ASSETS #####
############################
$BINARY validate-genesis $DATA/config/genesis.json

cp $DATA/config/app.toml $DEST/config/app.toml
cp $DATA/config/client.toml $DEST/config/client.toml
cp $DATA/config/config.toml $DEST/config/config.toml
cp $DATA/config/genesis.json $DEST/config/genesis.json

rm -fr $DEST/config/gentx
cp -r $DATA/config/gentx $DEST/config/gentx

###################
##### CLEANUP #####
###################
rm -fr ./scratch
