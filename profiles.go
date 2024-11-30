//go:build mage
// +build mage

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// getProfiles retrieves the profiles of multiple actors from the Bluesky API using the session
func getProfiles(session *CreateSessionResponse, actors []string) (map[string]interface{}, error) {
	// Create the request URL
	baseURL := getURL("/xrpc/app.bsky.actor.getProfiles")
	params := url.Values{}
	for _, actor := range actors {
		params.Add("actors", actor)
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

// GetProfiles retrieves the profiles of multiple actors from the Bluesky API and outputs the results
func (Tmp) GetProfiles(profiles string) error {
	// Authenticate to get the session
	session, err := authenticate()
	if err != nil {
		return err
	}

	// Split the comma-separated list of profiles into a slice of strings
	actors := strings.Split(profiles, ",")

	// Call getProfiles to retrieve the profiles
	profilesResponse, err := getProfiles(session, actors)
	if err != nil {
		return err
	}

	// Output the struct with JSON formatting and indentation
	formattedResponse, err := json.MarshalIndent(profilesResponse, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal response struct: %w", err)
	}
	fmt.Printf("Formatted Response:\n%s\n", string(formattedResponse))

	return nil
}

// getAuthorFeed retrieves the author feed from the Bluesky API using the session
func getAuthorFeed(session *CreateSessionResponse, author string, limit int, cursor, filter string, includePins bool) (map[string]interface{}, error) {
	// Create the request URL
	baseURL := getURL("/xrpc/app.bsky.feed.getAuthorFeed")
	params := url.Values{}
	params.Add("actor", author)
	if limit > 0 {
		params.Add("limit", fmt.Sprintf("%d", limit))
	}
	if cursor != "" {
		params.Add("cursor", cursor)
	}
	if filter != "" {
		params.Add("filter", filter)
	}
	params.Add("includePins", fmt.Sprintf("%t", includePins))
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

// GetAuthorFeed retrieves the author feed from the Bluesky API and outputs the results
func (Tmp) GetAuthorFeed(author string) error {
	// Authenticate to get the session
	session, err := authenticate()
	if err != nil {
		return err
	}

	limit := 100
	cursor := ""
	includePins := true
	// posts_with_replies, posts_no_replies, posts_with_media, posts_and_author_threads
	filter := "posts_with_replies"
	// Call getAuthorFeed to retrieve the author feed
	authorFeedResponse, err := getAuthorFeed(session, author, limit, cursor, filter, includePins)
	if err != nil {
		return err
	}

	// Output the struct with JSON formatting and indentation
	formattedResponse, err := json.MarshalIndent(authorFeedResponse, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal response struct: %w", err)
	}
	fmt.Printf("Formatted Response:\n%s\n", string(formattedResponse))

	return nil
}

// GetAuthorFeeds retrieves the author feed from the Bluesky API and outputs each item to the standard output
func (Tmp) GetAuthorFeeds(author string) error {
	// Authenticate to get the session
	session, err := authenticate()
	if err != nil {
		return err
	}

	cursor := ""
	limit := 100
	includePins := true
	// posts_with_replies, posts_no_replies, posts_with_media, posts_and_author_threads
	filter := "posts_with_replies"

	for {
		// Call getAuthorFeed to retrieve the author feed
		authorFeedResponse, err := getAuthorFeed(session, author, limit, cursor, filter, includePins)
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
