package main

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

func main() {
	// Парсинг аргументов командной строки
	host := flag.String("host", "", "Database host (required)")
	port := flag.String("port", "", "Database port (required)")
	dbName := flag.String("db", "", "Database name (required)")
	user := flag.String("user", "", "Database user")
	inputFile := flag.String("i", "tables.txt", "Input file with table list")
	outputFile := flag.String("o", "output.txt", "Output file for results")
	flag.Parse()

	// Получаем пароль из переменных окружения
	password := os.Getenv("PGPASSWORD")

	// Проверка обязательных параметров
	if *host == "" || *port == "" || *dbName == "" {
		log.Fatal("host, port and db parameters are required")
	}

	// Если пользователь не указан, используем значение по умолчанию
	if *user == "" {
		if envUser := os.Getenv("PGUSER"); envUser != "" {
			*user = envUser
		} else {
			*user = "postgres"
		}
	}

	// Формируем строку подключения
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		*host, *port, *user, password, *dbName,
	)

	// Подключаемся к базе данных
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("connection failed: %v", err)
	}
	defer db.Close()

	// Проверяем соединение
	if err := db.Ping(); err != nil {
		log.Fatalf("ping failed: %v", err)
	}

	// Открываем входной файл с таблицами
	inFile, err := os.Open(*inputFile)
	if err != nil {
		log.Fatalf("failed to open input file: %v", err)
	}
	defer inFile.Close()

	// Создаем выходной файл
	outFile, err := os.Create(*outputFile)
	if err != nil {
		log.Fatalf("failed to create output file: %v", err)
	}
	defer outFile.Close()

	writer := bufio.NewWriter(outFile)
	defer writer.Flush()

	// Обрабатываем каждую таблицу из входного файла
	scanner := bufio.NewScanner(inFile)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		tableName := strings.TrimSpace(scanner.Text())
		
		// Пропускаем пустые строки и комментарии
		if tableName == "" || strings.HasPrefix(tableName, "#") {
			continue
		}

		// Получаем первичный ключ для таблицы
		pkField, err := getPrimaryKey(db, tableName)
		if err != nil {
			log.Printf("%s: %v", tableName, err)
			fmt.Fprintf(writer, "%s ERROR: %v\n", tableName, err)
		} else {
			fmt.Fprintf(writer, "%s %s\n", tableName, pkField)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("error reading input file: %v", err)
	}
}

// Получение первого поля первичного ключа для таблицы
func getPrimaryKey(db *sql.DB, tableName string) (string, error) {
	// SQL-запрос для получения первого поля первичного ключа
	query := `
	SELECT a.attname
	FROM pg_index i
	JOIN pg_attribute a 
		ON a.attrelid = i.indrelid 
		AND a.attnum = i.indkey[0]
	WHERE i.indrelid = $1::regclass
	AND i.indisprimary
	LIMIT 1;
	`

	var pkField string
	err := db.QueryRow(query, tableName).Scan(&pkField)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("no primary key found for table '%s'", tableName)
		}
		return "", fmt.Errorf("query failed: %w", err)
	}

	return pkField, nil
}