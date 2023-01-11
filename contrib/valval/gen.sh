#!/bin/bash

peers=()

mkdir -p keys
for i in {1..11}
do
  home=kava-$i

  if [ ! -d $home ]
  then
    kava init val$i --home $home --chain-id kavamirror_2221-1 > /dev/null 2>&1

    rm -rf $home/data
    rm $home/config/genesis.json
    cp kava-1/config/init-data-directory.sh $home/config/init-data-directory.sh
    cp kava-1/config/priv_validator_state.json.example $home/config/priv_validator_state.json.example
  fi

  cp $home/config/priv_validator_key.json keys/priv_validator_key_$(($i-1)).json
  peers+=($(kava tendermint show-node-id --home $home)@$home:26656)
done

echo $(printf ",%s" "${peers[@]}")
