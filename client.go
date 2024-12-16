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
	"strings"
	"time"
)

// Client is a client for the Bluesky API
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

	// todo: add logic to use existing (cached) session
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

	client := &http.Client{Timeout: 10 * time.Second}
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

// GetProfile retrieves the profile for a given username and returns the profile data as a map
func (c *Client) GetProfile(actor string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/xrpc/app.bsky.actor.getProfile?actor=%s", c.BaseURL, url.QueryEscape(actor))

	res, err := c.SendRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var profile map[string]interface{}
	if err := json.Unmarshal(res, &profile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return profile, nil
}

// GetProfiles retrieves profiles from the Bluesky API using the client
func (c *Client) GetProfiles(actors []string) (map[string]interface{}, error) {
	if len(actors) > 25 {
		return nil, fmt.Errorf("too many actors: maximum allowed is 25")
	}

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

// SearchPosts searches posts in the Bluesky API
func (c *Client) SearchPosts(q string, limit int, cursor, sort, since, until, mentions, author, lang, domain, postURL string, tags []string) (map[string]interface{}, error) {
	baseURL := c.BaseURL + "/xrpc/app.bsky.feed.searchPosts"
	params := url.Values{}
	params.Add("q", q)
	if limit > 0 {
		params.Add("limit", fmt.Sprintf("%d", limit))
	}
	if cursor != "" {
		params.Add("cursor", cursor)
	}
	if sort != "" {
		params.Add("sort", sort)
	}
	if since != "" {
		params.Add("since", since)
	}
	if until != "" {
		params.Add("until", until)
	}
	if mentions != "" {
		params.Add("mentions", mentions)
	}
	if author != "" {
		params.Add("author", author)
	}
	if lang != "" {
		params.Add("lang", lang)
	}
	if domain != "" {
		params.Add("domain", domain)
	}
	if postURL != "" {
		params.Add("url", postURL)
	}
	for _, tag := range tags {
		params.Add("tag", tag)
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

// ListCreate creates a list in the Bluesky API
func (c *Client) ListCreate(purpose, name, description string, createdAt time.Time) (map[string]interface{}, error) {
	url := c.BaseURL + "/xrpc/com.atproto.repo.createRecord"

	request := CreateRecordRequest{
		Repo:       c.Session.DID,
		Collection: "app.bsky.graph.list",
		Record: struct {
			Name        string `json:"name"`
			Purpose     string `json:"purpose"`
			Description string `json:"description,omitempty"`
			CreatedAt   string `json:"createdAt"`
			Type        string `json:"$type"`
		}{
			Name:        name,
			Purpose:     purpose,
			Description: description,
			CreatedAt:   createdAt.Format(time.RFC3339),
			Type:        "app.bsky.graph.list",
		},
	}

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

// ListItem adds a member to a list in the Bluesky API
func (c *Client) ListItem(listURI, did string, createdAt time.Time) (map[string]interface{}, error) {
	url := c.BaseURL + "/xrpc/com.atproto.repo.createRecord"

	request := CreateRecordRequest{
		Repo:       c.Session.DID,
		Collection: "app.bsky.graph.listitem",
		Record: struct {
			Subject   string `json:"subject"`
			List      string `json:"list"`
			CreatedAt string `json:"createdAt"`
			Type      string `json:"$type"`
		}{
			Subject:   did,
			List:      listURI,
			CreatedAt: createdAt.Format(time.RFC3339),
			Type:      "app.bsky.graph.listitem",
		},
	}

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

// ListATURI parses the given URL and constructs the AT URI
func (c *Client) ListATURI(listURL string) (string, error) {
	// Remove any query parameters
	listURL = strings.Split(listURL, "?")[0]

	// Parse URL parts
	parsedURL, err := url.Parse(listURL)
	if err != nil {
		return "", fmt.Errorf("invalid list URL: %w", err)
	}

	pathComponents := strings.Split(parsedURL.Path, "/")
	if len(pathComponents) < 5 || !strings.Contains(listURL, "bsky.app/profile/") || !strings.Contains(listURL, "/lists/") {
		return "", fmt.Errorf("invalid list URL format")
	}

	handle := pathComponents[2]
	listId := pathComponents[4]

	// Get user's DID first
	profile, err := c.GetProfile(handle)
	if err != nil {
		return "", fmt.Errorf("failed to get profile: %w", err)
	}

	did, ok := profile["did"].(string)
	if !ok {
		return "", fmt.Errorf("failed to get DID from profile")
	}

	// Construct AT-URI
	listUri := fmt.Sprintf("at://%s/app.bsky.graph.list/%s", did, listId)
	return listUri, nil
}
