package zabbix_add_ip

//package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
)

// Zabbix API credentials and URL
const (
	zabbixURL   = "http://your_zabbix_server/api_jsonrpc.php"
	zabbixUser  = "your_zabbix_user"
	zabbixPass  = "your_zabbix_password"
	zabbixGroup = "int-test" // Zabbix group name
)

type ZabbixHost struct {
	Hostid     string `json:"hostid"`
	Name       string `json:"host"`
	DNS        string `json:"dns"`
	IP         string `json:"ip"`
	Interfaces []struct {
		IP  string `json:"ip"`
		DNS string `json:"dns"`
	} `json:"interfaces"`
}

type ZabbixAPIResponse struct {
	Result []ZabbixHost `json:"result"`
}

func getHostsFromZabbix() ([]ZabbixHost, error) {
	// Zabbix API request to get hosts from specific group
	requestBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "host.get",
		"params": map[string]interface{}{
			"output":           "extend",
			"selectInterfaces": "extend",
			"filter": map[string]interface{}{
				"groups": []map[string]string{
					{"name": zabbixGroup},
				},
			},
		},
		"auth": zabbixAuth(),
		"id":   1,
	}

	resp, err := makeZabbixRequest(requestBody)
	if err != nil {
		return nil, fmt.Errorf("error getting hosts from Zabbix: %w", err)
	}

	var zabbixResponse ZabbixAPIResponse
	if err := json.Unmarshal(resp, &zabbixResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling Zabbix response: %w", err)
	}

	return zabbixResponse.Result, nil
}

func updateHostIP(host ZabbixHost) error {
	// Zabbix API request to update host IP
	requestBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "host.update",
		"params": map[string]interface{}{
			"hostid": host.Hostid,
			"interfaces": []map[string]interface{}{
				{
					"dns": host.DNS,
					"ip":  host.IP,
				},
			},
		},
		"auth": zabbixAuth(),
		"id":   1,
	}

	_, err := makeZabbixRequest(requestBody)
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
	// Return the first IPv4 address found, or the first IPv6 if no IPv4 is found.
	for _, ip := range ips {
		if ip.To4() != nil {
			return ip.String(), nil
		}
	}
	return ips[0].String(), nil

}

func zabbixAuth() string {
	//TODO: Replace with secure method for storing Zabbix credentials.  Consider environment variables.
	return "YOUR_AUTH_TOKEN"
}

// Placeholder -  MUST BE IMPLEMENTED
func makeZabbixRequest(requestBody map[string]interface{}) ([]byte, error) {
	fmt.Println("makeZabbixRequest needs to be implemented")
	return nil, fmt.Errorf("makeZabbixRequest not implemented")
}

func main() {
	hosts, err := getHostsFromZabbix()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	for _, host := range hosts {
		if host.DNS != "" {
			ip, err := resolveDNS(host.DNS)
			if err != nil {
				fmt.Printf("Error resolving DNS for %s: %v\n", host.DNS, err)
				continue
			}
			//Update Zabbix only if DNS resolved successfully and IP is different from existing
			updateNeeded := true
			for _, iface := range host.Interfaces {
				if iface.IP == ip {
					updateNeeded = false
					break
				}
			}
			if updateNeeded {
				host.IP = ip
				err = updateHostIP(host)
				if err != nil {
					fmt.Printf("Error updating host %s: %v\n", host.Name, err)
				} else {
					fmt.Printf("Updated IP for host %s to %s\n", host.Name, ip)
				}
			} else {
				fmt.Printf("IP for host %s is already %s\n", host.Name, ip)
			}
		} else {
			fmt.Printf("Host %s has no DNS entry\n", host.Name)
		}
	}
}
