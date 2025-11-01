package encoding

import "encoding/base64"

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
