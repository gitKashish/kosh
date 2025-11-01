package model

import "github.com/gitKashish/kosh/src/internals/encoding"

type Credential struct {
	Label     string
	User      string
	Secret    string
	Ephemeral string
	Nonce     string
}

type CredentialData struct {
	Label     string
	User      string
	Secret    []byte
	Ephemeral []byte
	Nonce     []byte
}

func (c *Credential) GetRawData() *CredentialData {
	return &CredentialData{
		Label:     c.Label,
		User:      c.User,
		Secret:    encoding.DecodeBase64String(c.Secret),
		Ephemeral: encoding.DecodeBase64String(c.Ephemeral),
		Nonce:     encoding.DecodeBase64String(c.Nonce),
	}
}

func (c *CredentialData) EncodeToString() *Credential {
	return &Credential{
		Label:     c.Label,
		User:      c.User,
		Secret:    encoding.EncodeToBase64String(c.Secret),
		Ephemeral: encoding.EncodeToBase64String(c.Ephemeral),
		Nonce:     encoding.EncodeToBase64String(c.Nonce),
	}
}
