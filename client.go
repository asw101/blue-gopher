//go:build mage
// +build mage

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

// Client represents a client for the Bluesky API
type Client struct {
	BaseURL   string
	AuthToken string
	Session   CreateSessionResponse
}

// CreateSessionResponse represents the structure of the response from the createSession API
type CreateSessionResponse struct {
	DID    string `json:"did"`
	DIDDoc struct {
		Context            []string `json:"@context"`
		ID                 string   `json:"id"`
		AlsoKnownAs        []string `json:"alsoKnownAs"`
		VerificationMethod []struct {
			ID                 string `json:"id"`
			Type               string `json:"type"`
			Controller         string `json:"controller"`
			PublicKeyMultibase string `json:"publicKeyMultibase"`
		} `json:"verificationMethod"`
		Service []struct {
			ID              string `json:"id"`
			Type            string `json:"type"`
			ServiceEndpoint string `json:"serviceEndpoint"`
		} `json:"service"`
	} `json:"didDoc"`
	Handle          string `json:"handle"`
	Email           string `json:"email"`
	EmailConfirmed  bool   `json:"emailConfirmed"`
	EmailAuthFactor bool   `json:"emailAuthFactor"`
	AccessJwt       string `json:"accessJwt"`
	RefreshJwt      string `json:"refreshJwt"`
	Active          bool   `json:"active"`
}

// CreateRecordRequest represents the structure of the request to create a record
type CreateRecordRequest struct {
	Repo       string      `json:"repo"`
	Collection string      `json:"collection"`
	Rkey       string      `json:"rkey,omitempty"`
	Validate   bool        `json:"validate,omitempty"`
	Record     interface{} `json:"record"`
	SwapCommit string      `json:"swapCommit,omitempty"`
}

// NewClient creates a new Bluesky API client
func NewClient() (*Client, error) {
	client := &Client{}

	pdshost := os.Getenv("PDSHOST")
	// default to https://bsky.social
	if pdshost == "" {
		pdshost = "https://bsky.social"
	}
	client.BaseURL = pdshost

	_, err := client.CreateSession()
	if err != nil {
		return nil, err
	}

	return client, nil
}

// CreateSession authenticates to the Bluesky API using the provided credentials and sets the AuthToken on the client
func (c *Client) CreateSession() (*CreateSessionResponse, error) {
	user := os.Getenv("BLUESKY_HANDLE")
	pass := os.Getenv("BLUESKY_PASSWORD")

	url := c.BaseURL + "/xrpc/com.atproto.server.createSession"
	req := map[string]string{
		"identifier": user,
		"password":   pass,
	}
	body, err := c.SendRequest("POST", url, req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	var createSessionResponse CreateSessionResponse
	if err := json.Unmarshal(body, &createSessionResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	if createSessionResponse.AccessJwt == "" {
		return nil, fmt.Errorf("failed to authenticate: missing access token")
	}
	c.AuthToken = createSessionResponse.AccessJwt
	c.Session = createSessionResponse
	return &createSessionResponse, nil
}

// SendRequest makes a generic request to a given URL
func (c *Client) SendRequest(method, url string, requestBody interface{}) ([]byte, error) {
	var b []byte
	var err error
	if requestBody != nil {
		b, err = json.Marshal(requestBody)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status code %d: %s", res.StatusCode, body)
	}

	return body, nil
}

// GetAuthorFeed retrieves the author feed from the Bluesky API using the client
func (c *Client) GetAuthorFeed(actor string, limit int, cursor, filter string, includePins bool) (map[string]interface{}, error) {
	baseURL := c.BaseURL + "/xrpc/app.bsky.feed.getAuthorFeed"
	params := url.Values{}
	params.Set("actor", actor)
	params.Set("limit", fmt.Sprintf("%d", limit))
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	if filter != "" {
		params.Set("filter", filter)
	}
	params.Set("includePins", fmt.Sprintf("%t", includePins))
	requestURL := baseURL + "?" + params.Encode()

	body, err := c.SendRequest("GET", requestURL, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return result, nil
}

// GetProfiles retrieves profiles from the Bluesky API using the client
func (c *Client) GetProfiles(actors []string) (map[string]interface{}, error) {
	baseURL := c.BaseURL + "/xrpc/app.bsky.actor.getProfiles"
	params := url.Values{}
	for _, actor := range actors {
		params.Add("actors", actor)
	}

	body, err := c.SendRequest("GET", baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return result, nil
}

// GetAccounts retrieves the followers of a specified actor from the Bluesky API using the session
func (c *Client) GetAccounts(endpoint, actor string, limit int, cursor string) (map[string]interface{}, error) {
	baseURL := c.BaseURL + endpoint
	params := url.Values{}
	params.Add("actor", actor)
	if limit > 0 {
		params.Add("limit", fmt.Sprintf("%d", limit))
	}
	if cursor != "" {
		params.Add("cursor", cursor)
	}
	requestURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	body, err := c.SendRequest("GET", requestURL, nil)
	if err != nil {
		return nil, err
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return response, nil
}

// CreateRecord creates a record in the Bluesky API
func (c *Client) CreateRecord(request CreateRecordRequest) (map[string]interface{}, error) {
	url := c.BaseURL + "/xrpc/com.atproto.repo.createRecord"

	res, err := c.SendRequest("POST", url, request)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(res, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return result, nil
}
