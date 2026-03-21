package cmd

import (
	"fmt"
	"strings"
	"time"

	"git.plutolab.org/plutolab/kosh/internal/constants"
	"git.plutolab.org/plutolab/kosh/internal/logger"
	"git.plutolab.org/plutolab/kosh/internal/model"
	"github.com/spf13/cobra"
)

var (
	listLabel string
	listUser  string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Show a list of saved credentials",
	Args:  cobra.ExactArgs(0),

	RunE: func(cmd *cobra.Command, args []string) error {
		return runList(listLabel, listUser)
	},
}

func init() {
	listCmd.Flags().StringVarP(&listLabel, "label", "l", "", "filter creds that contain label string")
	listCmd.Flags().StringVarP(&listUser, "user", "u", "", "filter creds that contain user string")

	rootCmd.AddCommand(listCmd)
}

func runList(label string, user string) error {

	credentials, err := store.SearchCredentialByLabelOrUser(label, user)
	if err != nil {
		logger.Error("%s", constants.ErrCredentialMatchNotFound.Error())
		return err
	}

	displayCredentials(credentials, label, user)

	return nil
}

func displayCredentials(credentials []model.CredentialSummary, filterLabel, filterUser string) {
	// Show active filters
	filters := []string{}
	if filterLabel != "" || filterUser != "" {
		if filterLabel != "" {
			filters = append(filters, fmt.Sprintf("label contains '%s'", filterLabel))
		}
		if filterUser != "" {
			filters = append(filters, fmt.Sprintf("user contains '%s'", filterUser))
		}
	} else {
		filters = []string{"none"}
	}
	logger.Muted("filters: %s\n", strings.Join(filters, " and "))

	if len(credentials) == 0 {
		logger.Warn("%s", constants.ErrCredentialNotFound.Error())
		return
	}

	// Table header with separator
	fmt.Printf("%-4s %-18s %-18s %-20s %-20s %-20s %-6s\n", "ID", "LABEL", "USER", "CREATED AT", "UPDATED AT", "ACCESSED AT", "ACCESS COUNT")
	fmt.Printf("%s\n", strings.Repeat("─", 120))

	// Table rows
	for _, cred := range credentials {
		label := truncate(cred.Label, 18)
		user := truncate(cred.User, 18)
		createdAt := truncate(cred.CreatedAt.Local().Format(time.DateTime), 20)
		updatedAt := truncate(cred.UpdatedAt.Local().Format(time.DateTime), 20)
		accessedAt := truncate(cred.AccessedAt.Local().Format(time.DateTime), 20)
		fmt.Printf("%-4d %-18s %-18s %-20s %-20s %-20s %-6d\n", cred.Id, label, user, createdAt, updatedAt, accessedAt, cred.AccessCount)
	}

	fmt.Println()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
