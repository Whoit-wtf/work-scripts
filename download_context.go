package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func main() {
	// Параметры подключения (можно заменить на чтение из аргументов или конфига)
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
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Внимание: небезопасно для продакшена!
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

	// Создание SFTP-клиента
	sftpClient, err := sftp.NewClient(conn)
	if err != nil {
		return fmt.Errorf("SFTP client creation failed: %w", err)
	}
	defer sftpClient.Close()

	// Получение списка приложений
	appsDir := "/opt/solar"
	appDirs, err := sftpClient.ReadDir(appsDir)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", appsDir, err)
	}

	// Обработка каждого приложения
	for _, appDir := range appDirs {
		if !appDir.IsDir() {
			continue
		}

		appName := appDir.Name()
		remoteFile := fmt.Sprintf("%s/%s/config/context.xml", appsDir, appName)
		localDir := filepath.Join(localBaseDir, server, appName)
		localFile := filepath.Join(localDir, "context.xml")

		// Скачивание файла
		if err := downloadFile(sftpClient, remoteFile, localFile); err != nil {
			log.Printf("  [%s] Error: %v", appName, err)
		} else {
			log.Printf("  [%s] Downloaded successfully", appName)
		}
	}

	return nil
}

func downloadFile(sftpClient *sftp.Client, remotePath, localPath string) error {
	// Создание локальной директории
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("failed to create local directory: %w", err)
	}

	// Открытие удаленного файла
	srcFile, err := sftpClient.Open(remotePath)
	if err != nil {
		return fmt.Errorf("failed to open remote file: %w", err)
	}
	defer srcFile.Close()

	// Создание локального файла
	dstFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer dstFile.Close()

	// Копирование содержимого
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("copy failed: %w", err)
	}

	return nil
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