package uri

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/SabirDzh/VpnCLI/internal/domain"
	"net/url"
	"strconv"
	"strings"
)

type vmessJSON struct {
	Add  string `json:"add"`
	Port any    `json:"port"`
	ID   string `json:"id"`
	Aid  any    `json:"aid"`
	Net  string `json:"net"`
	Type string `json:"type"`
	TLS  string `json:"tls"`
	Host string `json:"host"`
	Path string `json:"path"`
	SNI  string `json:"sni"`
	PS   string `json:"ps"`
	Scy  string `json:"scy"`
	FP   string `json:"fp"`
}

// ParseVMess parses vmess://base64(json) in classic V2Ray format.
func ParseVMess(raw string) (domain.Profile, error) {
	body := strings.TrimPrefix(raw, "vmess://")
	body = strings.TrimSpace(body)
	data, err := decodeB64(body)
	if err != nil {
		return domain.Profile{}, fmt.Errorf("%w: vmess base64: %v", domain.ErrParse, err)
	}
	var v vmessJSON
	dec := json.NewDecoder(strings.NewReader(string(data)))
	if err := dec.Decode(&v); err != nil {
		return domain.Profile{}, fmt.Errorf("%w: vmess json: %v", domain.ErrParse, err)
	}
	if v.Add == "" || v.ID == "" {
		return domain.Profile{}, fmt.Errorf("%w: vmess: missing add/id", domain.ErrParse)
	}
	port, err := vmessPort(v.Port)
	if err != nil {
		return domain.Profile{}, err
	}
	aid := vmessInt(v.Aid)
	security := "none"
	if v.TLS == "tls" {
		security = "tls"
	}
	transport := v.Net
	if transport == "" {
		transport = "tcp"
	}
	p := domain.Profile{
		Name:     v.PS,
		Protocol: domain.ProtocolVMess,
		Endpoint: domain.Endpoint{Host: v.Add, Port: port},
		Settings: domain.ProtocolSettings{
			UUID:      v.ID,
			AlterID:   aid,
			Cipher:    v.Scy,
			Security:  security,
			Transport: normalizeTransport(transport),
			SNI:       first(url.Values{"sni": {v.SNI}, "host": {v.Host}}, "sni", "host"),
			FP:        v.FP,
			Path:      v.Path,
			Host:      v.Host,
		},
	}
	finish(&p)
	return p, nil
}

func vmessPort(v any) (uint16, error) {
	switch t := v.(type) {
	case float64:
		if t <= 0 || t > 65535 {
			return 0, fmt.Errorf("%w: bad vmess port", domain.ErrParse)
		}
		return uint16(t), nil
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(t))
		if err != nil || n <= 0 || n > 65535 {
			return 0, fmt.Errorf("%w: bad vmess port %q", domain.ErrParse, t)
		}
		return uint16(n), nil
	default:
		return 0, fmt.Errorf("%w: bad vmess port type", domain.ErrParse)
	}
}

func vmessInt(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case string:
		n, _ := strconv.Atoi(t)
		return n
	default:
		return 0
	}
}

func decodeB64(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if data, err := base64.RawURLEncoding.DecodeString(s); err == nil {
		return data, nil
	}
	if data, err := base64.URLEncoding.DecodeString(s); err == nil {
		return data, nil
	}
	if data, err := base64.RawStdEncoding.DecodeString(s); err == nil {
		return data, nil
	}
	return base64.StdEncoding.DecodeString(s)
}
