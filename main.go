package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/dgraph-io/dgo/v200"
	"github.com/dgraph-io/dgo/v200/protos/api"
	"google.golang.org/grpc"
	"log"
	"net/url"
)

var dgraphClient *dgo.Dgraph

func main() {

	// Dial a gRPC connection. The address to dial must be passed as parameter.
	conn, err := grpc.Dial("localhost:9080", grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Create a new Dgraph client.
	dgraphClient = dgo.NewDgraphClient(api.NewDgraphClient(conn))

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

	// Print the results.
	printResult(result)
	for _, node := range result["all"].([]interface{}) {
		nodeMap := node.(map[string]interface{})
		fmt.Printf("UID: %s, Name: %s\n", nodeMap["uid"], nodeMap["name"])
	}

	// check for combos
	comboFound := comboExists("Fire", "Water")
	fmt.Printf("comboExists: %t\n", comboFound)
	comboFound = comboExists("Fire", "Earth")
	fmt.Printf("comboExists: %t\n", comboFound)

	// test url encoding
	input := "Three's Company"
	fmt.Printf("name: %s\n", input)
	fmt.Printf("encodedName: %s\n", encodeInput(input))

	// test url construction
	encodedUrl := craftUrl(input, "Testing")
	fmt.Println(encodedUrl)
}

func encodeInput(str string) string {
	return url.QueryEscape(str)
}

func craftUrl(a string, b string) string {
	out := fmt.Sprintf(`https://neal.fun/api/infinite-craft/pair?first=%s&second=%s`,
		encodeInput(a),
		encodeInput(b),
	)
	return out
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

func comboExists(a string, b string) bool {
	// get nodes with a combination of input A and B
	query := fmt.Sprintf(`
		{
			queryCombo(func: type(Combo)) @filter(((eq(A, "%s") AND eq(B, "%s")) OR (eq(A, "%s") AND eq(B, "%s")))) {
				uid
			}
		}
	`, a, b, b, a)

	result, err := queryDgraph(dgraphClient, query)
	if err != nil {
		log.Fatal(err)
	}
	//printResult(result)

	// if the number of combos is greater than one, return true
	if len(result["queryCombo"].([]interface{})) > 0 {
		return true
	}
	return false
}
