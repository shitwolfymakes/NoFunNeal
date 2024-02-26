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

	query := `
		{
			queryResults(func: type(Combo)) {
				uid
			}
		}
	`
	nodes := countNodes(dgraphClient, query)
	fmt.Printf("Number of combinations: %d\n", nodes)

	query = `
		{
			queryResults(func: type(Result)) {
				uid
			}
		}
	`
	nodes = countNodes(dgraphClient, query)
	fmt.Printf("Number of unique types: %d\n", nodes)

	query = `
		{
			queryResults(func: type(Result)) @filter(eq(isNew, true)) {
				uid
				result
				encodedName
				emoji
				isNew
			}
		}
	`
	nodes = countNodes(dgraphClient, query)
	fmt.Printf("Number of discoveries: %d\n", nodes)

}
