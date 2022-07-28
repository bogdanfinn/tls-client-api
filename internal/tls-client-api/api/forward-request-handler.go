package api

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"

	http "github.com/bogdanfinn/fhttp"
	tls_client_wrapper "github.com/bogdanfinn/tls-client-api/internal/tls-client-api/tls-client-wrapper"
	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/apiserver"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type ForwardedRequestHandler struct {
	logger           log.Logger
	tlsClientWrapper tls_client_wrapper.TLSClientWrapper
}

type ForwardedRequestHandlerRequest struct {
	TLSClientIdentifier string            `json:"tlsClientIdentifier"`
	ProxyUrl            *string           `json:"proxyUrl"`
	Headers             map[string]string `json:"headers"`
	HeaderOrder         []string          `json:"headerOrder"`
	RequestUrl          string            `json:"requestUrl"`
	RequestMethod       string            `json:"requestMethod"`
	RequestBody         *string           `json:"requestBody"`
	RequestCookies      map[string]string `json:"requestCookies"` // TODO: implement
}

type ForwardedRequestHandlerResponse struct {
	StatusCode      int                 `json:"statusCode"`
	ResponseBody    string              `json:"responseBody"`
	ResponseHeaders map[string][]string `json:"responseHeaders"`
	ResponseCookies map[string]string   `json:"responseCookies"`
	SessionCookies  map[string]string   `json:"sessionCookies"`
}

func NewForwardedRequestHandler(ctx context.Context, config cfg.Config, logger log.Logger) (gin.HandlerFunc, error) {
	tlsClientWrapper, err := tls_client_wrapper.NewTLSClientWrapper(ctx, config, logger)

	if err != nil {
		return nil, fmt.Errorf("can not create tlsClientWrapper: %w", err)
	}

	handler := ForwardedRequestHandler{
		logger:           logger,
		tlsClientWrapper: tlsClientWrapper,
	}

	return apiserver.CreateJsonHandler(handler), nil
}

func (fh ForwardedRequestHandler) GetInput() interface{} {
	return &ForwardedRequestHandlerRequest{}
}

func (fh ForwardedRequestHandler) Handle(ctx context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	input, ok := request.Body.(*ForwardedRequestHandlerRequest)

	if !ok {
		return nil, fmt.Errorf("bad request body provided")
	}

	var tlsReq *http.Request
	var err error

	if input.RequestBody != nil {
		requestBody := bytes.NewBuffer([]byte(mdl.EmptyIfNil(input.RequestBody)))
		tlsReq, err = http.NewRequest(input.RequestMethod, input.RequestUrl, requestBody)
	} else {
		tlsReq, err = http.NewRequest(input.RequestMethod, input.RequestUrl, nil)
	}

	if input.RequestMethod == "" || input.RequestUrl == "" {
		return nil, fmt.Errorf("no request url or request method provided")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create request object: %w", err)
	}

	headers := http.Header{
		http.HeaderOrderKey: input.HeaderOrder,
	}

	for key, value := range input.Headers {
		headers[key] = []string{value}
	}

	tlsReq.Header = headers

	tlsResp, sessionCookies, err := fh.tlsClientWrapper.Do(input.TLSClientIdentifier, input.ProxyUrl, tlsReq)

	if err != nil {
		return nil, fmt.Errorf("failed to do tls-client request: %w", err)
	}

	defer tlsResp.Body.Close()

	respBodyBytes, err := ioutil.ReadAll(tlsResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	resp := ForwardedRequestHandlerResponse{
		StatusCode:      tlsResp.StatusCode,
		ResponseBody:    string(respBodyBytes),
		ResponseHeaders: tlsResp.Header,
		ResponseCookies: CookiesToMap(tlsResp.Cookies()),
		SessionCookies:  CookiesToMap(sessionCookies),
	}

	return apiserver.NewJsonResponse(resp), nil
}

func CookiesToMap(cookies []*http.Cookie) map[string]string {
	ret := make(map[string]string, 0)

	for _, c := range cookies {
		ret[c.Name] = c.String()
	}

	return ret
}
