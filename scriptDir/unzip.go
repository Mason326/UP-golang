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
	list := flag.Bool("l", false, "показать содержимое архива без распаковки")
	quiet := flag.Bool("q", false, "тихий режим (не выводить информацию)")
	test := flag.Bool("t", false, "проверить целостность архива")
	overwrite := flag.Bool("o", false, "перезаписывать существующие файлы без запроса")
	dir := flag.String("d", "", "извлечь в указанную директорию")
	exclude := flag.String("x", "", "исключить файлы по шаблону")
	include := flag.String("i", "", "включать только файлы по шаблону")
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
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Ошибка: требуется указать архив для распаковки")
		fmt.Fprintln(os.Stderr, "Использование: unzip [опции] архив.zip")
		os.Exit(1)
	}
	
	// Имя архива
	zipFile := args[0]
	
	// Проверяем существование архива
	if _, err := os.Stat(zipFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Ошибка: архив '%s' не найден\n", zipFile)
		os.Exit(1)
	}
	
	// Выполняем действие в зависимости от флагов
	var err error
	switch {
	case *list:
		err = listArchive(zipFile, *quiet, *include, *exclude)
	case *test:
		err = testArchive(zipFile, *quiet)
	default:
		// Распаковка
		targetDir := *dir
		if targetDir == "" && len(args) > 1 {
			targetDir = args[1]
		}
		err = extractArchive(zipFile, targetDir, *quiet, *overwrite, *include, *exclude)
	}
	
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`Использование: unzip [опции] архив.zip [файлы...] [ -d директория ]
Распаковывает ZIP архив.

Опции:
  -l    показать содержимое архива (без распаковки)
  -q    тихий режим (не выводить информацию)
  -d    извлечь в указанную директорию
  -h    показать эту справку

Примеры:
  unzip archive.zip
  unzip -l archive.zip
  unzip -d /tmp archive.zip
`)
}

func listArchive(zipFile string, quiet bool, includePattern, excludePattern string) error {
	// Открываем архив
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return fmt.Errorf("не удалось открыть архив: %v", err)
	}
	defer r.Close()
	
	if !quiet {
		fmt.Printf("Архив:  %s\n", zipFile)
		fmt.Println(strings.Repeat("=", 60))
		fmt.Printf("%-12s %-12s %-8s %-20s %s\n", 
			"Длина", "Сжатый", "Метод", "Дата", "Имя")
		fmt.Println(strings.Repeat("-", 60))
	}
	
	var totalFiles int
	var totalSize, totalCompressed uint64
	
	for _, f := range r.File {
		// Проверяем фильтры
		if !shouldProcess(f.Name, includePattern, excludePattern) {
			continue
		}
		
		if !quiet {
			// Определяем метод сжатия
			method := "Store"
			if f.Method == zip.Deflate {
				method = "Deflate"
			}
			
			// Форматируем дату
			date := f.Modified.Format("2006-01-02 15:04")
			
			fmt.Printf("%-12d %-12d %-8s %-20s %s\n",
				f.UncompressedSize64,
				f.CompressedSize64,
				method,
				date,
				f.Name)
		}
		
		totalFiles++
		totalSize += f.UncompressedSize64
		totalCompressed += f.CompressedSize64
	}
	
	if !quiet {
		fmt.Println(strings.Repeat("-", 60))
		fmt.Printf("Итого: %d файлов, %s (%s сжато)\n", 
			totalFiles,
			formatBytes(totalSize),
			formatBytes(totalCompressed))
	}
	
	return nil
}

func testArchive(zipFile string, quiet bool) error {
	// Открываем архив
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return fmt.Errorf("не удалось открыть архив: %v", err)
	}
	defer r.Close()
	
	if !quiet {
		fmt.Printf("Проверка архива: %s\n", zipFile)
	}
	
	checkedFiles := 0
	errorsFound := 0
	
	for _, f := range r.File {
		// Открываем файл в архиве
		rc, err := f.Open()
		if err != nil {
			if !quiet {
				fmt.Printf("Ошибка: не удалось открыть %s: %v\n", f.Name, err)
			}
			errorsFound++
			continue
		}
		
		// Читаем файл для проверки
		_, err = io.Copy(io.Discard, rc)
		rc.Close()
		
		if err != nil {
			if !quiet {
				fmt.Printf("Ошибка: поврежден %s: %v\n", f.Name, err)
			}
			errorsFound++
		} else {
			if !quiet {
				fmt.Printf("  OK: %s\n", f.Name)
			}
			checkedFiles++
		}
	}
	
	if !quiet {
		fmt.Println(strings.Repeat("-", 40))
		if errorsFound == 0 {
			fmt.Printf("✓ Проверка завершена успешно: %d файлов\n", checkedFiles)
		} else {
			fmt.Printf("✗ Найдено ошибок: %d из %d файлов\n", errorsFound, checkedFiles+errorsFound)
		}
	}
	
	if errorsFound > 0 {
		return fmt.Errorf("архив поврежден: %d ошибок", errorsFound)
	}
	
	return nil
}

