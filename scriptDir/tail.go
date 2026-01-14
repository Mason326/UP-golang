package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

func main() {
	// Флаги командной строки
	n := flag.Int("n", 10, "количество строк для вывода")
	follow := flag.Bool("f", false, "отслеживать изменения файла (только для одного файла)")
	quiet := flag.Bool("q", false, "не выводить заголовки с именами файлов")
	help := flag.Bool("help", false, "показать справку")
	
	// Кастомное использование
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Использование: %s [ПАРАМЕТРЫ]... [ФАЙЛЫ]...\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Выводит последние строки файлов (по умолчанию 10).\n\n")
		fmt.Fprintf(os.Stderr, "Параметры:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nПримеры:\n")
		fmt.Fprintf(os.Stderr, "  %s -n 20 file.txt           Вывести последние 20 строк файла\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -f file.txt             Отслеживать изменения файла в реальном времени\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s file1.txt file2.txt     Вывести последние строки нескольких файлов\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  cat file.txt | %s          Чтение из стандартного ввода\n", os.Args[0])
	}

	flag.Parse()

	// Показать справку если запрошено
	if *help {
		flag.Usage()
		os.Exit(0)
	}

	files := flag.Args()
	
	// Проверка для флага -f
	if *follow && len(files) == 0 {
		fmt.Fprintf(os.Stderr, "tail: опция -f требует указания файла\n")
		fmt.Fprintf(os.Stderr, "Попробуйте '%s --help' для получения дополнительной информации.\n", os.Args[0])
		os.Exit(1)
	}
	
	if *follow && len(files) > 1 {
		fmt.Fprintf(os.Stderr, "tail: опция -f поддерживает только один файл\n")
		fmt.Fprintf(os.Stderr, "Попробуйте '%s --help' для получения дополнительной информации.\n", os.Args[0])
		os.Exit(1)
	}

	// Чтение из stdin если файлы не указаны
	if len(files) == 0 {
		// Для stdin нельзя использовать флаг -f
		if *follow {
			fmt.Fprintf(os.Stderr, "tail: опция -f не поддерживается для стандартного ввода\n")
			os.Exit(1)
		}
		tailReader(os.Stdin, *n, "")
		return
	}

	multiFile := len(files) > 1
	
	// Если используется флаг -f, обрабатываем только первый файл
	if *follow {
		filename := files[0]
		tailFollow(filename, *n)
		return
	}
	
	// Обычный режим (без -f) для одного или нескольких файлов
	for i, filename := range files {
		if !*quiet && multiFile {
			if i > 0 {
				fmt.Println()
			}
			fmt.Printf("==> %s <==\n", filename)
		}

		file, err := os.Open(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "tail: невозможно открыть '%s' для чтения: %v\n", filename, err)
			continue
		}

		tailFile(file, *n)
		file.Close()
	}
}

// tailFile выводит последние n строк из файла
func tailFile(file *os.File, n int) {
	// Используем кольцевой буфер для эффективного хранения последних n строк
	if n <= 0 {
		return
	}

	ringBuffer := make([]string, 0, n)
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		line := scanner.Text()
		if len(ringBuffer) < n {
			ringBuffer = append(ringBuffer, line)
		} else {
			// Сдвигаем буфер
			ringBuffer = append(ringBuffer[1:], line)
		}
	}

	for _, line := range ringBuffer {
		fmt.Println(line)
	}
	
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "tail: ошибка чтения файла: %v\n", err)
	}
}

// tailReader выводит последние n строк из io.Reader
func tailReader(reader io.Reader, n int, prefix string) {
	if n <= 0 {
		return
	}

	ringBuffer := make([]string, 0, n)
	scanner := bufio.NewScanner(reader)
	
	for scanner.Scan() {
		line := scanner.Text()
		if len(ringBuffer) < n {
			ringBuffer = append(ringBuffer, line)
		} else {
			ringBuffer = append(ringBuffer[1:], line)
		}
	}

	for _, line := range ringBuffer {
		if prefix != "" {
			fmt.Printf("%s%s\n", prefix, line)
		} else {
			fmt.Println(line)
		}
	}
	
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "tail: ошибка чтения: %v\n", err)
	}
}

// tailFollow отслеживает изменения файла в реальном времени
func tailFollow(filename string, n int) {
	// Сначала выводим последние n строк файла
	file, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tail: невозможно открыть '%s': %v\n", filename, err)
		os.Exit(1)
	}
	
	tailFile(file, n)
	
	// Получаем информацию о файле для определения его текущего размера
	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "tail: невозможно получить информацию о файле: %v\n", err)
		file.Close()
		os.Exit(1)
	}
	
	// Запоминаем текущий размер файла
	lastPos := fileInfo.Size()
	file.Close()
	
	fmt.Fprintf(os.Stderr, "Отслеживание файла '%s'. Нажмите Ctrl+C для выхода...\n", filename)
	
	// Здесь нужно было бы использовать signal.Notify, но упростим
	
	// Бесконечный цикл для отслеживания изменений
	for {
		time.Sleep(100 * time.Millisecond) // Проверяем каждые 100ms
		
		// Открываем файл заново для проверки изменений
		file, err := os.Open(filename)
		if err != nil {
			// Если файл был удален, ждем его появления
			if os.IsNotExist(err) {
				time.Sleep(500 * time.Millisecond)
				continue
			}
			fmt.Fprintf(os.Stderr, "tail: ошибка при открытии файла: %v\n", err)
			return
		}
		
		// Получаем текущий размер файла
		fileInfo, err := file.Stat()
		if err != nil {
			fmt.Fprintf(os.Stderr, "tail: ошибка при получении информации о файле: %v\n", err)
			file.Close()
			return
		}
		
		currentSize := fileInfo.Size()
		
		// Если файл стал меньше (был перезаписан), начинаем с начала
		if currentSize < lastPos {
			fmt.Fprintf(os.Stderr, "\n--- Файл был перезаписан, начинаем с начала ---\n")
			lastPos = 0
		}
		
		// Если есть новые данные
		if currentSize > lastPos {
			// Переходим к позиции, где остановились
			_, err = file.Seek(lastPos, io.SeekStart)
			if err != nil {
				fmt.Fprintf(os.Stderr, "tail: ошибка при перемещении по файлу: %v\n", err)
				file.Close()
				return
			}
			
			// Читаем новые данные
			buffer := make([]byte, 4096)
			for {
				n, err := file.Read(buffer)
				if n > 0 {
					// Выводим прочитанные данные
					fmt.Print(string(buffer[:n]))
					lastPos += int64(n)
				}
				
				if err != nil {
					if err != io.EOF {
						fmt.Fprintf(os.Stderr, "tail: ошибка чтения: %v\n", err)
					}
					break
				}
			}
		}
		
		file.Close()
		
		// Проверяем, не пора ли выйти (для простоты - вечный цикл)
		// В реальном приложении здесь был бы select с каналом сигналов
	}
}