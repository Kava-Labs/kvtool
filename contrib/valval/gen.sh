#!/bin/bash

peers=()
num_validators=$(tr -d '[:space:]' <NUM_VALIDATORS)

# handle difference between GNU (ubuntu) and BSD (macos) sed
sed_edit() {
  if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    # GNU sed
    sed -i "$@"
  else
    # BSD sed
    sed -i '' "$@"
  fi
}

echo generating validator home directories with kava@"$(kava version)"

mkdir -p keys
for ((i = 1; i <= num_validators; i++)); do
  home=kava-$i

  already_generated=false
  # preserve node keys across runs of this script so they aren't regenerated every time
  if [ -d "$home" ]; then
    already_generated=true
    mv $home/config/priv_validator_key.json temp_priv_val_key.json
    mv $home/config/node_key.json temp_node_key.json
    rm -fr $home
  fi

  # (re)generate the kava home directory
  kava init val$i --home $home --chain-id kavamirror_2221-1 >/dev/null 2>&1

  # preserve original node keys if they existed
  if $already_generated; then
    mv -f temp_priv_val_key.json $home/config/priv_validator_key.json
    mv -f temp_node_key.json $home/config/node_key.json
  fi


  rm -f $home/config/genesis.json
  cp shared/init-data-directory.sh $home/config/init-data-directory.sh
  cp shared/priv_validator_state.json.example $home/config/priv_validator_state.json.example

  # move keys to shared dir (for use in update-validator-genesis)
  cp $home/config/priv_validator_key.json keys/priv_validator_key_$(($i - 1)).json
  peers+=("$(kava tendermint show-node-id --home $home)@$home:26656")

  # force use of rocksdb
  sed_edit -e "s#^db_backend = .*#db_backend = \"rocksdb\"#" $home/config/config.toml

  # update max_num_outbound_peers to be number of other validators
  sed_edit -e "s#^max_num_outbound_peers = .*#max_num_outbound_peers = $((num_validators - 1))#" $home/config/config.toml

  # include rocksdb configuration tweaks
  {
    echo
    echo '[rocksdb]'
    echo
    echo 'max_open_files = 4096'
    echo 'block_size = 16384'
  } >> $home/config/app.toml
done

echo Updating peer network topology.

# for performance, we set the first validators peers to all other validators
# all other validators are given only the first validator as a peer.

first_validator_addr="${peers[0]}" # the peer address of the first validator
persistent_peers=$(IFS=, echo "${peers[@]:1}")  # comma-delimited list of all other validator peer addresses
peer_list=$(echo "${persistent_peers[*]}" | tr ' ' ',')

# set first validator to have all other validators as peers
echo setting first validator to have \""$peer_list"\" as peers
sed_edit -e "s#^persistent_peers = .*#persistent_peers = \"$peer_list\"#" "kava-1/config/config.toml"

echo setting all other validators to have \""$first_validator_addr"\" as peer
# set all other validators to have 1st as their only peer
for ((i = 2; i <= num_validators; i++)); do
  configtoml=kava-$i/config/config.toml
  # replace existing persistent peers
  sed_edit -e "s#^persistent_peers = .*#persistent_peers = \"$first_validator_addr\"#" "$configtoml"
done
