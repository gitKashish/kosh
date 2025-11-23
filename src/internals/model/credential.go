package model

import (
	"time"

	"github.com/gitKashish/kosh/src/internals/encoding"
)

type Credential struct {
	Id          int
	Label       string
	User        string
	AccessCount int

	// crypto data
	Secret    string
	Ephemeral string
	Nonce     string

	// timestamps
	CreatedAt  time.Time
	UpdatedAt  time.Time
	AccessedAt time.Time
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

type CredentialData struct {
	Label     string
	User      string
	Secret    []byte
	Ephemeral []byte
	Nonce     []byte
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

type CredentialSummary struct {
	Id          int
	Label       string
	User        string
	AccessCount int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	AccessedAt  time.Time
}
