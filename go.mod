module github.com/kava-labs/kvtool

go 1.13

require (
	github.com/Jeffail/gabs/v2 v2.6.0
	github.com/cosmos/cosmos-sdk v0.45.4
	github.com/kava-labs/kava v0.17.3
	github.com/otiai10/copy v1.6.0
	github.com/spf13/cobra v1.4.0
	github.com/tendermint/tendermint v0.34.19
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)

replace (
	// Use the cosmos keyring code
	github.com/99designs/keyring => github.com/cosmos/keyring v1.1.7-0.20210622111912-ef00f8ac3d76
	// Use cosmos-sdk fork with backported fix for unsafe-reset-all
	github.com/cosmos/cosmos-sdk => github.com/kava-labs/cosmos-sdk v0.45.4-kava.1
	// See https://github.com/cosmos/cosmos-sdk/pull/10401, https://github.com/cosmos/cosmos-sdk/commit/0592ba6158cd0bf49d894be1cef4faeec59e8320
	github.com/gin-gonic/gin => github.com/gin-gonic/gin v1.7.0
	// Use the cosmos modified protobufs
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	// Use rocksdb 7.1.2
	github.com/tendermint/tm-db => github.com/kava-labs/tm-db v0.6.7-kava.1
	// Use ethermint fork that respects min-gas-price with NoBaseFee true and london enabled
	github.com/tharsis/ethermint => github.com/Kava-Labs/ethermint v0.14.0-kava-v17.2
	// Make sure that we use grpc compatible with cosmos
	google.golang.org/grpc => google.golang.org/grpc v1.33.2
)
