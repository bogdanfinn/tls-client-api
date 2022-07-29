package tls_client_wrapper

import (
	"context"
	"fmt"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type TLSClientWrapper interface {
	Do(tlsClientIdentifier string, proxy *string, cookies []*http.Cookie, req *http.Request) (*http.Response, []*http.Cookie, error)
}

type tlsClientWrapper struct {
	logger                  log.Logger
	tlsClientTimeoutSeconds int
}

func NewTLSClientWrapper(ctx context.Context, config cfg.Config, logger log.Logger) (TLSClientWrapper, error) {
	return &tlsClientWrapper{
		logger:                  logger,
		tlsClientTimeoutSeconds: config.GetInt("tls_client_timeout_seconds", 30),
	}, nil
}

func (w *tlsClientWrapper) Do(tlsClientIdentifier string, proxy *string, cookies []*http.Cookie, req *http.Request) (*http.Response, []*http.Cookie, error) {
	tlsClientProfile := w.getTlsClientProfile(tlsClientIdentifier)

	options := []tls_client.HttpClientOption{
		tls_client.WithTimeout(w.tlsClientTimeoutSeconds),
		tls_client.WithClientProfile(tlsClientProfile),
	}

	if proxy != nil && mdl.EmptyIfNil(proxy) != "" {
		options = append(options, tls_client.WithProxyUrl(mdl.EmptyIfNil(proxy)))
	}

	tlsClient, err := tls_client.NewHttpClient(w.logger, options...)

	if err != nil {
		return nil, nil, fmt.Errorf("could not create tls http client: %w", err)
	}

	if len(cookies) > 0 {
		tlsClient.SetCookies(req.URL, cookies)
	}

	resp, err := tlsClient.Do(req)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to get response: %w", err)
	}

	sessionCookies := tlsClient.GetCookies(req.URL)

	return resp, sessionCookies, nil
}

func (w *tlsClientWrapper) getTlsClientProfile(tlsClientIdentifier string) tls_client.ClientProfile {
	tlsClientProfile, ok := tls_client.MappedTLSClients[tlsClientIdentifier]

	if !ok {
		w.logger.Info("can not find supported tls client for %s - use default client", tlsClientIdentifier)
		return tls_client.DefaultClientProfile
	}

	return tlsClientProfile
}
