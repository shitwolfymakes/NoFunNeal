package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

var (
	// dgraph shit
	dgraphURI    = "localhost:9080"
	dgraphClient *dgo.Dgraph

	// mongoDB shit
	secretFiles = map[string]string{
		"MONGODB_ROOT_USERNAME_SECRET": "mongodb_root_username.txt",
		"MONGODB_ROOT_PASSWORD_SECRET": "mongodb_root_password.txt",
	}
	mongoURI    = "mongodb://localhost:27017"
	mongoDbName = "infinite-craft-metrics"
	client      *mongo.Client
	collection  *mongo.Collection

	// agent-wide shit
	agentId    string
	secretsDir string
	referer    = "https://neal.fun/infinite-craft/"
)

func init() {
	// Check if the directory path is provided as an argument
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <directory path>")
		os.Exit(1)
	}

	// Generate a new UUID
	id := uuid.New()
	agentId = id.String()
	fmt.Println("Agent UUID: " + agentId)

	preflightSecrets(os.Args[1])
	preflightDgraph()
	preflightMongoDb(agentId)
	preflightTests()
}

func preflightSecrets(dirPath string) {
	// Get the absolute path of the directory.
	absDirPath, err := filepath.Abs(dirPath)
	if err != nil {
		log.Fatal(err)
	}

	// Check that the directory exists.
	_, err = os.Stat(absDirPath)
	if os.IsNotExist(err) {
		log.Fatal(err)
	}

	// Set the global
	secretsDir = absDirPath

	// Make sure all the secrets exist.
	for _, value := range secretFiles {
		// Construct the file path by appending the directory path and the filename
		filePath := filepath.Join(absDirPath, value)
		_, err = os.Stat(filePath)
		if os.IsNotExist(err) {
			log.Fatal(err)
		}
	}
	fmt.Println("PREFLIGHT -- SECRETS: All secret files are found.")
}

func preflightDgraph() {
	// Dial a gRPC connection. The address to dial must be passed as parameter.
	conn, err := grpc.Dial(dgraphURI, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}

	// Create a new Dgraph client.
	dgraphClient = dgo.NewDgraphClient(api.NewDgraphClient(conn))

	// Perform a test query.
	query := `
		{
			all(func: has(result)) @filter(eq(result, "Water")) {
				uid
				result
			}
		}
	`
	_, err = queryDgraph(query)
	if err != nil {
		log.Fatal(err)
	}

	// Print the results.
	//printJSON(result)
	fmt.Println("PREFLIGHT -- DGRAPGH: Dgraph db accessible as expected.")
}

func preflightMongoDb(agentId string) {
	// Set client options for authentication
	clientOptions := options.Client().ApplyURI(mongoURI)
	clientOptions.Auth = &options.Credential{
		Username: getSecret("MONGODB_ROOT_USERNAME_SECRET"),
		Password: getSecret("MONGODB_ROOT_PASSWORD_SECRET"),
	}

	// Connect to MongoDB
	var err error
	client, err = mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	// Check the connection
	err = client.Ping(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}

	// Get a handle for your collection
	collection = client.Database(mongoDbName).Collection(agentId)
	fmt.Println("PREFLIGHT -- MONGODB: MongoDB db accessible as expected.")
}

func getSecret(secretName string) string {
	// Read the secret file
	secretFilePath := filepath.Join(secretsDir, secretFiles[secretName])
	secretBytes, err := os.ReadFile(secretFilePath)
	if err != nil {
		panic(err)
	}
	return string(secretBytes)
}

