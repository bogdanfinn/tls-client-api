package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	http "github.com/bogdanfinn/fhttp"
)

// wsConfigHeaderName is the local Upgrade request header carrying the base64url encoded bridge config.
const wsConfigHeaderName = "x-tls-client-ws-config"

// maxWsConfigHeaderSize limits how much metadata a client can send in the config header.
const maxWsConfigHeaderSize = 16 * 1024

// defaultWsHandshakeTimeoutMilliseconds is used when the config does not specify one.
const defaultWsHandshakeTimeoutMilliseconds = 10000

// wsProtocolResponseHeaderName/wsProtocolVersion are an optional, informational local Upgrade
// response header. Clients must not depend on it for normal operation.
const wsProtocolResponseHeaderName = "x-tls-client-ws-protocol"
const wsProtocolVersion = "1"

// WsHeaderValue accepts either a single string or an array of strings in JSON.
type WsHeaderValue []string

func (h *WsHeaderValue) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*h = WsHeaderValue{single}
		return nil
	}

	var multi []string
	if err := json.Unmarshal(data, &multi); err == nil {
		*h = WsHeaderValue(multi)
		return nil
	}

	return fmt.Errorf("header value must be a string or an array of strings")
}

// WsConfig is the decoded contents of the x-tls-client-ws-config header (protocol v1).
//
// withRandomTLSExtensionOrder/disableIPV4/disableIPV6 are the canonical field names matching
// tls-client-node's naming convention. The deprecated withRandomTlsExtensionOrder/disableIPv4/
// disableIPv6 aliases are still accepted for backward compatibility (see UnmarshalJSON); the
// canonical field wins whenever both are present in the same payload.
type WsConfig struct {
	Version                      int                      `json:"version"`
	RequestUrl                   string                   `json:"requestUrl"`
	TlsClientIdentifier          string                   `json:"tlsClientIdentifier"`
	ProxyUrl                     string                   `json:"proxyUrl"`
	Headers                      map[string]WsHeaderValue `json:"headers"`
	HeaderOrder                  []string                 `json:"headerOrder"`
	Protocols                    []string                 `json:"protocols"`
	HandshakeTimeoutMilliseconds int                      `json:"handshakeTimeoutMilliseconds"`
	ReadBufferSize               int                      `json:"readBufferSize"`
	WriteBufferSize              int                      `json:"writeBufferSize"`
	InsecureSkipVerify           bool                     `json:"insecureSkipVerify"`
	WithRandomTLSExtensionOrder  bool                     `json:"withRandomTLSExtensionOrder"`
	DisableIPV4                  bool                     `json:"disableIPV4"`
	DisableIPV6                  bool                     `json:"disableIPV6"`
	ServerNameOverwrite          string                   `json:"serverNameOverwrite"`
}

// UnmarshalJSON decodes the canonical fields normally, then falls back to the deprecated
// aliases (withRandomTlsExtensionOrder/disableIPv4/disableIPv6) for any canonical key that was
// not present in the payload. The canonical key always wins when both are supplied.
//
// The deprecated alias fields are declared directly on the same anonymous struct used for the
// decode (rather than via a second json.Unmarshal call into WsConfig) so that encoding/json
// resolves them as exact tag matches. Otherwise Go's case-insensitive fallback matching would
// let "withRandomTlsExtensionOrder" silently clobber the "withRandomTLSExtensionOrder" field
// since they only differ in case and no other field claims the alias's exact name.
func (c *WsConfig) UnmarshalJSON(data []byte) error {
	type wsConfigAlias WsConfig

	aux := struct {
		*wsConfigAlias
		WithRandomTlsExtensionOrder *bool `json:"withRandomTlsExtensionOrder"`
		DisableIPv4                 *bool `json:"disableIPv4"`
		DisableIPv6                 *bool `json:"disableIPv6"`
	}{
		wsConfigAlias: (*wsConfigAlias)(c),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	var presentKeys map[string]json.RawMessage
	if err := json.Unmarshal(data, &presentKeys); err != nil {
		return err
	}

	if _, ok := presentKeys["withRandomTLSExtensionOrder"]; !ok && aux.WithRandomTlsExtensionOrder != nil {
		c.WithRandomTLSExtensionOrder = *aux.WithRandomTlsExtensionOrder
	}

	if _, ok := presentKeys["disableIPV4"]; !ok && aux.DisableIPv4 != nil {
		c.DisableIPV4 = *aux.DisableIPv4
	}

	if _, ok := presentKeys["disableIPV6"]; !ok && aux.DisableIPv6 != nil {
		c.DisableIPV6 = *aux.DisableIPv6
	}

	return nil
}

// decodeWsConfig decodes and validates the base64url(JSON) bridge config header value.
func decodeWsConfig(raw string) (*WsConfig, error) {
	if raw == "" {
		return nil, fmt.Errorf("missing %s header", wsConfigHeaderName)
	}

	if len(raw) > maxWsConfigHeaderSize {
		return nil, fmt.Errorf("%s header exceeds maximum allowed size", wsConfigHeaderName)
	}

	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64url config: %w", err)
	}

	var cfg WsConfig
	if err := json.Unmarshal(decoded, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config json: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *WsConfig) validate() error {
	if c.Version != 1 {
		return fmt.Errorf("unsupported config version: %d", c.Version)
	}

	if c.RequestUrl == "" {
		return fmt.Errorf("requestUrl is required")
	}

	parsed, err := url.Parse(c.RequestUrl)
	if err != nil {
		return fmt.Errorf("invalid requestUrl: %w", err)
	}

	if parsed.Scheme != "ws" && parsed.Scheme != "wss" {
		return fmt.Errorf("requestUrl must use ws:// or wss:// scheme")
	}

	if c.TlsClientIdentifier == "" {
		return fmt.Errorf("tlsClientIdentifier is required")
	}

	if c.HandshakeTimeoutMilliseconds < 0 {
		return fmt.Errorf("handshakeTimeoutMilliseconds must not be negative")
	}

	if c.ReadBufferSize < 0 || c.WriteBufferSize < 0 {
		return fmt.Errorf("readBufferSize/writeBufferSize must not be negative")
	}

	return nil
}

func (c *WsConfig) effectiveHandshakeTimeoutMilliseconds() int {
	if c.HandshakeTimeoutMilliseconds <= 0 {
		return defaultWsHandshakeTimeoutMilliseconds
	}

	return c.HandshakeTimeoutMilliseconds
}

// buildRemoteHeaders converts the config headers/protocols/headerOrder into the fhttp
// header representation expected by tls-client's websocket dialer.
func (c *WsConfig) buildRemoteHeaders() http.Header {
	headers := http.Header{}

	for key, values := range c.Headers {
		for _, value := range values {
			headers.Add(key, value)
		}
	}

	if len(c.Protocols) > 0 {
		if _, exists := headers["Sec-Websocket-Protocol"]; !exists {
			headers.Set("Sec-WebSocket-Protocol", strings.Join(c.Protocols, ", "))
		}
	}

	if len(c.HeaderOrder) > 0 {
		order := make([]string, len(c.HeaderOrder))
		for i, key := range c.HeaderOrder {
			order[i] = strings.ToLower(key)
		}
		headers[http.HeaderOrderKey] = order
	}

	return headers
}

func containsString(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}

	return false
}
