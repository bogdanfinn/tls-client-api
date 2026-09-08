package api

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	localws "github.com/gorilla/websocket"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

// WebsocketBridgeHandler implements GET /api/ws: a generic WebSocket-to-WebSocket bridge
// between a local Node.js client and a remote target, tunnelled through bogdanfinn/tls-client
// so the remote connection gets tls-client's TLS fingerprinting, proxy support and profiles.
//
// The local connection is only upgraded (HTTP 101) after the remote websocket handshake has
// already succeeded. This is a plain gin.HandlerFunc rather than the apiserver JSON handler
// abstraction because it needs to perform an HTTP Upgrade instead of returning a JSON body.
type WebsocketBridgeHandler struct {
	logger log.Logger
}

func NewWebsocketBridgeHandler(ctx context.Context, config cfg.Config, logger log.Logger) (gin.HandlerFunc, error) {
	handler := &WebsocketBridgeHandler{
		logger: logger.WithChannel("ws-bridge"),
	}

	return handler.Handle, nil
}

func (h *WebsocketBridgeHandler) Handle(ginCtx *gin.Context) {
	connID := uuid.New().String()

	rawConfig := ginCtx.GetHeader(wsConfigHeaderName)

	wsCfg, err := decodeWsConfig(rawConfig)
	if err != nil {
		h.logger.Warn("ws bridge connection %s rejected: invalid config: %s", connID, err.Error())
		ginCtx.JSON(http.StatusBadRequest, gin.H{"error": "invalid websocket configuration"})
		return
	}

	tlsClient, err := buildRemoteHttpClient(wsCfg)
	if err != nil {
		h.logger.Warn("ws bridge connection %s rejected: can not build tls client: %s", connID, err.Error())
		ginCtx.JSON(http.StatusBadRequest, gin.H{"error": "invalid websocket configuration"})
		return
	}

	remoteHeaders := wsCfg.buildRemoteHeaders()
	handshakeTimeoutMs := wsCfg.effectiveHandshakeTimeoutMilliseconds()

	remoteWs, err := tls_client.NewWebsocket(tls_client.NewNoopLogger(),
		tls_client.WithTlsClient(tlsClient),
		tls_client.WithUrl(wsCfg.RequestUrl),
		tls_client.WithHeaders(remoteHeaders),
		tls_client.WithReadBufferSize(wsCfg.ReadBufferSize),
		tls_client.WithWriteBufferSize(wsCfg.WriteBufferSize),
		tls_client.WithHandshakeTimeoutMilliseconds(handshakeTimeoutMs),
	)
	if err != nil {
		tlsClient.CloseIdleConnections()
		h.logger.Warn("ws bridge connection %s rejected: can not build remote websocket: %s", connID, err.Error())
		ginCtx.JSON(http.StatusBadRequest, gin.H{"error": "invalid websocket configuration"})
		return
	}

	targetHost := safeHost(wsCfg.RequestUrl)
	connectStart := time.Now()

	remoteConn, err := remoteWs.Connect(ginCtx.Request.Context())
	if err != nil {
		tlsClient.CloseIdleConnections()

		status := http.StatusBadGateway
		if errors.Is(err, context.DeadlineExceeded) {
			status = http.StatusGatewayTimeout
		}

		h.logger.Warn(
			"ws bridge connection %s remote handshake failed target=%s identifier=%s proxy=%t duration=%s",
			connID, targetHost, wsCfg.TlsClientIdentifier, wsCfg.ProxyUrl != "", time.Since(connectStart),
		)

		ginCtx.JSON(status, gin.H{"error": "remote websocket handshake failed"})
		return
	}

	remoteProtocol := remoteConn.Subprotocol()
	if remoteProtocol != "" && !containsString(wsCfg.Protocols, remoteProtocol) {
		_ = remoteConn.Close()
		tlsClient.CloseIdleConnections()

		h.logger.Warn("ws bridge connection %s remote selected unrequested subprotocol %q", connID, remoteProtocol)
		ginCtx.JSON(http.StatusBadGateway, gin.H{"error": "remote selected unsupported subprotocol"})
		return
	}

	responseHeader := http.Header{}
	if remoteProtocol != "" {
		responseHeader.Set("Sec-WebSocket-Protocol", remoteProtocol)
	}
	// Optional debug/version negotiation metadata; clients must not depend on it.
	responseHeader.Set(wsProtocolResponseHeaderName, wsProtocolVersion)

	upgrader := localws.Upgrader{
		ReadBufferSize:  wsCfg.ReadBufferSize,
		WriteBufferSize: wsCfg.WriteBufferSize,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	localConn, err := upgrader.Upgrade(ginCtx.Writer, ginCtx.Request, responseHeader)
	if err != nil {
		_ = remoteConn.Close()
		tlsClient.CloseIdleConnections()

		h.logger.Warn("ws bridge connection %s local upgrade failed: %s", connID, err.Error())
		return
	}

	h.logger.Info(
		"ws bridge connection %s established target=%s identifier=%s proxy=%t duration=%s",
		connID, targetHost, wsCfg.TlsClientIdentifier, wsCfg.ProxyUrl != "", time.Since(connectStart),
	)

	session := &wsBridgeSession{
		connID:     connID,
		logger:     h.logger,
		localConn:  localConn,
		remoteConn: remoteConn,
		tlsClient:  tlsClient,
	}

	session.run()

	h.logger.Info("ws bridge connection %s closed", connID)
}

// safeHost extracts just the host portion of a URL for logging, never the full URL (which may
// contain sensitive query parameters).
func safeHost(rawUrl string) string {
	parsed, err := url.Parse(rawUrl)
	if err != nil {
		return "unknown"
	}

	return parsed.Host
}
