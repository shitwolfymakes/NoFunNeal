package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/dgraph-io/dgo/v200"
	"github.com/dgraph-io/dgo/v200/protos/api"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

var referer = "https://neal.fun/infinite-craft/"

var dgraphClient *dgo.Dgraph

var agentId string

func init() {
	// Generate a new UUID
	id := uuid.New()
	agentId = id.String()
	fmt.Println(agentId)
}

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
	printJSON(result)
	for _, node := range result["all"].([]interface{}) {
		nodeMap := node.(map[string]interface{})
		fmt.Printf("UID: %s, Name: %s\n", nodeMap["uid"], nodeMap["name"])
	}

	// check for combos
	comboFound := comboExists("Fire", "Water")
	fmt.Printf("comboExists: %t\n", comboFound)
	comboFound = comboExists("Fire", "Earth")
	fmt.Printf("comboExists: %t\n", comboFound)

	// check for duplicate combo
	comboFound = dupComboExists("Water", "Fire", "Steam")
	fmt.Printf("dupComboExists: %t\n", comboFound)
	comboFound = dupComboExists("Fire", "Water", "Steam")
	fmt.Printf("dupComboExists: %t\n", comboFound)

	// test url encoding
	input := "Water"
	fmt.Printf("name: %s\n", input)
	fmt.Printf("encodedName: %s\n", encodeInput(input))

	// test url construction
	encodedUrl := craftUrl(input, "Testing")
	fmt.Println(encodedUrl)

	// test sending a get request
	fmt.Printf("--- test sending a get request\n")
	response, err := sendGetRequest(encodedUrl, referer)
	if err != nil {
		log.Fatal(err)
	}
	printJSON(response)

	// TODO: log get response data to separate db for metrics
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

func sendGetRequest(url, referer string) (map[string]interface{}, error) {
	// Create a new request with a referer
	fmt.Printf("--- create request\n")
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	//req.Host = "neal.fun"
	//req.Header.Set("Accept", "*/*")
	//req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	//req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	//req.Header.Set("Alt-Used", "neal.fun")
	//req.Header.Set("Connection", "keep-alive")
	//req.Header.Set("DNT", "1")
	//req.Header.Set("Host", "neal.fun")
	req.Header.Set("Referer", referer)
	//req.Header.Set("Sec-Fetch-Dest", "empty")
	//req.Header.Set("Sec-Fetch-Mode", "cors")
	//req.Header.Set("Sec-Fetch-Sire", "same-origin")
	//req.Header.Set("Sec-GPC", "1")
	//req.Header.Set("TE", "trailers")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/115.0")

	// Print the request content
	dump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		return nil, err
	}
	fmt.Println("Request content:")
	fmt.Println(string(dump))

	// Send the request
	fmt.Printf("--- send request\n")
	//client := http.Client{}
	// Create a custom transport with the specified HTTP version
	transport := &http.Transport{
		// Specify the desired HTTP version here
		// For example, to use HTTP/2.0:
		// TLSNextProto: make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
		TLSNextProto: make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
	}
	client := &http.Client{
		Transport: transport,
	}
	reqTime := time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Print the HTTP status code
	fmt.Printf("--- return code\n")
	fmt.Println("HTTP Status Code:", resp.StatusCode)
	fmt.Println(resp)

	// Decode JSON response
	fmt.Printf("--- decode json\n")
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	// store status code in the result data before returning
	result["statusCode"] = resp.StatusCode
	result["reqTimestamp"] = reqTime
	result["respTimestamp"] = resp.Header.Get("Date")
	return result, nil
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

func printJSON(result map[string]interface{}) {
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
	//printJSON(result)

	// if the number of combos is greater than one, return true
	if len(result["queryCombo"].([]interface{})) > 0 {
		return true
	}
	return false
}

func dupComboExists(a, b, comboResult string) bool {
	// get nodes with a combination of input A and B
	query := fmt.Sprintf(`
		{
			queryCombo(func: type(Combo)) @filter(((eq(A, "%s") AND eq(B, "%s")) AND eq(ComboResult, "%s"))) {
				uid
			}
		}
	`, a, b, comboResult)

	result, err := queryDgraph(dgraphClient, query)
	if err != nil {
		log.Fatal(err)
	}
	//printJSON(result)

	// if the number of combos is greater than one, return true
	if len(result["queryCombo"].([]interface{})) > 0 {
		return true
	}
	return false
}
