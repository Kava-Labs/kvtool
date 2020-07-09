package binance

import (
	"strings"

	"github.com/tendermint/tendermint/crypto/tmhash"
)

func CalculateSwapID(randomNumberHash []byte, sender AccAddress, senderOtherChain string) []byte {
	senderOtherChain = strings.ToLower(senderOtherChain)
	data := randomNumberHash
	data = append(data, []byte(sender)...)
	data = append(data, []byte(senderOtherChain)...)
	return tmhash.Sum(data)
}
