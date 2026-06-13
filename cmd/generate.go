package cmd

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"slices"
	"strconv"
	"strings"

	"git.plutolab.org/plutolab/kosh/internal/constants"
	"git.plutolab.org/plutolab/kosh/internal/logger"
	"git.plutolab.org/plutolab/kosh/internal/ui"
	"github.com/spf13/cobra"
)

type CharGroup string
type RequireConfig map[CharGroup]int

var (
	genLength  int
	genUpper   bool
	genLower   bool
	genDigit   bool
	genSymbol  bool
	genRequire string
	genNoSave  bool
)

const (
	LowerCharGroup  = "lower"
	UpperCharGroup  = "upper"
	DigitCharGroup  = "digit"
	SymbolCharGroup = "symbol"
)

var generateCmd = &cobra.Command{
	Use:   "generate <label> <user>",
	Short: "Generate a strong password with specified restrictions",
	Long: `Generate a strong random password and store it securely in the vault.
The generated password is encrypted and copied to the clipboard.`,

	Example: `	Generate a default password:
	kosh generate github alice

	Generate a 32-character password with strict requirements:
    	kosh generate -l 32 --require "upper=2,lower=10,digit=5,symbol=3" email alice

	Generate a password without symbols:
    	kosh generate --symbol=false server root`,

	Args: cobra.RangeArgs(0, 2),

	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 && !genNoSave {
			logger.Error("%s", constants.ErrInvalidArguments.Error())
			return fmt.Errorf("wrong arguments got %d, want 2 (unless --no-save is used)", len(args))
		}

		return runGenerate(args...)
	},
}

func init() {
	generateCmd.Flags().IntVarP(&genLength, "length", "l", 20, "length of the password")
	generateCmd.Flags().BoolVar(&genUpper, "upper", true, "include uppercase letters")
	generateCmd.Flags().BoolVar(&genLower, "lower", true, "include uppercase letters")
	generateCmd.Flags().BoolVar(&genDigit, "digit", true, "include digits")
	generateCmd.Flags().BoolVar(&genSymbol, "symbol", true, "include special symbols")
	generateCmd.Flags().StringVarP(&genRequire, "require", "r", "", "password requirements (e.g., upper=2,digit=3)")
	generateCmd.Flags().BoolVarP(&genNoSave, "no-save", "n", false, "generate password but do not save it")

	rootCmd.AddCommand(generateCmd)
}

func runGenerate(args ...string) error {

	requirement, err := parseRequirement(genUpper, genLower, genDigit, genSymbol, genRequire)
	if err != nil {
		logger.Error("invalid `require` flag values")
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

	if requiredLength > genLength {
		logger.Warn("required length (%d characters) is greater than password length (%d characters)", requiredLength, genLength)
		confirm, err := ui.ConfirmYesNo(
			"generate password with the required length?",
			false,
		)

		if err != nil {
			logger.Error("%s", err.Error())
			return err
		}

		if !confirm {
			logger.Info(constants.MsgOperationAborted)
			return nil
		}

		genLength = requiredLength
	}

	generatedSecret, err := generatePassword(genLength, genUpper, genLower, genDigit, genSymbol, requirement)
	if err != nil {
		logger.Error("unable to generate credential")
		return err
	}

	// In case `--no-save` copy the password to clipboard, no need to fetch vault data or verify password
	if genNoSave {
		ui.CopyToClipboard(generatedSecret)
		logger.Info("%s", constants.MsgCopiedCredential)
		return nil
	}

	label := args[0]
	user := args[1]

	password, err := ui.ReadSecretField(constants.MsgEnterMasterPassword)
	if err != nil {
		logger.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}
	if err := vault.VerifyMasterPassword(password); err != nil {
		logger.Error("%s", constants.ErrIncorrectMasterPassword.Error())
		return err
	}

	err = vault.AddCredential(label, user, generatedSecret)
	if err != nil {
		logger.Debug("runGenerate:failed to add generated credential:%s", err.Error())
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
			logger.Error("invalid requirement field %s", param)
			return nil, fmt.Errorf("invalid requirement field %s", param)
		}
		group := CharGroup(strings.TrimSpace(fields[0]))
		str := strings.TrimSpace(fields[1])

		val, err := strconv.Atoi(str)
		if err != nil || val < 0 {
			logger.Error("invalid requirement count %s", str)
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
			logger.Error("contradicting requirement, %s", errMsg)
			return nil, fmt.Errorf("%s", errMsg)
		}

		requirement[group] = val
	}

	logger.Debug("final requirement %v", requirement)

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
