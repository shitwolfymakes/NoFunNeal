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
	"time"
)

var (
	// dgraph shit
	dgraphURI    = "localhost:9080"
	dgraphClient *dgo.Dgraph

	// mongoDB shit
	mongoURI    = "mongodb://localhost:27017"
	mongoDbName = "infinite-craft-metrics"
	client      *mongo.Client
	collection  *mongo.Collection

	// agent-wide shit
	agentId string
	referer = "https://neal.fun/infinite-craft/"
)

func init() {
	// Generate a new UUID
	id := uuid.New()
	agentId = id.String()
	fmt.Println(agentId)

	preflightDgraph()
	preflightMongoDb(agentId)
}

func preflightDgraph() {
	// Dial a gRPC connection. The address to dial must be passed as parameter.
	conn, err := grpc.Dial(dgraphURI, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}

	// Create a new Dgraph client.
	dgraphClient = dgo.NewDgraphClient(api.NewDgraphClient(conn))
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
	// Dial a gRPC connection. The address to dial must be passed as parameter.
	conn, err := grpc.Dial(dgraphURI, grpc.WithInsecure())
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
