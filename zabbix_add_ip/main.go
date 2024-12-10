package zabbix_add_ip

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
)

//package main

// Zabbix API credentials and URL
const (
	zabbixURL   = "http://your_zabbix_server/api_jsonrpc.php"
	zabbixUser  = "your_zabbix_user"
	zabbixPass  = "your_zabbix_password"
	zabbixGroup = "int-test" // Zabbix group name
)

type ZabbixHost struct {
	Hostid     string `json:"hostid"`
	Host       string `json:"host"`
	DNS        string `json:"dns"`
	Interfaces []struct {
		IP  string `json:"ip"`
		DNS string `json:"dns"`
	} `json:"interfaces"`
}

type ZabbixAPIResponse struct {
	Result []ZabbixHost `json:"result"`
	Error  string       `json:"error"`
}

type ZabbixAPIRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params"`
	Auth    string                 `json:"auth"`
	ID      int                    `json:"id"`
}

// Получение токена авторизации
func getAuthToken() (string, error) {
	authRequest := ZabbixAPIRequest{
		JSONRPC: "2.0",
		Method:  "user.login",
		Params: map[string]interface{}{
			"user":     zabbixUser,
			"password": zabbixPass,
		},
		ID: 1,
	}
	requestBody, err := json.Marshal(authRequest)
	if err != nil {
		return "", fmt.Errorf("error marshaling auth request: %w", err)
	}

	resp, err := makeZabbixRequest(http.MethodPost, zabbixURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("error during auth request: %w", err)
	}
	defer resp.Body.Close()

	var authResponse struct {
		Result string `json:"result"`
		Error  string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&authResponse); err != nil {
		return "", fmt.Errorf("error decoding auth response: %w", err)
	}

	if authResponse.Error != "" {
		return "", fmt.Errorf("Zabbix auth error: %s", authResponse.Error)
	}
	return authResponse.Result, nil
}

func getHostsFromZabbix(authToken string) ([]ZabbixHost, error) {
	requestBody := ZabbixAPIRequest{
		JSONRPC: "2.0",
		Method:  "host.get",
		Params: map[string]interface{}{
			"output":           "extend",
			"selectInterfaces": "extend",
			"filter": map[string]interface{}{
				"groups": []map[string]string{
					{"name": zabbixGroup},
				},
			},
		},
		Auth: authToken,
		ID:   1,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling get hosts request: %w", err)
	}
	resp, err := makeZabbixRequest(http.MethodPost, zabbixURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error getting hosts from Zabbix: %w", err)
	}
	defer resp.Body.Close()

	var zabbixResponse ZabbixAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&zabbixResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling Zabbix response: %w", err)
	}

	if zabbixResponse.Error != "" {
		return nil, fmt.Errorf("Zabbix API error: %s", zabbixResponse.Error)
	}
	return zabbixResponse.Result, nil
}

func updateHostIP(host ZabbixHost, authToken string) error {
	ip, err := resolveDNS(host.DNS)
	if err != nil {
		return fmt.Errorf("error resolving DNS for %s: %w", host.DNS, err)
	}

	requestBody := ZabbixAPIRequest{
		JSONRPC: "2.0",
		Method:  "host.update",
		Params: map[string]interface{}{
			"hostid": host.Hostid,
			"interfaces": []map[string]interface{}{
				{
					"dns": host.DNS,
					"ip":  ip,
				},
			},
		},
		Auth: authToken,
		ID:   1,
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("error marshaling update request: %w", err)
	}

	_, err = makeZabbixRequest(http.MethodPost, zabbixURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error updating host IP in Zabbix: %w", err)
	}
	return nil
}

func resolveDNS(hostName string) (string, error) {
	ips, err := net.LookupIP(hostName)
	if err != nil {
		return "", fmt.Errorf("error resolving DNS for %s: %w", hostName, err)
	}
	if len(ips) == 0 {
		return "", fmt.Errorf("no IP addresses found for %s", hostName)
	}
	return ips[0].String(), nil // Возвращаем первый найденный IP
}

func makeZabbixRequest(method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 status code: %d", resp.StatusCode)
	}
	return resp, nil
}

func main() {
	authToken, err := getAuthToken()
	if err != nil {
		fmt.Printf("Error getting auth token: %v\n", err)
		os.Exit(1)
	}

	hosts, err := getHostsFromZabbix(authToken)
	if err != nil {
		fmt.Printf("Error getting hosts from Zabbix: %v\n", err)
		os.Exit(1)
	}

	for _, host := range hosts {
		if host.DNS != "" {
			err := updateHostIP(host, authToken)
			if err != nil {
				fmt.Printf("Error updating host %s: %v\n", host.Host, err)
			} else {
				fmt.Printf("Successfully updated host %s\n", host.Host)
			}
		}
	}
}
