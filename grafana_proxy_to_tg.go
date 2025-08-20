package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// Конфигурация
var (
	telegramToken   = os.Getenv("TELEGRAM_TOKEN")
	listenPort      string
	targetWebhook   string
	maxMessageChars = 4096
)

// Структура для парсинга Grafana Alert
type GrafanaAlert struct {
	Title       string `json:"title"`
	Message     string `json:"message"`
	RuleName    string `json:"ruleName"`
	RuleURL     string `json:"ruleUrl"`
	State       string `json:"state"`
	EvalMatches []struct {
		Value  float64 `json:"value"`
		Metric string  `json:"metric"`
	} `json:"evalMatches"`
}

// Функция для отправки сообщения в Telegram
func sendToTelegram(chatID, text string) error {
	formData := url.Values{
		"chat_id": {chatID},
		"text":    {text},
	}

	resp, err := http.PostForm(targetWebhook, formData)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP error: %s, response: %s", resp.Status, string(body))
	}
	return nil
}

// Разбивка длинного текста на части
func splitLongMessage(text string, maxLength int) []string {
	var parts []string
	lines := strings.Split(text, "\n")
	currentPart := ""

	for _, line := range lines {
		if len(currentPart)+len(line)+1 > maxLength {
			if currentPart != "" {
				parts = append(parts, currentPart)
			}
			currentPart = line
		} else {
			if currentPart != "" {
				currentPart += "\n"
			}
			currentPart += line
		}
	}
	if currentPart != "" {
		parts = append(parts, currentPart)
	}
	return parts
}

// Обработчик входящих webhook-запросов
func grafanaWebhookHandler(chatID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading body", http.StatusInternalServerError)
			return
		}

		var alert GrafanaAlert
		if err := json.Unmarshal(body, &alert); err != nil {
			http.Error(w, "Error parsing JSON", http.StatusBadRequest)
			return
		}

		// Форматирование сообщения
		var message bytes.Buffer
		message.WriteString(fmt.Sprintf("⚠️ *%s*\n", alert.Title))
		message.WriteString(fmt.Sprintf("State: %s\n", alert.State))
		message.WriteString(fmt.Sprintf("Rule: %s\n", alert.RuleName))
		
		if alert.RuleURL != "" {
			message.WriteString(fmt.Sprintf("URL: %s\n", alert.RuleURL))
		}
		
		message.WriteString(fmt.Sprintf("\n%s\n", alert.Message))
		
		if len(alert.EvalMatches) > 0 {
			message.WriteString("\n*Metrics:*\n")
			for _, match := range alert.EvalMatches {
				message.WriteString(fmt.Sprintf("- %s: %.2f\n", match.Metric, match.Value))
			}
		}

		// Разбивка и отправка сообщений
		parts := splitLongMessage(message.String(), maxMessageChars)
		for i, part := range parts {
			if len(parts) > 1 {
				part = fmt.Sprintf("(%d/%d)\n%s", i+1, len(parts), part)
			}
			
			if err := sendToTelegram(chatID, part); err != nil {
				log.Printf("Error sending to Telegram: %v", err)
				http.Error(w, "Error sending message", http.StatusInternalServerError)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Processed"))
	}
}

func main() {
	// Обработка параметров командной строки
	var chatID string
	flag.StringVar(&chatID, "chat-id", "", "Telegram Chat ID (required)")
	flag.StringVar(&listenPort, "port", "8080", "Port to listen on")
	flag.Parse()

	// Проверка обязательных параметров
	if telegramToken == "" {
		log.Fatal("TELEGRAM_TOKEN environment variable must be set")
	}
	if chatID == "" {
		log.Fatal("chat-id parameter is required")
	}

	// Формирование URL вебхука Telegram
	targetWebhook = fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", telegramToken)

	// Настройка HTTP сервера
	http.HandleFunc("/webhook", grafanaWebhookHandler(chatID))
	log.Printf("Server listening on port %s", listenPort)
	log.Printf("Forwarding alerts to chat ID: %s", chatID)
	log.Fatal(http.ListenAndServe(":"+listenPort, nil))
}