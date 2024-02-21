package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/dgraph-io/dgo/v200"
	"github.com/dgraph-io/dgo/v200/protos/api"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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
			all(func: has(result)) {
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

type MetricData struct {
	UUID          string
	Result        string
	EncodedURL    string
	HTTPCode      int
	ReqTimestamp  string
	RespTimestamp string
}

func main() {
	preflightTests()
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
	response, metricData, err := sendGetRequest(encodedUrl)
	if err != nil {
		log.Fatal(err)
	}
	printJSON(response)
	println(metricData.UUID)

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

func sendGetRequest(url string) (map[string]interface{}, MetricData, error) {
	// Create a new request with proper headers
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, MetricData{}, err
	}
	req.Header.Set("Referer", referer)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/115.0")

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
		return nil, MetricData{}, err
	}
	defer resp.Body.Close()

	// Print the HTTP status code
	fmt.Println("HTTP Status Code:", resp.StatusCode)

	// Decode JSON response
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, MetricData{}, err
	}

	metricData := MetricData{
		UUID:          agentId,
		Result:        result["result"].(string),
		EncodedURL:    url,
		HTTPCode:      resp.StatusCode,
		ReqTimestamp:  reqTime,
		RespTimestamp: resp.Header.Get("Date"),
	}
	return result, metricData, nil
}

func sendMetrics(metricData MetricData) {
	_, err := collection.InsertOne(context.Background(), metricData)
	if err != nil {
		log.Fatal(err)
	}
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
	// check if result already exists
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
	resp, err := dgraphClient.NewReadOnlyTxn().Query(ctx, query)
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
			queryCombo(func: type(Combo)) @filter(((eq(A, "%s") AND eq(B, "%s")) OR (eq(A, "%s") AND eq(B, "%s")))) {
				uid
			}
		}
	`, a, b, b, a)

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
