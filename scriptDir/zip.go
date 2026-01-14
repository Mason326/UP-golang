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
	exclude := flag.String("e", "", "исключить файлы по шаблону (например, *.tmp)")

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
	err := createZip(zipName, filesToZip, *recursive, *quiet, *exclude)
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
  -e    исключить файлы по шаблону (например: *.tmp, *.log)
  -h    показать эту справку

Примеры:
  zip archive.zip file1.txt file2.txt
  zip -r archive.zip directory/
  zip -e "*.tmp" archive.zip *.txt
  zip -r -e "*.log" archive.zip logs/
  zip -q silent.zip file1 file2`)
}

func createZip(zipName string, files []string, recursive, quiet bool, excludePattern string) error {
	zipFile, err := os.Create(zipName)
	if err != nil {
		return fmt.Errorf("не удалось создать архив: %v", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	successCount := 0
	skippedCount := 0

	// Для каждого файла/директории
	for _, item := range files {
		info, err := os.Stat(item)
		if err != nil {
			if !quiet {
				fmt.Fprintf(os.Stderr, "Предупреждение: %s: %v\n", item, err)
			}
			continue
		}

		if info.IsDir() && recursive {
			// Рекурсивный обход директории
			err = filepath.Walk(item, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				// Проверяем, не нужно ли исключить этот файл
				if shouldExclude(path, excludePattern) {
					if !quiet {
						fmt.Printf("  пропущен (исключен): %s\n", path)
					}
					skippedCount++
					return nil
				}

				// Добавляем в архив
				err = addToZip(zipWriter, path, quiet)
				if err != nil {
					if !quiet {
						fmt.Fprintf(os.Stderr, "Предупреждение: %s: %v\n", path, err)
					}
					return nil // Продолжаем обход
				}

				successCount++
				return nil
			})

			if err != nil && !quiet {
				fmt.Fprintf(os.Stderr, "Предупреждение: ошибка обхода %s: %v\n", item, err)
			}
		} else {
			// Проверяем, не нужно ли исключить этот файл
			if shouldExclude(item, excludePattern) {
				if !quiet {
					fmt.Printf("  пропущен (исключен): %s\n", item)
				}
				skippedCount++
				continue
			}

			// Простой файл или директория без рекурсии
			err = addToZip(zipWriter, item, quiet)
			if err != nil {
				if !quiet {
					fmt.Fprintf(os.Stderr, "Предупреждение: %s: %v\n", item, err)
				}
				continue
			}
			successCount++
		}
	}

	if successCount == 0 {
		return fmt.Errorf("не удалось добавить ни одного файла в архив")
	}

	if !quiet {
		fmt.Printf("Добавлено элементов: %d\n", successCount)
		if skippedCount > 0 {
			fmt.Printf("Пропущено (исключено): %d\n", skippedCount)
		}
	}

	return nil
}

// shouldExclude проверяет, нужно ли исключить файл по шаблону
func shouldExclude(path string, pattern string) bool {
	if pattern == "" {
		return false
	}

	// Получаем только имя файла (без пути)
	filename := filepath.Base(path)

	// Проверяем соответствие шаблону
	matched, err := filepath.Match(pattern, filename)
	if err != nil {
		// Если ошибка в шаблоне, просто не исключаем
		return false
	}

	return matched
}

func addToZip(zipWriter *zip.Writer, path string, quiet bool) error {
	// Получаем информацию о файле/директории
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	// Если это директория
	if info.IsDir() {
		return addDirectoryToZip(zipWriter, path, info, quiet)
	}

	// Если это обычный файл
	return addFileToZip(zipWriter, path, info, quiet)
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

	// Устанавливаем имя файла в архиве
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
		fmt.Printf("  добавлен файл: %s\n", filename)
	}

	return nil
}

func addDirectoryToZip(zipWriter *zip.Writer, dirname string, info os.FileInfo, quiet bool) error {
	// Для директории создаем запись с / в конце
	header := &zip.FileHeader{
		Name:     dirname + "/", // Директории должны заканчиваться на /
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
