package api

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	localws "github.com/gorilla/websocket"
	"github.com/justtrackio/gosoline/pkg/apiserver/auth"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testApiKey = "test-api-key"

func init() {
	gin.SetMode(gin.TestMode)
}

// startRemoteEchoServer starts a local TLS websocket server that echoes every message back
// verbatim and forwards close frames, simulating the "remote target" the bridge connects to.
func startRemoteEchoServer(t *testing.T, subprotocols []string) *httptest.Server {
	t.Helper()

	upgrader := localws.Upgrader{
		Subprotocols: subprotocols,
		CheckOrigin:  func(r *http.Request) bool { return true },
	}

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			messageType, data, err := conn.ReadMessage()
			if err != nil {
				if closeErr, ok := err.(*localws.CloseError); ok {
					_ = conn.WriteMessage(localws.CloseMessage, localws.FormatCloseMessage(closeErr.Code, closeErr.Text))
				}
				return
			}

			if err := conn.WriteMessage(messageType, data); err != nil {
				return
			}
		}
	}))

	t.Cleanup(server.Close)

	return server
}

// startRemoteEchoServerWithCloseNotify behaves like startRemoteEchoServer but also reports the
// close code/reason it received from the bridge on the returned channel.
func startRemoteEchoServerWithCloseNotify(t *testing.T) (*httptest.Server, <-chan localws.CloseError) {
	t.Helper()

	closeCh := make(chan localws.CloseError, 1)
	upgrader := localws.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			messageType, data, err := conn.ReadMessage()
			if err != nil {
				if closeErr, ok := err.(*localws.CloseError); ok {
					closeCh <- *closeErr
					_ = conn.WriteMessage(localws.CloseMessage, localws.FormatCloseMessage(closeErr.Code, closeErr.Text))
				}
				return
			}

			if err := conn.WriteMessage(messageType, data); err != nil {
				return
			}
		}
	}))

	t.Cleanup(server.Close)

	return server, closeCh
}

// startRemoteEchoServerClosingOnTrigger echoes messages until it receives the text message
// "please-close", at which point it sends a close frame with a distinctive close code.
func startRemoteEchoServerClosingOnTrigger(t *testing.T) *httptest.Server {
	t.Helper()

	upgrader := localws.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			messageType, data, err := conn.ReadMessage()
			if err != nil {
				return
			}

			if messageType == localws.TextMessage && string(data) == "please-close" {
				_ = conn.WriteMessage(localws.CloseMessage, localws.FormatCloseMessage(4001, "server done"))
				return
			}

			if err := conn.WriteMessage(messageType, data); err != nil {
				return
			}
		}
	}))

	t.Cleanup(server.Close)

	return server
}

// startBridgeServer starts the /api/ws bridge handler behind the same auth middleware used in
// production, so tests also exercise authentication.
func startBridgeServer(t *testing.T) *httptest.Server {
	t.Helper()

	router := gin.New()

	authenticator := auth.NewConfigKeyAuthenticatorWithInterfaces(log.NewLogger(), []string{testApiKey}, auth.ProvideValueFromHeader(auth.HeaderApiKey))
	router.Use(auth.NewChainHandler(map[string]auth.Authenticator{auth.ByApiKey: authenticator}))

	handler := &WebsocketBridgeHandler{logger: log.NewLogger()}
	router.GET("/api/ws", handler.Handle)

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return server
}

func wsConfigHeaderValue(t *testing.T, cfg map[string]interface{}) string {
	t.Helper()

	data, err := json.Marshal(cfg)
	require.NoError(t, err)

	return base64.RawURLEncoding.EncodeToString(data)
}

func toWsUrl(httpUrl string) string {
	if strings.HasPrefix(httpUrl, "https://") {
		return "wss://" + strings.TrimPrefix(httpUrl, "https://")
	}

	return "ws://" + strings.TrimPrefix(httpUrl, "http://")
}

