package sign

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"io"
	"sort"
	"strings"
)

// ---------- V2 helpers you had (kept for reference in case) ----------
func GetSigned(params map[string]string, key string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	sb := strings.Builder{}
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte('&')
		}
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(params[k])
	}
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(sb.String()))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func GetSignedinter(params map[string]interface{}, key string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	sb := strings.Builder{}
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte('&')
		}
		sb.WriteString(k)
		sb.WriteByte('=')
		switch v := params[k].(type) {
		case bool:
			if v {
				sb.WriteString("true")
			} else {
				sb.WriteString("false")
			}
		default:
			sb.WriteString(fmt.Sprintf("%v", v))
		}
	}
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(sb.String()))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// ---------- V5 helpers ----------
func BuildQueryString(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(params[k])
	}
	return b.String()
}

// V5 payload = timestamp + apiKey + recvWindow + (queryString or bodyJSON)
func SignV5(timestamp, apiKey, recvWindow, queryOrBody, secret string) string {
	payload := timestamp + apiKey + recvWindow + queryOrBody
	h := hmac.New(sha256.New, []byte(secret))
	io.WriteString(h, payload)
	return fmt.Sprintf("%x", h.Sum(nil))
}
