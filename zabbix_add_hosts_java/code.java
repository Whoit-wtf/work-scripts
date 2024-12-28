
import com.fasterxml.jackson.databind.ObjectMapper;
import java.io.BufferedReader;
import java.io.FileReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.InetAddress;
import java.net.URL;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class ZabbixHostCreator {

    private static final String ZABBIX_URL = System.getenv("ZABBIX_URL");
    private static final String ZABBIX_USER = System.getenv("ZABBIX_USER");
    private static final String ZABBIX_PASSWORD = System.getenv("ZABBIX_PASSWORD");
    private static final String DNS_FILE = System.getenv("DNS_FILE");
    private static final String HOST_GROUP_NAME = System.getenv("HOST_GROUP");
    private static final ObjectMapper mapper = new ObjectMapper();


    public static void main(String[] args) {
        try {
            String token = getZabbixToken();
            String hostGroupId = getHostGroupId(token);

            try (BufferedReader reader = new BufferedReader(new FileReader(DNS_FILE))) {
                String dns;
                while ((dns = reader.readLine()) != null) {
                     dns = dns.trim();
                     if (dns.isEmpty()){
                         continue;
                     }

                    try {
                         String ip = getIpByDns(dns);
                         createHost(token, dns, ip, hostGroupId);
                     } catch (IOException e) {
                          System.err.println("Error during processing DNS " + dns + ": " + e.getMessage());
                     }
                }
            }


            System.out.println("Hosts creation finished.");
        } catch (IOException e) {
            System.err.println("Error: " + e.getMessage());
        }
    }

     private static String getIpByDns(String dns) throws IOException{
        try{
            InetAddress address = InetAddress.getByName(dns);
            return address.getHostAddress();
        } catch (Exception e){
            throw new IOException("Error getting ip address from dns: " + dns + e.getMessage());
        }
     }

    private static String getZabbixToken() throws IOException {
        Map<String, Object> authRequest = new HashMap<>();
        authRequest.put("jsonrpc", "2.0");
        authRequest.put("method", "user.login");
        Map<String, String> params = new HashMap<>();
        params.put("user", ZABBIX_USER);
        params.put("password", ZABBIX_PASSWORD);
        authRequest.put("params", params);
        authRequest.put("id", 1);

        String response = sendRequest(ZABBIX_URL, authRequest);
        Map<?, ?> jsonResponse = mapper.readValue(response, Map.class);

        if (jsonResponse.containsKey("error")) {
             Map<?, ?> error = (Map<?, ?>) jsonResponse.get("error");
            throw new IOException("Zabbix error: "+ error.get("message").toString() + ", code:"+ error.get("code").toString() + ", data:"+error.get("data").toString());
        }

        return jsonResponse.get("result").toString();
    }

    private static String getHostGroupId(String token) throws IOException {
        Map<String, Object> hostGroupRequest = new HashMap<>();
        hostGroupRequest.put("jsonrpc", "2.0");
        hostGroupRequest.put("method", "hostgroup.get");
        Map<String, Object> params = new HashMap<>();
        Map<String, String> filter = new HashMap<>();
        filter.put("name", HOST_GROUP_NAME);
        params.put("filter", filter);
        hostGroupRequest.put("params", params);
        hostGroupRequest.put("id", 1);

        String response = sendRequestWithAuth(ZABBIX_URL, hostGroupRequest, token);
        Map<?, ?> jsonResponse = mapper.readValue(response, Map.class);

        if (jsonResponse.containsKey("error")) {
             Map<?, ?> error = (Map<?, ?>) jsonResponse.get("error");
            throw new IOException("Zabbix error: "+ error.get("message").toString() + ", code:"+ error.get("code").toString() + ", data:"+error.get("data").toString());
        }


        List<?> result = (List<?>) jsonResponse.get("result");

        if (result != null && !result.isEmpty()) {
            Map<?,?> group = (Map<?,?>) result.get(0);
            if(group.containsKey("groupid")){
              return  group.get("groupid").toString();
            }

        }
        throw new IOException("Group with name '" + HOST_GROUP_NAME + "' not found.");
    }


     private static void createHost(String token, String dns, String ip, String hostGroupId) throws IOException {
        Map<String, Object> hostRequest = new HashMap<>();
        hostRequest.put("jsonrpc", "2.0");
        hostRequest.put("method", "host.create");


        Map<String, Object> params = new HashMap<>();
        params.put("host", dns);
        params.put("name", dns);
        List<Map<String, Object>> interfaces = Arrays.asList(createInterface(ip));
        params.put("interfaces", interfaces);

        List<Map<String, Object>> groups = Arrays.asList(createHostGroup(hostGroupId));
        params.put("groups", groups);

        hostRequest.put("params", params);
        hostRequest.put("id", 1);



        String response = sendRequestWithAuth(ZABBIX_URL, hostRequest, token);
        Map<?, ?> jsonResponse = mapper.readValue(response, Map.class);

          if (jsonResponse.containsKey("error")) {
             Map<?, ?> error = (Map<?, ?>) jsonResponse.get("error");
            throw new IOException("Zabbix error: "+ error.get("message").toString() + ", code:"+ error.get("code").toString() + ", data:"+error.get("data").toString());
        }


        if (jsonResponse.containsKey("result") &&  ((Map<?, ?>)jsonResponse.get("result")).containsKey("hostids")){
          System.out.println("Host '" + dns + "' created with ID: "+ ((List<?>)((Map<?, ?>)jsonResponse.get("result")).get("hostids")).get(0).toString());
        }else{
           throw  new IOException("Error create host " + dns + " error: "+ jsonResponse.get("result").toString());
        }

    }

    private static Map<String, Object> createInterface(String ip) {
        Map<String, Object> iface = new HashMap<>();
        iface.put("type", 1);
        iface.put("main", 1);
        iface.put("useip", 1);
        iface.put("ip", ip);
        iface.put("dns", "");
        iface.put("port", "10050");
        return iface;
    }


    private static Map<String, Object> createHostGroup(String groupId) {
        Map<String, Object> group = new HashMap<>();
        group.put("groupids", Arrays.asList(groupId));
        return group;
    }


    private static String sendRequest(String url, Map<String, Object> request) throws IOException {
        HttpURLConnection connection = (HttpURLConnection) new URL(url).openConnection();
        connection.setRequestMethod("POST");
        connection.setRequestProperty("Content-Type", "application/json");
        connection.setDoOutput(true);


        String jsonInputString = mapper.writeValueAsString(request);

        try (OutputStream os = connection.getOutputStream()) {
             byte[] input = jsonInputString.getBytes("utf-8");
             os.write(input, 0, input.length);
        }


        StringBuilder response = new StringBuilder();
        try (BufferedReader br = new BufferedReader(new InputStreamReader(connection.getInputStream(), "utf-8"))) {
            String responseLine;
            while ((responseLine = br.readLine()) != null) {
                response.append(responseLine.trim());
            }
        }
        connection.disconnect();

        return response.toString();
    }

     private static String sendRequestWithAuth(String url, Map<String, Object> request, String token) throws IOException {
        HttpURLConnection connection = (HttpURLConnection) new URL(url).openConnection();
        connection.setRequestMethod("POST");
        connection.setRequestProperty("Content-Type", "application/json");
        connection.setRequestProperty("Authorization", "Bearer "+token);
        connection.setDoOutput(true);


         String jsonInputString = mapper.writeValueAsString(request);

        try (OutputStream os = connection.getOutputStream()) {
            byte[] input = jsonInputString.getBytes("utf-8");
            os.write(input, 0, input.length);
        }


        StringBuilder response = new StringBuilder();
        try (BufferedReader br = new BufferedReader(new InputStreamReader(connection.getInputStream(), "utf-8"))) {
            String responseLine;
            while ((responseLine = br.readLine()) != null) {
                response.append(responseLine.trim());
            }
        }

         connection.disconnect();

        return response.toString();
    }
}
