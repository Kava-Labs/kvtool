#!/bin/bash

for i in {1..10}
do
  home=kava-$i

  cp $home/config/priv_validator_key.json keys/priv_validator_key_$((i-1)).json
done
