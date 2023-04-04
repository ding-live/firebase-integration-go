package ding

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

const (
	APIURL    = "https://api.ding.live/v1"
	AuthPath  = "authentication"
	CheckPath = "check"

	APIKeyHeader = "x-api-key"
)

var (
	ErrUnauthorized = errors.New("invalid credentials")
	ErrInvalidInput = errors.New("invalid input body")
	ErrRateLimited  = errors.New("rate limited")
)

// Client represents a Ding API client.
type Client struct {
	apiKey       string
	customerUUID string
	client       *http.Client
}

// Retrieves the configuration of the Ding API.
type Params struct {
	// The API key to use when making requests to the Ding API.
	// See the [authentication docs](https://docs.ding.live/api/authentication) for more information.
	APIKey string
	// The UUID of the customer to use when making requests to the Ding API.
	CustomerUUID string
}

func New(params *Params) *Client {
	return &Client{apiKey: params.APIKey, customerUUID: params.CustomerUUID, client: &http.Client{}}
}

// Authenticate sends a verification code to the given phone number.
func (c *Client) Authenticate(ctx context.Context, phoneNumber string) (string, error) {
	type request struct {
		PhoneNumber  string `json:"phone_number"`
		CustomerUUID string `json:"customer_uuid"`
	}

	type response struct {
		AuthenticationUUID string `json:"authentication_uuid"`
		Status             string `json:"status"`
	}

	body := request{PhoneNumber: phoneNumber, CustomerUUID: c.customerUUID}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/%s", APIURL, AuthPath), bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("new request: %v", err)
	}

	req.Header.Set(APIKeyHeader, c.apiKey)
	req.Header.Set("content-type", "application/json")

	res, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("post request: %v", err)
	}

	if res.StatusCode == http.StatusBadRequest {
		return "", ErrInvalidInput
	}
	if res.StatusCode == http.StatusUnauthorized {
		return "", ErrUnauthorized
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	var resp response
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return "", fmt.Errorf("decode response: %v", err)
	}

	if resp.Status == "rate_limited" {
		return "", ErrRateLimited
	}

	return resp.AuthenticationUUID, nil
}

// Check verifies the given code against the given phone number.
func (c *Client) Check(ctx context.Context, authUUID uuid.UUID, phoneNumber string, code string) (bool, error) {
	type request struct {
		CustomerUUID       string `json:"customer_uuid"`
		AuthenticationUUID string `json:"authentication_uuid"`
		CheckCode          string `json:"check_code"`
	}

	type response struct {
		AuthenticationUUID string `json:"authentication_uuid"`
		Status             string `json:"status"`
	}

	body := request{
		CustomerUUID:       c.customerUUID,
		AuthenticationUUID: authUUID.String(),
		CheckCode:          code,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return false, fmt.Errorf("marshal request: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/%s", APIURL, CheckPath), bytes.NewBuffer(bodyBytes))
	if err != nil {
		return false, fmt.Errorf("new request: %v", err)
	}

	req.Header.Set(APIKeyHeader, c.apiKey)
	req.Header.Set("content-type", "application/json")

	res, err := c.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("post request: %v", err)
	}

	if res.StatusCode == http.StatusBadRequest {
		return false, ErrInvalidInput
	}
	if res.StatusCode == http.StatusUnauthorized {
		return false, ErrUnauthorized
	}

	var resp response
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return false, fmt.Errorf("decode response: %v", err)
	}

	if resp.Status == "valid" {
		return true, nil
	}

	return false, nil
}
