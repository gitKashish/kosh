package constants

import "errors"

var (
	ErrInvalidArguments        = errors.New("invalid arguments")
	ErrIdMustBeInteger         = errors.New("id must be an integer")
	ErrVaultNotInitialized     = errors.New("vault not intialized")
	ErrFailedToInitializeVault = errors.New("unable to initialize vault")
	ErrFailedToFetchVaultInfo  = errors.New("unable to fetch vault info")

	ErrPasswordDoesNotMatch      = errors.New("password does not match")
	ErrIncorrectMasterPassword   = errors.New("incorrect master password")
	ErrLabelCannotBeCommand      = errors.New("credential label cannot be same as command")
	ErrSecretDoesNotMatch        = errors.New("credential secret does not match")
	ErrCredentialAlreadyExists   = errors.New("credential already exists")
	ErrFailedToFetchCredential   = errors.New("unable to fetch credential/s")
	ErrFailedToSaveCredential    = errors.New("unable to save credential")
	ErrFailedToDeleteCredential  = errors.New("unable to delete credential")
	ErrFailedToDecryptCredential = errors.New("unable to decrypt credential")
	ErrFailedToReadInput         = errors.New("unable to read input")

	ErrCredentialMatchNotFound = errors.New("credential match not found")
	ErrCredentialNotFound      = errors.New("no credential found")
)
