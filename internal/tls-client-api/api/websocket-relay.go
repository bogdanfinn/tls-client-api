package api

import (
	"sync"
	"time"

	tls_client "github.com/bogdanfinn/tls-client"
	remotews "github.com/bogdanfinn/websocket"
	localws "github.com/gorilla/websocket"
	"github.com/justtrackio/gosoline/pkg/log"
)

// wsCloseWriteWait bounds how long we wait to flush a close control frame before giving up.
const wsCloseWriteWait = 5 * time.Second

// maxCloseReasonBytes keeps close reasons within the RFC 6455 control frame payload limit (125
// bytes total, 2 of which are used by the status code).
const maxCloseReasonBytes = 123

// wsBridgeSession owns exactly one local <-> remote websocket pair and relays messages between
// them until either side terminates the connection.
type wsBridgeSession struct {
	connID     string
	logger     log.Logger
	localConn  *localws.Conn
	remoteConn *remotews.Conn
	tlsClient  tls_client.HttpClient

	closeOnce sync.Once
}

// run relays messages in both directions until the connection ends, then releases all
// resources owned by this session exactly once.
func (s *wsBridgeSession) run() {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		s.pumpLocalToRemote()
	}()

	go func() {
		defer wg.Done()
		s.pumpRemoteToLocal()
	}()

	wg.Wait()

	s.tlsClient.CloseIdleConnections()
}

func (s *wsBridgeSession) pumpLocalToRemote() {
	for {
		messageType, data, err := s.localConn.ReadMessage()
		if err != nil {
			s.handleLocalReadError(err)
			return
		}

		if err := s.remoteConn.WriteMessage(messageType, data); err != nil {
			s.handleRemoteWriteError(err)
			return
		}
	}
}

func (s *wsBridgeSession) pumpRemoteToLocal() {
	for {
		messageType, data, err := s.remoteConn.ReadMessage()
		if err != nil {
			s.handleRemoteReadError(err)
			return
		}

		if err := s.localConn.WriteMessage(messageType, data); err != nil {
			s.handleLocalWriteError(err)
			return
		}
	}
}

// handleLocalReadError runs when the Node side stops sending data (clean close or abnormal
// disconnect). On a clean close it forwards the exact close code/reason to the remote target.
func (s *wsBridgeSession) handleLocalReadError(err error) {
	s.closeOnce.Do(func() {
		if closeErr, ok := err.(*localws.CloseError); ok {
			deadline := time.Now().Add(wsCloseWriteWait)
			payload := remotews.FormatCloseMessage(closeErr.Code, truncateCloseReason(closeErr.Text))
			_ = s.remoteConn.WriteControl(remotews.CloseMessage, payload, deadline)
		}

		_ = s.remoteConn.Close()
		_ = s.localConn.Close()

		s.logger.Debug("ws bridge connection %s closed by local side", s.connID)
	})
}

// handleRemoteReadError runs when the remote target stops sending data (clean close or
// abnormal disconnect). On a clean close it forwards the exact close code/reason to Node.
func (s *wsBridgeSession) handleRemoteReadError(err error) {
	s.closeOnce.Do(func() {
		if closeErr, ok := err.(*remotews.CloseError); ok {
			deadline := time.Now().Add(wsCloseWriteWait)
			payload := localws.FormatCloseMessage(closeErr.Code, truncateCloseReason(closeErr.Text))
			_ = s.localConn.WriteControl(localws.CloseMessage, payload, deadline)
		}

		_ = s.localConn.Close()
		_ = s.remoteConn.Close()

		s.logger.Debug("ws bridge connection %s closed by remote side", s.connID)
	})
}

// handleRemoteWriteError runs on an internal bridge failure while forwarding local -> remote.
func (s *wsBridgeSession) handleRemoteWriteError(err error) {
	s.closeOnce.Do(func() {
		deadline := time.Now().Add(wsCloseWriteWait)
		payload := localws.FormatCloseMessage(localws.CloseInternalServerErr, "bridge write failure")
		_ = s.localConn.WriteControl(localws.CloseMessage, payload, deadline)

		_ = s.localConn.Close()
		_ = s.remoteConn.Close()

		s.logger.Warn("ws bridge connection %s failed forwarding local message to remote: %s", s.connID, err.Error())
	})
}

// handleLocalWriteError runs on an internal bridge failure while forwarding remote -> local.
func (s *wsBridgeSession) handleLocalWriteError(err error) {
	s.closeOnce.Do(func() {
		deadline := time.Now().Add(wsCloseWriteWait)
		payload := remotews.FormatCloseMessage(remotews.CloseInternalServerErr, "bridge write failure")
		_ = s.remoteConn.WriteControl(remotews.CloseMessage, payload, deadline)

		_ = s.remoteConn.Close()
		_ = s.localConn.Close()

		s.logger.Warn("ws bridge connection %s failed forwarding remote message to local: %s", s.connID, err.Error())
	})
}

func truncateCloseReason(text string) string {
	if len(text) <= maxCloseReasonBytes {
		return text
	}

	return text[:maxCloseReasonBytes]
}
