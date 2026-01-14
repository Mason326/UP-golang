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
	// Определение флагов
	countLines := flag.Bool("l", false, "count lines")
	countWords := flag.Bool("w", false, "count words")
	countBytes := flag.Bool("c", false, "count bytes")
	countChars := flag.Bool("m", false, "count characters")
	helpFlag := flag.Bool("help", false, "show help")
	
	flag.Parse()

	if *helpFlag {
		printSimpleHelp()
		return
	}

	// Если не указано никаких флагов, показываем всё
	showAll := !*countLines && !*countWords && !*countBytes && !*countChars

	args := flag.Args()

	if len(args) == 0 {
		// Чтение из stdin
		countStdin(showAll, *countLines, *countWords, *countBytes, *countChars)
	} else {
		// Обработка файлов
		processFiles(args, showAll, *countLines, *countWords, *countBytes, *countChars)
	}
}

func countStdin(showAll, countLines, countWords, countBytes, countChars bool) {
	// Читаем весь stdin
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wc: error reading stdin: %v\n", err)
		os.Exit(1)
	}

	counts := countData(data, "")
	printCountsSimple(counts, "", showAll, countLines, countWords, countBytes, countChars)
}

func processFiles(files []string, showAll, countLines, countWords, countBytes, countChars bool) {
	total := &FileCounts{}
	fileCount := 0

	for _, filename := range files {
		if filename == "-" {
			// stdin
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "wc: -: %v\n", err)
				continue
			}
			counts := countData(data, "-")
			printCountsSimple(counts, "-", showAll, countLines, countWords, countBytes, countChars)
			addCounts(total, counts)
			fileCount++
		} else {
			counts, err := countFile(filename)
			if err != nil {
				fmt.Fprintf(os.Stderr, "wc: %s: %v\n", filename, err)
				continue
			}
			printCountsSimple(counts, filename, showAll, countLines, countWords, countBytes, countChars)
			addCounts(total, counts)
			fileCount++
		}
	}

	// Итоги для нескольких файлов
	if fileCount > 1 {
		printCountsSimple(total, "total", showAll, countLines, countWords, countBytes, countChars)
	}
}

type FileCounts struct {
	Lines int
	Words int
	Bytes int
	Chars int
}

func countFile(filename string) (*FileCounts, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	
	return countData(data, filename), nil
}

func countData(data []byte, source string) *FileCounts {
	counts := &FileCounts{}
	
	// Байты
	counts.Bytes = len(data)
	
	// Конвертируем в строку для других подсчетов
	text := string(data)
	
	// Символы (руны)
	counts.Chars = len([]rune(text))
	
	// Строки
	lines := strings.Count(text, "\n")
	if len(text) > 0 && !strings.HasSuffix(text, "\n") {
		lines++ // Последняя строка без \n
	}
	counts.Lines = lines
	
	// Слова
	counts.Words = countWordsInText(text)
	
	return counts
}

func countWordsInText(text string) int {
	scanner := bufio.NewScanner(strings.NewReader(text))
	scanner.Split(bufio.ScanWords)
	
	words := 0
	for scanner.Scan() {
		words++
	}
	
	return words
}

func addCounts(total, counts *FileCounts) {
	total.Lines += counts.Lines
	total.Words += counts.Words
	total.Bytes += counts.Bytes
	total.Chars += counts.Chars
}

func printCountsSimple(counts *FileCounts, filename string, showAll, countLines, countWords, countBytes, countChars bool) {
	var parts []string
	
	if showAll || countLines {
		parts = append(parts, fmt.Sprintf("%7d", counts.Lines))
	}
	
	if showAll || countWords {
		parts = append(parts, fmt.Sprintf("%7d", counts.Words))
	}
	
	if countChars {
		parts = append(parts, fmt.Sprintf("%7d", counts.Chars))
	}
	
	if showAll || countBytes {
		parts = append(parts, fmt.Sprintf("%7d", counts.Bytes))
	}
	
	if filename != "" {
		parts = append(parts, filename)
	}
	
	fmt.Println(strings.Join(parts, " "))
}

func printSimpleHelp() {
	fmt.Println(`wc - word, line, character, and byte count

Usage: wc [option]... [file]...

Options:
  -c    print the byte counts
  -m    print the character counts
  -l    print the newline counts
  -w    print the word counts
      --help  display this help and exit

With no FILE, or when FILE is -, read standard input.
The default output format prints newline, word, and byte counts.

Examples:
  wc file.txt            count lines, words, and bytes
  wc -l file.txt         count lines only
  wc -w file.txt         count words only
  wc -c file.txt         count bytes only
  wc -m file.txt         count characters
  cat file.txt | wc      read from stdin
  wc file1 file2         multiple files`)
}