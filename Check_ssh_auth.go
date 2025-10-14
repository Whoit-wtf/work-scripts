package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

// Config представляет структуру для хранения параметров подключения
type SSHAuthConfig struct {
	Username   string
	PrivateKey string
	Passphrase string // Опционально, для зашифрованных ключей
	Timeout    time.Duration
}

// SSHCheckResult представляет результат проверки SSH
type SSHCheckResult struct {
	Server   string
	Success  bool
	Error    string
	Duration time.Duration
}

// readServerList читает файл со списком серверов
func readServerList(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var servers []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		server := scanner.Text()
		if server != "" {
			servers = append(servers, server)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return servers, nil
}

// parsePrivateKey читает и парсит приватный ключ
func parsePrivateKey(keyPath string, passphrase []byte) (ssh.Signer, error) {
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать файл ключа: %v", err)
	}

	if len(passphrase) > 0 {
		return ssh.ParsePrivateKeyWithPassphrase(keyData, passphrase)
	}

	return ssh.ParsePrivateKey(keyData)
}

// checkSSHAuth проверяет аутентификацию по SSH на сервере
func checkSSHAuth(server string, config *SSHAuthConfig) SSHCheckResult {
	startTime := time.Now()

	// Чтение и парсинг приватного ключа
	var passphrase []byte
	if config.Passphrase != "" {
		passphrase = []byte(config.Passphrase)
	}

	signer, err := parsePrivateKey(config.PrivateKey, passphrase)
	if err != nil {
		return SSHCheckResult{
			Server:  server,
			Success: false,
			Error:   fmt.Sprintf("Ошибка парсинга ключа: %v", err),
		}
	}

	// Конфигурация SSH клиента
	sshConfig := &ssh.ClientConfig{
		User: config.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Внимание: в продакшене используйте валидацию
		Timeout:         config.Timeout,
	}

	// Попытка подключения
	address := net.JoinHostPort(server, "22")
	client, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		return SSHCheckResult{
			Server:  server,
			Success: false,
			Error:   fmt.Sprintf("Ошибка подключения: %v", err),
		}
	}
	defer client.Close()

	// Проверка аутентификации путем создания сессии
	session, err := client.NewSession()
	if err != nil {
		return SSHCheckResult{
			Server:  server,
			Success: false,
			Error:   fmt.Sprintf("Ошибка создания сессии: %v", err),
		}
	}
	session.Close()

	duration := time.Since(startTime)
	return SSHCheckResult{
		Server:   server,
		Success:  true,
		Error:    "",
		Duration: duration,
	}
}

func main() {
	// Конфигурация (можно вынести в аргументы командной строки или конфиг файл)
	config := &SSHAuthConfig{
		Username:   "your-username",        // Замените на ваше имя пользователя
		PrivateKey: "/path/to/private/key", // Замените на путь к вашему приватному ключу
		Timeout:    10 * time.Second,
	}

	// Чтение списка серверов
	servers, err := readServerList("servers.txt") // Замените на ваш файл со списком серверов
	if err != nil {
		log.Fatalf("Ошибка чтения файла со списком серверов: %v", err)
	}

	fmt.Printf("Проверка SSH доступности для %d серверов...\n", len(servers))
	fmt.Println("==============================================")

	// Проверка каждого сервера
	for _, server := range servers {
		result := checkSSHAuth(server, config)

		if result.Success {
			fmt.Printf("✅ %s: Успешная аутентификация (время: %v)\n",
				result.Server, result.Duration)
		} else {
			fmt.Printf("❌ %s: Ошибка - %s\n",
				result.Server, result.Error)
		}
	}
}
