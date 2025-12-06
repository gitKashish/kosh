package cmd

import (
	"fmt"
	"time"

	"github.com/gitKashish/kosh/src/internals/dao"
	"github.com/gitKashish/kosh/src/internals/interaction"
	"github.com/gitKashish/kosh/src/internals/logger"
	"github.com/gitKashish/kosh/src/internals/search"
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
		logger.Error("missing arguments")
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
		logger.Error("unable to fetch credentials to search")
		return err
	}

	result := search.BestMatch(queryLabel, queryUser, credentials, time.Now())
	if result == nil {
		logger.Warn("suitable match not found")
		logger.Info("view existing credentials using 'list' command")
		return nil
	}
	logger.Debug("result score %f", result.Score)
	logger.Info("found credential - %s (%s)", result.Credential.Label, result.Credential.User)

	// get password from user
	password, err := interaction.ReadSecretField("master password > ")
	if err != nil {
		logger.Error("unable to read password")
		return err
	}

	vault, err := dao.GetVaultInfo()
	if err != nil {
		logger.Error("error getting vault info")
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
	logger.Info("copied secret to clipboard")
	return nil
}
