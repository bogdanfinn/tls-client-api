package api

import (
	"context"
	"fmt"
	"net/url"

	tls_client_cffi_src "github.com/bogdanfinn/tls-client/cffi_src"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/justtrackio/gosoline/pkg/apiserver"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type AddCookiesHandler struct {
	logger log.Logger
}

func NewAddCookiesHandler(ctx context.Context, config cfg.Config, logger log.Logger) (gin.HandlerFunc, error) {
	handler := AddCookiesHandler{
		logger: logger,
	}

	return apiserver.CreateJsonHandler(handler), nil
}

func (fh AddCookiesHandler) GetInput() interface{} {
	return &tls_client_cffi_src.AddCookiesToSessionInput{}
}

func (fh AddCookiesHandler) Handle(ctx context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	input, ok := request.Body.(*tls_client_cffi_src.AddCookiesToSessionInput)

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

	cookies := buildCookies(input.Cookies)

	if len(cookies) > 0 {
		tlsClient.SetCookies(u, cookies)
	}

	out := tls_client_cffi_src.CookiesFromSessionOutput{
		Id:      uuid.New().String(),
		Cookies: transformCookies(cookies),
	}

	return apiserver.NewJsonResponse(out), nil
}
