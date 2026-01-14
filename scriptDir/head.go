package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	// Определение флагов
	nLines := flag.Int("n", 10, "количество строк для вывода")
	nBytes := flag.Int("c", 0, "количество байт для вывода")
	quietMode := flag.Bool("q", false, "не выводить заголовки с именами файлов")
	verboseMode := flag.Bool("v", false, "всегда выводить заголовки с именами файлов")
	
	// Переопределяем стандартное использование флагов
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Использование: %s [ПАРАМЕТР]... [ФАЙЛ]...
Выводит первые 10 строк каждого ФАЙЛа в стандартный вывод.
Если ФАЙЛ не указан, читает стандартный ввод.

Параметры:
`, os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Примеры:
  %s -n 20 file.txt       Вывести первые 20 строк file.txt
  %s -c 100 file.txt      Вывести первые 100 байт file.txt
  %s -q file1 file2       Вывести первые 10 строк двух файлов без заголовков
  %s -v file.txt          Вывести первые 10 строк с заголовком файла
  cat file.txt | %s -n 5  Вывести первые 5 строк из stdin
`, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
	}

	flag.Parse()

	// Получаем список файлов из аргументов
	files := flag.Args()

	// Если файлы не указаны, читаем из stdin
	if len(files) == 0 {
		if *nBytes > 0 {
			readBytes(os.Stdin, *nBytes, "")
		} else {
			readLines(os.Stdin, *nLines, "")
		}
		return
	}

	// Обрабатываем несколько файлов
	showHeader := len(files) > 1
	if *quietMode {
		showHeader = false
	}
	if *verboseMode {
		showHeader = true
	}

	for i, filename := range files {
		file, err := os.Open(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "head: невозможно открыть '%s': %v\n", filename, err)
			continue
		}
		defer file.Close()

		if showHeader {
			if i > 0 {
				fmt.Println()
			}
			fmt.Printf("==> %s <==\n", filename)
		}

		if *nBytes > 0 {
			readBytes(file, *nBytes, filename)
		} else {
			readLines(file, *nLines, filename)
		}
	}
}

// Функция для чтения N строк
func readLines(reader io.Reader, n int, filename string) {
	scanner := bufio.NewScanner(reader)
	linesRead := 0

	for scanner.Scan() {
		if linesRead >= n {
			break
		}
		fmt.Println(scanner.Text())
		linesRead++
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "head: ошибка чтения '%s': %v\n", filename, err)
	}
}

// Функция для чтения N байт
func readBytes(reader io.Reader, n int, filename string) {
	// Используем io.LimitReader для ограничения количества байт
	limitedReader := io.LimitReader(reader, int64(n))
	
	// Просто копируем данные
	_, err := io.Copy(os.Stdout, limitedReader)
	if err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "head: ошибка чтения '%s': %v\n", filename, err)
	}
}
