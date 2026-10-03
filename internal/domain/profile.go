package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// Protocol — нейтральный идентификатор протокола, независимый от ядра.
type Protocol string

// Protocol constants for all supported proxy protocols.
const (
	// ProtocolVLESS is VLESS (optionally with REALITY/vision flow).
	ProtocolVLESS Protocol = "vless"
	// ProtocolVMess is classic VMess.
	ProtocolVMess Protocol = "vmess"
	// ProtocolTrojan is Trojan over TLS.
	ProtocolTrojan Protocol = "trojan"
	// ProtocolShadowsocks is Shadowsocks.
	ProtocolShadowsocks Protocol = "shadowsocks"
	// ProtocolHysteria2 is Hysteria2 over QUIC.
	ProtocolHysteria2 Protocol = "hysteria2"
)

// Valid reports whether p is a protocol supported by the MVP scope.
func (p Protocol) Valid() bool {
	switch p {
	case ProtocolVLESS, ProtocolVMess, ProtocolTrojan, ProtocolShadowsocks, ProtocolHysteria2:
		return true
	default:
		return false
	}
}

// Endpoint — адрес сервера.
type Endpoint struct {
	Host string `json:"host"`
	Port uint16 `json:"port"`
}

// ProtocolSettings — типизированные настройки протоколов MVP.
// Поля, не относящиеся к конкретному протоколу, остаются пустыми.
type ProtocolSettings struct {
	// VLESS / VMess.
	UUID string `json:"uuid,omitempty"`
	Flow string `json:"flow,omitempty"`
	// VMess extras.
	AlterID int    `json:"alterId,omitempty"`
	Cipher  string `json:"cipher,omitempty"`
	// Trojan / Shadowsocks.
	Password string `json:"password,omitempty"`
	Method   string `json:"method,omitempty"`
	// Transport / TLS.
	Security  string `json:"security,omitempty"` // none | tls | reality
	Transport string `json:"transport,omitempty"`
	SNI       string `json:"sni,omitempty"`
	FP        string `json:"fp,omitempty"`
	ALPN      string `json:"alpn,omitempty"`
	Path      string `json:"path,omitempty"`
	Host      string `json:"hostHeader,omitempty"`
	// REALITY.
	RealityPublicKey string `json:"realityPublicKey,omitempty"`
	RealityShortID   string `json:"realityShortId,omitempty"`
	// Hysteria2.
	ObfsType     string `json:"obfsType,omitempty"` // salamander | gecko
	ObfsPassword string `json:"obfsPassword,omitempty"`
	UpMbps       int    `json:"upMbps,omitempty"`
	DownMbps     int    `json:"downMbps,omitempty"`
	Insecure     bool   `json:"insecure,omitempty"`
	// Display name from URI fragment.
	Remark string `json:"-"`
}

// Identity returns the credential that distinguishes one profile from another.
func (s ProtocolSettings) Identity() string {
	if s.UUID != "" {
		return s.UUID
	}
	if s.Password != "" {
		return s.Method + ":" + s.Password
	}
	return ""
}

// Profile — нейтральная доменная модель сервера.
// Каждое ядро само переводит профиль в свой нативный конфиг.
type Profile struct {
	ID       string           `json:"id"`
	Name     string           `json:"name"`
	Protocol Protocol         `json:"protocol"`
	Source   string           `json:"source"`
	Endpoint Endpoint         `json:"endpoint"`
	Settings ProtocolSettings `json:"settings"`
	// Raw — нативный конфиг ядра, если профиль импортирован «как есть».
	Raw []byte `json:"raw,omitempty"`
}

// ManualSource marks profiles added by hand.
const ManualSource = "manual"

// SubscriptionSource returns Source value for subscription-owned profiles.
func SubscriptionSource(id string) string { return "subscription:" + id }

// ComputeID returns a stable hash that survives subscription refresh.
func ComputeID(p Protocol, host string, port uint16, identity string) string {
	h := sha256.Sum256([]byte(strings.ToLower(fmt.Sprintf("%s|%s|%d|%s", p, host, port, identity))))
	return hex.EncodeToString(h[:])[:16]
}
