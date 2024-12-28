package zabbix_add_hosts

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type ZabbixAuth struct {
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

type ZabbixHostGroup struct {
	GroupIDs []string `json:"groupids"`
}

type ZabbixHost struct {
	Host       string `json:"host"`
	Name       string `json:"name"`
	Interfaces []struct {
		Type  int    `json:"type"`
		Main  int    `json:"main"`
		Useip int    `json:"useip"`
		Ip    string `json:"ip"`
		Dns   string `json:"dns"`
		Port  string `json:"port"`
	} `json:"interfaces"`
	Groups []ZabbixHostGroup `json:"groups"`
}

func loadEnv() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
}

func getZabbixToken(url, user, password string) (string, error) {

	auth := ZabbixAuth{
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

	jsonData, err := json.Marshal(auth)
	if err != nil {
		return "", fmt.Errorf("error marshalling auth data: %w", err)
	}

	client := http.Client{
		Timeout: 10 * time.Second,
	}
	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}

	var zabbixResponse ZabbixResponse
	if err := json.Unmarshal(body, &zabbixResponse); err != nil {
		return "", fmt.Errorf("error unmarshalling response: %w, body:%s", err, string(body))
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
	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
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

func createHost(token, url, host ZabbixHost) (string, error) {

	request := ZabbixRequest{
		Jsonrpc: "2.0",
		Method:  "host.create",
		Params:  host,
		ID:      1,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("error marshalling host create request: %w", err)
	}

	client := http.Client{
		Timeout: 10 * time.Second,
	}
	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return "", fmt.Errorf("error creating host create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending host create request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading host create response: %w", err)
	}
	var zabbixResponse ZabbixResponse
	if err := json.Unmarshal(body, &zabbixResponse); err != nil {
		return "", fmt.Errorf("error unmarshalling host create response: %w, body:%s", err, string(body))
	}
	if zabbixResponse.Error.Message != "" {
		return "", fmt.Errorf("zabbix error: %s, code:%d, data:%s", zabbixResponse.Error.Message, zabbixResponse.Error.Code, zabbixResponse.Error.Data)
	}

	if result, ok := zabbixResponse.Result.(map[string]interface{}); ok {
		if ids, ok := result["hostids"].([]interface{}); ok && len(ids) > 0 {
			if id, ok := ids[0].(string); ok {
				return id, nil
			}
		}

	}
	return "", fmt.Errorf("error create host: %v", zabbixResponse.Result)
}

func getHostIp(dns string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip4", dns)
	if err != nil {
		return "", fmt.Errorf("error resolving IP for DNS %s: %w", dns, err)
	}
	if len(ips) > 0 {
		return ips[0].String(), nil
	}
	return "", fmt.Errorf("no IPs found for DNS %s", dns)
}

func main() {
	loadEnv()
	zabbixURL := os.Getenv("ZABBIX_URL")
	zabbixUser := os.Getenv("ZABBIX_USER")
	zabbixPassword := os.Getenv("ZABBIX_PASSWORD")
	dnsFile := os.Getenv("DNS_FILE")
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

	file, err := os.Open(dnsFile)
	if err != nil {
		log.Fatalf("Error opening DNS file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		dns := strings.TrimSpace(scanner.Text())
		if dns == "" {
			continue
		}

		ip, err := getHostIp(dns)
		if err != nil {
			log.Printf("error getting ip address %s %s", dns, err)
			continue
		}
		log.Printf("resolved ip %s of host %s", ip, dns)

		host := ZabbixHost{
			Host: dns,
			Name: dns,
			Interfaces: []struct {
				Type  int    `json:"type"`
				Main  int    `json:"main"`
				Useip int    `json:"useip"`
				Ip    string `json:"ip"`
				Dns   string `json:"dns"`
				Port  string `json:"port"`
			}{
				{
					Type:  1,
					Main:  1,
					Useip: 1,
					Ip:    ip,
					Dns:   "",
					Port:  "10050",
				},
			},
			Groups: []ZabbixHostGroup{
				{
					GroupIDs: []string{hostGroupId},
				},
			},
		}

		hostID, err := createHost(token, zabbixURL, host)
		if err != nil {
			log.Printf("Error creating host '%s': %v", dns, err)
			continue
		}
		log.Printf("Host '%s' created with ID: %s", dns, hostID)

	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading DNS file: %v", err)
	}

	log.Println("Hosts creation finished.")
}
