package model

import "github.com/gitKashish/kosh/src/internals/encoding"

type Vault struct {
	Salt      string
	PublicKey string
	Nonce     string
	Secret    string
}

type VaultData struct {
	Salt      []byte
	PublicKey []byte
	Nonce     []byte
	Secret    []byte
}

func (v *Vault) GetRawData() *VaultData {
	return &VaultData{
		Salt:      encoding.DecodeBase64String(v.Salt),
		PublicKey: encoding.DecodeBase64String(v.PublicKey),
		Nonce:     encoding.DecodeBase64String(v.Nonce),
		Secret:    encoding.DecodeBase64String(v.Secret),
	}
}

func (v *VaultData) EncodeToString() *Vault {
	return &Vault{
		Salt:      encoding.EncodeToBase64String(v.Salt),
		PublicKey: encoding.EncodeToBase64String(v.PublicKey),
		Nonce:     encoding.EncodeToBase64String(v.Nonce),
		Secret:    encoding.EncodeToBase64String(v.Secret),
	}
}
