package minitwit_api_simulater_test

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	baseURL    = "http://localhost:8081"
	database   = "minitwit.db"
	username   = "simulator"
	password   = "super_safe!"
	latestFile = "latest_processed_sim_action_id.txt"
	schemaFile = "../minitwit/schema.sql"
)

var (
	httpClient *http.Client
	headers    http.Header
)

func TestMain(m *testing.M) {
	httpClient = &http.Client{
		Timeout: time.Second * 10,
	}

	credentials := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	headers = http.Header{
		"Connection":    {"close"},
		"Content-Type":  {"application/json"},
		"Authorization": {"Basic " + credentials},
	}

	initDB()

	if _, err := os.Stat(latestFile); os.IsNotExist(err) {
		err := os.WriteFile(latestFile, []byte("0"), 0644)
		if err != nil {
			fmt.Printf("Failed to create latest file: %v\n", err)
			os.Exit(1)
		}
	}

	code := m.Run()

	os.Remove(database)
	os.Remove(latestFile)

	os.Exit(code)
}

func initDB() {
	os.Remove(database)

	db, err := sql.Open("sqlite3", database)
	if err != nil {
		fmt.Printf("Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	schemaBytes, err := os.ReadFile(schemaFile)
	if err != nil {
		fmt.Printf("Failed to read schema file: %v\n", err)
		os.Exit(1)
	}

	_, err = db.Exec(string(schemaBytes))
	if err != nil {
		fmt.Printf("Failed to execute schema: %v\n", err)
		os.Exit(1)
	}
}

func sendRequest(method, endpoint string, data interface{}, params map[string]interface{}) (*http.Response, error) {
	url := baseURL + endpoint
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if params != nil {
		q := req.URL.Query()
		for key, value := range params {
			q.Add(key, fmt.Sprintf("%v", value))
		}
		req.URL.RawQuery = q.Encode()
	}


	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON: %w", err)
		}
		req.Body = io.NopCloser(bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
	}

	return httpClient.Do(req)
}

func getLatest(t *testing.T) int {
	resp, err := sendRequest("GET", "/latest", nil, nil)
	require.NoError(t, err, "Failed to get latest value")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "Latest request failed with status: %d", resp.StatusCode)

	var result map[string]int
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err, "Failed to decode latest response")

	return result["latest"]
}

func TestLatest(t *testing.T) {
	// In TestLatest
	testUsername := fmt.Sprintf("test_%d", time.Now().UnixNano())
	data := map[string]string{
		"username": testUsername,
		"email":    testUsername + "@test.com",
		"pwd":      "foo",
	}
	params := map[string]interface{}{
		"latest": 1337,
	}

	resp, err := sendRequest("POST", "/register", data, params)
	require.NoError(t, err, "Failed to send request")
	require.Equal(t, http.StatusOK, resp.StatusCode, "Register request failed with status: %d", resp.StatusCode)
	resp.Body.Close()

	latest := getLatest(t)
	assert.Equal(t, 1337, latest, "Latest value was not updated correctly")
}

func TestRegister(t *testing.T) {
	data := map[string]string{
		"username": "a",
		"email":    "a@a.a",
		"pwd":      "a",
	}
	params := map[string]interface{}{
		"latest": 1,
	}

	resp, err := sendRequest("POST", "/register", data, params)
	require.NoError(t, err, "Failed to send register request")
	require.Equal(t, http.StatusOK, resp.StatusCode, "Register request failed with status: %d", resp.StatusCode)
	resp.Body.Close()
	latest := getLatest(t)
	assert.Equal(t, 1, latest, "Latest value was not updated correctly")
}

func TestCreateMsg(t *testing.T) {
	username := "a"
	data := map[string]string{
		"content": "Blub!",
	}
	params := map[string]interface{}{
		"latest": 2,
	}

	resp, err := sendRequest("POST", fmt.Sprintf("/msgs/%s", username), data, params)
	require.NoError(t, err, "Failed to send create message request")
	require.Equal(t, http.StatusNoContent, resp.StatusCode, "Create message request failed with status: %d", resp.StatusCode)
	resp.Body.Close()
	latest := getLatest(t)
	assert.Equal(t, 2, latest, "Latest value was not updated correctly")
}

type Message struct {
	Content string `json:"content"`
	PubDate string `json:"pub_date"`
	User    string `json:"user"`
}

func TestGetLatestUserMsgs(t *testing.T) {
	username := "a"
	params := map[string]interface{}{
		"no":     20,
		"latest": 3,
	}

	resp, err := sendRequest("GET", fmt.Sprintf("/msgs/%s", username), nil, params)
	require.NoError(t, err, "Failed to get user messages")
	require.Equal(t, http.StatusOK, resp.StatusCode, "Get user messages request failed with status: %d", resp.StatusCode)

	var messages []Message
	err = json.NewDecoder(resp.Body).Decode(&messages)
	require.NoError(t, err, "Failed to decode user messages response")
	resp.Body.Close()

	found := false
	for _, msg := range messages {
		if msg.Content == "Blub!" && msg.User == username {
			found = true
			break
		}
	}
	assert.True(t, found, "Could not find posted message in user messages")

	latest := getLatest(t)
	assert.Equal(t, 3, latest, "Latest value was not updated correctly")
}
func TestGetLatestMsgs(t *testing.T) {
	username := "a"
	params := map[string]interface{}{
		"no":     20,
		"latest": 4,
	}

	resp, err := sendRequest("GET", "/msgs", nil, params)
	require.NoError(t, err, "Failed to get messages")
	require.Equal(t, http.StatusOK, resp.StatusCode, "Error get messages request failed with status: %d", resp.StatusCode)

	var messages []Message
	err = json.NewDecoder(resp.Body).Decode(&messages)
	require.NoError(t, err, "Error failed to decode messages response")
	resp.Body.Close()

	found := false
	for _, msg := range messages {
		if msg.Content == "Blub!" && msg.User == username {
			found = true
			break
		}
	}
	assert.True(t, found, "Could not find posted message in all messages")

	latest := getLatest(t)
	assert.Equal(t, 4, latest, "Latest value was not updated correctly")
}

