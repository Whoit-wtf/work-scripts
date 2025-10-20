import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.file.Files;
import java.nio.file.Path;
import java.time.Duration;
import java.util.*;
import java.util.concurrent.ConcurrentHashMap;
import java.util.regex.Pattern;
import java.util.stream.Collectors;

public class PrometheusHikariStats {
    private static final String PROMETHEUS_URL = "http://localhost:9090";
    private static final String METRIC_NAME = "hikaricp_connections";
    private static final Duration TIMEOUT = Duration.ofSeconds(30);
    private static final Pattern LABEL_PATTERN = Pattern.compile("([a-zA-Z_][a-zA-Z0-9_]*)=\"([^\"]*)\"");
    
    private final HttpClient httpClient;
    private final String prometheusBaseUrl;

    public PrometheusHikariStats(String prometheusBaseUrl) {
        this.prometheusBaseUrl = prometheusBaseUrl;
        this.httpClient = HttpClient.newBuilder()
                .connectTimeout(TIMEOUT)
                .build();
    }

    public static void main(String[] args) throws Exception {
        String prometheusUrl = args.length > 0 ? args[0] : PROMETHEUS_URL;
        String outputFile = args.length > 1 ? args[1] : "hikari_stats.txt";
        
        PrometheusHikariStats statsCollector = new PrometheusHikariStats(prometheusUrl);
        List<PoolStats> stats = statsCollector.collectWeeklyStats();
        statsCollector.writeToFile(stats, outputFile);
        
        System.out.println("Статистика сохранена в файл: " + outputFile);
        System.out.println("Обработано пулов: " + stats.size());
    }

    public List<PoolStats> collectWeeklyStats() throws Exception {
        Map<String, PoolStats> statsMap = new ConcurrentHashMap<>();
        
        // Получаем текущие значения коннектов
        List<MetricData> currentConnections = queryPrometheus(METRIC_NAME);
        
        // Получаем максимальные значения за неделю
        List<MetricData> maxConnections = queryPrometheus("max_over_time(" + METRIC_NAME + "[7d])");
        
        // Обрабатываем текущие значения
        processMetricData(currentConnections, statsMap, true);
        
        // Обрабатываем максимальные значения
        processMetricData(maxConnections, statsMap, false);
        
        return new ArrayList<>(statsMap.values());
    }

    private List<MetricData> queryPrometheus(String query) throws Exception {
        String url = prometheusBaseUrl + "/api/v1/query?query=" + 
                    java.net.URLEncoder.encode(query, "UTF-8");
        
        HttpRequest request = HttpRequest.newBuilder()
                .uri(URI.create(url))
                .timeout(TIMEOUT)
                .build();
        
        HttpResponse<String> response = httpClient.send(request, HttpResponse.BodyHandlers.ofString());
        
        if (response.statusCode() != 200) {
            throw new RuntimeException("HTTP error: " + response.statusCode() + " - " + response.body());
        }
        
        return parsePrometheusResponse(response.body());
    }

    private List<MetricData> parsePrometheusResponse(String responseBody) {
        List<MetricData> result = new ArrayList<>();
        
        try {
            // Упрощенный парсинг JSON ответа Prometheus
            // В реальной программе лучше использовать JSON парсер
            String[] lines = responseBody.split("\n");
            
            for (String line : lines) {
                if (line.contains("\"metric\"") && line.contains("\"value\"")) {
                    MetricData data = parseMetricLine(line);
                    if (data != null) {
                        result.add(data);
                    }
                }
            }
        } catch (Exception e) {
            System.err.println("Ошибка парсинга ответа Prometheus: " + e.getMessage());
        }
        
        return result;
    }

    private MetricData parseMetricLine(String line) {
        try {
            // Извлекаем метки
            int metricStart = line.indexOf("\"metric\":") + 9;
            int metricEnd = line.indexOf("}", metricStart);
            String metricPart = line.substring(metricStart, metricEnd);
            
            Map<String, String> labels = parseLabels(metricPart);
            
            // Извлекаем значение
            int valueStart = line.indexOf("\"value\":") + 9;
            int valueEnd = line.indexOf("]", valueStart);
            String valuePart = line.substring(valueStart, valueEnd);
            String[] valueParts = valuePart.split(",");
            double value = Double.parseDouble(valueParts[1].replace("\"", "").trim());
            
            return new MetricData(labels, value);
        } catch (Exception e) {
            System.err.println("Ошибка парсинга метрики: " + e.getMessage());
            return null;
        }
    }

    private Map<String, String> parseLabels(String labelsString) {
        Map<String, String> labels = new HashMap<>();
        java.util.regex.Matcher matcher = LABEL_PATTERN.matcher(labelsString);
        
        while (matcher.find()) {
            labels.put(matcher.group(1), matcher.group(2));
        }
        
        return labels;
    }

    private void processMetricData(List<MetricData> metrics, Map<String, PoolStats> statsMap, boolean isCurrent) {
        for (MetricData metric : metrics) {
            String instance = metric.labels.getOrDefault("instance", "unknown");
            String app = metric.labels.getOrDefault("app", 
                      metric.labels.getOrDefault("application", 
                      metric.labels.getOrDefault("job", "unknown")));
            String pool = metric.labels.getOrDefault("pool", "default");
            
            String key = instance + "|" + app + "|" + pool;
            
            PoolStats stats = statsMap.computeIfAbsent(key, 
                k -> new PoolStats(instance, app, pool));
            
            if (isCurrent) {
                stats.totalConnections = metric.value;
            } else {
                stats.maxConnections = Math.max(stats.maxConnections, metric.value);
            }
        }
    }

    public void writeToFile(List<PoolStats> stats, String filename) throws Exception {
        List<String> lines = new ArrayList<>();
        lines.add("instance|appName|pool|total_connections|max_connections");
        
        stats.stream()
                .sorted(Comparator.comparing((PoolStats s) -> s.instance)
                        .thenComparing(s -> s.appName)
                        .thenComparing(s -> s.pool))
                .forEach(stat -> {
                    String line = String.format("%s|%s|%s|%.0f|%.0f",
                            stat.instance,
                            stat.appName,
                            stat.pool,
                            stat.totalConnections,
                            stat.maxConnections);
                    lines.add(line);
                });
        
        Files.write(Path.of(filename), lines);
    }

    // Классы для хранения данных
    static class MetricData {
        final Map<String, String> labels;
        final double value;
        
        MetricData(Map<String, String> labels, double value) {
            this.labels = labels;
            this.value = value;
        }
    }

    static class PoolStats {
        final String instance;
        final String appName;
        final String pool;
        double totalConnections;
        double maxConnections;
        
        PoolStats(String instance, String appName, String pool) {
            this.instance = instance;
            this.appName = appName;
            this.pool = pool;
            this.totalConnections = 0;
            this.maxConnections = 0;
        }
    }
}