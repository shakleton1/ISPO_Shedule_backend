package httpapi

import (
	"encoding/json"
	"strings"
)

func sanitizeAuditPayload(payload any) []byte {
	if payload == nil {
		return nil
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return b
	}
	sanitizeAny(&v)
	out, err := json.Marshal(v)
	if err != nil {
		return b
	}
	return out
}

func sanitizeAny(v *any) {
	switch t := (*v).(type) {
	case map[string]any:
		for k, val := range t {
			if isSensitiveKey(k) {
				t[k] = "<redacted>"
				continue
			}
			vv := val
			sanitizeAny(&vv)
			t[k] = vv
		}
	case []any:
		for i := range t {
			vv := t[i]
			sanitizeAny(&vv)
			t[i] = vv
		}
	}
}

func isSensitiveKey(k string) bool {
	k = strings.ToLower(k)
	k = strings.ReplaceAll(k, "-", "_")
	k = strings.ReplaceAll(k, " ", "_")
	sensitive := []string{
		"password",
		"password_hash",
		"access_token",
		"refresh_token",
		"jwt_secret",
		"api_key",
		"admin_key",
		"x_admin_key",
		"authorization",
	}
	for _, s := range sensitive {
		if k == s {
			return true
		}
	}
	return false
}
