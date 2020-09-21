#! /bin/bash
set -e

# remove any exisiting keys and chain state.
# WARNING this will delete private keys
rm -rf ~/.kvd
rm -rf ~/.kvcli

# setup kvcli for the localnet
kvcli config keyring-backend test
kvcli config chain-id kava-localnet
kvcli config broadcast-mode block

mnemonic[0]=$(jq '.kava.mnemonics[0]' config_templates/common/mnemonics.json | tr -d '"')
printf "${mnemonic[0]}\n" | kvcli keys add user0 --recover

mnemonic[1]=$(jq '.kava.mnemonics[1]' config_templates/common/mnemonics.json | tr -d '"')
printf "${mnemonic[0]}\n" | kvcli keys add user1 --recover

mnemonic[2]=$(jq '.kava.mnemonics[2]' config_templates/common/mnemonics.json | tr -d '"')
printf "${mnemonic[0]}\n" | kvcli keys add user2 --recover

mnemonic[3]=$(jq '.kava.mnemonics[3]' config_templates/common/mnemonics.json | tr -d '"')
printf "${mnemonic[0]}\n" | kvcli keys add user3 --recover

mnemonic[4]=$(jq '.kava.mnemonics[4]' config_templates/common/mnemonics.json | tr -d '"')
printf "${mnemonic[0]}\n" | kvcli keys add user4 --recover

mnemonic[5]=$(jq '.kava.mnemonics[5]' config_templates/common/mnemonics.json | tr -d '"')
printf "${mnemonic[0]}\n" | kvcli keys add user5 --recover

# move bnb onto chain

rndNum="f3fb64c0bb7d8aca716d43b9fd66c1f8ddd0a719e105573b443072b23533b216"
# generate this
timestamp="1600349462"
# use timestamp to calculate this (need to upgrade kvcli q bep3 calc-rnh)
rndNumHash="fa296b9b2b6bf160d57bb1af55aa8df3116e11b2c9770d5271c02f1cce8e76cf"
# specify rnh in this
# printf "password\n" | bnbcli token HTLT --amount 100000000:BNB --cross-chain --from user0 --height-span 20000 --recipient-addr bnb1zfa5vmsme2v3ttvqecfleeh2xtz5zghh49hfqe --recipient-other-chain kava1ypjp0m04pyp73hwgtc0dgkx0e9rrydecm054da --sender-other-chain kava1agcvt07tcw0tglu0hmwdecsnuxp2yd45f3avgm --chain-id Binance-Chain-Tigris --node tcp://localhost:26658

# calculate swapID using kvtool (need to upgrade to allow non mainnet deputy addresses)
swapID=""
# kvcli tx bep3 claim $swapID $rndNum --from user0

# Alternatively move everything to go:
# Need to import binance and optionally kava go-sdk. But kvtool needs kava, so really need to do the forking of binance go-sdk.

# Alternatively isolate this script into it's own go module for time being.

# Alternatively, just add cdps and ignore swaps for now.