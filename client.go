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

	"github.com/magefile/mage/mg"
)

// Client represents a client for the Bluesky API
type Client struct {
	BaseURL   string
	AuthToken string
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

// authenticate authenticates to the Bluesky API using the provided credentials and sets the AuthToken on the client
func (c *Client) authenticate() error {
	user := os.Getenv("BLUESKY_HANDLE")
	pass := os.Getenv("BLUESKY_PASSWORD")

	url := c.BaseURL + "/xrpc/com.atproto.server.createSession"
	req := map[string]string{
		"identifier": user,
		"password":   pass,
	}
	resp, err := c.SendRequest("POST", url, req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	// Decode the response body into the CreateSessionResponse struct
	var createSessionResponse CreateSessionResponse
	if err := json.Unmarshal(resp, &createSessionResponse); err != nil {
		return fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	if createSessionResponse.AccessJwt == "" {
		return fmt.Errorf("failed to authenticate: missing access token")
	}
	c.AuthToken = createSessionResponse.AccessJwt
	return nil
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

	err := client.authenticate()
	if err != nil {
		return nil, err
	}

	return client, nil
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

// ClientAuthorFeed retrieves the author feed from the Bluesky API using the client
func (c *Client) ClientAuthorFeed(actor string, limit int, cursor, filter string, includePins bool) (map[string]interface{}, error) {
	// Create the request URL
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

type Two mg.Namespace

// GetAuthorFeed2 retrieves the author feed and outputs the results
func (Two) GetAuthorFeed2(author string) error {
	c, err := NewClient()
	if err != nil {
		return err
	}

	limit := 100
	cursor := ""
	includePins := true
	// posts_with_replies, posts_no_replies, posts_with_media, posts_and_author_threads
	filter := "posts_with_replies"

	// Call ClientAuthorFeed to retrieve the author feed
	resp, err := c.ClientAuthorFeed(author, limit, cursor, filter, includePins)
	if err != nil {
		return err
	}

	b, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", b)

	return nil
}
