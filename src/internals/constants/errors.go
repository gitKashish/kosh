package constants

const (
	ErrInvalidArguments = "invalid arguments"
	ErrIdMustBeInteger  = "id must be an integer"

	ErrVaultNotInitialized     = "vault not intialized"
	ErrFailedToInitializeVault = "unable to initialize vault"
	ErrFailedToFetchVaultInfo  = "unable to fetch vault info"

	ErrPasswordDoesNotMatch    = "password does not match"
	ErrIncorrectMasterPassword = "incorrect master password"

	ErrLabelCannotBeCommand = "credential label cannot be same as command"
	ErrSecretDoesNotMatch   = "credential secret does not match"

	ErrFailedToFetchCredential   = "unable to fetch credential/s"
	ErrFailedToSaveCredential    = "unable to save credential"
	ErrFailedToDeleteCredential  = "unable to delete credential"
	ErrFailedToDecryptCredential = "unable to decrypt credential"
	ErrFailedToReadInput         = "unable to read input"

	ErrCredentialMatchNotFound = "credential match not found"
	ErrCredentialNotFound      = "no credential found"
)
