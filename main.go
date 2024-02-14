package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/dgraph-io/dgo/v200"
	"github.com/dgraph-io/dgo/v200/protos/api"
	"google.golang.org/grpc"
	"log"
)

func main() {
	// Dial a gRPC connection. The address to dial must be passed as parameter.
	conn, err := grpc.Dial("localhost:9080", grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Create a new Dgraph client.
	dgraphClient := dgo.NewDgraphClient(api.NewDgraphClient(conn))

	// Perform a query.
	query := `
		{
			all(func: has(name)) {
				uid
				name
			}
		}
	`
	result, err := queryDgraph(dgraphClient, query)
	if err != nil {
		log.Fatal(err)
	}
	printResult(result)

	// Print the results individually.
	for _, node := range result["all"].([]interface{}) {
		nodeMap := node.(map[string]interface{})
		fmt.Printf("UID: %s, Name: %s\n", nodeMap["uid"], nodeMap["name"])
	}
}

func queryDgraph(client *dgo.Dgraph, query string) (map[string]interface{}, error) {
	ctx := context.Background()
	resp, err := client.NewReadOnlyTxn().Query(ctx, query)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func printResult(result map[string]interface{}) {
	fmt.Println("Entire JSON block:")
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(jsonData))
}
