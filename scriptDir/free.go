package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

type MemoryInfo struct {
	Total     uint64
	Used      uint64
	Free      uint64
	Shared    uint64
	Buffers   uint64
	Cached    uint64
	Available uint64
	SwapTotal uint64
	SwapUsed  uint64
	SwapFree  uint64
}

type DisplayOptions struct {
	Bytes     bool
	KiloBytes bool
	MegaBytes bool
	GigaBytes bool
	Seconds   int
	Help      bool
}

func main() {
	opts := parseFlags()

	if opts.Help {
		printHelp()
		return
	}

	// Если указан интервал, запускаем цикл
	if opts.Seconds > 0 {
		runPeriodic(opts)
	} else {
		displayMemoryInfo(opts)
	}
}

func parseFlags() *DisplayOptions {
	opts := &DisplayOptions{}

	// Определение флагов (только указанные)
	flag.BoolVar(&opts.Bytes, "b", false, "отобразить память в байтах")
	flag.BoolVar(&opts.KiloBytes, "k", false, "отобразить память в килобайтах")
	flag.BoolVar(&opts.MegaBytes, "m", false, "отобразить память в мегабайтах")
	flag.BoolVar(&opts.GigaBytes, "g", false, "отобразить память в гигабайтах")
	flag.IntVar(&opts.Seconds, "s", 0, "повторять каждые N секунд")
	flag.BoolVar(&opts.Help, "help", false, "показать справку")

	flag.Parse()
	return opts
}

func getMemoryInfo() (*MemoryInfo, error) {
	data, err := ioutil.ReadFile("/proc/meminfo")
	if err != nil {
		return nil, err
	}

	info := &MemoryInfo{}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		if len(fields) < 2 {
			continue
		}

		key := strings.TrimSuffix(fields[0], ":")
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}

		// Значения в /proc/meminfo в килобайтах, конвертируем в байты
		value *= 1024

		switch key {
		case "MemTotal":
			info.Total = value
		case "MemFree":
			info.Free = value
		case "MemAvailable":
			info.Available = value
		case "Buffers":
			info.Buffers = value
		case "Cached":
			info.Cached = value
		case "Shmem":
			info.Shared = value
		case "SwapTotal":
			info.SwapTotal = value
		case "SwapFree":
			info.SwapFree = value
		}
	}

	// Рассчитываем использованную память
	info.Used = info.Total - info.Free - info.Buffers - info.Cached
	info.SwapUsed = info.SwapTotal - info.SwapFree

	return info, nil
}

func formatBytes(bytes uint64, opts *DisplayOptions) string {
	// Определяем единицу измерения
	if opts.Bytes {
		return fmt.Sprintf("%d", bytes)
	} else if opts.KiloBytes {
		return fmt.Sprintf("%d", bytes/1024)
	} else if opts.MegaBytes {
		return fmt.Sprintf("%d", bytes/(1024*1024))
	} else if opts.GigaBytes {
		return fmt.Sprintf("%d", bytes/(1024*1024*1024))
	}
	
	// По умолчанию - килобайты
	return fmt.Sprintf("%d", bytes/1024)
}

func displayMemoryInfo(opts *DisplayOptions) {
	info, err := getMemoryInfo()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка чтения информации о памяти: %v\n", err)
		os.Exit(1)
	}

	// Заголовок
	fmt.Println("               total        used        free      shared  buff/cache   available")
	
	// Основная память
	fmt.Printf("Mem:    %12s %12s %12s %12s %12s %12s\n",
		formatBytes(info.Total, opts),
		formatBytes(info.Used, opts),
		formatBytes(info.Free, opts),
		formatBytes(info.Shared, opts),
		formatBytes(info.Buffers+info.Cached, opts),
		formatBytes(info.Available, opts),
	)

	// Своп
	fmt.Printf("Swap:   %12s %12s %12s\n",
		formatBytes(info.SwapTotal, opts),
		formatBytes(info.SwapUsed, opts),
		formatBytes(info.SwapFree, opts),
	)
}

func runPeriodic(opts *DisplayOptions) {
	interval := opts.Seconds
	
	for {
		// Очищаем экран между итерациями
		fmt.Println()
		
		// Выводим время
		currentTime := time.Now().Format("15:04:05")
		fmt.Printf("Время: %s (обновление каждые %d секунд)\n\n", currentTime, interval)
		
		displayMemoryInfo(opts)
		
		// Ждем указанный интервал
		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func printHelp() {
	helpText := `Использование: free [ПАРАМЕТР]...
Отображает информацию об использовании памяти.

Параметры:
  -b, --bytes          отобразить память в байтах
  -k, --kilo           отобразить память в килобайтах (по умолчанию)
  -m, --mega           отобразить память в мегабайтах
  -g, --giga           отобразить память в гигабайтах
  -s N, --seconds N    повторять вывод каждые N секунд
      --help           показать эту справку

По умолчанию используется -k (килобайты).

Поля вывода:
  total        общий объем доступной памяти
  used         использованная память (рассчитывается как total - free - buffers/cache)
  free         неиспользованная память
  shared       память, используемая (в основном) tmpfs
  buff/cache   сумма buffers и cache
  available    оценка памяти, доступной для запуска новых приложений

Примеры:
  free                 Базовая информация в килобайтах
  free -b              Отобразить память в байтах
  free -k              Отобразить память в килобайтах (явно)
  free -m              Отобразить память в мегабайтах
  free -g              Отобразить память в гигабайтах
  free -s 2            Повторять вывод каждые 2 секунды
  free --help          Показать справку`

	fmt.Println(helpText)
}
