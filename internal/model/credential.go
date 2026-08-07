package model

import (
	"log/slog"
	"time"

	"git.plutolab.org/plutolab/kosh/internal/encoding"
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

func (c Credential) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("Id", c.Id),
		slog.String("Label", c.Label),
		slog.String("User", c.User),
		slog.Int("AccessCount", c.AccessCount),

		slog.String("Secret", "[REDACTED]"),
		slog.String("Ephemeral", "[REDACTED]"),
		slog.String("Nonce", "[REDACTED]"),
	)
}

type CredentialData struct {
	Id        int
	Label     string
	User      string
	Secret    []byte
	Ephemeral []byte
	Nonce     []byte
}

func (c CredentialData) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("Id", c.Id),
		slog.String("Label", c.Label),
		slog.String("User", c.User),

		slog.String("Secret", "[REDACTED]"),
		slog.String("Ephemeral", "[REDACTED]"),
		slog.String("Nonce", "[REDACTED]"),
	)
}

func (c *CredentialData) EncodeToString() *Credential {
	return &Credential{
		Id:        c.Id,
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

func (c CredentialSummary) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("Id", c.Id),
		slog.String("Label", c.Label),
		slog.String("User", c.User),
		slog.Int("AccessCount", c.AccessCount),
		slog.Time("CreatedAt", c.CreatedAt),
		slog.Time("UpdatedAt", c.UpdatedAt),
		slog.Time("AccessedAt", c.AccessedAt),
	)
}