func preflightTests() {
	fmt.Println("PREFLIGHT -- TESTS: RUNNING TESTS")
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
	fmt.Printf("result: %s\n", input)
	fmt.Printf("encodedName: %s\n", encodeInput(input))

	// test url construction
	encodedUrl := craftUrl(input, "Fire")
	fmt.Println(encodedUrl)

	// test sending a get request
	response, metricData := sendGetRequest(encodedUrl)
	printJSON(response)

	// Test sending metrics to MongoDB
	sendMetrics(metricData)

	// Steam is already in the DB, so let's test these functions with that
	// insert when already in there, fail gracefully
	insertResult(response)
	// remove when already in there
	removeResult(response["result"].(string))
	// remove when not in there, fail gracefully
	removeResult(response["result"].(string))
	// insert when not in there
	insertResult(response)

	// test operations on combinations
	a := "Water"
	b := "Fire"
	comboResult := "Steam"
	// insert when already in there, fail gracefully
	insertCombo(a, b, comboResult)
	// remove when already in there
	removeCombo(a, b, comboResult)
	// remove when not in there, fail gracefully
	removeCombo(a, b, comboResult)
	// insert when not in there
	insertCombo(a, b, comboResult)
	fmt.Println("PREFLIGHT -- TESTS: COMPLETED")

	// test response processing loop
	fmt.Println("PREFLIGHT -- RESPONSE PROCESSING LOOP: STARTED")
	// craft the url
	a, b = "Earth", "Wind"
	encodedUrl = craftUrl(a, b)
	// send the request
	response, metricData = sendGetRequest(encodedUrl)
	printJSON(response)
	// process the response
	processResponse(a, b, response)
	// send the metrics
	sendMetrics(metricData)

	// clean up created nodes
	removeResult(response["result"].(string))
	removeCombo(a, b, response["result"].(string))
	fmt.Println("PREFLIGHT -- RESPONSE PROCESSING LOOP: COMPLETED")

	// get a pair of results to combine
	fmt.Println("PREFLIGHT -- RANDOM COMBINANT SELECTION")
	a, b = getResultPair()
	fmt.Println(a + " | " + b)
	a, b = getResultPair()
	fmt.Println(a + " | " + b)
	a, b = getResultPair()
	fmt.Println(a + " | " + b)
	a, b = getResultPair()
	fmt.Println(a + " | " + b)

	// test the completed game loop
	fmt.Println("PREFLIGHT -- LOOP TEST")
	startTime := time.Now()
	// get a pair of results to combine
	a, b = getResultPair()
	fmt.Printf("Pair to be combined: %s, %s\n", a, b)
	// craft the url
	encodedUrl = craftUrl(a, b)
	// send the request
	response, metricData = sendGetRequest(encodedUrl)
	// process the response
	processResponse(a, b, response)
	// send the metrics
	sendMetrics(metricData)

	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)
	// Convert the duration to milliseconds
	elapsedTimeMilliseconds := float64(elapsedTime.Nanoseconds()) / 1000000.0
	fmt.Printf("Elapsed time: %.2f milliseconds\n", elapsedTimeMilliseconds)

	// clean up created nodes
	removeResult(response["result"].(string))
	removeCombo(a, b, response["result"].(string))
	fmt.Println("PREFLIGHT -- LOOP TEST COMPLETED")
}

type MetricData struct {
	UUID          string
	Result        string
	EncodedURL    string
	HTTPCode      int
	ReqTimestamp  string
	RespTimestamp string
}

func main() {
	// Channel to receive OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Channel to communicate between goroutines to end the loop
	endLoop := make(chan bool)

	// Start a goroutine to run the loop
	go func() {
		var i int = 0
		for {
			select {
			case <-endLoop:
				return
			default:
				runLoop()
				i++
				fmt.Printf("Completed iteration %d\n", i)
			}
		}
	}()

	// Wait for a signal
	<-sigChan

	// Send a signal to end the loop
	endLoop <- true

	fmt.Println("Program ended.")
}

func runLoop() {
	startTime := time.Now()
	// get a pair of results to combine
	a, b := getResultPair()
	fmt.Printf("Pair to be combined: %s, %s\n", a, b)
	if comboExists(a, b) {
		fmt.Println("This combination already exists, skipping...")
		return
	}
	// craft the url
	encodedUrl := craftUrl(a, b)
	// send the request
	response, metricData := sendGetRequest(encodedUrl)
	// process the response
	processResponse(a, b, response)
	// send the metrics
	sendMetrics(metricData)

	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)
	// Convert the duration to milliseconds
	elapsedTimeMilliseconds := float64(elapsedTime.Nanoseconds()) / 1000000.0
	fmt.Printf("Elapsed time: %.2f milliseconds\n", elapsedTimeMilliseconds)

	// wait here, so we only wait after an API call
	time.Sleep(time.Second * 2)
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

func sendGetRequest(url string) (map[string]interface{}, MetricData) {
	// Create a new request with proper headers
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Referer", referer)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Safari/605.1.1")

	// Send the request
	transport := &http.Transport{
		// API doesn't accept HTTP/1.1, so use HTTP/2.0:
		TLSNextProto: make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
	}
	client := &http.Client{
		Transport: transport,
	}
	reqTime := time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// Print the HTTP status code
	//fmt.Println("HTTP Status Code:", resp.StatusCode)
	if resp.StatusCode == 403 {
		log.Fatal("403 error, restarting")
	}

	// Decode JSON response
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		log.Fatal(err)
	}

	metricData := MetricData{
		UUID:          agentId,
		Result:        result["result"].(string),
		EncodedURL:    url,
		HTTPCode:      resp.StatusCode,
		ReqTimestamp:  reqTime,
		RespTimestamp: resp.Header.Get("Date"),
	}
	return result, metricData
}

func sendMetrics(metricData MetricData) {
	_, err := collection.InsertOne(context.Background(), metricData)
	if err != nil {
		log.Fatal(err)
	}
}

func processResponse(a, b string, response map[string]interface{}) {
	// store result
	insertResult(response)
	// store combo
	insertCombo(a, b, response["result"].(string))
}

func getResultPair() (string, string) {
	// Query for the total number of nodes
	query := `
        {
			count(func: type(Result)) {
				count(uid)
			}
		}
    `
	response, err := queryDgraph(query)
	if err != nil {
		log.Fatal(err)
	}
	results := response["count"].([]interface{})
	numNodes := int(results[0].(map[string]interface{})["count"].(float64))

	// Query for two random nodes
	return getRandomResult(numNodes), getRandomResult(numNodes)
}

