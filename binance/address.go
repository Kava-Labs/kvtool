package binance

import (
	"errors"
	"fmt"

	"github.com/tendermint/tendermint/libs/bech32"
)

const Prefix = "bnb"

type AccAddress []byte

func (bz AccAddress) String() string {
	bech32Addr, err := bech32.ConvertAndEncode(Prefix, bz)
	if err != nil {
		panic(err)
	}
	return bech32Addr
}

// AccAddressFromBech32 to create an AccAddress from a bech32 string
func AccAddressFromBech32(address string) (addr AccAddress, err error) {
	bz, err := GetFromBech32(address, Prefix)
	if err != nil {
		return nil, err
	}
	return AccAddress(bz), nil
}

// GetFromBech32 to decode a bytestring from a bech32-encoded string
func GetFromBech32(bech32str, prefix string) ([]byte, error) {
	if len(bech32str) == 0 {
		return nil, errors.New("decoding bech32 address failed: must provide an address")
	}
	hrp, bz, err := bech32.DecodeAndConvert(bech32str)
	if err != nil {
		return nil, err
	}

	if hrp != prefix {
		return nil, fmt.Errorf("invalid bech32 prefix. Expected %s, Got %s", prefix, hrp)
	}

	return bz, nil
}
