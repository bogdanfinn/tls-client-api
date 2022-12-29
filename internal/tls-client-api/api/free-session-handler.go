package api

import (
	"context"
	"fmt"

	tls_client_cffi_src "github.com/bogdanfinn/tls-client/cffi_src"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/justtrackio/gosoline/pkg/apiserver"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type FreeSessionHandler struct {
	logger log.Logger
}

func NewFreeSessionHandler(ctx context.Context, config cfg.Config, logger log.Logger) (gin.HandlerFunc, error) {
	handler := FreeSessionHandler{
		logger: logger,
	}

	return apiserver.CreateJsonHandler(handler), nil
}

func (fh FreeSessionHandler) GetInput() interface{} {
	return &tls_client_cffi_src.DestroySessionInput{}
}

func (fh FreeSessionHandler) Handle(ctx context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	input, ok := request.Body.(*tls_client_cffi_src.DestroySessionInput)

	if !ok {
		err := tls_client_cffi_src.NewTLSClientError(fmt.Errorf("bad request body provided"))
		return handleErrorResponse(fh.logger, "", false, err)
	}

	tls_client_cffi_src.RemoveSession(input.SessionId)

	out := tls_client_cffi_src.DestroyOutput{
		Id:      uuid.New().String(),
		Success: true,
	}

	return apiserver.NewJsonResponse(out), nil
}
