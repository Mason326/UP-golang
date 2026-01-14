package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

type DiskInfo struct {
	Drive      string
	Size       uint64
	Free       uint64
	Used       uint64
	UsePercent float64
}

type DfOptions struct {
	All       bool   // -a, --all
	BlockSize string // -B, --block-size=SIZE
	Kilo      bool   // -k (like --block-size=1K)
	Human     bool   // -h, --human-readable
	Help      bool   // --help
}

func main() {
	opts := parseFlags()

	if opts.Help {
		printHelp()
		return
	}

	if runtime.GOOS != "windows" {
		fmt.Println("Эта версия df предназначена для Windows")
		fmt.Println("Для Linux используйте системную команду df")
		return
	}

	showWindowsDisks(opts)
}

func parseFlags() *DfOptions {
	opts := &DfOptions{}

	// Определение флагов
	flag.BoolVar(&opts.All, "a", false, "include all filesystems")
	flag.BoolVar(&opts.All, "all", false, "include all filesystems")
	flag.StringVar(&opts.BlockSize, "B", "", "scale sizes by SIZE before printing them")
	flag.StringVar(&opts.BlockSize, "block-size", "", "scale sizes by SIZE before printing them")
	flag.BoolVar(&opts.Kilo, "k", false, "like --block-size=1K")
	flag.BoolVar(&opts.Human, "h", false, "human readable output")
	flag.BoolVar(&opts.Human, "human-readable", false, "human readable output")
	flag.BoolVar(&opts.Help, "help", false, "show help")

	flag.Parse()
	return opts
}

func showWindowsDisks(opts *DfOptions) {
	// Получаем информацию о дисках
	disks := getAllDisksInfo(opts.All)
	
	// Определяем размер блока для вывода
	blockSize := getBlockSize(opts)
	scaleFactor := getScaleFactor(blockSize)
	blockSizeLabel := getBlockSizeLabel(blockSize)
	
	// Заголовок
	if opts.Human {
		fmt.Printf("%-10s %15s %15s %15s %6s %s\n", 
			"Filesystem", "Size", "Used", "Avail", "Use%", "Mounted on")
	} else {
		if blockSizeLabel == "B" {
			fmt.Printf("%-10s %15s %15s %15s %6s %s\n", 
				"Filesystem", "Bytes", "Used", "Available", "Use%", "Mounted on")
		} else {
			fmt.Printf("%-10s %15s %15s %15s %6s %s\n", 
				"Filesystem", blockSizeLabel+"-blocks", "Used", "Available", "Use%", "Mounted on")
		}
	}

	// Выводим информацию по каждому диску
	for _, disk := range disks {
		printDiskInfo(disk, opts, scaleFactor, blockSizeLabel)
	}
	
	// Выводим итоговую информацию
	if opts.All && len(disks) > 0 {
		printTotalInfo(disks, opts, scaleFactor, blockSizeLabel)
	}
}

func getAllDisksInfo(showAll bool) []DiskInfo {
	var disks []DiskInfo
	
	// Получаем список всех дисков
	allDrives := getAllDrives()
	
	for _, drive := range allDrives {
		size, free, err := getDiskSpace(drive)
		if err != nil {
			continue
		}
		
		// Пропускаем диски с нулевым размером если не указан -a
		if !showAll && size == 0 {
			continue
		}
		
		used := size - free
		usePercent := 0.0
		if size > 0 {
			usePercent = float64(used) / float64(size) * 100
		}
		
		disks = append(disks, DiskInfo{
			Drive:      drive,
			Size:       size,
			Free:       free,
			Used:       used,
			UsePercent: usePercent,
		})
	}
	
	return disks
}

func getAllDrives() []string {
	var drives []string
	
	// Используем Windows API для получения списка дисков
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getLogicalDrives := kernel32.NewProc("GetLogicalDrives")
	
	ret, _, _ := getLogicalDrives.Call()
	if ret == 0 {
		// Fallback: проверяем диски от A: до Z:
		for drive := 'A'; drive <= 'Z'; drive++ {
			drivePath := string(drive) + ":\\"
			_, err := os.Stat(drivePath)
			if err == nil {
				drives = append(drives, drivePath)
			}
		}
		return drives
	}
	
	// Разбираем битовую маску дисков
	driveBits := uint32(ret)
	for drive := 0; drive < 26; drive++ {
		if driveBits&(1<<drive) != 0 {
			driveLetter := string(rune('A'+drive)) + ":\\"
			drives = append(drives, driveLetter)
		}
	}
	
	return drives
}

