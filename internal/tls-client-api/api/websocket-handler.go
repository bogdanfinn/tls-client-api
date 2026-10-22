package api

import (
	"context"
	"fmt"
	"time"

	tls_client_cffi_src "github.com/bogdanfinn/tls-client/cffi_src"
	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/apiserver"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

// wsReadTimeoutMargin leaves time to write the read response before the server write timeout hits.
const wsReadTimeoutMargin = time.Second

type WsConnectHandler struct {
	logger log.Logger
}

func NewWsConnectHandler(ctx context.Context, config cfg.Config, logger log.Logger) (gin.HandlerFunc, error) {
	handler := WsConnectHandler{
		logger: logger,
	}

	return apiserver.CreateJsonHandler(handler), nil
}

func (wh WsConnectHandler) GetInput() interface{} {
	return &tls_client_cffi_src.WsConnectInput{}
}

func (wh WsConnectHandler) Handle(ctx context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	input, ok := request.Body.(*tls_client_cffi_src.WsConnectInput)

	if !ok {
		err := tls_client_cffi_src.NewTLSClientError(fmt.Errorf("bad request body provided"))
		return handleErrorResponse(wh.logger, "", false, err)
	}

	out, clientErr := tls_client_cffi_src.WsConnect(*input)
	if clientErr != nil {
		withSession := input.SessionId != nil && *input.SessionId != ""
		sessionId := ""
		if withSession {
			sessionId = *input.SessionId
		}

		return handleErrorResponse(wh.logger, sessionId, withSession, clientErr)
	}

	return apiserver.NewJsonResponse(out), nil
}

type WsReadHandler struct {
	logger         log.Logger
	maxReadTimeout time.Duration
}

func NewWsReadHandler(ctx context.Context, config cfg.Config, logger log.Logger) (gin.HandlerFunc, error) {
	settings := &apiserver.Settings{}
	config.UnmarshalKey("api", settings)

	handler := WsReadHandler{
		logger:         logger,
		maxReadTimeout: settings.Timeout.Write - wsReadTimeoutMargin,
	}

	return apiserver.CreateJsonHandler(handler), nil
}

func (wh WsReadHandler) GetInput() interface{} {
	return &tls_client_cffi_src.WsReadInput{}
}

func (wh WsReadHandler) Handle(ctx context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	input, ok := request.Body.(*tls_client_cffi_src.WsReadInput)

	if !ok {
		err := tls_client_cffi_src.NewTLSClientError(fmt.Errorf("bad request body provided"))
		return handleErrorResponse(wh.logger, "", false, err)
	}

	// Without a bound the read outlives the HTTP request and the next message would be consumed
	// after the server already dropped the response, so the message would be lost.
	maxReadTimeoutMilliseconds := int(wh.maxReadTimeout.Milliseconds())
	if input.TimeoutMilliseconds <= 0 || input.TimeoutMilliseconds > maxReadTimeoutMilliseconds {
		input.TimeoutMilliseconds = maxReadTimeoutMilliseconds
	}

	out, clientErr := tls_client_cffi_src.WsRead(*input)
	if clientErr != nil {
		return handleErrorResponse(wh.logger, "", false, clientErr)
	}

	return apiserver.NewJsonResponse(out), nil
}

type WsWriteHandler struct {
	logger log.Logger
}

func NewWsWriteHandler(ctx context.Context, config cfg.Config, logger log.Logger) (gin.HandlerFunc, error) {
	handler := WsWriteHandler{
		logger: logger,
	}

	return apiserver.CreateJsonHandler(handler), nil
}

func (wh WsWriteHandler) GetInput() interface{} {
	return &tls_client_cffi_src.WsWriteInput{}
}

func (wh WsWriteHandler) Handle(ctx context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	input, ok := request.Body.(*tls_client_cffi_src.WsWriteInput)

	if !ok {
		err := tls_client_cffi_src.NewTLSClientError(fmt.Errorf("bad request body provided"))
		return handleErrorResponse(wh.logger, "", false, err)
	}

	out, clientErr := tls_client_cffi_src.WsWrite(*input)
	if clientErr != nil {
		return handleErrorResponse(wh.logger, "", false, clientErr)
	}

	return apiserver.NewJsonResponse(out), nil
}

type WsCloseHandler struct {
	logger log.Logger
}

func NewWsCloseHandler(ctx context.Context, config cfg.Config, logger log.Logger) (gin.HandlerFunc, error) {
	handler := WsCloseHandler{
		logger: logger,
	}

	return apiserver.CreateJsonHandler(handler), nil
}

func (wh WsCloseHandler) GetInput() interface{} {
	return &tls_client_cffi_src.WsCloseInput{}
}

func (wh WsCloseHandler) Handle(ctx context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	input, ok := request.Body.(*tls_client_cffi_src.WsCloseInput)

	if !ok {
		err := tls_client_cffi_src.NewTLSClientError(fmt.Errorf("bad request body provided"))
		return handleErrorResponse(wh.logger, "", false, err)
	}

	out, clientErr := tls_client_cffi_src.WsClose(*input)
	if clientErr != nil {
		return handleErrorResponse(wh.logger, "", false, clientErr)
	}

	return apiserver.NewJsonResponse(out), nil
}

type WsStatsHandler struct {
	logger log.Logger
}

func NewWsStatsHandler(ctx context.Context, config cfg.Config, logger log.Logger) (gin.HandlerFunc, error) {
	handler := WsStatsHandler{
		logger: logger,
	}

	return apiserver.CreateJsonHandler(handler), nil
}

func (wh WsStatsHandler) GetInput() interface{} {
	return &tls_client_cffi_src.WsStatsInput{}
}

func (wh WsStatsHandler) Handle(ctx context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	input, ok := request.Body.(*tls_client_cffi_src.WsStatsInput)

	if !ok {
		err := tls_client_cffi_src.NewTLSClientError(fmt.Errorf("bad request body provided"))
		return handleErrorResponse(wh.logger, "", false, err)
	}

	out, clientErr := tls_client_cffi_src.WsStats(*input)
	if clientErr != nil {
		return handleErrorResponse(wh.logger, "", false, clientErr)
	}

	return apiserver.NewJsonResponse(out), nil
}
