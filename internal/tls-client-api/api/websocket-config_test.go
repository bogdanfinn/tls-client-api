package api

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	http "github.com/bogdanfinn/fhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func encodeConfigJson(t *testing.T, raw string) string {
	t.Helper()

	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func TestDecodeWsConfig_MalformedBase64(t *testing.T) {
	_, err := decodeWsConfig("not-valid-base64url!!!")
	require.Error(t, err)
}

func TestDecodeWsConfig_MalformedJson(t *testing.T) {
	_, err := decodeWsConfig(encodeConfigJson(t, `{"version": 1, "requestUrl":`))
	require.Error(t, err)
}

func TestDecodeWsConfig_MissingHeader(t *testing.T) {
	_, err := decodeWsConfig("")
	require.Error(t, err)
}

func TestDecodeWsConfig_UnsupportedVersion(t *testing.T) {
	raw := `{"version": 2, "requestUrl": "wss://example.com/ws", "tlsClientIdentifier": "chrome_133"}`
	_, err := decodeWsConfig(encodeConfigJson(t, raw))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported config version")
}

func TestDecodeWsConfig_InvalidRequestUrlScheme(t *testing.T) {
	raw := `{"version": 1, "requestUrl": "http://example.com/ws", "tlsClientIdentifier": "chrome_133"}`
	_, err := decodeWsConfig(encodeConfigJson(t, raw))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ws:// or wss://")
}

func TestDecodeWsConfig_InvalidRequestUrlMalformed(t *testing.T) {
	raw := `{"version": 1, "requestUrl": "://not-a-url", "tlsClientIdentifier": "chrome_133"}`
	_, err := decodeWsConfig(encodeConfigJson(t, raw))
	require.Error(t, err)
}

func TestDecodeWsConfig_MissingRequestUrl(t *testing.T) {
	raw := `{"version": 1, "tlsClientIdentifier": "chrome_133"}`
	_, err := decodeWsConfig(encodeConfigJson(t, raw))
	require.Error(t, err)
}

func TestDecodeWsConfig_MissingTlsClientIdentifier(t *testing.T) {
	raw := `{"version": 1, "requestUrl": "wss://example.com/ws"}`
	_, err := decodeWsConfig(encodeConfigJson(t, raw))
	require.Error(t, err)
}

func TestDecodeWsConfig_ConfigHeaderTooLarge(t *testing.T) {
	huge := make([]byte, maxWsConfigHeaderSize+1)
	for i := range huge {
		huge[i] = 'a'
	}
	_, err := decodeWsConfig(string(huge))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum")
}

func TestDecodeWsConfig_Valid(t *testing.T) {
	raw := `{
		"version": 1,
		"requestUrl": "wss://example.com/socket.io/?EIO=4&transport=websocket",
		"tlsClientIdentifier": "chrome_133",
		"handshakeTimeoutMilliseconds": 5000
	}`

	cfg, err := decodeWsConfig(encodeConfigJson(t, raw))
	require.NoError(t, err)
	assert.Equal(t, "wss://example.com/socket.io/?EIO=4&transport=websocket", cfg.RequestUrl)
	assert.Equal(t, "chrome_133", cfg.TlsClientIdentifier)
	assert.Equal(t, 5000, cfg.effectiveHandshakeTimeoutMilliseconds())
}

func TestDecodeWsConfig_DefaultHandshakeTimeout(t *testing.T) {
	raw := `{"version": 1, "requestUrl": "wss://example.com/ws", "tlsClientIdentifier": "chrome_133"}`
	cfg, err := decodeWsConfig(encodeConfigJson(t, raw))
	require.NoError(t, err)
	assert.Equal(t, defaultWsHandshakeTimeoutMilliseconds, cfg.effectiveHandshakeTimeoutMilliseconds())
}

func TestWsHeaderValue_SingleString(t *testing.T) {
	var cfg WsConfig
	raw := `{
		"version": 1,
		"requestUrl": "wss://example.com/ws",
		"tlsClientIdentifier": "chrome_133",
		"headers": {"Origin": "https://example.com"}
	}`

	require.NoError(t, json.Unmarshal([]byte(raw), &cfg))
	assert.Equal(t, WsHeaderValue{"https://example.com"}, cfg.Headers["Origin"])
}

func TestWsHeaderValue_StringArray(t *testing.T) {
	var cfg WsConfig
	raw := `{
		"version": 1,
		"requestUrl": "wss://example.com/ws",
		"tlsClientIdentifier": "chrome_133",
		"headers": {"Cookie": ["a=1", "b=2"]}
	}`

	require.NoError(t, json.Unmarshal([]byte(raw), &cfg))
	assert.Equal(t, WsHeaderValue{"a=1", "b=2"}, cfg.Headers["Cookie"])
}

func TestWsHeaderValue_InvalidType(t *testing.T) {
	var cfg WsConfig
	raw := `{
		"version": 1,
		"requestUrl": "wss://example.com/ws",
		"tlsClientIdentifier": "chrome_133",
		"headers": {"Cookie": 123}
	}`

	err := json.Unmarshal([]byte(raw), &cfg)
	require.Error(t, err)
}

func TestBuildRemoteHeaders_MultiValueAndOrder(t *testing.T) {
	cfg := &WsConfig{
		Headers: map[string]WsHeaderValue{
			"Origin": {"https://example.com"},
			"Cookie": {"a=1", "b=2"},
		},
		HeaderOrder: []string{"Host", "Upgrade", "Connection", "Origin", "Cookie"},
	}

	headers := cfg.buildRemoteHeaders()

	assert.Equal(t, []string{"https://example.com"}, headers["Origin"])
	assert.Equal(t, []string{"a=1", "b=2"}, headers["Cookie"])
	assert.Equal(t, []string{"host", "upgrade", "connection", "origin", "cookie"}, headers[http.HeaderOrderKey])
}

