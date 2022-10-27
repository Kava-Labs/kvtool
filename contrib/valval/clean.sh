#!/bin/bash

for i in {1..10}
do
  home=kava-$i

  rm -rf $home/data
done

