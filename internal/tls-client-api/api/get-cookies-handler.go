package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	tls_client_cffi_src "github.com/bogdanfinn/tls-client/cffi_src"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/justtrackio/gosoline/pkg/apiserver"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type GetCookiesHandler struct {
	logger log.Logger
}

func NewGetCookiesHandler(ctx context.Context, config cfg.Config, logger log.Logger) (gin.HandlerFunc, error) {
	handler := GetCookiesHandler{
		logger: logger,
	}

	return apiserver.CreateJsonHandler(handler), nil
}

func (fh GetCookiesHandler) GetInput() interface{} {
	return &tls_client_cffi_src.GetCookiesFromSessionInput{}
}

func (fh GetCookiesHandler) Handle(ctx context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	input, ok := request.Body.(*tls_client_cffi_src.GetCookiesFromSessionInput)

	if !ok {
		err := tls_client_cffi_src.NewTLSClientError(fmt.Errorf("bad request body provided"))
		return handleErrorResponse(fh.logger, "", false, err)
	}

	tlsClient, err := tls_client_cffi_src.GetClient(input.SessionId)

	if err != nil {
		clientErr := tls_client_cffi_src.NewTLSClientError(err)
		return handleErrorResponse(fh.logger, input.SessionId, true, clientErr)
	}

	u, parsErr := url.Parse(input.Url)
	if parsErr != nil {
		clientErr := tls_client_cffi_src.NewTLSClientError(parsErr)
		return handleErrorResponse(fh.logger, input.SessionId, true, clientErr)
	}

	cookies := tlsClient.GetCookies(u)

	out := tls_client_cffi_src.CookiesFromSessionOutput{
		Id:      uuid.New().String(),
		Cookies: transformCookies(cookies),
	}

	jsonResponse, marshallError := json.Marshal(out)

	if marshallError != nil {
		clientErr := tls_client_cffi_src.NewTLSClientError(marshallError)
		return handleErrorResponse(fh.logger, input.SessionId, true, clientErr)
	}

	return apiserver.NewJsonResponse(jsonResponse), nil
}
