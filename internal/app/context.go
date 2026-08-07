package app

import (
	"git.plutolab.org/plutolab/kosh/internal/config"
	"git.plutolab.org/plutolab/kosh/internal/core"
	"git.plutolab.org/plutolab/kosh/internal/storage"
)

type Context struct {
	Config  *config.Config
	Store   storage.Store
	Vault   core.VaultService
	Profile core.ProfileService
}
