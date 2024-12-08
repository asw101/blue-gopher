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
	var body io.Reader
	if requestBody != nil {
		jsonBody, err := json.Marshal(requestBody)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		body = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status code %d: %s", resp.StatusCode, responseBody)
	}

	return responseBody, nil
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

	// Make the request
	responseBody, err := c.SendRequest("GET", requestURL, nil)
	if err != nil {
		return nil, err
	}

	// Unmarshal the response
	var result map[string]interface{}
	if err := json.Unmarshal(responseBody, &result); err != nil {
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
	authorFeedResponse, err := c.ClientAuthorFeed(author, limit, cursor, filter, includePins)
	if err != nil {
		return err
	}

	// Output the results
	responseBody, err := json.MarshalIndent(authorFeedResponse, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal response body: %w", err)
	}
	fmt.Println(string(responseBody))

	return nil
}
