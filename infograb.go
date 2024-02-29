package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
	"google.golang.org/grpc"
	"log"
)

func countNodes(dgraphClient *dgo.Dgraph, query string) int {
	// Create a new transaction
	txn := dgraphClient.NewReadOnlyTxn()

	// Execute the query
	response, err := txn.Query(context.Background(), query)
	if err != nil {
		log.Fatal(err)
	}

	// Unmarshal the response
	var result map[string]interface{}
	if err := json.Unmarshal(response.Json, &result); err != nil {
		log.Fatal(err)
	}

	// Get the number of nodes returned
	nodes := result["queryResults"].([]interface{})
	return len(nodes)
}

func countNodesPaginated(dgraphClient *dgo.Dgraph, query string) int {
	ctx := context.Background()
	first := 1000
	totalCount := 0

	for {
		// Create a new transaction
		txn := dgraphClient.NewTxn()

		// craft query
		query := fmt.Sprintf(`
			{
				%s, first: %d, offset: %d) {
					uid
				}
			}
		`, query, first, totalCount)

		// Execute the query with pagination
		response, err := txn.Query(ctx, query)
		if err != nil {
			log.Fatal(err)
		}

		// Unmarshal the response
		var result map[string]interface{}
		if err := json.Unmarshal(response.Json, &result); err != nil {
			log.Fatal(err)
		}

		// Get the number of nodes returned
		nodes := result["queryResults"].([]interface{})
		totalCount += len(nodes)

		// Check if there are more results to fetch
		if len(nodes) < first {
			break
		}

		// Set the starting point for the next query
		query = fmt.Sprintf("%s offset %d", query, totalCount)

		// Close the transaction
		txn.Discard(ctx)
	}

	return totalCount
}

func newDgraphClient() (*dgo.Dgraph, error) {
	conn, err := grpc.Dial("localhost:9080", grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return dgo.NewDgraphClient(
		api.NewDgraphClient(conn),
	), nil
}

func main() {
	// Set up the Dgraph client
	dgraphClient, err := newDgraphClient()
	if err != nil {
		log.Fatal(err)
	}

	nodes := countNodesPaginated(dgraphClient, `queryResults(func: type(Combo)`)
	fmt.Printf("Number of combinations: %d\n", nodes)

	nodes = countNodesPaginated(dgraphClient, `queryResults(func: type(Result)`)
	fmt.Printf("Number of unique types: %d\n", nodes)

	query := `
		{
			queryResults(func: type(Result)) @filter(eq(isNew, true)) {
				uid
			}
		}
	`
	nodes = countNodes(dgraphClient, query)
	fmt.Printf("Number of discoveries: %d\n", nodes)

}
