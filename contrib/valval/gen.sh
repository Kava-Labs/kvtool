#!/bin/bash

peers=()
num_validators=$(tr -d '[:space:]' <NUM_VALIDATORS)
there_is_a_new_validator=false

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
done

if [ "$there_is_a_new_validator" = true ]; then
  echo The number of validators has changed. Updating the persistent peers of each one.

  for ((i = 1; i <= num_validators; i++)); do
    configtoml=kava-$i/config/config.toml
    echo "$configtoml"

    persistent_peers=()
    for ((j = 0; j < num_validators; j++)); do
      if [[ $j -ne $((i - 1)) ]]; then
        persistent_peers+=("${peers[j]}")
      fi
    done

    # make comma-delimited
    peer_list=$(echo "${persistent_peers[*]}" | tr ' ' ',')

    # replace existing persistent peers
    sed -i '' -e "s#^persistent_peers = .*#persistent_peers = \"$peer_list\"#" "$configtoml"
  done
else
  echo No change to the number of validators was made.
  printf "%s" "${peers[*]}"
fi
