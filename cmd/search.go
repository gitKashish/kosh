package cmd

import (
	"errors"
	"strings"
	"time"

	"git.plutolab.org/plutolab/kosh/internal/constants"
	"git.plutolab.org/plutolab/kosh/internal/logger"
	"git.plutolab.org/plutolab/kosh/internal/model"
	"git.plutolab.org/plutolab/kosh/internal/search"
	"git.plutolab.org/plutolab/kosh/internal/ui"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <label> <user>",
	Short: "Retrieve a credential via fuzzy search",
	Args:  cobra.RangeArgs(0, 2),

	RunE: func(cmd *cobra.Command, args []string) error {
		credentials, err := store.GetAllCredentials()
		if err != nil {
			logger.Error("%s", constants.ErrFailedToFetchCredential.Error())
			return nil
		}

		var result *search.SearchResult
		if len(args) == 0 { // Interactive Search
			result, err = runInteractiveSearch(credentials)
			if err != nil {
				if errors.Is(err, constants.ErrSearchCancelled) {
					logger.Warn(constants.MsgOperationAborted)
					return nil
				}
				return err
			}
		} else { // Search by command args
			var label, user string
			label = args[0]
			if len(args) > 1 {
				user = args[1]
			}
			result = runSearchByLabelAndUser(credentials, label, user)
		}
		
		return runSearch(result)
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
}

func runSearch(result *search.SearchResult) error {
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

	// decrypt secret using master password
	secret, err := vault.DecryptCredential(&result.Credential, password)
	if err != nil {
		logger.Debug("runSearch:failed to decrypt credential")
		return err
	}

	// increment access count by 1 on successful search
	store.UpdateCredentialAccessCount(result.Credential.Id, 1, time.Now())
	ui.CopyToClipboard([]byte(secret))
	logger.Info(constants.MsgCopiedCredential)
	return nil
}

func runSearchByLabelAndUser(credentials []model.Credential, queryLabel, queryUser string) *search.SearchResult {
	// find matches and return the best match
	result := search.BestMatches(queryLabel, queryUser, credentials, time.Now())
	if len(result) == 0 {
		return nil
	}
	return &result[0]
}

func runInteractiveSearch(credentials []model.Credential) (*search.SearchResult, error) {
	result, err := ui.InteractiveSearch(
		constants.MsgCredentialSearch,
		func (query string) []search.SearchResult {
			return searchCredentialsFromList(query, credentials)
		},
	)
	if err != nil {
		logger.Debug("runInteractiveSearch:failed run interactive search:%s", err.Error())
		return nil, err
	}

	return &result, nil
}

func searchCredentialsFromList(query string, list []model.Credential) []search.SearchResult {
	if strings.TrimSpace(query) == "" {
		return nil
	}
	var label, user string
	parts := strings.Split(query, " ")
	label = parts[0]
	if len(parts) > 1 {
		user = parts[1]
	}
	result := search.BestMatches(label, user, list, time.Now())
	return result[:min(len(result), 5)] // filter out top 5 results
}
