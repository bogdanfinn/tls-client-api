package api

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/apiserver"
	"github.com/justtrackio/gosoline/pkg/apiserver/auth"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

func DefineRouter(ctx context.Context, config cfg.Config, logger log.Logger) (*apiserver.Definitions, error) {
	d := &apiserver.Definitions{}
	d.Use(apiserver.Cors(config))
	logger = logger.WithChannel("tls-client-api")

	authenticate := auth.NewChainHandler(map[string]auth.Authenticator{
		auth.ByApiKey: auth.NewConfigKeyAuthenticator(config, logger, auth.ProvideValueFromHeader(auth.HeaderApiKey)),
	})

	d.Use(authenticate)

	forwardedRequestHandler, err := NewForwardedRequestHandler(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create forwardedRequestHandler: %w", err)
	}

	d.POST("api/forward", forwardedRequestHandler)

	getCookiesHandler, err := NewGetCookiesHandler(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create getCookiesHandler: %w", err)
	}

	d.POST("api/cookies", getCookiesHandler)

	freeAllRequestHandler, err := NewFreeAllHandler(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create freeAllRequestHandler: %w", err)
	}

	d.GET("api/free-all", freeAllRequestHandler)

	freeSessionRequestHandler, err := NewFreeSessionHandler(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create freeSessionRequestHandler: %w", err)
	}

	d.POST("api/free-session", freeSessionRequestHandler)

	return d, nil
}
