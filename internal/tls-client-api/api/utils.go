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

func buildCookies(cookies []tls_client_cffi_src.Cookie) []*http.Cookie {
	var ret []*http.Cookie

	for _, cookie := range cookies {
		ret = append(ret, &http.Cookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Path:     cookie.Path,
			Domain:   cookie.Domain,
			MaxAge:   cookie.MaxAge,
			Expires:  cookie.Expires.Time,
			Secure:   cookie.Secure,
			HttpOnly: cookie.HttpOnly,
		})
	}

	return ret
}

func transformCookies(cookies []*http.Cookie) []tls_client_cffi_src.Cookie {
	var ret []tls_client_cffi_src.Cookie

	for _, cookie := range cookies {
		ret = append(ret, tls_client_cffi_src.Cookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Path:     cookie.Path,
			Domain:   cookie.Domain,
			MaxAge:   cookie.MaxAge,
			Secure:   cookie.Secure,
			HttpOnly: cookie.HttpOnly,
			Expires: tls_client_cffi_src.Timestamp{
				Time: cookie.Expires,
			},
		})
	}

	return ret
}