func extractArchive(zipFile, targetDir string, quiet, overwrite bool, includePattern, excludePattern string) error {
	// Определяем целевую директорию
	if targetDir == "" {
		targetDir = "."
	}
	
	// Создаем целевую директорию если нужно
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("не удалось создать директорию %s: %v", targetDir, err)
	}
	
	// Открываем архив
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return fmt.Errorf("не удалось открыть архив: %v", err)
	}
	defer r.Close()
	
	if !quiet {
		fmt.Printf("Распаковка архива: %s\n", zipFile)
		fmt.Printf("В директорию: %s\n", targetDir)
		fmt.Println(strings.Repeat("-", 40))
	}
	
	extractedFiles := 0
	skippedFiles := 0
	
	for _, f := range r.File {
		// Проверяем фильтры
		if !shouldProcess(f.Name, includePattern, excludePattern) {
			if !quiet {
				fmt.Printf("  пропущен (фильтр): %s\n", f.Name)
			}
			skippedFiles++
			continue
		}
		
		// Извлекаем файл
		err := extractFile(f, targetDir, overwrite, quiet)
		if err != nil {
			if !quiet {
				fmt.Printf("Ошибка: %s: %v\n", f.Name, err)
			}
			continue
		}
		
		extractedFiles++
	}
	
	if !quiet {
		fmt.Println(strings.Repeat("-", 40))
		fmt.Printf("Извлечено файлов: %d\n", extractedFiles)
		if skippedFiles > 0 {
			fmt.Printf("Пропущено (фильтр): %d\n", skippedFiles)
		}
	}
	
	if extractedFiles == 0 {
		return fmt.Errorf("не извлечено ни одного файла")
	}
	
	return nil
}

func extractFile(f *zip.File, targetDir string, overwrite, quiet bool) error {
	// Создаем полный путь
	path := filepath.Join(targetDir, f.Name)
	
	// Проверяем, является ли это директорией
	if f.FileInfo().IsDir() {
		// Создаем директорию
		return os.MkdirAll(path, f.Mode())
	}
	
	// Проверяем, существует ли уже файл
	if !overwrite {
		if _, err := os.Stat(path); err == nil {
			if !quiet {
				fmt.Printf("  пропущен (существует): %s\n", f.Name)
			}
			return nil
		}
	}
	
	// Создаем родительские директории если нужно
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("не удалось создать директорию %s: %v", dir, err)
	}
	
	// Открываем файл в архиве
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("не удалось открыть в архиве: %v", err)
	}
	defer rc.Close()
	
	// Создаем файл на диске
	outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return fmt.Errorf("не удалось создать файл %s: %v", path, err)
	}
	defer outFile.Close()
	
	// Копируем данные
	_, err = io.Copy(outFile, rc)
	if err != nil {
		return fmt.Errorf("ошибка копирования %s: %v", f.Name, err)
	}
	
	if !quiet {
		fmt.Printf("  извлечен: %s\n", f.Name)
	}
	
	return nil
}

func shouldProcess(filename, includePattern, excludePattern string) bool {
	// Получаем только имя файла (без пути) для проверки паттернов
	baseName := filepath.Base(filename)
	
	// Проверяем паттерн включения
	if includePattern != "" {
		matched, err := filepath.Match(includePattern, baseName)
		if err != nil || !matched {
			return false
		}
	}
	
	// Проверяем паттерн исключения
	if excludePattern != "" {
		matched, err := filepath.Match(excludePattern, baseName)
		if err == nil && matched {
			return false
		}
	}
	
	return true
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
