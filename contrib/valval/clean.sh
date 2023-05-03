#!/bin/bash

for i in {1..13}
do
  home=kava-$i

  rm -rf $home/data
  rm -rf $home/config/addrbook.json
done
