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
