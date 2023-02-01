package api

import (
	"context"
	"fmt"

	tls_client_cffi_src "github.com/bogdanfinn/tls-client/cffi_src"
	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/apiserver"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type ForwardedRequestHandler struct {
	logger log.Logger
}

func NewForwardedRequestHandler(ctx context.Context, config cfg.Config, logger log.Logger) (gin.HandlerFunc, error) {
	handler := ForwardedRequestHandler{
		logger: logger,
	}

	return apiserver.CreateJsonHandler(handler), nil
}

func (fh ForwardedRequestHandler) GetInput() interface{} {
	return &tls_client_cffi_src.RequestInput{}
}

func (fh ForwardedRequestHandler) Handle(ctx context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	input, ok := request.Body.(*tls_client_cffi_src.RequestInput)

	if !ok {
		err := tls_client_cffi_src.NewTLSClientError(fmt.Errorf("bad request body provided"))
		return handleErrorResponse(fh.logger, "", false, err)
	}

	tlsClient, sessionId, withSession, err := tls_client_cffi_src.CreateClient(*input)

	if err != nil {
		return handleErrorResponse(fh.logger, sessionId, withSession, err)
	}

	req, err := tls_client_cffi_src.BuildRequest(*input)

	if err != nil {
		clientErr := tls_client_cffi_src.NewTLSClientError(err)
		return handleErrorResponse(fh.logger, sessionId, withSession, clientErr)
	}

	cookies := buildCookies(input.RequestCookies)

	if len(cookies) > 0 {
		tlsClient.SetCookies(req.URL, cookies)
	}

	resp, reqErr := tlsClient.Do(req)

	if reqErr != nil {
		clientErr := tls_client_cffi_src.NewTLSClientError(fmt.Errorf("failed to do request: %w", reqErr))
		return handleErrorResponse(fh.logger, sessionId, withSession, clientErr)
	}

	sessionCookies := tlsClient.GetCookies(resp.Request.URL)

	response, err := tls_client_cffi_src.BuildResponse(sessionId, withSession, resp, sessionCookies, *input)

	if err != nil {
		return handleErrorResponse(fh.logger, sessionId, withSession, err)
	}

	return apiserver.NewJsonResponse(response), nil
}
