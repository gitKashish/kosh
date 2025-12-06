package cmd

import (
	"bytes"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/gitKashish/kosh/src/internals/dao"
	"github.com/gitKashish/kosh/src/internals/logger"
	"github.com/gitKashish/kosh/src/internals/model"
)

func init() {
	Commands["list"] = CommandInfo{
		Exec:        ListCmd,
		Description: "List all credentials associated to a lable or user",
	}
}

func ListCmd(args ...string) error {

	// command flags
	flagSet := flag.NewFlagSet("list", flag.ContinueOnError)

	var buf bytes.Buffer
	flagSet.SetOutput(&buf)

	userFlag := flagSet.String("user", "", "filter by username")
	labelFlag := flagSet.String("label", "", "filter by label")

	if err := flagSet.Parse(args); err != nil {
		if err != flag.ErrHelp {
			errorMessage := strings.Split(buf.String(), "\n")[0]
			logger.Error("%s\n", errorMessage)
		}
		printListHelp()
		return err
	}

	// if no flag filter but has positional args then use them as user filter
	if *userFlag == "" && *labelFlag == "" && len(flagSet.Args()) > 0 {
		*userFlag = flagSet.Arg(0)
	}

	credentials, err := dao.SearchCredentialByLabelOrUser(*labelFlag, *userFlag)
	if err != nil {
		logger.Error("unable to get matching credential")
		return err
	}

	displayCredentials(credentials, *labelFlag, *userFlag)

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
		logger.Warn("no credential found")
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

func printListHelp() {
	logger.Info("Usage: kosh list [options]")
	fmt.Println()
	fmt.Println("List stored credentials with optional filtering")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --label <text>    Filter by label (partial match)")
	fmt.Println("  --user <text>     Filter by username (partial match)")
	fmt.Println("  -h, --help        Show this help")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  kosh list                                # List all credentials")
	fmt.Println("  kosh list pluto                          # Search users containing 'pluto'")
	fmt.Println("  kosh list --user pluto                   # Same as above")
	fmt.Println("  kosh list --label github                 # Search labels containing 'github'")
	fmt.Println("  kosh list --user pluto --label github    # Search users contating 'pluto' and labels containing 'github'")
}
