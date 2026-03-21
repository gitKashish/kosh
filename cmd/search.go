package cmd

import (
	"time"

	"git.plutolab.org/plutolab/kosh/internal/constants"
	"git.plutolab.org/plutolab/kosh/internal/logger"
	"git.plutolab.org/plutolab/kosh/internal/search"
	"git.plutolab.org/plutolab/kosh/internal/ui"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <label> <user>",
	Short: "Retrieve a credential via fuzzy search",
	Args:  cobra.RangeArgs(1, 2),

	RunE: func(cmd *cobra.Command, args []string) error {
		var label, user string

		label = args[0]
		if len(args) > 1 {
			user = args[1]
		}

		return runSearch(label, user)
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
}

func runSearch(queryLabel, queryUser string) error {
	credentials, err := store.GetAllCredentials()
	if err != nil {
		logger.Error("%s", constants.ErrFailedToFetchCredential.Error())
		return err
	}

	result := search.BestMatch(queryLabel, queryUser, credentials, time.Now())
	if result == nil {
		logger.Warn("%s", constants.ErrCredentialMatchNotFound.Error())
		logger.Info(constants.MsgListCredentialWithList)
		return nil
	}
	logger.Debug("result score %f", result.Score)
	logger.Info("found credential - %s (%s)", result.Credential.Label, result.Credential.User)

	// get password from user
	password, err := ui.ReadSecretField(constants.MsgEnterMasterPassword)
	if err != nil {
		logger.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}

	vault, err := store.GetVaultInfo()
	if err != nil {
		logger.Error("%s", constants.ErrFailedToFetchVaultInfo.Error())
		return err
	}
	vaultData := vault.GetRawData()

	secret, err := extractSecret(result.Credential.GetRawData(), vaultData, []byte(password))
	if err != nil {
		return err
	}

	// increment access count by 1 on successful search
	store.UpdateCredentialAccessCount(result.Credential.Id, 1, time.Now())
	ui.CopyToClipboard(secret)
	logger.Info(constants.MsgCopiedCredential)
	return nil
}
