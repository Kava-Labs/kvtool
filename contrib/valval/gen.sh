#!/bin/bash

peers=()
num_validators=$(tr -d '[:space:]' <NUM_VALIDATORS)
there_is_a_new_validator=true

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

mkdir -p keys
for ((i = 1; i <= num_validators; i++)); do
  home=kava-$i

  # only generate home dirs that don't exist
  if [ ! -d $home ]; then
    there_is_a_new_validator=true
    kava init val$i --home $home --chain-id kavamirror_2221-1 >/dev/null 2>&1

    rm -rf $home/data
    rm $home/config/genesis.json
    cp kava-1/config/init-data-directory.sh $home/config/init-data-directory.sh
    cp kava-1/config/priv_validator_state.json.example $home/config/priv_validator_state.json.example
  fi

  cp $home/config/priv_validator_key.json keys/priv_validator_key_$(($i - 1)).json
  peers+=("$(kava tendermint show-node-id --home $home)@$home:26656")

  # force use of rocksdb
  sed_edit -e "s#^db_backend = .*#db_backend = \"rocksdb\"#" $home/config/config.toml

  # update max_num_outbound_peers to be number of other validators
  sed_edit -e "s#^max_num_outbound_peers = .*#max_num_outbound_peers = $((num_validators - 1))#" $home/config/config.toml
done

if [ "$there_is_a_new_validator" = true ]; then
  echo The number of validators has changed. Updating peer network topology.

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
else
  echo No change to the number of validators was made.
  printf "%s" "${peers[*]}"
fi
