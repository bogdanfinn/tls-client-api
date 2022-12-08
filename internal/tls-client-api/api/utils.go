package api

import (
	http "github.com/bogdanfinn/fhttp"
	tls_client_cffi_src "github.com/bogdanfinn/tls-client/cffi_src"
	"github.com/google/uuid"
	"github.com/justtrackio/gosoline/pkg/apiserver"
	"github.com/justtrackio/gosoline/pkg/log"
)

func handleErrorResponse(logger log.Logger, sessionId string, withSession bool, err *tls_client_cffi_src.TLSClientError) (*apiserver.Response, error) {
	logger.Error("error during api request handling: %w", err)

	resp := tls_client_cffi_src.Response{
		Id:      uuid.New().String(),
		Status:  0,
		Body:    err.Error(),
		Headers: nil,
		Cookies: nil,
	}

	if withSession {
		resp.SessionId = sessionId
	}

	return apiserver.NewJsonResponse(resp), nil
}

func buildCookies(cookies []tls_client_cffi_src.CookieInput) []*http.Cookie {
	var ret []*http.Cookie

	for _, cookie := range cookies {
		ret = append(ret, &http.Cookie{
			Name:    cookie.Name,
			Value:   cookie.Value,
			Path:    cookie.Path,
			Domain:  cookie.Domain,
			Expires: cookie.Expires.Time,
		})
	}

	return ret
}
