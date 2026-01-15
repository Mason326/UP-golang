package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"syscall"
)

func main() {
	// Флаги
	all := flag.Bool("a", false, "include all filesystems")
	allLong := flag.Bool("all", false, "include all filesystems")
	human := flag.Bool("h", false, "human readable output")
	humanLong := flag.Bool("human-readable", false, "human readable output")
	blockSize := flag.String("B", "1K", "scale sizes by SIZE before printing them")
	blockSizeLong := flag.String("block-size", "1K", "scale sizes by SIZE before printing them")
	help := flag.Bool("help", false, "show help")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTION]... [FILE]...\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Show filesystem disk space usage.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  -a, --all            include all filesystems\n")
		fmt.Fprintf(os.Stderr, "  -h, --human-readable print sizes in human readable format\n")
		fmt.Fprintf(os.Stderr, "  -B, --block-size=SIZE  scale sizes by SIZE before printing them\n")
		fmt.Fprintf(os.Stderr, "      --help           show this help\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s                    show disk usage\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -h                show in human readable format\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -B 1M             show in 1M blocks\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s /home             show /home filesystem\n", os.Args[0])
	}

	flag.Parse()

	// Объединяем короткие и длинные флаги
	showAll := *all || *allLong
	humanReadable := *human || *humanLong
	
	// Используем значение из короткого или длинного флага
	bs := *blockSize
	if *blockSizeLong != "1K" {
		bs = *blockSizeLong
	}

	if *help {
		flag.Usage()
		return
	}

	// Получаем информацию о файловых системах
	filesystems, err := getFilesystems(showAll)
	if err != nil {
		fmt.Fprintf(os.Stderr, "df: error: %v\n", err)
		os.Exit(1)
	}

	// Показываем результат
	printFilesystems(filesystems, humanReadable, bs)
}

type Filesystem struct {
	Device     string
	MountPoint string
	Total      uint64
	Used       uint64
	Available  uint64
	UsePercent float64
}

