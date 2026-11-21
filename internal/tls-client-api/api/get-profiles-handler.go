package api

import (
	"context"
	"sort"

	"github.com/bogdanfinn/tls-client/profiles"
	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/apiserver"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type GetProfilesHandler struct{}

func NewGetProfilesHandler(ctx context.Context, config cfg.Config, logger log.Logger) (gin.HandlerFunc, error) {
	return apiserver.CreateHandler(GetProfilesHandler{}), nil
}

func (GetProfilesHandler) Handle(ctx context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	identifiers := make([]string, 0, len(profiles.MappedTLSClients))
	for identifier := range profiles.MappedTLSClients {
		identifiers = append(identifiers, identifier)
	}
	sort.Strings(identifiers)

	return apiserver.NewJsonResponse(identifiers), nil
}