func dialBridge(t *testing.T, bridgeServer *httptest.Server, apiKey string, cfg map[string]interface{}) (*localws.Conn, *http.Response, error) {
	t.Helper()

	header := http.Header{}
	if apiKey != "" {
		header.Set(auth.HeaderApiKey, apiKey)
	}
	header.Set(wsConfigHeaderName, wsConfigHeaderValue(t, cfg))

	dialer := localws.Dialer{HandshakeTimeout: 5 * time.Second}

	return dialer.Dial(toWsUrl(bridgeServer.URL)+"/api/ws", header)
}

func echoConfig(remote *httptest.Server) map[string]interface{} {
	return map[string]interface{}{
		"version":             1,
		"requestUrl":          toWsUrl(remote.URL),
		"tlsClientIdentifier": "chrome_133",
		"insecureSkipVerify":  true,
	}
}

func TestWebsocketBridge_RequiresAuthentication(t *testing.T) {
	remote := startRemoteEchoServer(t, nil)
	bridge := startBridgeServer(t)

	conn, resp, err := dialBridge(t, bridge, "", echoConfig(remote))
	require.Error(t, err)
	require.NotNil(t, resp)
	assert.Nil(t, conn)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestWebsocketBridge_WrongApiKey(t *testing.T) {
	remote := startRemoteEchoServer(t, nil)
	bridge := startBridgeServer(t)

	conn, resp, err := dialBridge(t, bridge, "wrong-key", echoConfig(remote))
	require.Error(t, err)
	require.NotNil(t, resp)
	assert.Nil(t, conn)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestWebsocketBridge_MalformedConfigRejectedBeforeUpgrade(t *testing.T) {
	bridge := startBridgeServer(t)

	header := http.Header{}
	header.Set(auth.HeaderApiKey, testApiKey)
	header.Set(wsConfigHeaderName, "not-valid-base64!!")

	dialer := localws.Dialer{}
	conn, resp, err := dialer.Dial(toWsUrl(bridge.URL)+"/api/ws", header)

	require.Error(t, err)
	assert.Nil(t, conn)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestWebsocketBridge_UnsupportedVersionRejectedBeforeUpgrade(t *testing.T) {
	bridge := startBridgeServer(t)

	cfg := map[string]interface{}{
		"version":             2,
		"requestUrl":          "wss://example.com/ws",
		"tlsClientIdentifier": "chrome_133",
	}

	conn, resp, err := dialBridge(t, bridge, testApiKey, cfg)
	require.Error(t, err)
	assert.Nil(t, conn)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestWebsocketBridge_InvalidRequestUrlRejectedBeforeUpgrade(t *testing.T) {
	bridge := startBridgeServer(t)

	cfg := map[string]interface{}{
		"version":             1,
		"requestUrl":          "http://example.com/ws",
		"tlsClientIdentifier": "chrome_133",
	}

	conn, resp, err := dialBridge(t, bridge, testApiKey, cfg)
	require.Error(t, err)
	assert.Nil(t, conn)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestWebsocketBridge_RemoteFailureBeforeUpgrade(t *testing.T) {
	bridge := startBridgeServer(t)

	cfg := map[string]interface{}{
		"version":                      1,
		"requestUrl":                   "wss://127.0.0.1:1/",
		"tlsClientIdentifier":          "chrome_133",
		"handshakeTimeoutMilliseconds": 2000,
	}

	conn, resp, err := dialBridge(t, bridge, testApiKey, cfg)
	require.Error(t, err)
	assert.Nil(t, conn)
	require.NotNil(t, resp)
	assert.NotEqual(t, http.StatusSwitchingProtocols, resp.StatusCode)
	assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
}

func TestWebsocketBridge_TextAndBinaryRelay(t *testing.T) {
	remote := startRemoteEchoServer(t, nil)
	bridge := startBridgeServer(t)

	conn, resp, err := dialBridge(t, bridge, testApiKey, echoConfig(remote))
	require.NoError(t, err)
	require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	defer conn.Close()

	require.NoError(t, conn.WriteMessage(localws.TextMessage, []byte("hello")))
	mt, data, err := conn.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, localws.TextMessage, mt)
	assert.Equal(t, "hello", string(data))

	payload := []byte{0x00, 0x01, 0x02, 0xff}
	require.NoError(t, conn.WriteMessage(localws.BinaryMessage, payload))
	mt, data, err = conn.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, localws.BinaryMessage, mt)
	assert.Equal(t, payload, data)

	engineIoMessage := `42["event",{"x":1}]`
	require.NoError(t, conn.WriteMessage(localws.TextMessage, []byte(engineIoMessage)))
	mt, data, err = conn.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, localws.TextMessage, mt)
	assert.Equal(t, engineIoMessage, string(data))
}

// startConnectProxy starts a minimal HTTP CONNECT proxy for testing proxyUrl handling; it
// reports how many CONNECT tunnels it has established.
func startConnectProxy(t *testing.T) (addr string, connectCount *int32) {
	t.Helper()

	var count int32

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			go serveConnectTunnel(conn, &count)
		}
	}()

	return listener.Addr().String(), &count
}

