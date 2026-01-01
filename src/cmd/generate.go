package cmd

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"flag"
	"fmt"
	"math/big"
	"slices"
	"strconv"
	"strings"

	"git.plutolab.org/plutolab/kosh/src/internals/crypto"
	"git.plutolab.org/plutolab/kosh/src/internals/dao"
	"git.plutolab.org/plutolab/kosh/src/internals/interaction"
	"git.plutolab.org/plutolab/kosh/src/internals/logger"
	"git.plutolab.org/plutolab/kosh/src/internals/model"
	"golang.org/x/crypto/curve25519"
)

type CharGroup string
type RequireConfig map[CharGroup]int

const (
	LowerCharGroup  = "lower"
	UpperCharGroup  = "upper"
	DigitCharGroup  = "digit"
	SymbolCharGroup = "symbol"
)

func init() {
	Commands["generate"] = CommandInfo{
		Exec:        generateCmd,
		Usage:       "kosh generate <label> <user> [options]",
		Description: "generate a strong password and store as credential",
	}
}

func generateCmd(args ...string) error {
	flagSet := flag.NewFlagSet("generate", flag.ContinueOnError)

	length := flagSet.Int("length", 20, "length of the password")
	upper := flagSet.Bool("upper", true, "include uppercase letters")
	lower := flagSet.Bool("lower", true, "include lowercase letters")
	digit := flagSet.Bool("digit", true, "include digits")
	symbol := flagSet.Bool("symbol", true, "include special symbols")
	require := flagSet.String("require", "", "password requirements")

	var buf bytes.Buffer
	flagSet.SetOutput(&buf)
	if err := flagSet.Parse(args); err != nil {
		logger.Error("%s", strings.Split(buf.String(), "\n")[0])
		return err
	}
	logger.Debug("length %d, upper %t, lower %t, digit %t, symbol %t, require %s", *length, *upper, *lower, *digit, *symbol, *require)

	// positional arguments
	if len(flagSet.Args()) < 2 {
		logger.Error("wrong arguments got %d, want %d", len(flagSet.Args()), 2)
		return fmt.Errorf("invalid arguments provided")
	}

	label := flagSet.Arg(0)
	user := flagSet.Arg(1)

	requirement, err := parseRequirement(*upper, *lower, *digit, *symbol, *require)
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

	if requiredLength > *length {
		logger.Warn("required length (%d characters) is greater than password length (%d characters)", requiredLength, *length)
		confirm, err := interaction.ConfirmYesNo(
			"generate password with the required length?",
			false,
		)

		if err != nil {
			logger.Error("%s", err.Error())
			return err
		}

		if !confirm {
			logger.Info("opeartion aborted")
			return nil
		}

		*length = requiredLength
	}

	// fetch vault info
	// fetch vault info
	vault, err := dao.GetVaultInfo()
	if err != nil {
		return err
	}
	vaultData := vault.GetRawData()

	// verify master password
	password, err := interaction.ReadSecretField("master password > ")
	if err != nil {
		logger.Error("error reading master password")
		return err
	}

	unlockKey := crypto.GenerateSymmetricKey([]byte(password), vaultData.Salt)
	if _, err := crypto.DecryptSecret(unlockKey, vaultData.Secret, vaultData.Nonce); err != nil {
		logger.Error("master password is incorrect")
		return err
	}

	// ensure that label is not a command
	if _, found := Commands[label]; found {
		logger.Error("label cannot be same as an existing command")
		logger.Info("list existing commands with 'help' command")
		return nil
	}

	// check if same credential already exists or not
	if cred, _ := dao.GetCredentialByLabelAndUser(label, user); cred != nil {
		overwrite, err := interaction.ConfirmYesNo(
			"credential already exists. do you want to overwrite?",
			false,
		)

		if err != nil {
			logger.Error("failed to read input")
			return err
		}

		if !overwrite {
			logger.Info("operation aborted")
			return nil
		}

		confirm, err := interaction.ConfirmWithText(
			"this operation cannot be undone. are you sure?",
			fmt.Sprintf("overwrite %s %s", label, user),
		)
		if err != nil {
			logger.Error("error confirming with text prompt")
			return err
		}

		if !confirm {
			logger.Info("operation aborted")
			return nil
		}
	}

	generatedSecret, err := generatePassword(*length, *upper, *lower, *digit, *symbol, requirement)
	if err != nil {
		logger.Error("unable to generate credential")
		return err
	}

	ephemeralPrivateKey, ephemeralPublicKey := crypto.GenerateAsymmetricKeyPair()

	// generate symmetric shared secret
	encryptionKey, _ := curve25519.X25519(ephemeralPrivateKey, vaultData.PublicKey)

	// hash to get 32 bit consistent key for encryption
	key := sha256.Sum256(encryptionKey)

	cipher, nonce := crypto.EncryptSecret(key[:], []byte(generatedSecret))

	credential := model.CredentialData{
		Label:     label,
		User:      user,
		Nonce:     nonce,
		Secret:    cipher,
		Ephemeral: ephemeralPublicKey,
	}

	// save credential
	err = dao.AddCredential(credential.EncodeToString())
	if err != nil {
		logger.Error("unable to save credential")
	} else {
		interaction.CopyToClipboard(generatedSecret)
		logger.Info("saved credential to vault")
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
