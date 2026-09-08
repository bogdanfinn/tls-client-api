package api

import (
	"fmt"

	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

// buildRemoteHttpClient builds a dedicated tls-client HttpClient for a single websocket bridge
// connection, reusing the same client-profile lookup table as the rest of tls-client-api.
// The client is never registered in the shared session map: the bridge owns and releases it.
func buildRemoteHttpClient(cfg *WsConfig) (tls_client.HttpClient, error) {
	clientProfile, ok := profiles.MappedTLSClients[cfg.TlsClientIdentifier]
	if !ok {
		return nil, fmt.Errorf("unknown tlsClientIdentifier: %s", cfg.TlsClientIdentifier)
	}

	options := []tls_client.HttpClientOption{
		tls_client.WithClientProfile(clientProfile),
		// WebSocket connections require HTTP/1.1.
		tls_client.WithForceHttp1(),
	}

	if cfg.WithRandomTLSExtensionOrder {
		options = append(options, tls_client.WithRandomTLSExtensionOrder())
	}

	if cfg.InsecureSkipVerify {
		options = append(options, tls_client.WithInsecureSkipVerify())
	}

	if cfg.DisableIPV4 {
		options = append(options, tls_client.WithDisableIPV4())
	}

	if cfg.DisableIPV6 {
		options = append(options, tls_client.WithDisableIPV6())
	}

	if cfg.ServerNameOverwrite != "" {
		options = append(options, tls_client.WithServerNameOverwrite(cfg.ServerNameOverwrite))
	}

	if cfg.ProxyUrl != "" {
		options = append(options, tls_client.WithProxyUrl(cfg.ProxyUrl))
	}

	return tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
}