func serveConnectTunnel(conn net.Conn, count *int32) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		return
	}

	if req.Method != http.MethodConnect {
		_, _ = conn.Write([]byte("HTTP/1.1 405 Method Not Allowed\r\n\r\n"))
		return
	}

	target, err := net.Dial("tcp", req.Host)
	if err != nil {
		_, _ = conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer target.Close()

	atomic.AddInt32(count, 1)
	_, _ = conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	if buffered := reader.Buffered(); buffered > 0 {
		leftover := make([]byte, buffered)
		_, _ = io.ReadFull(reader, leftover)
		_, _ = target.Write(leftover)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); _, _ = io.Copy(target, conn) }()
	go func() { defer wg.Done(); _, _ = io.Copy(conn, target) }()
	wg.Wait()
}

func TestWebsocketBridge_ProxyConfiguration(t *testing.T) {
	remote := startRemoteEchoServer(t, nil)
	proxyAddr, connectCount := startConnectProxy(t)
	bridge := startBridgeServer(t)

	cfg := echoConfig(remote)
	cfg["proxyUrl"] = "http://" + proxyAddr

	conn, resp, err := dialBridge(t, bridge, testApiKey, cfg)
	require.NoError(t, err)
	require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	defer conn.Close()

	require.NoError(t, conn.WriteMessage(localws.TextMessage, []byte("via-proxy")))
	_, data, err := conn.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, "via-proxy", string(data))

	assert.GreaterOrEqual(t, atomic.LoadInt32(connectCount), int32(1), "expected the remote connection to be tunnelled through the configured proxy")
}

func TestWebsocketBridge_ResponseHasProtocolVersionHeader(t *testing.T) {
	remote := startRemoteEchoServer(t, nil)
	bridge := startBridgeServer(t)

	conn, resp, err := dialBridge(t, bridge, testApiKey, echoConfig(remote))
	require.NoError(t, err)
	require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	defer conn.Close()

	assert.Equal(t, wsProtocolVersion, resp.Header.Get(wsProtocolResponseHeaderName))
}

func TestWebsocketBridge_LocalCloseSentToRemote(t *testing.T) {
	remote, closeCh := startRemoteEchoServerWithCloseNotify(t)
	bridge := startBridgeServer(t)

	conn, resp, err := dialBridge(t, bridge, testApiKey, echoConfig(remote))
	require.NoError(t, err)
	require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	defer conn.Close()

	require.NoError(t, conn.WriteMessage(localws.CloseMessage, localws.FormatCloseMessage(localws.CloseNormalClosure, "bye")))

	select {
	case closeErr := <-closeCh:
		assert.Equal(t, localws.CloseNormalClosure, closeErr.Code)
		assert.Equal(t, "bye", closeErr.Text)
	case <-time.After(3 * time.Second):
		t.Fatal("remote did not receive the close frame forwarded from local")
	}
}

