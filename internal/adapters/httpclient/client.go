package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	ErrRateLimited      = errors.New("provider rate limited request")
	ErrResponseTooLarge = errors.New("provider response exceeds configured size limit")
	ErrUpstream         = errors.New("provider upstream failure")
)

type Options struct {
	BaseURL      string
	Timeout      time.Duration
	MaxBodyBytes int64
	MaxAttempts  int
	UserAgent    string
	HTTPClient   *http.Client
}

type Client struct {
	baseURL      *url.URL
	httpClient   *http.Client
	timeout      time.Duration
	maxBodyBytes int64
	maxAttempts  int
	userAgent    string
}

type StatusError struct {
	StatusCode int
	RetryAfter time.Duration
}

func (error StatusError) Error() string {
	return fmt.Sprintf("provider returned HTTP %d", error.StatusCode)
}

func (error StatusError) Unwrap() error {
	if error.StatusCode == http.StatusTooManyRequests {
		return ErrRateLimited
	}
	return ErrUpstream
}

func New(options Options) (*Client, error) {
	baseURL, err := url.Parse(options.BaseURL)
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, errors.New("HTTP client requires an absolute base URL")
	}
	if baseURL.Scheme != "https" && baseURL.Scheme != "http" {
		return nil, errors.New("HTTP client base URL must use HTTP or HTTPS")
	}
	if options.Timeout <= 0 || options.MaxBodyBytes <= 0 || options.MaxAttempts < 1 || strings.TrimSpace(options.UserAgent) == "" {
		return nil, errors.New("HTTP client options must use positive bounds and a user agent")
	}
	client := options.HTTPClient
	if client == nil {
		client = &http.Client{}
	}
	return &Client{baseURL: baseURL, httpClient: client, timeout: options.Timeout, maxBodyBytes: options.MaxBodyBytes, maxAttempts: options.MaxAttempts, userAgent: options.UserAgent}, nil
}

func (client *Client) GetJSON(ctx context.Context, path string, query url.Values, destination any) error {
	return client.getJSON(ctx, path, query, nil, destination)
}

func (client *Client) GetJSONWithHeaders(ctx context.Context, path string, query url.Values, headers http.Header, destination any) error {
	return client.getJSON(ctx, path, query, headers, destination)
}

func (client *Client) PostJSON(ctx context.Context, path string, payload any, destination any) error {
	return client.postJSON(ctx, path, payload, nil, destination)
}

func (client *Client) PostJSONWithHeaders(ctx context.Context, path string, payload any, headers http.Header, destination any) error {
	return client.postJSON(ctx, path, payload, headers, destination)
}

func (client *Client) postJSON(ctx context.Context, path string, payload any, headers http.Header, destination any) error {
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		return errors.New("provider path must be an absolute path within the configured base URL")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode provider request: %w", err)
	}
	requestURL := client.requestURL(path, nil)
	for attempt := 1; attempt <= client.maxAttempts; attempt++ {
		requestContext, cancel := context.WithTimeout(ctx, client.timeout)
		request, err := http.NewRequestWithContext(requestContext, http.MethodPost, requestURL.String(), bytes.NewReader(body))
		if err == nil {
			request.Header.Set("Accept", "application/json")
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("User-Agent", client.userAgent)
			for name, values := range headers {
				for _, value := range values {
					request.Header.Add(name, value)
				}
			}
			response, requestError := client.httpClient.Do(request)
			if requestError == nil {
				err = client.decodeResponse(response, destination)
				_ = response.Body.Close()
			} else {
				err = requestError
			}
		}
		cancel()
		if err == nil {
			return nil
		}
		if !isRetryable(err) || attempt == client.maxAttempts {
			return err
		}
		if err := waitForRetry(ctx, retryDelay(err, attempt)); err != nil {
			return err
		}
	}
	return ErrUpstream
}

func (client *Client) getJSON(ctx context.Context, path string, query url.Values, headers http.Header, destination any) error {
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		return errors.New("provider path must be an absolute path within the configured base URL")
	}

	requestURL := client.requestURL(path, query)
	for attempt := 1; attempt <= client.maxAttempts; attempt++ {
		requestContext, cancel := context.WithTimeout(ctx, client.timeout)
		request, err := http.NewRequestWithContext(requestContext, http.MethodGet, requestURL.String(), nil)
		if err == nil {
			request.Header.Set("Accept", "application/json")
			request.Header.Set("User-Agent", client.userAgent)
			for name, values := range headers {
				for _, value := range values {
					request.Header.Add(name, value)
				}
			}
			response, requestError := client.httpClient.Do(request)
			if requestError == nil {
				err = client.decodeResponse(response, destination)
				_ = response.Body.Close()
			} else {
				err = requestError
			}
		}
		cancel()
		if err == nil {
			return nil
		}
		if !isRetryable(err) || attempt == client.maxAttempts {
			return err
		}
		if err := waitForRetry(ctx, retryDelay(err, attempt)); err != nil {
			return err
		}
	}
	return ErrUpstream
}

func (client *Client) decodeResponse(response *http.Response, destination any) error {
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return StatusError{StatusCode: response.StatusCode, RetryAfter: retryAfter(response.Header.Get("Retry-After"))}
	}
	body := &io.LimitedReader{R: response.Body, N: client.maxBodyBytes + 1}
	decoder := json.NewDecoder(body)
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode provider response: %w", err)
	}
	if _, err := io.Copy(io.Discard, body); err != nil {
		return fmt.Errorf("read provider response: %w", err)
	}
	if body.N == 0 {
		return ErrResponseTooLarge
	}
	return nil
}

func (client *Client) requestURL(path string, query url.Values) *url.URL {
	requestURL := *client.baseURL
	requestURL.Path = path
	requestURL.RawPath = ""
	parameters := requestURL.Query()
	for name, values := range query {
		if _, fixed := parameters[name]; fixed {
			continue
		}
		parameters[name] = append([]string(nil), values...)
	}
	requestURL.RawQuery = parameters.Encode()
	return &requestURL
}

func isRetryable(err error) bool {
	var statusError StatusError
	return errors.As(err, &statusError) && (statusError.StatusCode == http.StatusTooManyRequests || statusError.StatusCode >= http.StatusInternalServerError)
}

func retryDelay(err error, attempt int) time.Duration {
	var statusError StatusError
	if errors.As(err, &statusError) && statusError.RetryAfter > 0 {
		return statusError.RetryAfter
	}
	return time.Duration(attempt) * 100 * time.Millisecond
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func retryAfter(value string) time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}