func getDiskSpace(drive string) (uint64, uint64, error) {
	var freeBytes, totalBytes uint64
	
	drivePtr, err := syscall.UTF16PtrFromString(drive)
	if err != nil {
		return 0, 0, err
	}

	// Вызываем GetDiskFreeSpaceExW
	ret, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("GetDiskFreeSpaceExW").Call(
		uintptr(unsafe.Pointer(drivePtr)),
		uintptr(unsafe.Pointer(&freeBytes)),
		uintptr(unsafe.Pointer(&totalBytes)),
		0,
	)

	if ret == 0 {
		return 0, 0, fmt.Errorf("не удалось получить информацию о диске")
	}

	return totalBytes, freeBytes, nil
}

func getBlockSize(opts *DfOptions) string {
	// Определяем размер блока в зависимости от флагов
	if opts.Human {
		return "human"
	}
	
	if opts.Kilo {
		return "1K"
	}
	
	if opts.BlockSize != "" {
		return opts.BlockSize
	}
	
	// По умолчанию - 1K блоки
	return "1K"
}

func getScaleFactor(blockSize string) uint64 {
	switch blockSize {
	case "human":
		return 1 // Будет обработан отдельно
	case "1K", "K", "k":
		return 1024
	case "1M", "M", "m":
		return 1024 * 1024
	case "1G", "G", "g":
		return 1024 * 1024 * 1024
	case "1T", "T", "t":
		return 1024 * 1024 * 1024 * 1024
	case "1P", "P", "p":
		return 1024 * 1024 * 1024 * 1024 * 1024
	case "1E", "E", "e":
		return 1024 * 1024 * 1024 * 1024 * 1024 * 1024
	default:
		// Пробуем распарсить число
		if len(blockSize) > 0 {
			// Убираем возможный префикс "1"
			size := blockSize
			if len(blockSize) > 1 && blockSize[0] == '1' {
				size = blockSize[1:]
			}
			
			// Определяем множитель по последнему символу
			lastChar := size[len(size)-1]
			multiplier := uint64(1)
			
			switch lastChar {
			case 'K', 'k':
				multiplier = 1024
				size = size[:len(size)-1]
			case 'M', 'm':
				multiplier = 1024 * 1024
				size = size[:len(size)-1]
			case 'G', 'g':
				multiplier = 1024 * 1024 * 1024
				size = size[:len(size)-1]
			case 'T', 't':
				multiplier = 1024 * 1024 * 1024 * 1024
				size = size[:len(size)-1]
			case 'P', 'p':
				multiplier = 1024 * 1024 * 1024 * 1024 * 1024
				size = size[:len(size)-1]
			case 'E', 'e':
				multiplier = 1024 * 1024 * 1024 * 1024 * 1024 * 1024
				size = size[:len(size)-1]
			}
			
			// Пробуем распарсить число
			if len(size) > 0 {
				var num uint64
				n, err := fmt.Sscanf(size, "%d", &num)
				if err == nil && n == 1 {
					return num * multiplier
				}
			} else {
				return multiplier
			}
		}
		
		// По умолчанию 1024
		return 1024
	}
}

func getBlockSizeLabel(blockSize string) string {
	switch blockSize {
	case "human":
		return "human"
	case "1K", "K", "k":
		return "1K"
	case "1M", "M", "m":
		return "1M"
	case "1G", "G", "g":
		return "1G"
	case "1T", "T", "t":
		return "1T"
	case "1P", "P", "p":
		return "1P"
	case "1E", "E", "e":
		return "1E"
	default:
		// Возвращаем как есть
		return blockSize
	}
}

