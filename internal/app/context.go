package app

import (
	"plutolab.org/kosh/internal/config"
	"plutolab.org/kosh/internal/core"
	"plutolab.org/kosh/internal/storage"
)

type Context struct {
	Config  *config.Config
	Store   storage.Store
	Vault   core.VaultService
	Profile core.ProfileService
}
