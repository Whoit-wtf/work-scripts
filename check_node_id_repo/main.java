import java.io.IOException;
import java.nio.file.*;
import java.nio.file.attribute.BasicFileAttributes;
import java.io.BufferedWriter;
import java.io.FileWriter;
import java.util.regex.Matcher;
import java.util.regex.Pattern;
import java.util.stream.Stream;

public class FindNodeIds {

    public static void main(String[] args) {
        // Определяем корневую директорию для поиска.
        String rootDir = "."; // Текущая директория по умолчанию

        // Определяем имя файла для вывода результатов.
        String outputFile = "node_ids.txt";

        // Регулярное выражение для поиска "node-id:".
        Pattern nodeIDRegex = Pattern.compile("node-id:\\s*([a-zA-Z0-9-]+)");

        try (BufferedWriter writer = new BufferedWriter(new FileWriter(outputFile))) {
            Files.walkFileTree(Paths.get(rootDir), new SimpleFileVisitor<Path>() {
                @Override
                public FileVisitResult visitFile(Path file, BasicFileAttributes attrs) throws IOException {
                    // Проверяем, что это обычный файл и у него расширение .yaml или .yml
                    if (!Files.isDirectory(file) && (file.toString().toLowerCase().endsWith(".yaml") || file.toString().toLowerCase().endsWith(".yml"))) {
                        try {
                            String content = Files.readString(file);
                            Matcher matcher = nodeIDRegex.matcher(content);

                            if (matcher.find()) {
                                String nodeID = matcher.group(1);

                                // Получаем имя родительской директории.
                                String dirName = file.getParent().getFileName().toString();

                                // Записываем результат в файл.
                                String outputLine = String.format("%s - %s%n", dirName, nodeID);
                                writer.write(outputLine);
                                System.out.println("Найдено: " + outputLine.trim() + " в файле: " + file); // Выводим найденные строки в консоль.
                            }
                        } catch (IOException e) {
                            System.err.println("Ошибка при чтении файла: " + file + " - " + e.getMessage());
                            // Продолжаем обход, даже если не удалось прочитать файл.
                        }
                    }
                    return FileVisitResult.CONTINUE;
                }

                @Override
                public FileVisitResult visitFileFailed(Path file, IOException exc) throws IOException {
                    System.err.println("Ошибка доступа к файлу/директории: " + file + " - " + exc.getMessage());
                    // Продолжаем обход, даже если есть ошибка доступа.
                    return FileVisitResult.CONTINUE;
                }
            });
            System.out.println("Поиск завершен. Результаты записаны в: " + outputFile);

        } catch (IOException e) {
            System.err.println("Ошибка при создании/записи в файл: " + outputFile + " - " + e.getMessage());
        }
    }
}