func printDiskInfo(disk DiskInfo, opts *DfOptions, scaleFactor uint64, blockSizeLabel string) {
	var sizeStr, usedStr, freeStr string
	driveName := disk.Drive[:2] + ":" // Берем только букву диска
	
	if opts.Human {
		sizeStr = formatHuman(disk.Size)
		usedStr = formatHuman(disk.Used)
		freeStr = formatHuman(disk.Free)
	} else {
		if scaleFactor == 1 {
			// Байты
			sizeStr = fmt.Sprintf("%d", disk.Size)
			usedStr = fmt.Sprintf("%d", disk.Used)
			freeStr = fmt.Sprintf("%d", disk.Free)
		} else {
			// Масштабированные блоки
			sizeStr = fmt.Sprintf("%d", disk.Size/scaleFactor)
			usedStr = fmt.Sprintf("%d", disk.Used/scaleFactor)
			freeStr = fmt.Sprintf("%d", disk.Free/scaleFactor)
		}
	}
	
	fmt.Printf("%-10s %15s %15s %15s %5.0f%% %s\n",
		driveName, sizeStr, usedStr, freeStr, disk.UsePercent, disk.Drive)
}

func printTotalInfo(disks []DiskInfo, opts *DfOptions, scaleFactor uint64, blockSizeLabel string) {
	var totalSize, totalUsed, totalFree uint64
	for _, disk := range disks {
		totalSize += disk.Size
		totalUsed += disk.Used
		totalFree += disk.Free
	}
	
	totalUsePercent := 0.0
	if totalSize > 0 {
		totalUsePercent = float64(totalUsed) / float64(totalSize) * 100
	}
	
	var sizeStr, usedStr, freeStr string
	if opts.Human {
		sizeStr = formatHuman(totalSize)
		usedStr = formatHuman(totalUsed)
		freeStr = formatHuman(totalFree)
	} else {
		if scaleFactor == 1 {
			sizeStr = fmt.Sprintf("%d", totalSize)
			usedStr = fmt.Sprintf("%d", totalUsed)
			freeStr = fmt.Sprintf("%d", totalFree)
		} else {
			sizeStr = fmt.Sprintf("%d", totalSize/scaleFactor)
			usedStr = fmt.Sprintf("%d", totalUsed/scaleFactor)
			freeStr = fmt.Sprintf("%d", totalFree/scaleFactor)
		}
	}
	
	fmt.Printf("%-10s %15s %15s %15s %5.0f%% %s\n",
		"total", sizeStr, usedStr, freeStr, totalUsePercent, "")
}

func formatHuman(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
		PB = TB * 1024
		EB = PB * 1024
	)

	switch {
	case bytes >= EB:
		return fmt.Sprintf("%.1fE", float64(bytes)/float64(EB))
	case bytes >= PB:
		return fmt.Sprintf("%.1fP", float64(bytes)/float64(PB))
	case bytes >= TB:
		return fmt.Sprintf("%.1fT", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.1fG", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1fM", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1fK", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

func printHelp() {
	helpText := `Использование: df [ПАРАМЕТР]... [ФАЙЛ]...
Показывает информацию об использовании дискового пространства на Windows.

Параметры:
  -a, --all            показать все файловые системы (включая пустые)
  -B, --block-size=РАЗМЕР  использовать блоки указанного РАЗМЕРа
  -h, --human-readable выводить размеры в удобочитаемом формате
  -k                   равносильно --block-size=1K
      --help           показать эту справку и выйти

По умолчанию размеры выводятся в 1K-блоках.

Блочный размер может быть указан в следующих форматах:
  K, M, G, T, P, E, Z, Y  (блок 1024)
  KB, MB, GB, TB, PB, EB, ZB, YB  (блок 1024)
  K, M, G, T, P, E, Z, Y  (блок 1000)

Примеры размеров:
  1K    = 1024 байт
  1M    = 1048576 байт
  1G    = 1073741824 байт
  1024  = 1024 байт
  1KB   = 1000 байт
  1MB   = 1000000 байт

Флаги:
  -a    Показать все диски, включая пустые и недоступные
  -B    Указать размер блока для вывода
  -k    Выводить в килобайтах (1K блоках)
  -h    Выводить в удобочитаемом формате (автоматически подбирать единицы)

Примеры:
  df              Показать информацию о дисках
  df -h           Удобочитаемый формат
  df -k           В килобайтах
  df -B 1M        В мегабайтах
  df -a           Все диски
  df -B 1024      В блоках по 1024 байта
  df --help       Справка
`
	fmt.Println(helpText)
}