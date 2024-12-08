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

	"github.com/magefile/mage/mg"
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

	_, err := client.createSession()
	if err != nil {
		return nil, err
	}

	return client, nil
}

// createSession authenticates to the Bluesky API using the provided credentials and sets the AuthToken on the client
func (c *Client) createSession() (*CreateSessionResponse, error) {
	user := os.Getenv("BLUESKY_HANDLE")
	pass := os.Getenv("BLUESKY_PASSWORD")

	url := c.BaseURL + "/xrpc/com.atproto.server.createSession"
	req := map[string]string{
		"identifier": user,
		"password":   pass,
	}
	resp, err := c.SendRequest("POST", url, req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Decode the response body into the CreateSessionResponse struct
	var createSessionResponse CreateSessionResponse
	if err := json.Unmarshal(resp, &createSessionResponse); err != nil {
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

// GetProfiles retrieves profiles from the Bluesky API using the client
func (c *Client) GetProfiles(actors []string) (map[string]interface{}, error) {
	// Create the request URL
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

// getAccounts retrieves the followers of a specified actor from the Bluesky API using the session
func (c *Client) GetAccounts(endpoint, actor string, limit int, cursor string) (map[string]interface{}, error) {
	// Create the request URL with query parameters
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

	// Decode the response body into a generic map
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return response, nil
}

// createRecord creates a record in the Bluesky API
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

type Bs mg.Namespace

// GetAuthorFeed retrieves a single page of an author feed and outputs the results
func (Bs) GetAuthorFeed(author string) error {
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
	resp, err := c.GetAuthorFeed(author, limit, cursor, filter, includePins)
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

// GetAuthorFeeds retrieves the author feed from the Bluesky API and outputs each item to the standard output
func (Bs) GetAuthorFeeds(author string) error {
	c, err := NewClient()
	if err != nil {
		return err
	}

	limit := 100
	cursor := ""
	includePins := true
	// posts_with_replies, posts_no_replies, posts_with_media, posts_and_author_threads
	filter := "posts_with_replies"

	for {
		// Call getAuthorFeed to retrieve the author feed
		authorFeedResponse, err := c.GetAuthorFeed(author, limit, cursor, filter, includePins)
		if err != nil {
			return err
		}

		// Print each feed item to the standard output
		if feed, ok := authorFeedResponse["feed"].([]interface{}); ok {
			for _, item := range feed {
				formattedItem, err := json.Marshal(item)
				if err != nil {
					return fmt.Errorf("failed to marshal feed item: %w", err)
				}
				fmt.Printf("%s\n", formattedItem)
			}
		}

		// Check if there is a next cursor
		if nextCursor, ok := authorFeedResponse["cursor"].(string); ok && nextCursor != "" {
			cursor = nextCursor
		} else {
			break
		}
	}

	return nil
}

// GetProfiles retrieves the profiles of multiple actors from the Bluesky API and outputs the results
func (Bs) GetProfiles(profiles string) error {
	c, err := NewClient()
	if err != nil {
		return err
	}

	actors := strings.Split(profiles, ",")
	profilesResponse, err := c.GetProfiles(actors)
	if err != nil {
		return err
	}

	val, ok := profilesResponse["profiles"]
	if !ok {
		return fmt.Errorf("profiles not found")
	}

	list, ok := val.([]interface{})
	if !ok {
		fmt.Errorf("cannot type assert profiles to []interface{}")
	}
	for _, x := range list {
		formattedResponse, err := json.Marshal(x)
		if err != nil {
			return fmt.Errorf("failed to marshal response struct: %w", err)
		}
		fmt.Printf("%s\n", formattedResponse)
	}

	return nil
}

// GetFollowers retrieves the followers of a specified actor from the Bluesky API and outputs the results
func (Bs) GetFollowers(actor string) error {
	c, err := NewClient()
	if err != nil {
		return err
	}
	limit := 100
	cursor := ""
	for {
		accountsResponse, err := c.GetAccounts("/xrpc/app.bsky.graph.getFollowers", actor, limit, cursor)
		if err != nil {
			return err
		}

		if val, ok := accountsResponse["followers"]; ok {
			accounts, ok := val.([]interface{})
			if !ok {
				return fmt.Errorf("Cannot type assert followers to []interface{}")
			}
			for _, x := range accounts {
				formattedResponse, err := json.Marshal(x)
				if err != nil {
					return fmt.Errorf("failed to marshal response struct: %w", err)
				}
				fmt.Printf("%s\n", formattedResponse)
			}
		}

		val, ok := accountsResponse["cursor"]
		if !ok {
			break
		}
		cursor = val.(string)
		if cursor == "" {
			break
		}
	}
	return nil
}

// GetFollows retrieves the followers of a specified actor from the Bluesky API and outputs the results
func (Bs) GetFollows(actor string) error {
	c, err := NewClient()
	if err != nil {
		return err
	}
	limit := 100
	cursor := ""
	for {
		accountsResponse, err := c.GetAccounts("/xrpc/app.bsky.graph.getFollows", actor, limit, cursor)
		if err != nil {
			return err
		}

		if val, ok := accountsResponse["follows"]; ok {
			accounts := val.([]interface{})
			if !ok {
				return fmt.Errorf("Cannot type assert follows to []interface{}")
			}
			for _, x := range accounts {
				formattedResponse, err := json.Marshal(x)
				if err != nil {
					return fmt.Errorf("failed to marshal response struct: %w", err)
				}
				fmt.Printf("%s\n", formattedResponse)
			}
		}

		val, ok := accountsResponse["cursor"]
		if !ok {
			break
		}
		cursor = val.(string)
		if cursor == "" {
			break
		}
	}
	return nil
}

// CreateSession authenticates to the Bluesky API using the provided credentials and returns a CreateSessionResponse
func (Bs) CreateSession() error {
	c, err := NewClient()
	if err != nil {
		return err
	}

	createSessionResponse, err := c.createSession()
	if err != nil {
		return err
	}
	// Output the struct with JSON formatting and indentation
	formattedResponse, err := json.MarshalIndent(createSessionResponse, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", formattedResponse)
	return nil
}

// CreateRecord creates a new post in the Bluesky API
func (Bs) CreateRecord(text string) error {
	c, err := NewClient()
	if err != nil {
		return err
	}

	// Create the record request
	request := CreateRecordRequest{
		Repo:       c.Session.Handle,
		Collection: "app.bsky.feed.post",
		Record: map[string]interface{}{
			"text":      text,
			"createdAt": time.Now().UTC().Format(time.RFC3339),
		},
	}

	resp, err := c.CreateRecord(request)

	b, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	fmt.Printf("%s\n", b)
	return nil
}
