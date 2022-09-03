package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

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
	SessionId           *string           `json:"sessionId"`
	TLSClientIdentifier string            `json:"tlsClientIdentifier"`
	Ja3String           string            `json:"ja3String"`
	ProxyUrl            *string           `json:"proxyUrl"`
	Headers             map[string]string `json:"headers"`
	HeaderOrder         []string          `json:"headerOrder"`
	RequestUrl          string            `json:"requestUrl"`
	RequestMethod       string            `json:"requestMethod"`
	RequestBody         *string           `json:"requestBody"`
	RequestCookies      []CookieInput     `json:"requestCookies"`
}

type CookieInput struct {
	Name    string    `json:"name"`
	Value   string    `json:"value"`
	Path    string    `json:"path"`
	Domain  string    `json:"domain"`
	Expires Timestamp `json:"expires"`
}

type Timestamp struct {
	time.Time
}

func (p *Timestamp) UnmarshalJSON(bytes []byte) error {
	var raw int64
	err := json.Unmarshal(bytes, &raw)

	if err != nil {
		return fmt.Errorf("error decoding timestamp: %w", err)
	}

	*&p.Time = time.Unix(raw, 0)
	return nil
}

type ForwardedRequestHandlerResponse struct {
	SessionId string              `json:"sessionId"`
	Status    int                 `json:"status"`
	Body      string              `json:"body"`
	Headers   map[string][]string `json:"headers"`
	Cookies   map[string]string   `json:"cookies"`
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
		return fh.handleErrorResponse(nil, fmt.Errorf("bad request body provided"))
	}

	tlsReq, err := fh.buildRequest(input)

	if err != nil {
		return fh.handleErrorResponse(nil, fmt.Errorf("failed to create request object: %w", err))
	}

	tlsResp, sessionId, sessionCookies, err := fh.tlsClientWrapper.Do(input.SessionId, input.TLSClientIdentifier, input.Ja3String, input.ProxyUrl, BuildCookies(input.RequestCookies), tlsReq)

	if err != nil {
		return fh.handleErrorResponse(sessionId, fmt.Errorf("failed to do tls-client request: %w", err))
	}

	defer tlsResp.Body.Close()

	respBodyBytes, err := ioutil.ReadAll(tlsResp.Body)
	if err != nil {
		return fh.handleErrorResponse(sessionId, fmt.Errorf("failed to read response body: %w", err))
	}

	resp := ForwardedRequestHandlerResponse{
		SessionId: mdl.EmptyIfNil(sessionId),
		Status:    tlsResp.StatusCode,
		Body:      string(respBodyBytes),
		Headers:   tlsResp.Header,
		Cookies:   CookiesToMap(sessionCookies),
	}

	return apiserver.NewJsonResponse(resp), nil
}

func (fh ForwardedRequestHandler) buildRequest(input *ForwardedRequestHandlerRequest) (*http.Request, error) {
	var tlsReq *http.Request
	var err error

	if input.RequestBody != nil && mdl.EmptyIfNil(input.RequestBody) != "" {
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

	return tlsReq, nil
}

func (fh ForwardedRequestHandler) handleErrorResponse(sessionId *string, err error) (*apiserver.Response, error) {
	fh.logger.Error("error during api request forwarding: %w", err)

	resp := ForwardedRequestHandlerResponse{
		SessionId: mdl.EmptyIfNil(sessionId),
		Status:    0,
		Body:      err.Error(),
		Headers:   nil,
		Cookies:   nil,
	}

	return apiserver.NewJsonResponse(resp), nil
}
