package encoding

import (
	"encoding/base64"
	"log/slog"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// DecodeBase64String decodes text encoded as base64 and returns original data as []byte if decoding fails.
func DecodeBase64String(data string) []byte {
	content, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return []byte(data)
	}
	return content
}

// EncodeToBase64String encodes byte data into a string with base64 encoding.
func EncodeToBase64String(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func RemoveAccent(s string) (string, error) {
	t := transform.Chain(
		norm.NFD,
		runes.Remove(runes.In(unicode.Mn)),
		norm.NFC,
	)

	clean, _, err := transform.String(t, s)
	if err != nil {
		slog.Debug("failed to transform and remove accents", "error", err)
		return "", err
	}

	return clean, nil
}
