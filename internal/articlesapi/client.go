package articlesapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// todo: To optimize the number of rps to an external server,
// it must support batching. This will allow us to increase
// throughput with fewer requests to the server.

const (
	baseURL = "https://jsonmock.hackerrank.com/api/articles"
	// rate limiting: the external server definitely having rate limit per ipAddress
	// therefore it is better to be able to regulate it from our side as well
	// to avoid possible ban (will explain).
	// todo: possible to add run argument
	MaxRPSPerCurrentHost = 10.0
	burstPerSecond       = 1
)

type Client struct {
	logger     *zap.Logger
	httpClient *http.Client
	limiter    *rate.Limiter
}

func New(
	logger *zap.Logger,
) *Client {
	return &Client{
		logger: logger,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				MaxConnsPerHost:       1,
				MaxIdleConns:          1,
				MaxIdleConnsPerHost:   1,
				IdleConnTimeout:       60 * time.Second,
				ResponseHeaderTimeout: 30 * time.Second,
				ExpectContinueTimeout: 0,
				ForceAttemptHTTP2:     true,
			},
		},
		// gradually distributed requests smoothly(1 req / 100ms)
		limiter: rate.NewLimiter(rate.Limit(MaxRPSPerCurrentHost), burstPerSecond),
	}
}

func (c *Client) FetchPage(ctx context.Context, page int) (*Response, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s?page=%d", baseURL, page), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// todo: it case of network issues "keep alive":
		// logs, retry with backoff,
		// checking all possible errors to prevent fall of the app
		// with errors.Is or errors.As including RestAPI errors
		// like (4xx / 5xx / 429), ratelimit, panic recovery ...
		c.logger.Error("external API error", zap.Error(err))

		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d for page %d", resp.StatusCode, page)
	}

	var apiResp Response
	// for a good boost of performance(x3 minimum) and to avoid reflection under the hood
	// better to use codegen for marshal/unmarshal for example:
	// https://github.com/mailru/easyjson
	if err = json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode page %d: %w", page, err)
	}

	return &apiResp, nil
}