func TestRegisterB(t *testing.T) {
	data := map[string]string{
		"username": "b",
		"email":    "b@b.b",
		"pwd":      "b",
	}
	params := map[string]interface{}{
		"latest": 5,
	}

	resp, err := sendRequest("POST", "/register", data, params)
	require.NoError(t, err, "Failed to send register request for user B")
	require.Equal(t, http.StatusOK, resp.StatusCode, "Register request for user B failed with status: %d", resp.StatusCode)
	resp.Body.Close()

	latest := getLatest(t)
	assert.Equal(t, 5, latest, "Latest value was not updated correctly")
}

func TestRegisterC(t *testing.T) {
	data := map[string]string{
		"username": "c",
		"email":    "c@c.c",
		"pwd":      "c",
	}
	params := map[string]interface{}{
		"latest": 6,
	}

	resp, err := sendRequest("POST", "/register", data, params)
	require.NoError(t, err, "Failed to send register request for user C")
	require.Equal(t, http.StatusOK, resp.StatusCode, "Register request for user C failed with status: %d", resp.StatusCode)
	resp.Body.Close()

	latest := getLatest(t)
	assert.Equal(t, 6, latest, "Latest value was not updated correctly")
}

type FollowsResponse struct {
	Follows []string `json:"follows"`
}

func TestFollowUser(t *testing.T) {
	username := "a"

	followData := map[string]string{
		"follow": "b",
	}
	params := map[string]interface{}{
		"latest": 7,
	}

	resp, err := sendRequest("POST", fmt.Sprintf("/fllws/%s", username), followData, params)
	require.NoError(t, err, "Failed to follow user B")
	require.Equal(t, http.StatusOK, resp.StatusCode, "Follow user B request failed with status: %d", resp.StatusCode)
	resp.Body.Close()

	followData = map[string]string{
		"follow": "c",
	}
	params = map[string]interface{}{
		"latest": 8,
	}

	resp, err = sendRequest("POST", fmt.Sprintf("/fllws/%s", username), followData, params)
	require.NoError(t, err, "Failed to follow user C")
	require.Equal(t, http.StatusOK, resp.StatusCode, "Follow user C request failed with status: %d", resp.StatusCode)
	resp.Body.Close()

	params = map[string]interface{}{
		"no":     20,
		"latest": 9,
	}

	resp, err = sendRequest("GET", fmt.Sprintf("/fllws/%s", username), nil, params)
	require.NoError(t, err, "Failed to get follows")
	require.Equal(t, http.StatusOK, resp.StatusCode, "Get follows request failed with status: %d", resp.StatusCode)

	var followsResponse FollowsResponse
	err = json.NewDecoder(resp.Body).Decode(&followsResponse)
	require.NoError(t, err, "Failed to decode follows response")
	resp.Body.Close()

	assert.Contains(t, followsResponse.Follows, "b", "User b not found in follows list")
	assert.Contains(t, followsResponse.Follows, "c", "User c not found in follows list")

	latest := getLatest(t)
	assert.Equal(t, 9, latest, "Latest value was not updated correctly")
}

func TestUnfollowUser(t *testing.T) {
	username := "a"

	unfollowData := map[string]string{
		"unfollow": "b",
	}
	params := map[string]interface{}{
		"latest": 10,
	}

	resp, err := sendRequest("POST", fmt.Sprintf("/fllws/%s", username), unfollowData, params)
	require.NoError(t, err, "Failed to unfollow user B")
	require.Equal(t, http.StatusOK, resp.StatusCode, "Unfollow user B request failed with status: %d", resp.StatusCode)
	resp.Body.Close()

	params = map[string]interface{}{
		"no":     20,
		"latest": 11,
	}

	resp, err = sendRequest("GET", fmt.Sprintf("/fllws/%s", username), nil, params)
	require.NoError(t, err, "Failed to get follows after unfollowing")
	require.Equal(t, http.StatusOK, resp.StatusCode, "Get follows request failed after unfollowing with status: %d", resp.StatusCode)

	var followsResponse FollowsResponse
	err = json.NewDecoder(resp.Body).Decode(&followsResponse)
	require.NoError(t, err, "Failed to decode follows response after unfollowing")
	resp.Body.Close()

	assert.NotContains(t, followsResponse.Follows, "b", "User b still in follows list after unfollowing")

	assert.Contains(t, followsResponse.Follows, "c", "User c not found in follows list after unfollowing user B")

	latest := getLatest(t)
	assert.Equal(t, 11, latest, "Latest value was not updated correctly")
}
