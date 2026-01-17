package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	// Парсинг аргументов
	width := flag.Int("w", 6, "width of line numbers")
	start := flag.Int("s", 1, "start numbering at")
	inc := flag.Int("i", 1, "increment line numbers by")
	separator := flag.String("sep", "\t", "separator between number and line")
	numberAll := flag.Bool("a", false, "number all lines (including blanks)")
	help := flag.Bool("h", false, "show help")
	
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Использование: nl [опции] [файл...]\n")
		fmt.Fprintf(os.Stderr, "Нумерует строки файлов.\n\n")
		fmt.Fprintf(os.Stderr, "Опции:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nПримеры:\n")
		fmt.Fprintf(os.Stderr, "  nl file.txt              # Нумеровать строки file.txt\n")
		fmt.Fprintf(os.Stderr, "  nl -a file.txt          # Нумеровать все строки\n")
		fmt.Fprintf(os.Stderr, "  nl -s 10 -i 2 file.txt  # Начать с 10, шаг 2\n")
		fmt.Fprintf(os.Stderr, "  cat file.txt | nl       # Нумеровать вывод команды\n")
	}
	
	flag.Parse()
	
	if *help {
		flag.Usage()
		return
	}
	
	// Получаем файлы для обработки
	files := flag.Args()
	if len(files) == 0 {
		// Чтение из stdin
		numberLines(os.Stdin, *width, *start, *inc, *separator, *numberAll, "stdin")
	} else {
		// Обработка каждого файла
		for _, filename := range files {
			if filename == "-" {
				numberLines(os.Stdin, *width, *start, *inc, *separator, *numberAll, "stdin")
			} else {
				file, err := os.Open(filename)
				if err != nil {
					fmt.Fprintf(os.Stderr, "nl: не удалось открыть файл %s: %v\n", filename, err)
					continue
				}
				numberLines(file, *width, *start, *inc, *separator, *numberAll, filename)
				file.Close()
			}
		}
	}
}

func numberLines(reader io.Reader, width, start, inc int, separator string, numberAll bool, source string) {
	scanner := bufio.NewScanner(reader)
	lineNum := start
	
	for scanner.Scan() {
		line := scanner.Text()
		
		// Решаем, нумеровать ли эту строку
		shouldNumber := numberAll || strings.TrimSpace(line) != ""
		
		if shouldNumber {
			// Форматируем номер с заданной шириной
			numStr := fmt.Sprintf("%*d", width, lineNum)
			fmt.Printf("%s%s%s\n", numStr, separator, line)
			lineNum += inc
		} else {
			// Пустая строка без номера
			indent := strings.Repeat(" ", width)
			fmt.Printf("%s%s\n", indent, line)
		}
	}
	
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "nl: ошибка чтения %s: %v\n", source, err)
	}
}