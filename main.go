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
	secret_files = map[string]string{
		"MONGODB_ROOT_USERNAME":                    "mongodb_root_username.txt",
		"MONGODB_ROOT_PASSWORD_SECRET":             "mongodb_root_password.txt",
		"MONGO_EXPRESS_ADMIN_USERNAME_SECRET":      "mongo_express_admin_username.txt",
		"MONGO_EXPRESS_ADMIN_PASSWORD_SECRET":      "mongo_express_admin_password.txt",
		"MONGO_EXPRESS_BASIC_AUTH_USERNAME_SECRET": "mongo_express_basic_auth_username.txt",
		"MONGO_EXPRESS_BASIC_AUTH_PASSWORD_SECRET": "mongo_express_basic_auth_password.txt",
	}
	mongoURI    = "mongodb://localhost:27017"
	mongoDbName = "infinite-craft-metrics"
	client      *mongo.Client
	collection  *mongo.Collection

	// agent-wide shit
	agentId string
	referer = "https://neal.fun/infinite-craft/"
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

	preflightDgraph()
	preflightMongoDb(agentId)
	preflightSecrets(os.Args[1])
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
			all(func: has(name)) {
				uid
				name
			}
		}
	`
	_, err = queryDgraph(dgraphClient, query)
	if err != nil {
		log.Fatal(err)
	}

	// Print the results.
	//printJSON(result)
	fmt.Println("PREFLIGHT -- DGRAPGH: Dgraph db accessible as expected.")
}

func preflightMongoDb(agentId string) {
	// Connect to MongoDB
	var err error
	client, err = mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURI))
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

	// Make sure all the secrets exist.
	for _, value := range secret_files {
		// Construct the file path by appending the directory path and the filename
		filePath := filepath.Join(absDirPath, value)
		_, err = os.Stat(filePath)
		if os.IsNotExist(err) {
			log.Fatal(err)
		}
	}
	fmt.Println("PREFLIGHT -- SECRETS: All secret files are found.")
}

func getSecret(secretName string) string {
	// Get the path to the secrets directory
	secretsDir := "./secrets/"

	// Read the secret file
	secretFilePath := secretsDir + secret_files[secretName]
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
	response, metricData, err := sendGetRequest(encodedUrl)
	if err != nil {
		log.Fatal(err)
	}
	printJSON(response)
	println(metricData.UUID)

	// TODO: log get response data to separate db for metrics
	sendMetrics(metricData)
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
