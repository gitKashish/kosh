package cmd

import (
	"database/sql"
	"time"

	"git.plutolab.org/plutolab/kosh/internal/constants"
	"git.plutolab.org/plutolab/kosh/internal/logger"
	"git.plutolab.org/plutolab/kosh/internal/ui"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get <label> <user>",
	Short: "Retrieve credential by exact label and user",
	Args:  cobra.ExactArgs(2),

	RunE: func(cmd *cobra.Command, args []string) error {
		return runGet(args[0], args[1])
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}

func runGet(desiredGroup string, desiredUser string) error {
	// fetch credential info
	credential, err := store.GetCredentialByLabelAndUser(desiredGroup, desiredUser)
	if credential == nil && err == sql.ErrNoRows {
		// credential does not exist
		logger.Error("%s", constants.ErrCredentialMatchNotFound.Error())
		return nil
	}

	if err != nil {
		return err
	}

	// get password from user
	password, err := ui.ReadSecretField(constants.MsgEnterMasterPassword)
	if err != nil {
		logger.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}

	secret, err := vault.DecryptCredential(credential, password)
	if err != nil {
		return err
	}
	// on successful access update the access info for the credential,
	// increment access count by 2 on get because it has been fetched
	// with intention meaning that user might be wanting this more
	store.UpdateCredentialAccessCount(credential.Id, 2, time.Now())

	ui.CopyToClipboard([]byte(secret))
	logger.Info(constants.MsgCopiedCredential)
	return nil
}
