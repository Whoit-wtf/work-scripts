import kotlinx.serialization.*
import kotlinx.serialization.json.*
import okhttp3.*

@Serializable
data class ZabbixRequest<T>(
    val jsonrpc: String = "2.0",
    val method: String,
    val params: T,
    val id: Int = 1,
    val auth: String? = null
)

@Serializable
data class ZabbixResponse<T>(
    val jsonrpc: String,
    val result: T? = null,
    val error: ZabbixError? = null,
    val id: Int
)

@Serializable
data class ZabbixError(
    val code: Int,
    val message: String,
    val data: String
)

fun main() {
    val zabbixUrl = "http://your-zabbix-server/api_jsonrpc.php" // Замените на ваш URL
    val username = "your_username"
    val password = "your_password"

    val client = OkHttpClient()
    val json = Json { ignoreUnknownKeys = true }

    // 1. Авторизация
    val authToken = zabbixApiCall<String>(
        client, json, zabbixUrl,
        method = "user.login",
        params = mapOf("user" to username, "password" to password)
    ) ?: run {
        println("Failed to authenticate")
        return
    }

    // 2. Получаем ID группы "solar"
    val groupIdList = zabbixApiCall<List<JsonObject>>(
        client, json, zabbixUrl,
        method = "hostgroup.get",
        params = mapOf("filter" to mapOf("name" to listOf("solar")))
        , auth = authToken
    ) ?: run {
        println("Failed to get host group")
        return
    }

    if (groupIdList.isEmpty()) {
        println("Group 'solar' not found")
        return
    }

    val groupId = groupIdList[0]["groupid"]?.jsonPrimitive?.content ?: run {
        println("Group ID not found")
        return
    }

    // 3. Получаем хосты из группы
    val hosts = zabbixApiCall<List<JsonObject>>(
        client, json, zabbixUrl,
        method = "host.get",
        params = mapOf(
            "groupids" to groupId,
            "output" to listOf("hostid", "name")
        ),
        auth = authToken
    ) ?: run {
        println("Failed to get hosts")
        return
    }

    // 4. Для каждого хоста получаем последний элемент cpu iowait time
    // Предполагаем, что ключ элемента - system.cpu.util[,iowait]
    for (host in hosts) {
        val hostId = host["hostid"]?.jsonPrimitive?.content ?: continue
        val hostName = host["name"]?.jsonPrimitive?.content ?: "Unknown"

        // Получаем item с ключом system.cpu.util[,iowait]
        val items = zabbixApiCall<List<JsonObject>>(
            client, json, zabbixUrl,
            method = "item.get",
                        params = mapOf(
                "hostids" to hostId,
                "search" to mapOf("key_" to "system.cpu.util[,iowait]"),
                "output" to listOf("itemid")
            ),
            auth = authToken
        ) ?: continue

        if (items.isEmpty()) continue

        val itemId = items[0]["itemid"]?.jsonPrimitive?.content ?: continue

        // Получаем последнее значение элемента
        val history = zabbixApiCall<List<JsonObject>>(
            client, json, zabbixUrl,
            method = "history.get",
            params = mapOf(
                "itemids" to itemId,
                "sortfield" to "clock",
                "sortorder" to "DESC",
                "limit" to 1,
                "output" to "extend",
                "history" to 0 // 0 - float, 3 - unsigned int, зависит от типа данных
            ),
            auth = authToken
        ) ?: continue

        if (history.isEmpty()) continue

        val valueStr = history[0]["value"]?.jsonPrimitive?.content ?: continue
        val value = valueStr.toDoubleOrNull() ?: continue

        if (value > 30) {
            println("Host: $hostName, CPU iowait: $value%")
        }
    }
}

// Универсальная функция для вызова Zabbix API
inline fun <reified T> zabbixApiCall(
    client: OkHttpClient,
    json: Json,
    url: String,
    method: String,
    params: Any,
    auth: String? = null
): T? {
    val requestBody = json.encodeToString(
        ZabbixRequest(
            method = method,
            params = params,
            auth = auth
        )
    ).toRequestBody("application/json".toMediaType())

    val request = Request.Builder()
        .url(url)
        .post(requestBody)
        .build()

    client.newCall(request).execute().use { response ->
        if (!response.isSuccessful) {
            println("HTTP error: ${response.code}")
            return null
        }
        val body = response.body?.string() ?: return null
        val resp = json.decodeFromString<ZabbixResponse<T>>(body)
        if (resp.error != null) {
            println("Zabbix API error: ${resp.error.message} - ${resp.error.data}")
            return null
        }
        return resp.result
    }
}
