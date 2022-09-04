package tls_client_wrapper

import (
	"context"
	"fmt"
	"sync"

	http "github.com/bogdanfinn/fhttp"
	"github.com/bogdanfinn/fhttp/http2"
	tls_client "github.com/bogdanfinn/tls-client"
	tls "github.com/bogdanfinn/utls"
	"github.com/google/uuid"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type CustomTlsClient struct {
	Ja3String         string            `json:"ja3String"`
	H2Settings        map[uint16]uint32 `json:"h2Settings"`
	H2SettingsOrder   []uint16          `json:"h2SettingsOrder"`
	PseudoHeaderOrder []string          `json:"pseudoHeaderOrder"`
	ConnectionFlow    uint32            `json:"connectionFlow"`
	PriorityFrames    []PriorityFrames  `json:"priorityFrames"`
}

type PriorityFrames struct {
	StreamID      uint32 `json:"streamID"`
	PriorityParam struct {
		StreamDep uint32 `json:"streamDep"`
		Exclusive bool   `json:"exclusive"`
		Weight    uint8  `json:"weight"`
	} `json:"priorityParam"`
}

type TLSClientWrapper interface {
	BuildTlsClientProfile(tlsClientIdentifier string, customTlsClientProfile *CustomTlsClient) (tls_client.ClientProfile, error)
	Do(sessionId *string, clientProfile tls_client.ClientProfile, proxy *string, followRedirects bool, cookies []*http.Cookie, req *http.Request) (*http.Response, *string, []*http.Cookie, error)
}

type tlsClientWrapper struct {
	sync.Mutex
	logger                   log.Logger
	clients                  map[string]tls_client.HttpClient
	tlsClientTimeoutSeconds  int
	tlsClientFollowRedirects bool
}

func NewTLSClientWrapper(ctx context.Context, config cfg.Config, logger log.Logger) (TLSClientWrapper, error) {
	return &tlsClientWrapper{
		logger:                   logger,
		tlsClientTimeoutSeconds:  config.GetInt("tls_client_timeout_seconds", 30),
		tlsClientFollowRedirects: config.GetBool("tls_client_follow_redirects", false),
		clients:                  make(map[string]tls_client.HttpClient),
	}, nil
}

func (w *tlsClientWrapper) BuildTlsClientProfile(tlsClientIdentifier string, customTlsClientProfile *CustomTlsClient) (tls_client.ClientProfile, error) {
	var clientProfile tls_client.ClientProfile

	if tlsClientIdentifier != "" {
		clientProfile = w.getTlsClientProfile(tlsClientIdentifier)
	}

	if customTlsClientProfile != nil {
		clientHelloId, h2Settings, h2SettingsOrder, pseudoHeaderOrder, connectionFlow, priorityFrames, err := w.getCustomTlsClientProfile(customTlsClientProfile)

		if err != nil {
			return tls_client.ClientProfile{}, fmt.Errorf("can not build http client out of custom tls client information: %w", err)
		}

		clientProfile = tls_client.NewClientProfile(clientHelloId, h2Settings, h2SettingsOrder, pseudoHeaderOrder, connectionFlow, priorityFrames)
	}

	return clientProfile, nil
}

func (w *tlsClientWrapper) Do(sessionId *string, clientProfile tls_client.ClientProfile, proxy *string, followRedirects bool, cookies []*http.Cookie, req *http.Request) (*http.Response, *string, []*http.Cookie, error) {
	tlsClient, newSessionId, err := w.getTlsClient(sessionId, clientProfile, proxy, followRedirects)

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

func (w *tlsClientWrapper) getTlsClient(sessionId *string, clientProfile tls_client.ClientProfile, proxy *string, followRedirects bool) (tls_client.HttpClient, *string, error) {
	w.Lock()
	defer w.Unlock()

	newSessionId := uuid.New().String()
	if mdl.EmptyIfNil(sessionId) != "" {
		newSessionId = mdl.EmptyIfNil(sessionId)
	}

	client, ok := w.clients[newSessionId]

	if ok {
		w.logger.Info("found client in cache for session id: %s", newSessionId)

		modifiedClient, changed, err := w.handleModification(client, proxy, followRedirects)
		if err != nil {
			return nil, mdl.Box(newSessionId), fmt.Errorf("failed to modify existing client: %w", err)
		}

		if changed {
			w.clients[newSessionId] = modifiedClient
		}

		return modifiedClient, mdl.Box(newSessionId), nil
	}

	options := []tls_client.HttpClientOption{
		tls_client.WithTimeout(w.tlsClientTimeoutSeconds),
		tls_client.WithClientProfile(clientProfile),
	}

	if !w.tlsClientFollowRedirects {
		options = append(options, tls_client.WithNotFollowRedirects())
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

func (w *tlsClientWrapper) getCustomTlsClientProfile(customClientDefinition *CustomTlsClient) (tls.ClientHelloID, map[http2.SettingID]uint32, []http2.SettingID, []string, uint32, []http2.Priority, error) {
	specFactory, err := tls_client.GetSpecFactorFromJa3String(customClientDefinition.Ja3String)

	if err != nil {
		return tls.ClientHelloID{}, nil, nil, nil, 0, nil, err
	}

	h2Settings := make(map[http2.SettingID]uint32)
	for key, value := range customClientDefinition.H2Settings {
		h2Settings[http2.SettingID(key)] = value
	}

	var h2SettingsOrder []http2.SettingID
	for _, order := range customClientDefinition.H2SettingsOrder {
		h2SettingsOrder = append(h2SettingsOrder, http2.SettingID(order))
	}

	pseudoHeaderOrder := customClientDefinition.PseudoHeaderOrder
	connectionFlow := customClientDefinition.ConnectionFlow

	var priorityFrames []http2.Priority
	for _, priority := range customClientDefinition.PriorityFrames {
		priorityFrames = append(priorityFrames, http2.Priority{
			StreamID: priority.StreamID,
			PriorityParam: http2.PriorityParam{
				StreamDep: priority.PriorityParam.StreamDep,
				Exclusive: priority.PriorityParam.Exclusive,
				Weight:    priority.PriorityParam.Weight,
			},
		})
	}

	clientHelloId := tls.ClientHelloID{
		Client:      "Custom",
		Version:     "1",
		Seed:        nil,
		SpecFactory: specFactory,
	}

	return clientHelloId, h2Settings, h2SettingsOrder, pseudoHeaderOrder, connectionFlow, priorityFrames, nil
}

func (w *tlsClientWrapper) handleModification(client tls_client.HttpClient, proxyUrl *string, followRedirects bool) (tls_client.HttpClient, bool, error) {
	changed := false

	if proxyUrl != nil && client.GetProxy() != *proxyUrl {
		err := client.SetProxy(*proxyUrl)
		if err != nil {
			return nil, false, fmt.Errorf("failed to change proxy url of client: %w", err)
		}

		changed = true
	}

	if client.GetFollowRedirect() != followRedirects {
		client.SetFollowRedirect(followRedirects)
		changed = true
	}

	return client, changed, nil
}
