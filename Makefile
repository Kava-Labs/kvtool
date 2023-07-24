install:
	go install -ldflags "-X github.com/kava-labs/kvtool/config/generate.ConfigTemplatesDir=$(CURDIR)/config/templates"

generate-kava-genesis:
	bash ./config/generate/genesis/generate-kava-genesis.sh

# when keys are added or changed, use me. we don'd replace keys by default because they include
# creation time, so they create noise by always creating a diff.
generate-kava-genesis-with-keys:
	REPLACE_ACCOUNT_KEYS=true bash ./config/generate/genesis/generate-kava-genesis.sh

generate-ibc-genesis:
	CHAIN_ID=kavalocalnet_8889-2 DEST=./config/templates/ibcchain/master/initstate/.kava DENOM=uatom SKIP_INCENTIVES=true bash ./config/generate/genesis/generate-kava-genesis.sh

# utility command to get posted_prices for pricefeed genesis w/ current market prices
# make -s get-updated-prices | pbcopy
# (-s flag omits the command in the output)
# then paste into ./config/generate/genesis/pricefeed.json's posted_prices
get-updated-prices:
	kava --node https://rpc.kava.io:443 q pricefeed prices --output json | \
	jq '.prices |= map( .oracle_address = "$$oracle" | .expiry = "2050-01-01T00:00:00Z" ) | .prices'
