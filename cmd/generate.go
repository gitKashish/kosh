package cmd

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"math/big"
	"slices"
	"strconv"
	"strings"

	"git.plutolab.org/plutolab/kosh/internal/app"
	"git.plutolab.org/plutolab/kosh/internal/constants"
	"git.plutolab.org/plutolab/kosh/internal/ui"
	"github.com/spf13/cobra"
)

type CharGroup string
type RequireConfig map[CharGroup]int

type generateOptions struct {
	length  int
	upper   bool
	lower   bool
	digit   bool
	symbol  bool
	require string
	noSave  bool
}

const (
	LowerCharGroup  = "lower"
	UpperCharGroup  = "upper"
	DigitCharGroup  = "digit"
	SymbolCharGroup = "symbol"
)

func NewCmdGenerate(ctx *app.Context) *cobra.Command {
	opts := &generateOptions{}

	generateCmd := &cobra.Command{
		Use:   "generate [label] [user]",
		Short: "Generate a strong random password and store it",
		Long: `Generate a cryptographically random password and save it as a credential.

Both a label and a user are required, and the generated secret is encrypted into
the active profile's vault after the master password is verified. Pass --no-save
to only generate a password: it is copied to the clipboard, nothing is written to
the vault, and the label and user arguments can be omitted.

The character pool is controlled by --upper, --lower, --digit and --symbol, each
enabled by default and disabled with "--flag=false". Use --require to demand a
minimum count from a group, for example "upper=2,digit=3"; requiring characters
from a group that has been disabled is rejected. If the required counts add up to
more than --length, kosh asks whether to grow the password to fit them.

Saving a password does not print it. Retrieve it later with "kosh get" or
"kosh search".`,

		Example: `  Generate and save a default 20-character password:
    kosh generate github alice

  Generate a 32-character password with strict requirements:
    kosh generate -l 32 --require "upper=2,lower=10,digit=5,symbol=3" email alice

  Generate a password without symbols:
    kosh generate --symbol=false server root

  Copy a throwaway password to the clipboard without saving it:
    kosh generate --no-save`,

		Args: cobra.RangeArgs(0, 2),

		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 && !opts.noSave {
				ui.Error("%s", constants.ErrInvalidArguments.Error())
				return fmt.Errorf("wrong arguments got %d, want 2 (unless --no-save is used)", len(args))
			}

			var label, user string
			if len(args) >= 2 {
				label = args[0]
				user = args[1]
			}
			return runGenerate(cmd, ctx, opts, label, user)
		},
	}

	generateCmd.Flags().IntVarP(&opts.length, "length", "l", 20, "length of the password")
	generateCmd.Flags().BoolVar(&opts.upper, "upper", true, "include uppercase letters")
	generateCmd.Flags().BoolVar(&opts.lower, "lower", true, "include lowercase letters")
	generateCmd.Flags().BoolVar(&opts.digit, "digit", true, "include digits")
	generateCmd.Flags().BoolVar(&opts.symbol, "symbol", true, "include special symbols")
	generateCmd.Flags().StringVarP(&opts.require, "require", "r", "", "password requirements (e.g., upper=2,digit=3)")
	generateCmd.Flags().BoolVarP(&opts.noSave, "no-save", "n", false, "generate password but do not save it")

	return generateCmd
}

func runGenerate(_ *cobra.Command, ctx *app.Context, opts *generateOptions, label, user string) error {

	requirement, err := parseRequirement(opts.upper, opts.lower, opts.digit, opts.symbol, opts.require)
	if err != nil {
		ui.Error("invalid `require` flag values")
		return err
	}

	// check length and get confirmation
	requiredLength := 0
	for key, value := range requirement {
		validKey := slices.Contains(
			[]CharGroup{LowerCharGroup, UpperCharGroup, DigitCharGroup, SymbolCharGroup},
			key,
		)

		if validKey {
			requiredLength += value
		}
	}

	if requiredLength > opts.length {
		ui.Warn("required length (%d characters) is greater than password length (%d characters)", requiredLength, opts.length)
		confirm, err := ui.ConfirmYesNo(
			"generate password with the required length?",
			false,
		)

		if err != nil {
			ui.Error("%s", err.Error())
			return err
		}

		if !confirm {
			ui.Info(constants.MsgOperationAborted)
			return nil
		}

		opts.length = requiredLength
	}

	generatedSecret, err := generatePassword(opts.length, opts.upper, opts.lower, opts.digit, opts.symbol, requirement)
	if err != nil {
		ui.Error("unable to generate credential")
		return err
	}

	// In case `--no-save` copy the password to clipboard, no need to fetch vault data or verify password
	if opts.noSave {
		ui.CopyToClipboard(generatedSecret)
		ui.Info(constants.MsgCredentialCopiedToClipboard)
		return nil
	}

	password, err := ui.ReadSecretField(constants.MsgEnterMasterPassword)
	if err != nil {
		ui.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}
	if err := ctx.Vault.VerifyMasterPassword(password); err != nil {
		ui.Error("%s", constants.ErrIncorrectMasterPassword.Error())
		return err
	}

	err = ctx.Vault.AddCredential(label, user, generatedSecret)
	if err != nil {
		return err
	}

	return nil
}