func TestWebsocketBridge_RemoteCloseSentToLocal(t *testing.T) {
	remote := startRemoteEchoServerClosingOnTrigger(t)
	bridge := startBridgeServer(t)

	conn, resp, err := dialBridge(t, bridge, testApiKey, echoConfig(remote))
	require.NoError(t, err)
	require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	defer conn.Close()

	require.NoError(t, conn.WriteMessage(localws.TextMessage, []byte("please-close")))

	_, _, err = conn.ReadMessage()
	require.Error(t, err)

	closeErr, ok := err.(*localws.CloseError)
	require.True(t, ok, "expected a close error, got %T: %v", err, err)
	assert.Equal(t, 4001, closeErr.Code)
	assert.Equal(t, "server done", closeErr.Text)
}

func TestWebsocketBridge_SubprotocolNegotiation(t *testing.T) {
	remote := startRemoteEchoServer(t, []string{"chatv2", "chat"})
	bridge := startBridgeServer(t)

	cfg := echoConfig(remote)
	cfg["protocols"] = []string{"chat"}

	conn, resp, err := dialBridge(t, bridge, testApiKey, cfg)
	require.NoError(t, err)
	require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	defer conn.Close()

	assert.Equal(t, "chat", conn.Subprotocol())
}

func TestWebsocketBridge_ConcurrentConnections(t *testing.T) {
	remote := startRemoteEchoServer(t, nil)
	bridge := startBridgeServer(t)

	const concurrency = 10

	var wg sync.WaitGroup
	errCh := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			conn, resp, err := dialBridge(t, bridge, testApiKey, echoConfig(remote))
			if err != nil {
				errCh <- fmt.Errorf("connection %d: dial failed: %w", i, err)
				return
			}
			defer conn.Close()

			if resp.StatusCode != http.StatusSwitchingProtocols {
				errCh <- fmt.Errorf("connection %d: unexpected status %d", i, resp.StatusCode)
				return
			}

			msg := fmt.Sprintf("hello-%d", i)
			if err := conn.WriteMessage(localws.TextMessage, []byte(msg)); err != nil {
				errCh <- fmt.Errorf("connection %d: write failed: %w", i, err)
				return
			}

			_, data, err := conn.ReadMessage()
			if err != nil {
				errCh <- fmt.Errorf("connection %d: read failed: %w", i, err)
				return
			}

			if string(data) != msg {
				errCh <- fmt.Errorf("connection %d: unexpected echo %q", i, data)
				return
			}

			_ = conn.WriteMessage(localws.CloseMessage, localws.FormatCloseMessage(localws.CloseNormalClosure, ""))
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Error(err)
	}
}

func TestWebsocketBridge_NoGoroutineLeakAfterClose(t *testing.T) {
	remote := startRemoteEchoServer(t, nil)
	bridge := startBridgeServer(t)

	runtime.GC()
	baseline := runtime.NumGoroutine()

	for i := 0; i < 5; i++ {
		conn, resp, err := dialBridge(t, bridge, testApiKey, echoConfig(remote))
		require.NoError(t, err)
		require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)

		require.NoError(t, conn.WriteMessage(localws.TextMessage, []byte("ping")))
		_, _, err = conn.ReadMessage()
		require.NoError(t, err)

		require.NoError(t, conn.WriteMessage(localws.CloseMessage, localws.FormatCloseMessage(localws.CloseNormalClosure, "")))
		_, _, _ = conn.ReadMessage()
		require.NoError(t, conn.Close())
	}

	deadline := time.Now().Add(3 * time.Second)
	for runtime.NumGoroutine() > baseline+5 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}

	assert.LessOrEqual(t, runtime.NumGoroutine(), baseline+5, "goroutines appear to have leaked after connections closed")
}
