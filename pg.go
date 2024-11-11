//go:build mage
// +build mage

package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/lib/pq"
	"github.com/magefile/mage/mg"
)

type Pg mg.Namespace

// getConnection returns a PostgreSQL database connection
func getConnection() (*sql.DB, error) {
	connStr := "user=user dbname=user sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the database: %w", err)
	}
	return db, nil
}

// ListTables lists all tables in the PostgreSQL database
func (Pg) ListTables() error {
	db, err := getConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	rows, err := db.Query("SELECT table_name FROM information_schema.tables WHERE table_schema = 'public'")
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	fmt.Println("Tables in the database:")
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}
		fmt.Println(tableName)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error occurred during row iteration: %w", err)
	}

	return nil
}

// CreateBlueskyTable creates a table for storing JSON objects
func (Pg) CreateBlueskyTable() error {
	db, err := getConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	query := `
	CREATE TABLE IF NOT EXISTS bluesky (
		id SERIAL PRIMARY KEY,
		name TEXT,
		data JSONB NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	)`
	_, err = db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	fmt.Println("Table 'bluesky' created successfully")
	return nil
}

// DropBlueskyTable drops the bluesky table
func (Pg) DropBlueskyTable() error {
	db, err := getConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	query := "DROP TABLE IF EXISTS bluesky"
	_, err = db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to drop table: %w", err)
	}

	fmt.Println("Table 'bluesky' dropped successfully")
	return nil
}

// ImportJsonFile imports JSON lines from a file into the bluesky table
func (Pg) ImportJsonFile(filePath, name string) error {
	db, err := getConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		jsonLine := scanner.Text()
		_, err := db.Exec("INSERT INTO bluesky (name, data) VALUES ($1, $2)", name, jsonLine)
		if err != nil {
			return fmt.Errorf("failed to insert JSON line: %w", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	fmt.Println("JSON lines imported successfully")
	return nil
}

// QueryHandles queries the bluesky table and selects the "handle" from the JSON column, filtered by name
func (Pg) QueryHandles(name string) error {
	db, err := getConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	rows, err := db.Query("SELECT data->>'handle' AS handle FROM bluesky WHERE name = $1", name)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	fmt.Println("Handles in the bluesky table with name:", name)
	for rows.Next() {
		var handle string
		if err := rows.Scan(&handle); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}
		fmt.Println(handle)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error occurred during row iteration: %w", err)
	}

	return nil
}

// Query runs an arbitrary query against the bluesky table and outputs the results as JSON lines
func (Pg) Query(query string) error {
	db, err := getConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	rows, err := db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		result := make(map[string]interface{})
		for i, col := range columns {
			result[col] = values[i]
		}

		jsonLine, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}

		fmt.Println(string(jsonLine))
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error occurred during row iteration: %w", err)
	}

	return nil
}

// Query2 runs an arbitrary query against the bluesky table and outputs the results as JSON lines
func (Pg) Query2(query string) error {
	db, err := getConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	rows, err := db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		result := make(map[string]interface{})
		for i, col := range columns {
			result[col] = values[i]
		}

		if data, ok := result["data"].([]byte); ok {
			fmt.Println(string(data))
		} else {
			jsonLine, err := json.Marshal(result)
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonLine))
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error occurred during row iteration: %w", err)
	}

	return nil
}
