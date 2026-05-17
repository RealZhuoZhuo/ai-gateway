package common

import (
	"encoding/base64"
	"encoding/json"
)

func Base64URLJSON(value any) string {
	payload, _ := json.Marshal(value)
	return base64.RawURLEncoding.EncodeToString(payload)
}
