package main

import (
	"flag"
	"fmt"
	"os"
	"time"
    "strings"
)

func main() {
	// Парсинг флагов
	noCreate := flag.Bool("c", false, "не создавать файлы, если они не существуют")
	reference := flag.String("r", "", "использовать время указанного файла")
	timeStr := flag.String("t", "", "установить указанное время [[CC]YY]MMDDhhmm[.ss]")
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
		fmt.Fprintln(os.Stderr, "Ошибка: не указаны файлы")
		fmt.Fprintln(os.Stderr, "Использование: touch [опции] файл...")
		os.Exit(1)
	}
	
	// Определяем время для установки
	var targetTime time.Time
	var err error
	
	switch {
	case *reference != "":
		// Используем время другого файла
		targetTime, err = getFileTime(*reference)
		if err != nil {
			fmt.Fprintf(os.Stderr, "touch: ошибка доступа к %s: %v\n", *reference, err)
			os.Exit(1)
		}
	case *timeStr != "":
		// Используем указанное время
		targetTime, err = parseTime(*timeStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "touch: неверный формат времени '%s': %v\n", *timeStr, err)
			os.Exit(1)
		}
	default:
		// Используем текущее время
		targetTime = time.Now()
	}
	
	// Обрабатываем каждый файл
	allSuccess := true
	for _, filename := range args {
		err := processFile(filename, targetTime, *noCreate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "touch: %s: %v\n", filename, err)
			allSuccess = false
		}
	}
	
	if !allSuccess {
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`Использование: touch [опции] файл...
Изменяет время доступа и модификации файлов.

Опции:
  -a        изменить только время доступа
  -c        не создавать файлы, если они не существуют
  -r файл   использовать время указанного файла
  -t время  установить указанное время [[CC]YY]MMDDhhmm[.ss]
  -h        показать эту справку

Примеры:
  touch file.txt               # Создать файл или обновить время
  touch -c file.txt           # Только обновить время существующего файла
  touch -r ref.txt file.txt   # Использовать время ref.txt
  touch -t 202312251530.45 file.txt  # Установить конкретное время
  touch file1.txt file2.txt   # Обработать несколько файлов`)
}

func getFileTime(filename string) (time.Time, error) {
	info, err := os.Stat(filename)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

func parseTime(timeStr string) (time.Time, error) {
	// Поддерживаем разные форматы:
	// MMDDhhmm[.ss]
	// [[CC]YY]MMDDhhmm[.ss]
	
	// Определяем длину строки
	length := len(timeStr)
	
	var year, month, day, hour, minute, second int
	
	switch length {
	case 8: // MMDDhhmm
		// Год не указан - используем текущий
		year = time.Now().Year()
		fmt.Sscanf(timeStr, "%2d%2d%2d%2d", &month, &day, &hour, &minute)
		
	case 10: // YYMMDDhhmm (2 цифры года)
		fmt.Sscanf(timeStr, "%2d%2d%2d%2d%2d", &year, &month, &day, &hour, &minute)
		// Преобразуем 2 цифры года в 4
		if year >= 0 && year <= 68 {
			year += 2000
		} else {
			year += 1900
		}
		
	case 12: // CCYYMMDDhhmm (4 цифры года)
		fmt.Sscanf(timeStr, "%4d%2d%2d%2d%2d", &year, &month, &day, &hour, &minute)
		
	case 11, 13, 15: // С секундами MMDDhhmm.ss или YYMMDDhhmm.ss или CCYYMMDDhhmm.ss
		// Определяем позицию точки
		dotIndex := strings.Index(timeStr, ".")
		if dotIndex == -1 {
			return time.Time{}, fmt.Errorf("неверный формат времени")
		}
		
		// Парсим основную часть
		mainPart := timeStr[:dotIndex]
		secPart := timeStr[dotIndex+1:]
		
		switch len(mainPart) {
		case 8: // MMDDhhmm
			year = time.Now().Year()
			fmt.Sscanf(mainPart, "%2d%2d%2d%2d", &month, &day, &hour, &minute)
		case 10: // YYMMDDhhmm
			fmt.Sscanf(mainPart, "%2d%2d%2d%2d%2d", &year, &month, &day, &hour, &minute)
			if year >= 0 && year <= 68 {
				year += 2000
			} else {
				year += 1900
			}
		case 12: // CCYYMMDDhhmm
			fmt.Sscanf(mainPart, "%4d%2d%2d%2d%2d", &year, &month, &day, &hour, &minute)
		default:
			return time.Time{}, fmt.Errorf("неверный формат времени")
		}
		
		// Парсим секунды
		if len(secPart) == 2 {
			fmt.Sscanf(secPart, "%2d", &second)
		} else {
			return time.Time{}, fmt.Errorf("неверный формат секунд")
		}
		
	default:
		return time.Time{}, fmt.Errorf("неверный формат времени")
	}
	
	// Проверяем корректность даты
	if month < 1 || month > 12 || day < 1 || day > 31 ||
	   hour < 0 || hour > 23 || minute < 0 || minute > 59 ||
	   second < 0 || second > 59 {
		return time.Time{}, fmt.Errorf("неверные значения даты/времени")
	}
	
	// Создаем время
	return time.Date(year, time.Month(month), day, hour, minute, second, 0, time.Local), nil
}

func processFile(filename string, targetTime time.Time, noCreate bool) error {
	// Проверяем существование файла
	_, err := os.Stat(filename)
	fileExists := err == nil
	
	if !fileExists {
		if noCreate {
			// Не создаем файл
			return fmt.Errorf("файл не существует (используйте без -c для создания)")
		}
		
		// Создаем файл
		file, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("не удалось создать файл: %v", err)
		}
		file.Close()
	}
	
	// Изменяем время файла
	return os.Chtimes(filename, targetTime, targetTime)
}