func TestBuildRemoteHeaders_ProtocolsSetSecWebsocketProtocol(t *testing.T) {
	cfg := &WsConfig{
		Protocols: []string{"chat", "superchat"},
	}

	headers := cfg.buildRemoteHeaders()

	assert.Equal(t, "chat, superchat", headers.Get("Sec-WebSocket-Protocol"))
}

func TestBuildRemoteHeaders_DoesNotOverwriteExplicitProtocolHeader(t *testing.T) {
	cfg := &WsConfig{
		Headers: map[string]WsHeaderValue{
			"Sec-WebSocket-Protocol": {"explicit-protocol"},
		},
		Protocols: []string{"chat"},
	}

	headers := cfg.buildRemoteHeaders()

	assert.Equal(t, "explicit-protocol", headers.Get("Sec-WebSocket-Protocol"))
}

func TestBuildRemoteHttpClient_UnknownIdentifier(t *testing.T) {
	_, err := buildRemoteHttpClient(&WsConfig{TlsClientIdentifier: "not-a-real-identifier"})
	require.Error(t, err)
}

func TestBuildRemoteHttpClient_KnownIdentifier(t *testing.T) {
	client, err := buildRemoteHttpClient(&WsConfig{TlsClientIdentifier: "chrome_133"})
	require.NoError(t, err)
	require.NotNil(t, client)
	client.CloseIdleConnections()
}

func TestDecodeWsConfig_CanonicalFieldNames(t *testing.T) {
	raw := `{
		"version": 1,
		"requestUrl": "wss://example.com/ws",
		"tlsClientIdentifier": "chrome_136",
		"withRandomTLSExtensionOrder": true,
		"disableIPV4": true,
		"disableIPV6": true
	}`

	cfg, err := decodeWsConfig(encodeConfigJson(t, raw))
	require.NoError(t, err)
	assert.True(t, cfg.WithRandomTLSExtensionOrder)
	assert.True(t, cfg.DisableIPV4)
	assert.True(t, cfg.DisableIPV6)
}

func TestDecodeWsConfig_DeprecatedAliasFieldNames(t *testing.T) {
	raw := `{
		"version": 1,
		"requestUrl": "wss://example.com/ws",
		"tlsClientIdentifier": "chrome_136",
		"withRandomTlsExtensionOrder": true,
		"disableIPv4": true,
		"disableIPv6": true
	}`

	cfg, err := decodeWsConfig(encodeConfigJson(t, raw))
	require.NoError(t, err)
	assert.True(t, cfg.WithRandomTLSExtensionOrder)
	assert.True(t, cfg.DisableIPV4)
	assert.True(t, cfg.DisableIPV6)
}

func TestDecodeWsConfig_CanonicalFieldWinsOverDeprecatedAlias(t *testing.T) {
	raw := `{
		"version": 1,
		"requestUrl": "wss://example.com/ws",
		"tlsClientIdentifier": "chrome_136",
		"withRandomTLSExtensionOrder": false,
		"withRandomTlsExtensionOrder": true,
		"disableIPV4": false,
		"disableIPv4": true,
		"disableIPV6": false,
		"disableIPv6": true
	}`

	cfg, err := decodeWsConfig(encodeConfigJson(t, raw))
	require.NoError(t, err)
	assert.False(t, cfg.WithRandomTLSExtensionOrder)
	assert.False(t, cfg.DisableIPV4)
	assert.False(t, cfg.DisableIPV6)
}

func TestDecodeWsConfig_DeprecatedAliasUsedWhenCanonicalAbsent(t *testing.T) {
	raw := `{
		"version": 1,
		"requestUrl": "wss://example.com/ws",
		"tlsClientIdentifier": "chrome_136",
		"disableIPv4": true
	}`

	cfg, err := decodeWsConfig(encodeConfigJson(t, raw))
	require.NoError(t, err)
	assert.True(t, cfg.DisableIPV4)
	assert.False(t, cfg.DisableIPV6)
}

// TestDecodeWsConfig_NodeLikePayload mirrors the exact payload shape sent by tls-client-node.
func TestDecodeWsConfig_NodeLikePayload(t *testing.T) {
	raw := `{
		"version": 1,
		"requestUrl": "wss://localhost:9999/socket.io/?EIO=4&transport=websocket",
		"tlsClientIdentifier": "chrome_136",
		"proxyUrl": "",
		"headers": {
			"Origin": "https://example.com",
			"User-Agent": "Mozilla/5.0",
			"Cookie": "session=abc"
		},
		"headerOrder": ["host", "upgrade", "connection", "origin", "user-agent", "cookie"],
		"protocols": [],
		"handshakeTimeoutMilliseconds": 10000,
		"readBufferSize": 0,
		"writeBufferSize": 0,
		"insecureSkipVerify": true,
		"withRandomTLSExtensionOrder": false,
		"disableIPV4": false,
		"disableIPV6": false,
		"serverNameOverwrite": ""
	}`

	cfg, err := decodeWsConfig(encodeConfigJson(t, raw))
	require.NoError(t, err)

	assert.Equal(t, 1, cfg.Version)
	assert.Equal(t, "wss://localhost:9999/socket.io/?EIO=4&transport=websocket", cfg.RequestUrl)
	assert.Equal(t, "chrome_136", cfg.TlsClientIdentifier)
	assert.True(t, cfg.InsecureSkipVerify)
	assert.Equal(t, WsHeaderValue{"session=abc"}, cfg.Headers["Cookie"])

	headers := cfg.buildRemoteHeaders()
	assert.Equal(t, []string{"host", "upgrade", "connection", "origin", "user-agent", "cookie"}, headers[http.HeaderOrderKey])
}
