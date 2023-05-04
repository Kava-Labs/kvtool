#!/bin/bash

peers=()
numPeers=13

mkdir -p keys
for i in $(seq 1 $numPeers)
do
  home=kava-$i

  if [ ! -d $home ]
  then
    kava init val$i --home $home --chain-id kavamirror_2221-1 > /dev/null 2>&1

    rm -rf $home/data
    rm $home/config/genesis.json
    cp kava-1/config/init-data-directory.sh $home/config/init-data-directory.sh
    cp kava-1/config/priv_validator_state.json.example $home/config/priv_validator_state.json.example
  else
    echo "kava-$i already exists"
  fi

  cp $home/config/priv_validator_key.json keys/priv_validator_key_$(($i-1)).json
  peers+=($(kava tendermint show-node-id --home $home)@$home:26656)
done

echo "Updating persistent_peers"
echo $(printf ",%s" "${peers[@]}")

for i in $(seq 1 $numPeers)
do
  home=kava-$i

  peersStr=$(echo ",%s" "${peers[@]}")
  # Remove self from peers
  # Filter out own match
  # Join with commas
  # Remove trailing comma
  persistent_peers=$(echo "$peersStr" | grep -oE '[0-9a-f]*@kava-[0-9]+:26656' | grep -v "kava-${i}:"  | tr '\n' ',' | sed 's/,$//')

  # Replace in config file
  sed -i'' "s/persistent_peers = \".*\"/persistent_peers = \"${persistent_peers}\"/" "${home}/config/config.toml"
done

echo "Done!"
