package constants

// Prompts shown when kosh asks the user for input.
//
// Inline prompts are read on the same line as the question, so they end with
// ": " to leave the cursor after the colon. Block prompts are rendered on a
// line of their own - confirmations and option lists - and carry no trailing
// colon or space.
//
// Everything here is lowercase and unpunctuated, except for questions, which
// end with a question mark.
const (
	// Inline prompts: master password.
	MsgEnterMasterPassword   = "enter master password: "
	MsgConfirmMasterPassword = "confirm master password: "

	// Inline prompts: credential fields. The wording follows the vault's own
	// vocabulary - a credential is a label, a user and a secret.
	MsgEnterCredentialLabel    = "enter credential label: "
	MsgEnterCredentialUser     = "enter credential user: "
	MsgEnterCredentialSecret   = "enter credential secret: "
	MsgConfirmCredentialSecret = "confirm credential secret: "

	// Inline prompts: interactive search.
	MsgSearchProfile    = "search profile: "
	MsgSearchCredential = "search credential: "

	// Block prompt: followed by the list of fields to choose from.
	MsgSelectCredentialField = "select the credential field to update"

	// Block prompts: confirmations. Each states the action and its
	// consequence on its own, so callers never have to compose them.
	//
	// Only the ones that destroy a secret claim to be permanent. Renaming a
	// credential's label or user can be undone by renaming it back, so that
	// confirmation stays neutral.
	MsgConfirmCredentialOverwrite = "overwrite the existing credential? this cannot be undone"
	MsgConfirmCredentialDelete    = "delete this credential permanently?"
	MsgConfirmProfileDelete       = "delete this profile and all its credentials permanently?"
	MsgConfirmCredentialChange    = "apply this change to the credential?"
	MsgOperationIsPermanent       = "this operation is permanent and cannot be undone"
)
