
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;

import java.io.IOException;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.util.Iterator;

public class ZabbixCpuIowaitChecker {

    private static final String ZABBIX_URL = "http://your-zabbix-server/api_jsonrpc.php"; // Замените на ваш URL
    private static final String USERNAME = "your_username";
    private static final String PASSWORD = "your_password";

    private static final ObjectMapper mapper = new ObjectMapper();
    private static final HttpClient client = HttpClient.newHttpClient();

    public static void main(String[] args) throws IOException, InterruptedException {
        String authToken = login();
        if (authToken == null) {
            System.err.println("Authentication failed");
            return;
        }

        String groupId = getGroupId(authToken, "solar");
        if (groupId == null) {
            System.err.println("Group 'solar' not found");
            return;
        }

        JsonNode hosts = getHosts(authToken, groupId);
        if (hosts == null || !hosts.isArray()) {
            System.err.println("Failed to get hosts");
            return;
        }

        for (JsonNode host : hosts) {
            String hostId = host.get("hostid").asText();
            String hostName = host.get("name").asText();

            String itemId = getItemId(authToken, hostId, "system.cpu.util[,iowait]");
            if (itemId == null) continue;

            Double lastValue = getLastValue(authToken, itemId);
            if (lastValue != null && lastValue > 30) {
                System.out.printf("Host: %s, CPU iowait: %.2f%%%n", hostName, lastValue);
            }
        }
    }

    private static String login() throws IOException, InterruptedException {
        String json = """
                {
                  "jsonrpc": "2.0",
                  "method": "user.login",
                  "params": {
                    "user": "%s",
                    "password": "%s"
                  },
                  "id": 1,
                  "auth": null
                }
                """.formatted(USERNAME, PASSWORD);

        JsonNode response = sendRequest(json);
        if (response.has("result")) {
            return response.get("result").asText();
        } else {
            System.err.println("Login error: " + response);
            return null;
        }
    }

    private static String getGroupId(String auth, String groupName) throws IOException, InterruptedException {
        String json = """
                {
                  "jsonrpc": "2.0",
                  "method": "hostgroup.get",
                  "params": {
                    "filter": {
                      "name": ["%s"]
                    }
                  },
                  "auth": "%s",
                  "id": 2
                }
                """.formatted(groupName, auth);

        JsonNode response = sendRequest(json);
        JsonNode result = response.get("result");
        if (result != null && result.isArray() && result.size() > 0) {
            return result.get(0).get("groupid").asText();
        }
        return null;
    }

    private static JsonNode getHosts(String auth, String groupId) throws IOException, InterruptedException {
        String json = """
                {
                  "jsonrpc": "2.0",
                  "method": "host.get",
                  "params": {
                    "groupids": "%s",
                    "output": ["hostid", "name"]
                  },
                  "auth": "%s",
                  "id": 3
                }
                """.formatted(groupId, auth);

        JsonNode response = sendRequest(json);
        return response.get("result");
    }

    private static String getItemId(String auth, String hostId, String key) throws IOException, InterruptedException {
        String json = """
                {
                  "jsonrpc": "2.0",
                  "method": "item.get",
                  "params": {
                    "hostids": "%s",
                    "search": {
                      "key_": "%s"
                    },
                    "output": ["itemid"]
                  },
                  "auth": "%s",
                  "id": 4
                }
                """.formatted(hostId, key, auth);

        JsonNode response = sendRequest(json);
        JsonNode result = response.get("result");
        if (result != null && result.isArray() && result.size() > 0) {
            return result.get(0).get("itemid").asText();
        }
        return null;
    }

    private static Double getLastValue(String auth, String itemId) throws IOException, InterruptedException {
        String json = """
                {
                  "jsonrpc": "2.0",
                  "method": "history.get",
                  "params": {
                    "itemids": "%s",
                    "sortfield": "clock",
                    "sortorder": "DESC",
                    "limit": 1,
                    "output": "extend",
                    "history": 0
                  },
                  "auth": "%s",
                  "id": 5
                }
                """.formatted(itemId, auth);

        JsonNode response = sendRequest(json);
        JsonNode result = response.get("result");
        if (result != null && result.isArray() && result.size() > 0) {
            String valueStr = result.get(0).get("value").asText();
            try {
                return Double.parseDouble(valueStr);
            } catch (NumberFormatException e) {
                return null;
            }
        }
        return null;
    }

    private static JsonNode sendRequest(String json) throws IOException, InterruptedException {
        HttpRequest request = HttpRequest.newBuilder()
                .uri(URI.create(ZABBIX_URL))
                .header("Content-Type", "application/json")
                .POST(HttpRequest.BodyPublishers.ofString(json))
                .build();

        HttpResponse<String> response = client.send(request, HttpResponse.BodyHandlers.ofString());
        return mapper.readTree(response.body());
    }
}