func getFilesystems(showAll bool) ([]Filesystem, error) {
	var filesystems []Filesystem

	// Читаем /proc/mounts или /etc/mtab
	mountsFile := "/proc/mounts"
	if _, err := os.Stat(mountsFile); os.IsNotExist(err) {
		mountsFile = "/etc/mtab"
	}

	file, err := os.Open(mountsFile)
	if err != nil {
		return nil, fmt.Errorf("cannot open %s: %v", mountsFile, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		device := fields[0]
		mountPoint := fields[1]
		fsType := fields[2]

		// Фильтруем специальные файловые системы если не указан -a
		if !showAll && isSpecialFilesystem(device, mountPoint, fsType) {
			continue
		}

		// Получаем статистику файловой системы
		var stat syscall.Statfs_t
		if err := syscall.Statfs(mountPoint, &stat); err != nil {
			// Пропускаем файловые системы, к которым нет доступа
			continue
		}

		// Рассчитываем размеры
		blockSize := uint64(stat.Bsize)
		totalBlocks := uint64(stat.Blocks)
		freeBlocks := uint64(stat.Bfree)
		availableBlocks := uint64(stat.Bavail)

		total := totalBlocks * blockSize
		available := availableBlocks * blockSize
		used := total - freeBlocks*blockSize

		usePercent := 0.0
		if total > 0 {
			usePercent = float64(used) / float64(total) * 100
		}

		filesystems = append(filesystems, Filesystem{
			Device:     device,
			MountPoint: mountPoint,
			Total:      total,
			Used:       used,
			Available:  available,
			UsePercent: usePercent,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %v", mountsFile, err)
	}

	return filesystems, nil
}

func isSpecialFilesystem(device, mountPoint, fsType string) bool {
	// Список специальных файловых систем
	specialTypes := map[string]bool{
		"proc":       true,
		"sysfs":      true,
		"devtmpfs":   true,
		"devpts":     true,
		"tmpfs":      true,
		"cgroup":     true,
		"cgroup2":    true,
		"overlay":    true,
		"debugfs":    true,
		"tracefs":    true,
		"securityfs": true,
		"configfs":   true,
		"fusectl":    true,
		"pstore":     true,
		"binfmt_misc": true,
		"autofs":     true,
	}

	// Проверяем тип файловой системы
	if specialTypes[fsType] {
		return true
	}

	// Специальные точки монтирования
	specialMounts := []string{
		"/proc", "/sys", "/dev", "/run", "/sys/fs/cgroup",
		"/dev/shm", "/dev/pts", "/run/lock", "/run/user",
	}

	for _, m := range specialMounts {
		if mountPoint == m || strings.HasPrefix(mountPoint, m+"/") {
			return true
		}
	}

	// Специальные устройства
	if device == "none" || device == "udev" || device == "tmpfs" ||
		strings.HasPrefix(device, "/dev/loop") || strings.HasPrefix(device, "overlay") {
		return true
	}

	return false
}

func printFilesystems(filesystems []Filesystem, humanReadable bool, blockSize string) {
	// Определяем заголовок
	var header string
	if humanReadable {
		header = "Size"
	} else {
		switch blockSize {
		case "1K", "1024":
			header = "1K-blocks"
		case "1M", "1048576":
			header = "1M-blocks"
		case "1G", "1073741824":
			header = "1G-blocks"
		default:
			header = blockSize + "-blocks"
		}
	}

	// Выводим заголовок
	fmt.Printf("%-25s %10s %10s %10s %5s %s\n",
		"Filesystem", header, "Used", "Available", "Use%", "Mounted on")

	// Выводим информацию о каждой файловой системе
	for _, fs := range filesystems {
		printFilesystem(fs, humanReadable, blockSize)
	}
}

func printFilesystem(fs Filesystem, humanReadable bool, blockSize string) {
	var totalStr, usedStr, availStr string

	if humanReadable {
		totalStr = formatHuman(fs.Total)
		usedStr = formatHuman(fs.Used)
		availStr = formatHuman(fs.Available)
	} else {
		scale := getScaleFactor(blockSize)
		totalStr = fmt.Sprintf("%d", fs.Total/scale)
		usedStr = fmt.Sprintf("%d", fs.Used/scale)
		availStr = fmt.Sprintf("%d", fs.Available/scale)
	}

	// Обрезаем слишком длинные имена устройств
	device := fs.Device
	if len(device) > 25 {
		device = "..." + device[len(device)-22:]
	}

	fmt.Printf("%-25s %10s %10s %10s %5.0f%% %s\n",
		device, totalStr, usedStr, availStr, fs.UsePercent, fs.MountPoint)
}

func getScaleFactor(blockSize string) uint64 {
	switch blockSize {
	case "1K", "1024":
		return 1024
	case "1M", "1048576":
		return 1024 * 1024
	case "1G", "1073741824":
		return 1024 * 1024 * 1024
	case "512":
		return 512
	case "2K", "2048":
		return 2048
	case "4K", "4096":
		return 4096
	default:
		// Пробуем распарсить число
		return parseBlockSize(blockSize)
	}
}

func parseBlockSize(s string) uint64 {
	if len(s) == 0 {
		return 1024 // По умолчанию
	}

	multiplier := uint64(1)
	lastChar := s[len(s)-1]

	switch lastChar {
	case 'K', 'k':
		multiplier = 1024
		s = s[:len(s)-1]
	case 'M', 'm':
		multiplier = 1024 * 1024
		s = s[:len(s)-1]
	case 'G', 'g':
		multiplier = 1024 * 1024 * 1024
		s = s[:len(s)-1]
	case 'T', 't':
		multiplier = 1024 * 1024 * 1024 * 1024
		s = s[:len(s)-1]
	}

	// Пробуем распарсить число
	var num uint64
	n, err := fmt.Sscanf(s, "%d", &num)
	if err != nil || n != 1 {
		return 1024 // По умолчанию
	}

	return num * multiplier
}

func formatHuman(bytes uint64) string {
	const (
		KB = 1024.0
		MB = KB * 1024.0
		GB = MB * 1024.0
		TB = GB * 1024.0
		PB = TB * 1024.0
	)

	switch {
	case bytes >= PB:
		return fmt.Sprintf("%.1fP", float64(bytes)/PB)
	case bytes >= TB:
		return fmt.Sprintf("%.1fT", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.1fG", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1fM", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1fK", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
