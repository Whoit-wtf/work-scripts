package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

func main() {
	// Параметры подключения
	username := "your_username"
	privateKeyPath := "/path/to/private/key"
	serverListFile := "servers.txt"
	localBaseDir := "./downloads"

	// Чтение приватного ключа
	privateKeyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		log.Fatalf("Failed to read private key: %v", err)
	}

	// Парсинг приватного ключа
	signer, err := ssh.ParsePrivateKey(privateKeyBytes)
	if err != nil {
		log.Fatalf("Failed to parse private key: %v", err)
	}

	// Конфигурация SSH
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	// Чтение списка серверов
	servers, err := readLines(serverListFile)
	if err != nil {
		log.Fatalf("Failed to read server list: %v", err)
	}

	// Обработка каждого сервера
	for _, server := range servers {
		server = strings.TrimSpace(server)
		if server == "" {
			continue
		}

		log.Printf("Connecting to %s...", server)
		err := processServer(server, sshConfig, localBaseDir)
		if err != nil {
			log.Printf("Error processing %s: %v", server, err)
		}
	}
}

func processServer(server string, config *ssh.ClientConfig, localBaseDir string) error {
	// Подключение по SSH
	conn, err := ssh.Dial("tcp", server+":22", config)
	if err != nil {
		return fmt.Errorf("SSH connection failed: %w", err)
	}
	defer conn.Close()

	// Получение списка приложений
	apps, err := listApplications(conn)
	if err != nil {
		return fmt.Errorf("failed to list applications: %w", err)
	}

	// Обработка каждого приложения
	for _, app := range apps {
		remotePath := fmt.Sprintf("/opt/solar/%s/config/context.xml", app)
		localDir := filepath.Join(localBaseDir, server, app)
		localPath := filepath.Join(localDir, "context.xml")

		// Скачивание файла с помощью cat
		if err := downloadViaCat(conn, remotePath, localPath); err != nil {
			log.Printf("  [%s] Error: %v", app, err)
		} else {
			log.Printf("  [%s] Downloaded successfully", app)
		}
	}

	return nil
}

func listApplications(conn *ssh.Client) ([]string, error) {
	session, err := conn.NewSession()
	if err != nil {
		return nil, fmt.Errorf("session creation failed: %w", err)
	}
	defer session.Close()

	// Выполнение команды для получения списка приложений
	output, err := session.Output("ls -1 /opt/solar")
	if err != nil {
		return nil, fmt.Errorf("command failed: %w", err)
	}

	var apps []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		app := strings.TrimSpace(scanner.Text())
		if app != "" {
			apps = append(apps, app)
		}
	}

	return apps, nil
}

func downloadViaCat(conn *ssh.Client, remotePath, localPath string) error {
	// Создание локальной директории
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("failed to create local directory: %w", err)
	}

	// Создание файла для записи
	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Создание SSH-сессии
	session, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("session creation failed: %w", err)
	}
	defer session.Close()

	// Перенаправление вывода команды в файл
	session.Stdout = file

	// Экранирование пути для безопасной передачи
	escapedPath := escapeShellArg(remotePath)
	cmd := fmt.Sprintf("cat %s", escapedPath)

	// Выполнение команды
	if err := session.Run(cmd); err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}

	return nil
}

func escapeShellArg(s string) string {
	return "'" + strings.Replace(s, "'", "'\"'\"'", -1) + "'"
}

func readLines(filename string) ([]string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	var result []string
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result, nil
}