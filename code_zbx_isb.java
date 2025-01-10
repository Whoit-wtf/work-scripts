
import com.fasterxml.jackson.databind.ObjectMapper;
import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class ZabbixHostUpdater {

    private static final String ZABBIX_URL = System.getenv("ZABBIX_URL");
    private static final String ZABBIX_USER = System.getenv("ZABBIX_USER");
    private static final String ZABBIX_PASSWORD = System.getenv("ZABBIX_PASSWORD");
     private static final String HOST_GROUP_NAME = System.getenv("HOST_GROUP");
    private static final ObjectMapper mapper = new ObjectMapper();

    public static void main(String[] args) {
        try {
            String token = getZabbixToken();
            String hostGroupId = getHostGroupId(token);
            List<Map<String, String>> hosts = getHostsByGroupId(token, hostGroupId);

             System.out.printf("Found %d hosts in group %s%n",hosts.size(),HOST_GROUP_NAME);

            for (Map<String, String> host : hosts) {
                String newName = host.get("name") + ".isb";
                 boolean update = updateHost(token, host.get("hostid"), newName);
               if (update){
                 System.out.printf("Host '%s' updated with new name: %s%n", host.get("host"), newName);
               }
            }

             System.out.println("Hosts update finished.");

        } catch (IOException e) {
            System.err.println("Error: " + e.getMessage());
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

     private static List<Map<String, String>> getHostsByGroupId(String token, String groupId) throws IOException {
       Map<String, Object> hostRequest = new HashMap<>();
       hostRequest.put("jsonrpc", "2.0");
       hostRequest.put("method", "host.get");
       Map<String, Object> params = new HashMap<>();
        params.put("output", Arrays.asList("hostid","host","name"));
       params.put("groupids", Arrays.asList(groupId));
       hostRequest.put("params", params);
        hostRequest.put("id", 1);


       String response = sendRequestWithAuth(ZABBIX_URL, hostRequest, token);
        Map<?, ?> jsonResponse = mapper.readValue(response, Map.class);

        if (jsonResponse.containsKey("error")) {
             Map<?, ?> error = (Map<?, ?>) jsonResponse.get("error");
            throw new IOException("Zabbix error: "+ error.get("message").toString() + ", code:"+ error.get("code").toString() + ", data:"+error.get("data").toString());
        }


        List<Map<String,String>> hosts =  (List<Map<String, String>>) jsonResponse.get("result");

        return  hosts;
    }

    private static boolean updateHost(String token, String hostId, String newName) throws IOException {
        Map<String, Object> hostRequest = new HashMap<>();
        hostRequest.put("jsonrpc", "2.0");
        hostRequest.put("method", "host.update");
        Map<String, Object> params = new HashMap<>();
        params.put("hostid", hostId);
        params.put("name", newName);
        hostRequest.put("params", params);
        hostRequest.put("id", 1);


      String response = sendRequestWithAuth(ZABBIX_URL, hostRequest, token);
       Map<?, ?> jsonResponse = mapper.readValue(response, Map.class);

        if (jsonResponse.containsKey("error")) {
             Map<?, ?> error = (Map<?, ?>) jsonResponse.get("error");
            throw new IOException("Zabbix error: "+ error.get("message").toString() + ", code:"+ error.get("code").toString() + ", data:"+error.get("data").toString());
        }


       if (jsonResponse.containsKey("result") &&  ((Map<?, ?>)jsonResponse.get("result")).containsKey("hostids")){
          return true;
       }else {
           throw new IOException("Error update host "+ hostId + " response: " +jsonResponse.get("result").toString() );
       }

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
