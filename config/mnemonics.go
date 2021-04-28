package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

func LoadKavaWallets(filename string) (NamedWallets, error) {
	var accountJSON KavaAccounts
	bz, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bz, &accountJSON)
	if err != nil {
		return nil, err
	}
	var wallets []NamedWallet
	for _, wallet := range accountJSON.Kava.Wallets {
		wallets = append(wallets, wallet)
	}
	return wallets, nil
}

type KavaAccounts struct {
	Kava Kava `json:"kava"`
}
type NamedWallet struct {
	Name     string `json:"name" yaml:"name"`
	Address  string `json:"address" yaml:"address"`
	Mnemonic string `json:"mnemonic" yaml:"mnemonic"`
}
type NamedValidatorWallet struct {
	NamedWallet `json:"named_wallet" yaml:"named_wallet"`

	ConsPubkey string `json:"cons_pubkey" yaml:"cons_pubkey"`
	ValAddress string `json:"val_address" yaml:"val_address"`
}
type Kava struct {
	Wallets    []NamedWallet          `json:"wallets" yaml:"wallets"`
	Validators []NamedValidatorWallet `json:"validators" yaml:"validators"`
}

func (nw NamedWallet) String() string {
	return fmt.Sprintf(`Name: %s
	Address: %s
	Mnemonic: %s`,
		nw.Name, nw.Address, nw.Mnemonic)
}

type NamedWallets []NamedWallet

func (nws NamedWallets) String() string {
	out := ""
	for _, nw := range nws {
		out += nw.String()
	}
	return out
}

func (nws NamedWallets) GetWalletByName(name string) (NamedWallet, error) {
	for _, nw := range nws {
		if name == nw.Name {
			return nw, nil
		}
	}
	return NamedWallet{}, fmt.Errorf("no wallet associated with name %s found", name)
}
