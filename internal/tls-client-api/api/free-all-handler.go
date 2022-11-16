package api

import (
	"context"

	tls_client_cffi_src "github.com/bogdanfinn/tls-client/cffi_src"
	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/apiserver"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type FreeAllHandler struct {
	logger log.Logger
}

func NewFreeAllHandler(ctx context.Context, config cfg.Config, logger log.Logger) (gin.HandlerFunc, error) {
	handler := FreeAllHandler{
		logger: logger,
	}

	return apiserver.CreateHandler(handler), nil
}

func (fh FreeAllHandler) Handle(ctx context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	err := tls_client_cffi_src.DestroyTlsClientSessions()

	if err != nil {
		clientErr := tls_client_cffi_src.NewTLSClientError(err)
		return handleErrorResponse(fh.logger, "", false, clientErr)
	}

	out := tls_client_cffi_src.FreeOutput{
		Success: true,
	}

	return apiserver.NewJsonResponse(out), nil
}
