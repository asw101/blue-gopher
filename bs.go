//go:build mage
// +build mage

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/magefile/mage/mg"
)

type Tmp mg.Namespace

// Hello says hello
func (Tmp) Hello() error {
	return errors.New("not implemented")
}

// GetFollowersResponse represents the structure of the response from the getFollowers API
type GetFollowersResponse struct {
	Subject   ProfileView   `json:"subject"`
	Cursor    string        `json:"cursor"`
	Followers []ProfileView `json:"followers"`
}

// ProfileView represents the profile view structure
type ProfileView struct {
	Did      string `json:"did"`
	Handle   string `json:"handle"`
	Avatar   string `json:"avatar"`
	FullName string `json:"fullName"`
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

func getURL(path string) string {
	pdshost := os.Getenv("PDSHOST")
	// default to https://bsky.social
	if pdshost == "" {
		pdshost = "https://bsky.social"
	}
	return pdshost + path
}

// authenticate authenticates to the Bluesky API using the provided credentials and returns a CreateSessionResponse
func authenticate() (*CreateSessionResponse, error) {
	user := os.Getenv("BLUESKY_HANDLE")
	pass := os.Getenv("BLUESKY_PASSWORD")

	// Create the request body
	requestBody, err := json.Marshal(map[string]string{
		"identifier": user,
		"password":   pass,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Send the POST request
	url := getURL("/xrpc/com.atproto.server.createSession")
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read and output the response body to the standard output
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Decode the response body into the CreateSessionResponse struct
	var createSessionResponse CreateSessionResponse
	if err := json.Unmarshal(body, &createSessionResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	// Check the response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to authenticate: %s", resp.Status)
	}

	return &createSessionResponse, nil
}

// authenticate authenticates to the Bluesky API using the provided credentials and returns a CreateSessionResponse
func (Tmp) Authenticate() error {
	createSessionResponse, err := authenticate()
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

// CreateSession authenticates to the Bluesky API and outputs the pretty-printed JSON response
func (Tmp) CreateSession() error {
	session, err := authenticate()
	if err != nil {
		return err
	}

	// Output the struct with JSON formatting and indentation
	formattedResponse, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal response struct: %w", err)
	}
	fmt.Printf("%s\n", formattedResponse)

	return nil
}

// createRecord creates a record in the Bluesky API
func createRecord(request CreateRecordRequest, session *CreateSessionResponse) ([]byte, error) {
	// Create the request body
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create the HTTP request
	url := getURL("/xrpc/com.atproto.repo.createRecord")
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+session.AccessJwt)
	req.Header.Set("Content-Type", "application/json")

	// Send the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read and output the response body to the standard output
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check the response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to create record: %s", resp.Status)
	}

	return body, nil
}

// CreateRecord creates a new post in the Bluesky API
func (Tmp) CreateRecord(text string) error {
	// Authenticate to get the session
	session, err := authenticate()
	if err != nil {
		return err
	}

	// Create the record request
	request := CreateRecordRequest{
		Repo:       session.Handle,
		Collection: "app.bsky.feed.post",
		Record: map[string]interface{}{
			"text":      text,
			"createdAt": time.Now().UTC().Format(time.RFC3339),
		},
	}

	// Call createRecord to create the new post
	resp, err := createRecord(request, session)

	fmt.Printf("%s\n", resp)

	return err
}

// getAccounts retrieves the followers of a specified actor from the Bluesky API using the session
func getAccounts(session *CreateSessionResponse, endpoint, actor string, limit int, cursor string) (map[string]interface{}, error) {
	// Create the request URL with query parameters
	baseURL := getURL(endpoint)
	params := url.Values{}
	params.Add("actor", actor)
	if limit > 0 {
		params.Add("limit", fmt.Sprintf("%d", limit))
	}
	if cursor != "" {
		params.Add("cursor", cursor)
	}
	requestURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Create the HTTP request
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+session.AccessJwt)

	// Send the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read and output the response body to the standard output
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Decode the response body into a generic map
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return response, nil
}

// GetFollowers retrieves the followers of a specified actor from the Bluesky API and outputs the results
func (Tmp) GetFollowers(actor string) error {
	session, err := authenticate()
	if err != nil {
		return err
	}
	limit := 100
	cursor := ""
	for {
		followersResponse, err := getAccounts(session, "/xrpc/app.bsky.graph.getFollowers", actor, limit, cursor)
		if err != nil {
			return err
		}

		if val, ok := followersResponse["followers"]; ok {
			followers := val.([]interface{})
			for _, x := range followers {
				formattedResponse, err := json.Marshal(x)
				if err != nil {
					return fmt.Errorf("failed to marshal response struct: %w", err)
				}
				fmt.Printf("%s\n", formattedResponse)
			}
		}

		val, ok := followersResponse["cursor"]
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
func (Tmp) GetFollows(actor string) error {
	session, err := authenticate()
	if err != nil {
		return err
	}
	limit := 100
	cursor := ""
	for {
		followersResponse, err := getAccounts(session, "/xrpc/app.bsky.graph.getFollows", actor, limit, cursor)
		if err != nil {
			return err
		}

		if val, ok := followersResponse["follows"]; ok {
			followers := val.([]interface{})
			for _, x := range followers {
				formattedResponse, err := json.Marshal(x)
				if err != nil {
					return fmt.Errorf("failed to marshal response struct: %w", err)
				}
				fmt.Printf("%s\n", formattedResponse)
			}
		}

		val, ok := followersResponse["cursor"]
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