func parseRequirement(upper, lower, digit, symbol bool, requireStr string) (RequireConfig, error) {
	requireList := strings.Split(requireStr, ",")
	requirement := make(map[CharGroup]int)

	if strings.TrimSpace(requireStr) == "" {
		return requirement, nil
	}

	for _, param := range requireList {
		fields := strings.Split(param, "=")
		if len(fields) != 2 {
			slog.Debug("invalid requirement field", "param", param)
			return nil, fmt.Errorf("invalid requirement field %s", param)
		}
		group := CharGroup(strings.TrimSpace(fields[0]))
		str := strings.TrimSpace(fields[1])

		val, err := strconv.Atoi(str)
		if err != nil || val < 0 {
			slog.Debug("invalid requirement count", "count", str)
			return nil, fmt.Errorf("invalid requirement count %s", str)
		}

		var errMsg string
		switch group {
		case LowerCharGroup:
			if !lower && val > 0 {
				errMsg = "lowercase letters not allowed but required"
			}
		case UpperCharGroup:
			if !upper && val > 0 {
				errMsg = "uppercase letters not allowed but required"
			}
		case DigitCharGroup:
			if !digit && val > 0 {
				errMsg = "digits not allowed but required"
			}
		case SymbolCharGroup:
			if !symbol && val > 0 {
				errMsg = "symbols not allowed but required"
			}
		}

		if errMsg != "" {
			slog.Debug("contradicting requirement", "contradiction", errMsg)
			return nil, fmt.Errorf("%s", errMsg)
		}

		requirement[group] = val
	}

	slog.Debug("final requirement", "req", requirement)

	return requirement, nil
}

func generatePassword(length int, upper, lower, digit, symbol bool, require RequireConfig) ([]byte, error) {
	var (
		lowerChars  = "abcdefghijklmnopqrstuvwxyz"
		upperChars  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		digitChars  = "0123456789"
		symbolChars = "!@#$%^&*()-_=+[]{}<>?/|"
	)

	var password []byte
	var pool string

	// required characters
	addRequired := func(group CharGroup, chars string) error {
		count := require[group]
		for range count {
			c, err := randomChar(chars)
			if err != nil {
				return err
			}
			password = append(password, c)
		}
		return nil
	}

	if lower {
		pool += lowerChars
		if err := addRequired(LowerCharGroup, lowerChars); err != nil {
			return nil, err
		}
	}

	if upper {
		pool += upperChars
		if err := addRequired(UpperCharGroup, upperChars); err != nil {
			return nil, err
		}
	}

	if digit {
		pool += digitChars
		if err := addRequired(DigitCharGroup, digitChars); err != nil {
			return nil, err
		}
	}

	if symbol {
		pool += symbolChars
		if err := addRequired(SymbolCharGroup, symbolChars); err != nil {
			return nil, err
		}
	}

	// fill remaining length
	for len(password) < length {
		c, err := randomChar(pool)
		if err != nil {
			return nil, err
		}
		password = append(password, c)
	}

	// shuffle everything
	for i := len(password) - 1; i > 0; i-- {
		j, err := randomInt(i + 1)
		if err != nil {
			return nil, err
		}
		password[i], password[j] = password[j], password[i]
	}

	return password, nil
}

func randomInt(max int) (int, error) {
	if max <= 0 {
		return 0, fmt.Errorf("max must be greater than 0")
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		// extremely rare: crypto/rand failure (system entropy issue)
		return 0, err
	}
	return int(n.Int64()), nil
}

func randomChar(chars string) (byte, error) {
	i, err := randomInt(len(chars))
	if err != nil {
		return 0, err
	}
	return chars[i], nil
}
