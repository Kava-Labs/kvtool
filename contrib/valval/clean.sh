#!/bin/bash

for i in {1..11}
do
  home=kava-$i

  rm -rf $home/data
done
