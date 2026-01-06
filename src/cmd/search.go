package cmd

import (
	"fmt"
	"time"

	"git.plutolab.org/plutolab/kosh/src/internals/constants"
	"git.plutolab.org/plutolab/kosh/src/internals/dao"
	"git.plutolab.org/plutolab/kosh/src/internals/interaction"
	"git.plutolab.org/plutolab/kosh/src/internals/logger"
	"git.plutolab.org/plutolab/kosh/src/internals/search"
)

func init() {
	Commands["search"] = CommandInfo{
		Exec:        SearchCmd,
		Description: "fuzzy search a credential and copy the best match.",
		Usage:       "kosh search <label> <user>",
	}
}

func SearchCmd(args ...string) error {
	if len(args) < 1 {
		logger.Error(constants.ErrInvalidArguments)
		HelpCmd()
		return fmt.Errorf("missing arguments, got %d, want %d", len(args), 2)
	}

	var queryLabel, queryUser string
	queryLabel = args[0]
	if len(args) > 1 {
		queryUser = args[1]
	}

	credentials, err := dao.GetAllCredentials()
	if err != nil {
		logger.Error(constants.ErrFailedToFetchCredential)
		return err
	}

	result := search.BestMatch(queryLabel, queryUser, credentials, time.Now())
	if result == nil {
		logger.Warn(constants.ErrCredentialMatchNotFound)
		logger.Info(constants.MsgListCredentialWithList)
		return nil
	}
	logger.Debug("result score %f", result.Score)
	logger.Info("found credential - %s (%s)", result.Credential.Label, result.Credential.User)

	// get password from user
	password, err := interaction.ReadSecretField(constants.MsgEnterMasterPassword)
	if err != nil {
		logger.Error(constants.ErrFailedToReadInput)
		return err
	}

	vault, err := dao.GetVaultInfo()
	if err != nil {
		logger.Error(constants.ErrFailedToFetchVaultInfo)
		return err
	}
	vaultData := vault.GetRawData()

	secret, err := extractSecret(result.Credential.GetRawData(), vaultData, []byte(password))
	if err != nil {
		return err
	}

	// increment access count by 1 on successful search
	dao.UpdateCredentialAccessCount(result.Credential.Id, 1, time.Now())
	interaction.CopyToClipboard(secret)
	logger.Info(constants.MsgCopiedCredential)
	return nil
}
