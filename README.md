# Blue Gopher

Bluesky, Go, Postgres

## Usage

```
$ go run main.go
Targets:
  bs:createRecord          <text> creates a new post
  bs:createSession         authenticates to the Bluesky API using the BLUESKY_HANDLE and BLUESKY_PASSWORD env vars and outputs the response
  bs:getAuthorFeed         <author> retrieves a single page of an author feed and outputs the response
  bs:getAuthorFeeds        <authors> retrieves the author feed and outputs the results
  bs:getFollowers          <actor> retrieves the followers of a specified actor and outputs the results
  bs:getFollows            <actor> retrieves the followers of a specified actor and outputs the results
  bs:getProfiles           <profiles> retrieves the profiles of multiple actors and outputs the results
  hello:hello              says hello
  pg:createBlueskyTable    creates a table for storing JSON objects
  pg:dropBlueskyTable      drops the bluesky table
  pg:importJsonFile        imports JSON lines from a file into the bluesky table
  pg:listTables            lists all tables in the PostgreSQL database
  pg:query                 runs an arbitrary query against the bluesky table and outputs the results as JSON lines
  pg:query2                runs an arbitrary query against the bluesky table and outputs the results as JSON lines
  pg:queryHandles          queries the bluesky table and selects the "handle" from the JSON column, filtered by name
  ```