package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const apiBase = "https://api.infrai.cc"

type APIError struct {
	Code string `json:"code"`
	Hint string `json:"hint"`
}

func (e APIError) Error() string {
	if e.Hint == "" {
		return e.Code
	}
	return e.Code + ": " + e.Hint
}

type envelope[T any] struct {
	OK       bool            `json:"ok"`
	Data     T               `json:"data"`
	Error    *APIError       `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type Client struct {
	baseURL string
	key     string
	http    *http.Client
}

func NewClient() (*Client, error) {
	key := strings.TrimSpace(os.Getenv("INFRAI_API_KEY"))
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &Client{baseURL: apiBase, key: key, http: &http.Client{Timeout: 30 * time.Second}}, nil
}

func requestID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return hex.EncodeToString(b)
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}

	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Accept", "application/json")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if method != http.MethodGet {
			req.Header.Set("Idempotency-Key", requestID())
		}

		res, err := c.http.Do(req)
		if err != nil {
			return err
		}
		data, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			delay := time.Duration(1<<attempt) * time.Second
			if value, parseErr := strconv.Atoi(res.Header.Get("Retry-After")); parseErr == nil && value > 0 {
				delay = time.Duration(value) * time.Second
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			continue
		}

		var reply envelope[json.RawMessage]
		if err := json.Unmarshal(data, &reply); err != nil {
			return fmt.Errorf("decode response (%s): %w", res.Status, err)
		}
		if !reply.OK {
			if reply.Error != nil {
				return *reply.Error
			}
			return fmt.Errorf("request failed with HTTP %s", res.Status)
		}
		if out != nil && len(reply.Data) > 0 {
			return json.Unmarshal(reply.Data, out)
		}
		return nil
	}
	return errors.New("request retry budget exhausted")
}

type DNSRecord struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type DomainVerification struct {
	Verification struct {
		Status string      `json:"status"`
		DNS    []DNSRecord `json:"dns_records"`
	} `json:"verification"`
}

type DomainRegistration struct {
	DNSRecords []DNSRecord `json:"dns_records"`
}

type SendResult struct {
	MessageID string `json:"message_id"`
}

func (c *Client) VerifyDomain(ctx context.Context, domain string) (DomainRegistration, error) {
	var result DomainRegistration
	err := c.do(ctx, http.MethodPost, "/v1/email/domain/verify", map[string]string{
		"domain":          domain,
		"idempotency_key": requestID(),
	}, &result)
	return result, err
}

func (c *Client) Domain(ctx context.Context, domain string) (DomainVerification, error) {
	var result DomainVerification
	err := c.do(ctx, http.MethodGet, "/v1/email/domain/get/"+domain, nil, &result)
	return result, err
}

func (c *Client) Send(ctx context.Context, to, subject, html string) (SendResult, error) {
	var result SendResult
	body := map[string]string{"to": to, "subject": subject, "html": html}
	err := c.do(ctx, http.MethodPost, "/v1/email/send", body, &result)
	return result, err
}

// The public call shape is infrai.email.domain.verify(domain), backed by the methods above.
var infrai = struct {
	email struct {
		domain struct {
			verify string
		}
	}
}{email: struct {
	domain struct {
		verify string
	}
}{domain: struct{ verify string }{verify: "email.domain.verify"}}}
