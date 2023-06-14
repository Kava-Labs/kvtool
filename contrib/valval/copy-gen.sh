#!/bin/bash

num_validators=$(tr -d '[:space:]' <NUM_VALIDATORS)

for ((i = 1; i <= num_validators; i++)); do
  home=kava-$i

  echo copying genesis to "$home"
  cp updated-genesis.json $home/config/genesis.json
done
