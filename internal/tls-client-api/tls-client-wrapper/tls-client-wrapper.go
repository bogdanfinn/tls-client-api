package tls_client_wrapper

import (
	"context"
	"fmt"
	"sync"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/google/uuid"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type TLSClientWrapper interface {
	Do(sessionId *string, tlsClientIdentifier string, proxy *string, cookies []*http.Cookie, req *http.Request) (*http.Response, *string, []*http.Cookie, error)
}

type tlsClientWrapper struct {
	sync.Mutex
	logger                  log.Logger
	clients                 map[string]tls_client.HttpClient
	tlsClientTimeoutSeconds int
}

func NewTLSClientWrapper(ctx context.Context, config cfg.Config, logger log.Logger) (TLSClientWrapper, error) {
	return &tlsClientWrapper{
		logger:                  logger,
		tlsClientTimeoutSeconds: config.GetInt("tls_client_timeout_seconds", 30),
		clients:                 make(map[string]tls_client.HttpClient),
	}, nil
}

func (w *tlsClientWrapper) Do(sessionId *string, tlsClientIdentifier string, proxy *string, cookies []*http.Cookie, req *http.Request) (*http.Response, *string, []*http.Cookie, error) {
	tlsClient, newSessionId, err := w.getTlsClient(sessionId, tlsClientIdentifier, proxy)

	if err != nil {
		return nil, newSessionId, nil, fmt.Errorf("could not create tls http client: %w", err)
	}

	if len(cookies) > 0 {
		tlsClient.SetCookies(req.URL, cookies)
	}

	resp, err := tlsClient.Do(req)

	if err != nil {
		return nil, newSessionId, nil, fmt.Errorf("failed to get response: %w", err)
	}

	sessionCookies := tlsClient.GetCookies(req.URL)

	return resp, newSessionId, sessionCookies, nil
}

func (w *tlsClientWrapper) getTlsClient(sessionId *string, tlsClientIdentifier string, proxy *string) (tls_client.HttpClient, *string, error) {
	w.Lock()
	defer w.Unlock()

	newSessionId := uuid.New().String()
	if mdl.EmptyIfNil(sessionId) != "" {
		newSessionId = mdl.EmptyIfNil(sessionId)
	}

	client, ok := w.clients[newSessionId]

	if ok {
		w.logger.Info("found client in cache for session id: %s", newSessionId)
		return client, mdl.Box(newSessionId), nil
	}

	tlsClientProfile := w.getTlsClientProfile(tlsClientIdentifier)

	options := []tls_client.HttpClientOption{
		tls_client.WithTimeout(w.tlsClientTimeoutSeconds),
		tls_client.WithClientProfile(tlsClientProfile),
	}

	if proxy != nil && mdl.EmptyIfNil(proxy) != "" {
		options = append(options, tls_client.WithProxyUrl(mdl.EmptyIfNil(proxy)))
	}

	tlsClient, err := tls_client.NewHttpClient(w.logger, options...)

	w.clients[newSessionId] = tlsClient
	w.logger.Info("stored new client for session: %s", newSessionId)

	return tlsClient, mdl.Box(newSessionId), err
}

func (w *tlsClientWrapper) getTlsClientProfile(tlsClientIdentifier string) tls_client.ClientProfile {
	tlsClientProfile, ok := tls_client.MappedTLSClients[tlsClientIdentifier]

	if !ok {
		w.logger.Info("can not find supported tls client for %s - use default client", tlsClientIdentifier)
		return tls_client.DefaultClientProfile
	}

	return tlsClientProfile
}
