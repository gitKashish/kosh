package constants

import "errors"

// Errors surfaced to the user.
//
// Each name says what went wrong with what, subject-first, and the message
// spells out the same thing in the same order, so ErrFailedToSaveCredential
// always reads "failed to save credential". Operations that broke use "failed
// to <verb> <subject>"; states that are simply wrong are stated as they are.
//
// Messages are lowercase, unpunctuated and free of articles, because the UI
// prints them after its own prefix.
var (
	// Input and arguments.
	ErrInvalidArguments  = errors.New("invalid arguments")
	ErrIdMustBeInteger   = errors.New("id must be an integer")
	ErrFailedToReadInput = errors.New("failed to read input")

	// Vault and storage.
	ErrVaultNotInitialized       = errors.New("vault is not initialized")
	ErrTargetVaultNotInitialized = errors.New("target profile vault is not initialized")
	ErrFailedToInitializeVault   = errors.New("failed to initialize vault")
	ErrFailedToInitializeStore   = errors.New("failed to initialize vault storage")
	ErrFailedToFetchVaultInfo    = errors.New("failed to fetch vault info")

	// Master password and secret entry.
	ErrIncorrectMasterPassword = errors.New("incorrect master password")
	ErrPasswordDoesNotMatch    = errors.New("passwords do not match")
	ErrSecretDoesNotMatch      = errors.New("secrets do not match")

	// Credentials. ErrCredentialNotFound reports a lookup by id or by label and
	// user that hit nothing; ErrCredentialMatchNotFound reports a search or
	// filter that came back empty.
	ErrCredentialNotFound        = errors.New("credential not found")
	ErrCredentialMatchNotFound   = errors.New("no matching credential found")
	ErrCredentialAlreadyExists   = errors.New("credential already exists")
	ErrLabelCannotBeCommand      = errors.New("credential label cannot be a command name")
	ErrFailedToFetchCredential   = errors.New("failed to fetch credential")
	ErrFailedToSaveCredential    = errors.New("failed to save credential")
	ErrFailedToDeleteCredential  = errors.New("failed to delete credential")
	ErrFailedToDecryptCredential = errors.New("failed to decrypt credential")

	// Profiles.
	ErrInvalidProfileName        = errors.New("invalid profile name")
	ErrReservedProfileName       = errors.New("profile name is reserved by the operating system")
	ErrProfileDoesNotExist       = errors.New("profile does not exist")
	ErrProfileAlreadyExists      = errors.New("profile already exists")
	ErrFailedToFetchProfile      = errors.New("failed to fetch profile")
	ErrCannotDeleteActiveProfile = errors.New("cannot delete active profile")
	ErrCannotCopyToActiveProfile = errors.New("cannot copy credential to active profile")

	// Flow control: the user backed out rather than anything going wrong.
	ErrSearchCancelled  = errors.New("search cancelled")
	ErrOperationAborted = errors.New("operation aborted")
)
