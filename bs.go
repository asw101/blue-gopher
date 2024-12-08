//go:build mage
// +build mage

package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/magefile/mage/mg"
)

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
		authorFeedResponse, err := c.GetAuthorFeed(author, limit, cursor, filter, includePins)
		if err != nil {
			return err
		}

		if feed, ok := authorFeedResponse["feed"].([]interface{}); ok {
			for _, item := range feed {
				formattedItem, err := json.Marshal(item)
				if err != nil {
					return fmt.Errorf("failed to marshal feed item: %w", err)
				}
				fmt.Printf("%s\n", formattedItem)
			}
		}

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
