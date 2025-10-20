package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// PrometheusResponse представляет структуру ответа Prometheus API
type PrometheusResponse struct {
	Data struct {
		Result []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// MetricResult хранит данные метрики с распарсенными значениями
type MetricResult struct {
	Labels  map[string]string
	Value   float64
	SortKey string
}

func main() {
	prometheusURL := "http://localhost:9090/api/v1/query"

	// Выполняем запросы
	maxConnections := queryPrometheus(prometheusURL, "hikaricp_connections_max")
	maxOverTime := queryPrometheus(prometheusURL, "max_over_time(hikaricp_connections{}[7d])")

	// Создаем мапу для объединения результатов
	results := make(map[string][2]float64)

	// Обрабатываем первый запрос
	for _, mr := range maxConnections {
		key := mr.SortKey
		results[key] = [2]float64{mr.Value, 0}
	}

	// Объединяем со вторым запросом
	for _, mr := range maxOverTime {
		key := mr.SortKey
		if existing, exists := results[key]; exists {
			existing[1] = mr.Value
			results[key] = existing
		}
	}

	// Подготавливаем данные для вывода
	var output []string
	for key, values := range results {
		if values[0] != 0 && values[1] != 0 {
			parts := strings.Split(key, "|")
			output = append(output, fmt.Sprintf("%s|%.0f|%.0f", key, values[0], values[1]))
		}
	}

	// Сортируем вывод для удобства
	sort.Strings(output)

	// Выводим результаты
	fmt.Println("appName|instance|job|nodeName|pool|connections_max|max_over_time_7d")
	for _, line := range output {
		fmt.Println(line)
	}
}

// queryPrometheus выполняет запрос к Prometheus и возвращает результаты
func queryPrometheus(baseURL, query string) []MetricResult {
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest("GET", baseURL, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Параметры запроса
	q := url.Values{}
	q.Add("query", query)
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var pr PrometheusResponse
	if err := json.Unmarshal(body, &pr); err != nil {
		log.Fatal(err)
	}

	var results []MetricResult
	for _, result := range pr.Data.Result {
		// Извлекаем значение
		if len(result.Value) < 2 {
			continue
		}

		valStr, ok := result.Value[1].(string)
		if !ok {
			continue
		}

		value, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			continue
		}

		// Формируем ключ сортировки
		labels := []string{
			result.Metric["appName"],
			result.Metric["instance"],
			result.Metric["job"],
			result.Metric["nodeName"],
			result.Metric["pool"],
		}
		sortKey := strings.Join(labels, "|")

		results = append(results, MetricResult{
			Labels:  result.Metric,
			Value:   value,
			SortKey: sortKey,
		})
	}

	return results
}
