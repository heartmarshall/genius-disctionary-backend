package resolver

import (
	"encoding/base64"
)

// encodeCursor encodes an ID string into a base64 cursor.
func encodeCursor(id string) string {
	return base64.StdEncoding.EncodeToString([]byte(id))
}

// decodeCursor decodes a base64 cursor into an ID string.
func decodeCursor(cursor string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
