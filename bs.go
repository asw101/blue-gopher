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

// CreateRecordResponse represents the structure of the response from the createRecord API
type CreateRecordResponse struct {
	URI              string `json:"uri"`
	CID              string `json:"cid"`
	Commit           string `json:"commit"`
	ValidationStatus string `json:"validationStatus"`
}

// Authenticate authenticates to the Bluesky API using the provided credentials and returns a CreateSessionResponse
func Authenticate() (*CreateSessionResponse, error) {
	user := os.Getenv("BLUESKY_HANDLE")
	pass := os.Getenv("BLUESKY_PASSWORD")

	fmt.Printf("BLUESKY_HANDLE: %s\n", user)
	fmt.Printf("BLUESKY_PASSWORD: %s\n", pass)

	// Create the request body
	requestBody, err := json.Marshal(map[string]string{
		"identifier": user,
		"password":   pass,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Output the JSON request body to the standard output
	fmt.Printf("Request Body:\n%s\n", string(requestBody))

	// Send the POST request
	url := os.Getenv("PDSHOST") + "/xrpc/com.atproto.server.createSession"
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
	fmt.Printf("Response Body:\n%s\n", string(body))

	// Decode the response body into the CreateSessionResponse struct
	var createSessionResponse CreateSessionResponse
	if err := json.Unmarshal(body, &createSessionResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	// Output the struct with JSON formatting and indentation
	formattedResponse, err := json.MarshalIndent(createSessionResponse, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response struct: %w", err)
	}
	fmt.Printf("Formatted Response:\n%s\n", string(formattedResponse))

	// Check the response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to authenticate: %s", resp.Status)
	}

	fmt.Println("Successfully connected to Bluesky")

	return &createSessionResponse, nil
}

// CreateSession authenticates to the Bluesky API and outputs the pretty-printed JSON response
func (Tmp) CreateSession() error {
	session, err := Authenticate()
	if err != nil {
		return err
	}

	// Output the struct with JSON formatting and indentation
	formattedResponse, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal response struct: %w", err)
	}
	fmt.Printf("Formatted Response:\n%s\n", string(formattedResponse))

	return nil
}

// createRecord creates a record in the Bluesky API
func createRecord(request CreateRecordRequest, session *CreateSessionResponse) (*CreateRecordResponse, error) {
	// Create the request body
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Output the JSON request body to the standard output
	fmt.Printf("Request Body:\n%s\n", string(requestBody))

	// Create the HTTP request
	url := os.Getenv("PDSHOST") + "/xrpc/com.atproto.repo.createRecord"
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
	fmt.Printf("Response Body:\n%s\n", string(body))

	// Decode the response body into the CreateRecordResponse struct
	var createRecordResponse CreateRecordResponse
	if err := json.Unmarshal(body, &createRecordResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	// Output the struct with JSON formatting and indentation
	formattedResponse, err := json.MarshalIndent(createRecordResponse, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response struct: %w", err)
	}
	fmt.Printf("Formatted Response:\n%s\n", string(formattedResponse))

	// Check the response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to create record: %s", resp.Status)
	}

	return &createRecordResponse, nil
}

// CreateRecord creates a new post in the Bluesky API
func (Tmp) CreateRecord() error {
	// Authenticate to get the session
	session, err := Authenticate()
	if err != nil {
		return err
	}

	// Create the record request
	request := CreateRecordRequest{
		Repo:       session.Handle,
		Collection: "app.bsky.feed.post",
		Record: map[string]interface{}{
			"text":      "Hello world! I posted this via the API (via @golang.org).",
			"createdAt": time.Now().UTC().Format(time.RFC3339),
		},
	}

	// Call createRecord to create the new post
	_, err = createRecord(request, session)
	return err
}

// GetFollowers retrieves the followers of a specified actor from the Bluesky API
func (Tmp) GetFollowers(actor string, limit int, cursor string) error {
	// Create the request URL with query parameters
	baseURL := os.Getenv("PDSHOST") + "/xrpc/app.bsky.graph.getFollowers"
	params := url.Values{}
	params.Add("actor", actor)
	if limit > 0 {
		params.Add("limit", fmt.Sprintf("%d", limit))
	}
	if cursor != "" {
		params.Add("cursor", cursor)
	}
	requestURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Send the GET request
	resp, err := http.Get(requestURL)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read and output the response body to the standard output
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Decode the response body into the GetFollowersResponse struct
	var getFollowersResponse GetFollowersResponse
	if err := json.Unmarshal(body, &getFollowersResponse); err != nil {
		return fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	// Output the struct with JSON formatting and indentation
	formattedResponse, err := json.MarshalIndent(getFollowersResponse, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal response struct: %w", err)
	}
	fmt.Printf("Formatted Response:\n%s\n", string(formattedResponse))

	return nil
}
