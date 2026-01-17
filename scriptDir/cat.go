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
	showAll := flag.Bool("A", false, "показать все непечатаемые символы")
	numberNonblank := flag.Bool("b", false, "нумеровать только непустые строки")
	showEnds := flag.Bool("E", false, "выводить $ в конце каждой строки")
	numberLines := flag.Bool("n", false, "нумеровать все строки")
	helpFlag := flag.Bool("help", false, "показать справку")

	flag.Parse()

	// Если запрошена справка
	if *helpFlag {
		printHelp()
		return
	}

	// Получаем список файлов из аргументов
	files := flag.Args()

	// Если файлы не указаны, читаем из stdin
	if len(files) == 0 {
		processFile(os.Stdin, "", *numberNonblank, *numberLines, *showAll, *showEnds)
		return
	}

	// Обрабатываем несколько файлов
	for _, filename := range files {
		if filename == "-" {
			// Специальный случай: читать из stdin
			processFile(os.Stdin, "", *numberNonblank, *numberLines, *showAll, *showEnds)
			continue
		}

		file, err := os.Open(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cat: %s: %v\n", filename, err)
			continue
		}
		defer file.Close()

		processFile(file, filename, *numberNonblank, *numberLines, *showAll, *showEnds)
	}
}

// Обработка одного файла
func processFile(reader io.Reader, filename string, numberNonblank, numberLines, showAll, showEnds bool) {
	scanner := bufio.NewScanner(reader)
	lineNumber := 0

	for scanner.Scan() {
		line := scanner.Bytes() // Используем Bytes для обработки непечатаемых символов

		// Нумерация строк
		shouldNumber := false
		prefix := ""

		// Проверяем, пустая ли строка
		isEmpty := len(line) == 0

		if numberNonblank && !isEmpty {
			lineNumber++
			shouldNumber = true
		} else if numberLines {
			lineNumber++
			shouldNumber = true
		}

		if shouldNumber {
			prefix = fmt.Sprintf("%6d\t", lineNumber)
		}

		// Вывод строки с учетом флагов
		fmt.Print(prefix)
		printLine(line, showAll, showEnds)
		fmt.Println() // Новая строка после каждой строки
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "cat: ошибка чтения '%s': %v\n", filename, err)
	}
}

// Вывод строки с учетом флагов
func printLine(line []byte, showAll, showEnds bool) {
	if showAll {
		// Показать все непечатаемые символы
		for _, b := range line {
			switch {
			case b == '\t':
				fmt.Print("^I")
			case b < 32:
				// Управляющие символы
				fmt.Printf("^%c", b+64)
			case b == 127:
				// Символ DEL
				fmt.Print("^?")
			case b > 127:
				// Расширенные символы
				fmt.Printf("M-^%c", b-128+64)
			default:
				fmt.Printf("%c", b)
			}
		}
		if showEnds {
			fmt.Print("$")
		}
	} else if showEnds {
		// Только показывать $ в конце строки
		fmt.Print(string(line))
		fmt.Print("$")
	} else {
		// Простой вывод
		fmt.Print(string(line))
	}
}

// Функция для вывода справки
func printHelp() {
	helpText := `Использование: cat [ПАРАМЕТР]... [ФАЙЛ]...
Объединяет файлы и выводит их содержимое в стандартный вывод.
Если ФАЙЛ не указан или указан как -, читает стандартный ввод.

Параметры:
  -A           показать все непечатаемые символы
  -b           нумеровать только непустые строки
  -E           выводить $ в конце каждой строки
  -n           нумеровать все строки
  -h, -help    показать эту справку и выйти

Примеры:
  cat file.txt                    Вывести содержимое файла
  cat -n file.txt                 Вывести с номерами строк
  cat -b file.txt                 Нумеровать только непустые строки
  cat -E file.txt                 Показать концы строк
  cat -A file.txt                 Показать все непечатаемые символы
`

	fmt.Println(helpText)
}
