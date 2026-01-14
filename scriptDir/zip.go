package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Парсинг флагов
	recursive := flag.Bool("r", false, "рекурсивно обходить директории")
	quiet := flag.Bool("q", false, "тихий режим (не выводить информацию)")
	help := flag.Bool("h", false, "показать справку")
	
	flag.Usage = func() {
		printHelp()
	}
	
	flag.Parse()
	
	// Проверяем флаг помощи
	if *help {
		printHelp()
		return
	}
	
	// Проверяем аргументы
	args := flag.Args()
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Ошибка: требуется указать имя zip файла и файлы для архивирования")
		fmt.Fprintln(os.Stderr, "Использование: zip [опции] archive.zip file1 file2 ...")
		os.Exit(1)
	}
	
	// Первый аргумент - имя архива
	zipName := args[0]
	
	// Проверяем расширение .zip
	if !strings.HasSuffix(strings.ToLower(zipName), ".zip") {
		zipName = zipName + ".zip"
	}
	
	// Остальные аргументы - файлы для архивирования
	filesToZip := args[1:]
	
	// Создаем архив
	err := createZip(zipName, filesToZip, *recursive, *quiet)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}
	
	if !*quiet {
		fmt.Printf("Архив создан: %s\n", zipName)
	}
}

func printHelp() {
	fmt.Println(`Использование: zip [опции] архив.zip файл1 файл2 ...
Создает ZIP архив из указанных файлов.

Опции:
  -r    рекурсивно обходить директории
  -q    тихий режим (не выводить информацию о процессе)
  -h    показать эту справку

Примеры:
  zip archive.zip file1.txt file2.txt
  zip -r archive.zip directory/
  zip archive.zip *.txt
  zip -q silent.zip file1 file2`)
}

func createZip(zipName string, files []string, recursive, quiet bool) error {
	// Создаем или открываем файл архива
	zipFile, err := os.Create(zipName)
	if err != nil {
		return fmt.Errorf("не удалось создать архив: %v", err)
	}
	defer zipFile.Close()
	
	// Создаем zip writer
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()
	
	// Собираем все файлы для архивации
	var allFiles []string
	for _, pattern := range files {
		matchedFiles, err := expandPattern(pattern, recursive)
		if err != nil {
			return err
		}
		allFiles = append(allFiles, matchedFiles...)
	}
	
	// Удаляем дубликаты
	allFiles = removeDuplicates(allFiles)
	
	if len(allFiles) == 0 {
		return fmt.Errorf("не найдено файлов для архивации")
	}
	
	// Архивируем каждый файл
	successCount := 0
	for _, file := range allFiles {
		err := addToZip(zipWriter, file, quiet)
		if err != nil {
			if !quiet {
				fmt.Fprintf(os.Stderr, "Предупреждение: не удалось добавить %s: %v\n", file, err)
			}
			continue
		}
		successCount++
	}
	
	if successCount == 0 {
		return fmt.Errorf("не удалось добавить ни одного файла в архив")
	}
	
	if !quiet {
		fmt.Printf("Добавлено файлов: %d\n", successCount)
	}
	
	return nil
}

func expandPattern(pattern string, recursive bool) ([]string, error) {
	// Если это обычный файл (не содержит wildcards)
	if !containsWildcard(pattern) {
		return []string{pattern}, nil
	}
	
	// Используем filepath.Glob для wildcards
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("ошибка при поиске файлов по шаблону %s: %v", pattern, err)
	}
	
	// Если нужна рекурсия, обходим директории
	if recursive {
		var result []string
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				continue
			}
			
			if info.IsDir() {
				// Рекурсивно обходим директорию
				dirFiles, err := walkDir(match)
				if err != nil {
					return nil, err
				}
				result = append(result, dirFiles...)
			} else {
				result = append(result, match)
			}
		}
		return result, nil
	}
	
	return matches, nil
}

func containsWildcard(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}

func walkDir(root string) ([]string, error) {
	var entries []string
	
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Добавляем и файлы и директории
		entries = append(entries, path)
		
		return nil
	})
	
	return entries, err
}

func addToZip(zipWriter *zip.Writer, filename string, quiet bool) error {
	// Получаем информацию о файле
	info, err := os.Stat(filename)
	if err != nil {
		return err
	}
	
	// Если это директория - создаем запись директории
	if info.IsDir() {
		return addDirectoryToZip(zipWriter, filename, info, quiet)
	}
	
	// Если это обычный файл
	return addFileToZip(zipWriter, filename, info, quiet)
}

func addFileToZip(zipWriter *zip.Writer, filename string, info os.FileInfo, quiet bool) error {
	// Открываем файл
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Создаем заголовок файла в архиве
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	
	// Устанавливаем имя файла в архиве (относительный путь)
	header.Name = filename
	
	// Устанавливаем метод сжатия (Deflate по умолчанию)
	header.Method = zip.Deflate
	
	// Создаем запись в архиве
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	
	// Копируем содержимое файла в архив
	_, err = io.Copy(writer, file)
	if err != nil {
		return err
	}
	
	if !quiet {
		fmt.Printf("  добавлен: %s\n", filename)
	}
	
	return nil
}

func addDirectoryToZip(zipWriter *zip.Writer, dirname string, info os.FileInfo, quiet bool) error {
	// Для директории создаем запись с / в конце
	header := &zip.FileHeader{
		Name:     dirname + "/",  // Директории должны заканчиваться на /
		Modified: info.ModTime(),
	}
	
	// Устанавливаем права доступа
	header.SetMode(info.Mode())
	
	// Метод сжатия для директорий всегда Store (без сжатия)
	header.Method = zip.Store
	
	// Создаем запись директории
	_, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	
	if !quiet {
		fmt.Printf("  добавлена директория: %s/\n", dirname)
	}
	
	return nil
}

func removeDuplicates(files []string) []string {
	seen := make(map[string]bool)
	var result []string
	
	for _, file := range files {
		// Преобразуем в абсолютный путь для устранения дубликатов
		absPath, err := filepath.Abs(file)
		if err != nil {
			absPath = file
		}
		
		if !seen[absPath] {
			seen[absPath] = true
			result = append(result, file)
		}
	}
	
	return result
}