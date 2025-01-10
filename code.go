package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// Zabbix API Request Structs
type ZabbixAuthRequest struct {
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  struct {
		User     string `json:"user"`
		Password string `json:"password"`
	} `json:"params"`
	ID int `json:"id"`
}

type ZabbixRequest struct {
	Jsonrpc string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      int         `json:"id"`
}

// Zabbix API Response Structs
type ZabbixResponse struct {
	Jsonrpc string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    string `json:"data"`
	} `json:"error,omitempty"`
	ID int `json:"id"`
}
type ZabbixHost struct {
	Hostid string `json:"hostid"`
	Host   string `json:"host"`
	Name   string `json:"name"`
}

// Utility function to load environment variables from .env file
func loadEnv() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
}

func getZabbixToken(url, user, password string) (string, error) {
	authRequest := ZabbixAuthRequest{
		Jsonrpc: "2.0",
		Method:  "user.login",
		Params: struct {
			User     string `json:"user"`
			Password string `json:"password"`
		}{
			User:     user,
			Password: password,
		},
		ID: 1,
	}
	jsonAuthRequest, err := json.Marshal(authRequest)
	if err != nil {
		return "", fmt.Errorf("error marshalling json request : %w", err)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonAuthRequest))
	if err != nil {
		return "", fmt.Errorf("error creating request : %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error performing request : %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}
	var zabbixResponse ZabbixResponse
	if err := json.Unmarshal(body, &zabbixResponse); err != nil {
		return "", fmt.Errorf("error unmarshaling response body : %w body:%s", err, string(body))
	}
	if zabbixResponse.Error.Message != "" {
		return "", fmt.Errorf("zabbix error: %s, code:%d, data:%s", zabbixResponse.Error.Message, zabbixResponse.Error.Code, zabbixResponse.Error.Data)
	}

	token, ok := zabbixResponse.Result.(string)
	if !ok {
		return "", fmt.Errorf("unexpected response format: %v", zabbixResponse.Result)
	}
	return token, nil
}

func getHostGroupId(token, url, groupName string) (string, error) {
	params := map[string]interface{}{
		"filter": map[string]interface{}{
			"name": groupName,
		},
	}
	request := ZabbixRequest{
		Jsonrpc: "2.0",
		Method:  "hostgroup.get",
		Params:  params,
		ID:      1,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("error marshalling hostgroup request: %w", err)
	}

	client := http.Client{
		Timeout: 10 * time.Second,
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating hostgroup request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending hostgroup request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading hostgroup response: %w", err)
	}
	var zabbixResponse ZabbixResponse
	if err := json.Unmarshal(body, &zabbixResponse); err != nil {
		return "", fmt.Errorf("error unmarshalling hostgroup response: %w, body:%s", err, string(body))
	}
	if zabbixResponse.Error.Message != "" {
		return "", fmt.Errorf("zabbix error: %s, code:%d, data:%s", zabbixResponse.Error.Message, zabbixResponse.Error.Code, zabbixResponse.Error.Data)
	}

	if result, ok := zabbixResponse.Result.([]interface{}); ok && len(result) > 0 {
		if group, ok := result[0].(map[string]interface{}); ok {
			if id, ok := group["groupid"].(string); ok {
				return id, nil
			}
		}
	}

	return "", fmt.Errorf("group with name '%s' not found", groupName)
}

func getHostsByGroupId(token, url, groupId string) ([]ZabbixHost, error) {
	params := map[string]interface{}{
		"output":   []string{"hostid", "host", "name"},
		"groupids": []string{groupId},
	}
	request := ZabbixRequest{
		Jsonrpc: "2.0",
		Method:  "host.get",
		Params:  params,
		ID:      1,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshalling host get request: %w", err)
	}

	client := http.Client{
		Timeout: 10 * time.Second,
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating host get request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending host get request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading host get response: %w", err)
	}
	var zabbixResponse ZabbixResponse
	if err := json.Unmarshal(body, &zabbixResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling host get response: %w, body:%s", err, string(body))
	}
	if zabbixResponse.Error.Message != "" {
		return nil, fmt.Errorf("zabbix error: %s, code:%d, data:%s", zabbixResponse.Error.Message, zabbixResponse.Error.Code, zabbixResponse.Error.Data)
	}

	var hosts []ZabbixHost
	if result, ok := zabbixResponse.Result.([]interface{}); ok {
		for _, hostData := range result {
			if host, ok := hostData.(map[string]interface{}); ok {
				hostItem := ZabbixHost{
					Hostid: host["hostid"].(string),
					Host:   host["host"].(string),
					Name:   host["name"].(string),
				}
				hosts = append(hosts, hostItem)
			}
		}
	}

	return hosts, nil
}
func updateHost(token, url, hostId, newName string) (bool, error) {
	params := map[string]interface{}{
		"hostid": hostId,
		"name":   newName,
	}

	request := ZabbixRequest{
		Jsonrpc: "2.0",
		Method:  "host.update",
		Params:  params,
		ID:      1,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return false, fmt.Errorf("error marshalling host update request: %w", err)
	}
	client := http.Client{
		Timeout: 10 * time.Second,
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return false, fmt.Errorf("error creating host update request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("error sending host update request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("error reading host update response: %w", err)
	}

	var zabbixResponse ZabbixResponse
	if err := json.Unmarshal(body, &zabbixResponse); err != nil {
		return false, fmt.Errorf("error unmarshalling host update response: %w, body:%s", err, string(body))
	}
	if zabbixResponse.Error.Message != "" {
		return false, fmt.Errorf("zabbix error: %s, code:%d, data:%s", zabbixResponse.Error.Message, zabbixResponse.Error.Code, zabbixResponse.Error.Data)
	}

	if result, ok := zabbixResponse.Result.(map[string]interface{}); ok {
		if _, ok := result["hostids"].([]interface{}); ok {
			return true, nil
		}
	}
	return false, fmt.Errorf("error update host: %v", zabbixResponse.Result)
}

func main() {
	loadEnv()
	zabbixURL := os.Getenv("ZABBIX_URL")
	zabbixUser := os.Getenv("ZABBIX_USER")
	zabbixPassword := os.Getenv("ZABBIX_PASSWORD")
	hostGroupName := os.Getenv("HOST_GROUP")

	token, err := getZabbixToken(zabbixURL, zabbixUser, zabbixPassword)
	if err != nil {
		log.Fatalf("Error getting Zabbix token: %v", err)
	}
	log.Println("Successfully logged in to Zabbix")

	hostGroupId, err := getHostGroupId(token, zabbixURL, hostGroupName)
	if err != nil {
		log.Fatalf("Error getting host group ID: %v", err)
	}

	hosts, err := getHostsByGroupId(token, zabbixURL, hostGroupId)
	if err != nil {
		log.Fatalf("Error getting hosts by group id: %v", err)
	}

	log.Printf("Found %d hosts in group %s", len(hosts), hostGroupName)

	for _, host := range hosts {
		newName := host.Name + ".isb"
		ok, err := updateHost(token, zabbixURL, host.Hostid, newName)
		if err != nil {
			log.Printf("Error update host %s with id %s: %v", host.Name, host.Hostid, err)
			continue
		}
		if ok {
			log.Printf("Host '%s' updated with new name: %s", host.Host, newName)
		}
	}

	log.Println("Hosts updated finished.")
}
