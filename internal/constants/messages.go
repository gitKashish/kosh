package constants

// Status messages printed back to the user.
//
// Names read subject-first, then what happened to it (MsgCredentialSaved, not
// MsgSavedCredential), so related messages sort together. Single-line messages
// are lowercase and unpunctuated; a completed action ends with "successfully"
// so every confirmation sounds the same.
const (
	// Vault lifecycle.
	MsgVaultInitialized        = "vault initialized successfully"
	MsgVaultAlreadyInitialized = "vault already initialized"

	// Credential lifecycle.
	MsgCredentialSaved   = "credential saved successfully"
	MsgCredentialUpdated = "credential updated successfully"
	MsgCredentialDeleted = "credential deleted successfully"
	MsgCredentialCopied  = "credential copied successfully"

	// Clipboard. Distinct from MsgCredentialCopied, which reports a copy into
	// another profile.
	MsgCredentialCopiedToClipboard = "credential copied to clipboard"

	// Profile lifecycle.
	MsgProfileCreated  = "profile created successfully"
	MsgProfileDeleted  = "profile deleted successfully"
	MsgProfileSwitched = "profile switched successfully"

	// Outcome of a confirmation the user declined.
	MsgOperationAborted = "operation aborted"

	// Hints pointing at the command that answers the question just raised.
	MsgHintListCommands    = "run `kosh help` to list the available commands"
	MsgHintListCredentials = "run `kosh list` to list the stored credentials"
)

// Bodies of the caution block shown before a destructive action, with the
// headers they appear under.
//
// These are prose rather than one-liners: each names what is about to be lost,
// then states that it cannot be recovered, with the stakes in capitals.
//
// The block is reserved for losing a secret that nothing replaces - a deletion,
// or an overwrite of an entry the user was not setting out to change. Replacing
// a secret through "kosh update" leaves a working credential behind and is
// confirmed inline instead, so the block keeps its weight.
const (
	MsgCautionDestructive = "DESTRUCTIVE ACTION"
	MsgCautionOverwrite   = "CREDENTIAL OVERWRITE"

	MsgWarnCredentialDelete = `This credential will be deleted PERMANENTLY.
The operation is IRREVERSIBLE and the secret is IRRECOVERABLE.`

	MsgWarnCredentialOverwrite = `This label and user already hold a credential and it will be OVERWRITTEN.
The operation is IRREVERSIBLE and the current secret is IRRECOVERABLE.`

	MsgWarnTargetCredentialOverwrite = `The target profile already holds this credential and it will be OVERWRITTEN.
The operation is IRREVERSIBLE and the current secret is IRRECOVERABLE.`

	MsgWarnProfileDelete = `This profile and every credential in it will be deleted PERMANENTLY.
The operation is IRREVERSIBLE and the secrets are IRRECOVERABLE.`
)
