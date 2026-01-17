package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	// Определение флагов
	termType := flag.String("T", "", "указать тип терминала")
	versionFlag := flag.Bool("V", false, "показать версию")
	noScroll := flag.Bool("x", false, "не очищать историю прокрутки")
	helpFlag := flag.Bool("help", false, "показать справку")
	
	flag.Parse()

	if *helpFlag {
		printSimpleHelp()
		return
	}

	if *versionFlag {
		printSimpleVersion() 
		return
	}

	clearScreen(*termType, *noScroll)
}

func clearScreen(termType string, noScroll bool) {
	// Определяем фактический тип терминала
	actualTerm := termType
	if actualTerm == "" {
		actualTerm = os.Getenv("TERM")
		if actualTerm == "" {
			actualTerm = "xterm" // значение по умолчанию
		}
	}
	
	// Получаем escape-последовательность для очистки
	seq := getClearSequence(actualTerm, noScroll)
	fmt.Print(seq)
}

func getClearSequence(term string, noScroll bool) string {
	termLower := strings.ToLower(term)
	
	// Специальная обработка для "тупых" терминалов
	if termLower == "dumb" {
		if noScroll {
			// Много пустых строк для видимой области
			return "\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n"
		} else {
			// Еще больше строк + возврат каретки
			return "\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\r"
		}
	}
	
	// Для всех остальных терминалов используем ANSI escape-последовательности
	if noScroll {
		// Очистка только видимой области
		return "\033[2J"
	} else {
		// Полная очистка
		return "\033[2J\033[H"
	}
}

func printSimpleHelp() {
	fmt.Println(`clear - очистка экрана терминала

Использование: clear [опции]

Опции:
  -T тип       указать тип терминала
  -V           показать версию
  -x           не очищать историю прокрутки
  --help       показать справку

Примеры:
  clear           очистить экран
  clear -T xterm  очистить для терминала xterm
  clear -x        очистить видимую область
  clear -V        показать версию
  clear --help    показать справку

Типы терминалов:
  xterm, linux, vt100, vt220, screen, tmux, dumb

Переменная TERM:
  Если не указан -T, используется значение из $TERM
  Если $TERM не установлена, используется "xterm"`)
}

func printSimpleVersion() {
	fmt.Println("clear 1.0")
}
