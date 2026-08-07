package cmd

import (
	"errors"
	"log/slog"
	"strings"
	"time"

	"git.plutolab.org/plutolab/kosh/internal/app"
	"git.plutolab.org/plutolab/kosh/internal/constants"
	"git.plutolab.org/plutolab/kosh/internal/model"
	"git.plutolab.org/plutolab/kosh/internal/search"
	"git.plutolab.org/plutolab/kosh/internal/ui"
	"github.com/spf13/cobra"
)

func NewCmdSearch(ctx *app.Context) *cobra.Command {
	searchCmd := &cobra.Command{
		Use:   "search [label] [user]",
		Short: "Copy a credential found by fuzzy search (default command)",
		Long: `Find a credential by fuzzy search and copy its secret to the clipboard.

With no arguments the search is interactive: type to filter the vault live and
press enter to pick from the top matches. With arguments, the best match for the
given label (and optional user) is selected straight away.

Matching is approximate, so partial and misspelled queries still work. Results
are ranked mostly on how closely the label and user match, with recently and
frequently used credentials nudged higher.

This is the default command: any argument that is not a known subcommand is
passed to it, which makes "kosh github" the same as "kosh search github". The
master password is requested only after a match is found, and the secret goes to
the clipboard, never to the terminal.`,

		Example: `  Pick a credential interactively:
    kosh search

  Search by label:
    kosh search github

  Narrow the search with a user:
    kosh search github alice

  Shorthand form - the search subcommand is implied:
    kosh github alice`,

		Args: cobra.RangeArgs(0, 2),

		RunE: func(cmd *cobra.Command, args []string) error {
			credentials, err := ctx.Store.GetAllCredentials()
			if err != nil {
				slog.Debug("failed to fetch credential for search", "error", err)
				return nil
			}

			var result *search.SearchResult
			if len(args) == 0 { // Interactive Search
				result, err = runInteractiveSearch(credentials)
				if err != nil {
					if errors.Is(err, constants.ErrSearchCancelled) {
						ui.Info(constants.MsgOperationAborted)
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

			return runSearch(cmd, ctx, result)
		},
	}
	return searchCmd
}

func runSearch(_ *cobra.Command, ctx *app.Context, result *search.SearchResult) error {
	if result == nil {
		ui.Warn("%s", constants.ErrCredentialMatchNotFound.Error())
		ui.Info(constants.MsgHintListCredentials)
		return nil
	}

	slog.Debug("best match", "score", result.Score)
	ui.Info("found credential - %s (%s)", result.Credential.Label, result.Credential.User)

	// get password from user
	password, err := ui.ReadSecretField(constants.MsgEnterMasterPassword)
	if err != nil {
		ui.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}

	// decrypt secret using master password
	secret, err := ctx.Vault.DecryptCredential(&result.Credential, password)
	if err != nil {
		return err
	}

	ui.CopyToClipboard(secret)
	ui.Info(constants.MsgCredentialCopiedToClipboard)

	// increment access count by 1 on successful search
	ctx.Store.UpdateCredentialAccessCount(result.Credential.Id, 1, time.Now())
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
		constants.MsgSearchCredential,
		func(query string) []search.SearchResult {
			return searchCredentialsFromList(query, credentials)
		},
	)
	if err != nil {
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