func getRandomResult(numNodes int) string {
	offset := rand.Intn(numNodes)
	query := fmt.Sprintf(`
		{
			result(func: type(Result), first: 1, offset: %d) {
				result
			}
		}
	`, offset)
	response, err := queryDgraph(query)
	if err != nil {
		log.Fatal(err)
	}

	results := response["result"].([]interface{})
	result := results[0].(map[string]interface{})["result"].(string)
	return result
}

func removeResult(name string) {
	// get uid of result by name
	response := getNodeByTypeAndName("Result", name)
	results := response["queryResult"].([]interface{})
	if len(results) == 0 {
		fmt.Printf("Result \"%s\" not found in database.\n", name)
		return
	}

	// extract uid
	uid := results[0].(map[string]interface{})["uid"].(string)
	// remove node by uid
	deleteNode(uid)
	fmt.Printf("Result \"%s\" removed successfully.\n", name)

}

func removeCombo(a, b, name string) {
	// get uid of combo by name, A, and B
	query := fmt.Sprintf(`
		{
			queryCombo(func: type(Combo)) @filter(((eq(A, "%s") AND eq(B, "%s")) AND eq(ComboResult, "%s"))) {
				uid
			}
		}
	`, a, b, name)
	response, err := queryDgraph(query)
	if err != nil {
		log.Fatal(err)
	}
	combos := response["queryCombo"].([]interface{})
	if len(combos) == 0 {
		fmt.Printf("Combo \"%s\" not found in database.\n", name)
		return
	}

	// extract uid
	uid := combos[0].(map[string]interface{})["uid"].(string)
	// remove node by uid
	deleteNode(uid)
	fmt.Printf("Combo \"%s\" removed successfully.\n", name)
}

func insertCombo(a, b, comboResult string) {
	// check if combo already exists
	if dupComboExists(a, b, comboResult) {
		fmt.Printf("Combo \"%s\" already exists.\n", comboResult)
		return
	}

	// Define the data to be inserted
	data := map[string]interface{}{
		"dgraph.type": "Combo",
		"A":           a,
		"B":           b,
		"ComboResult": comboResult,
	}

	insertNode(data)
	fmt.Printf("Combo \"%s\" inserted successfully.\n", comboResult)
}

func deleteNode(uid string) {
	ctx := context.Background()

	// Create a new transaction
	txn := dgraphClient.NewTxn()
	defer txn.Discard(ctx)

	// Create a new mutation
	mu := &api.Mutation{
		CommitNow:  true,
		DeleteJson: []byte(`{"uid":"` + uid + `"}`),
	}

	// Execute the mutation
	_, err := txn.Mutate(ctx, mu)
	if err != nil {
		log.Fatal(err)
	}
}

func insertResult(result map[string]interface{}) {
	name := result["result"].(string)

	// check if result already exists
	if nodeExists("Result", name) {
		fmt.Printf("Result \"%s\" already exists.\n", name)
		return
	}

	// Define the data to be inserted
	data := result
	data["encodedName"] = encodeInput(result["result"].(string))
	data["dgraph.type"] = "Result"
	insertNode(data)
	fmt.Printf("Result \"%s\" inserted successfully.\n", name)
}

func insertNode(data map[string]interface{}) {
	ctx := context.Background()

	// Create a new transaction
	txn := dgraphClient.NewTxn()
	defer txn.Discard(ctx)

	// Create a new mutation
	mu := &api.Mutation{
		CommitNow: true,
	}

	// Add a setJson operation to the mutation
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Fatal(err)
	}
	mu.SetJson = jsonData

	// Execute the mutation
	_, err = txn.Mutate(ctx, mu)
	if err != nil {
		log.Fatal(err)
	}
}

func queryDgraph(query string) (map[string]interface{}, error) {
	ctx := context.Background()
	response, err := dgraphClient.NewReadOnlyTxn().Query(ctx, query)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(response.Json, &result); err != nil {
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

func nodeExists(nodeType, name string) bool {
	response := getNodeByTypeAndName(nodeType, name)

	// if the number of combos is greater than one, return true
	if len(response["queryResult"].([]interface{})) > 0 {
		return true
	}
	return false
}

func getNodeByTypeAndName(nodeType, name string) map[string]interface{} {
	// construct query string
	query := fmt.Sprintf(`
		{
			queryResult(func: type("%s")) @filter(((eq(result, "%s")))) {
				uid
			}
		}
	`, nodeType, name)

	response, err := queryDgraph(query)
	if err != nil {
		log.Fatal(err)
	}
	return response
}

func comboExists(a string, b string) bool {
	// get nodes with a combination of input A and B
	query := fmt.Sprintf(`
		{
			queryCombo(func: type(Combo)) @filter(eq(A, "%s") AND eq(B, "%s")) {
				uid
			}
		}
	`, a, b)

	result, err := queryDgraph(query)
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

	result, err := queryDgraph(query)
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
