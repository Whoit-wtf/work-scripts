package main

import (
 "fmt"
 "io/ioutil"
 "os"
 "path/filepath"
 "regexp"
 "strings"
)

func main() {
 // Определяем корневую директорию для поиска.
 rootDir := "." // Текущая директория по умолчанию, можно изменить.

 // Определяем имя файла для вывода результатов.
 outputFile := "node_ids.txt"

 // Регулярное выражение для поиска "node-id:".
 nodeIDRegex := regexp.MustCompile(`node-id:\s*([a-zA-Z0-9-]+)`)

 // Открываем файл для записи (с очисткой предыдущего содержимого).
 outFile, err := os.Create(outputFile)
 if err != nil {
  fmt.Println("Ошибка при создании файла:", err)
  return
 }
 defer outFile.Close()

 // Функция для рекурсивного обхода директорий.
 err = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
  if err != nil {
   fmt.Println("Ошибка доступа к файлу/директории:", path, err)
   return err // Продолжаем обход, даже если есть ошибка доступа.
  }

  // Проверяем, что это обычный файл и у него расширение .yaml или .yml
  if !info.IsDir() && (strings.HasSuffix(strings.ToLower(info.Name()), ".yaml") || strings.HasSuffix(strings.ToLower(info.Name()), ".yml")) {
   // Читаем содержимое файла.
   content, err := ioutil.ReadFile(path)
   if err != nil {
    fmt.Println("Ошибка при чтении файла:", path, err)
    return nil // Продолжаем обход, даже если не удалось прочитать файл.
   }

   // Ищем node-id с помощью регулярного выражения.
   match := nodeIDRegex.FindSubmatch(content)
   if len(match) > 1 {
    nodeID := string(match[1])

    // Получаем имя родительской директории.
    dirName := filepath.Base(filepath.Dir(path))

    // Записываем результат в файл.
    outputLine := fmt.Sprintf("%s - %s\n", dirName, nodeID)
    _, err = outFile.WriteString(outputLine)
    if err != nil {
     fmt.Println("Ошибка при записи в файл:", outputFile, err)
     return nil // Продолжаем обход, даже если не удалось записать в файл.
    }

    fmt.Println("Найдено:", outputLine[:len(outputLine)-1], "в файле:", path) // Выводим найденные строки в консоль.
   }
  }

  return nil
 })

 if err != nil {
  fmt.Println("Ошибка при обходе директорий:", err)
 } else {
  fmt.Println("Поиск завершен. Результаты записаны в:", outputFile)
 }
}
