package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

const (
	prometheusURL = "http://localhost:9090" // Замените на адрес вашего Prometheus
	metricName    = "hikaricp_connections"
	outputFile    = "hikaricp_stats.txt"
)

// PrometheusResponse представляет структуру ответа от Prometheus API
type PrometheusResponse struct {
	Data struct {
		Result []struct {
			Metric struct {
				Instance string `json:"instance"`
				AppName  string `json:"appName"`
				Pool     string `json:"pool"`
			} `json:"metric"`
			Values [][]interface{} `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

func main() {
	ctx := context.Background()
	client := &http.Client{Timeout: 30 * time.Second}

	// Получаем данные за последнюю неделю
	end := time.Now()
	start := end.Add(-7 * 24 * time.Hour)
	step := "1h" // Шаг для агрегации данных

	query := fmt.Sprintf("%s[7d]", metricName)
	stats, err := fetchPrometheusStats(ctx, client, query, start, end, step)
	if err != nil {
		log.Fatal("Ошибка при получении данных: ", err)
	}

	// Записываем результаты в файл
	if err := writeResultsToFile(stats); err != nil {
		log.Fatal("Ошибка при записи в файл: ", err)
	}

	fmt.Printf("Статистика сохранена в файл %s\n", outputFile)
}

func fetchPrometheusStats(
	ctx context.Context,
	client *http.Client,
	query string,
	start, end time.Time,
	step string,
) (map[string]map[string]map[string]*PoolStats, error) {
	// Формируем URL запроса
	params := url.Values{}
	params.Add("query", query)
	params.Add("start", start.Format(time.RFC3339))
	params.Add("end", end.Format(time.RFC3339))
	params.Add("step", step)

	reqURL := fmt.Sprintf("%s/api/v1/query_range?%s", prometheusURL, params.Encode())

	// Выполняем запрос
	resp, err := client.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result PrometheusResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Агрегируем статистику
	stats := make(map[string]map[string]map[string]*PoolStats)
	for _, series := range result.Data.Result {
		instance := series.Metric.Instance
		appName := series.Metric.AppName
		pool := series.Metric.Pool

		if _, exists := stats[instance]; !exists {
			stats[instance] = make(map[string]map[string]*PoolStats)
		}
		if _, exists := stats[instance][appName]; !exists {
			stats[instance][appName] = make(map[string]*PoolStats)
		}

		if _, exists := stats[instance][appName][pool]; !exists {
			stats[instance][appName][pool] = &PoolStats{}
		}

		// Обрабатываем все значения временного ряда
		for _, value := range series.Values {
			if len(value) < 2 {
				continue
			}

			strVal, ok := value[1].(string)
			if !ok {
				continue
			}

			val, err := strconv.ParseFloat(strVal, 64)
			if err != nil {
				continue
			}

			stats[instance][appName][pool].Update(val)
		}
	}

	return stats, nil
}

// PoolStats хранит статистику по пулу соединений
type PoolStats struct {
	Total float64
	Max   float64
}

func (ps *PoolStats) Update(value float64) {
	ps.Total = value // Используем последнее значение как общее количество
	if value > ps.Max {
		ps.Max = value
	}
}

func writeResultsToFile(stats map[string]map[string]map[string]*PoolStats) error {
	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	for instance, appMap := range stats {
		for appName, poolMap := range appMap {
			for pool, poolStats := range poolMap {
				line := fmt.Sprintf("%s|%s|%s|%.0f|%.0f\n",
					instance,
					appName,
					pool,
					poolStats.Total,
					poolStats.Max,
				)
				if _, err := file.WriteString(line); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
