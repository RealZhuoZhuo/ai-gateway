package providers

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestMakeKlingToken(t *testing.T) {
	now := time.Unix(1000, 0)
	token, err := makeKlingToken("ak", "sk", now)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token has %d parts, want 3", len(parts))
	}

	var header map[string]string
	if err := decodeJWTPart(parts[0], &header); err != nil {
		t.Fatal(err)
	}
	if header["alg"] != "HS256" || header["typ"] != "JWT" {
		t.Fatalf("header = %#v", header)
	}

	var claims map[string]any
	if err := decodeJWTPart(parts[1], &claims); err != nil {
		t.Fatal(err)
	}
	if claims["iss"] != "ak" {
		t.Fatalf("iss = %v, want ak", claims["iss"])
	}
	if int64(claims["exp"].(float64)) != 2800 {
		t.Fatalf("exp = %v, want 2800", claims["exp"])
	}
	if int64(claims["nbf"].(float64)) != 995 {
		t.Fatalf("nbf = %v, want 995", claims["nbf"])
	}
}

func decodeJWTPart(part string, out any) error {
	raw, err := base64.RawURLEncoding.DecodeString(part)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}
