package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"syscall"
	"time"
)

const dotCharacter = 46

// printHelp выводит справку по использованию программы
func printHelp() {
	programName := os.Args[0]
	fmt.Printf(`Использование: %s [КЛЮЧ]... [ФАЙЛ]...
Выводит информацию о файлах (по умолчанию в текущем каталоге).

Ключи:
  -a, --all                  не игнорировать записи, начинающиеся с .
  -h                         с -l: выводить размеры в читаемом для человека виде
                               (например, 1K 234M 2G)
  -l                         использовать длинный формат вывода
  -r, --reverse              обратный порядок сортировки
  -R, --recursive            выводить подкаталоги рекурсивно
      --help                 показать эту справку и выйти
`, programName)
}

func main() {
	// Варианты флагов
	recursiveFlag := flag.Bool("R", false, "List subdirectories recursively")
	allFlag := flag.Bool("a", false, "Do not ignore entries starting with .")
	longListingFlag := flag.Bool("l", false, "Use a long listing format")
	reverseFlag := flag.Bool("r", false, "Reverse order while sorting")
	humanReadableFlag := flag.Bool("h", false, "With -l, print sizes in human readable format (e.g., 1K 234M 2G)")
	helpFlag := flag.Bool("help", false, "Display this help and exit")

	// Устанавливаем кастомную функцию использования
	flag.Usage = func() {
		printHelp()
	}

	// Парсим флаги
	flag.Parse()

	// Проверяем флаг help
	if *helpFlag {
		printHelp()
		return
	}

	// Дополнительная проверка для --help в аргументах
	for _, arg := range os.Args[1:] {
		if arg == "--help" {
			printHelp()
			return
		}
	}

	// Получаем массив введенных директорий
	inputDirs := flag.Args()

	if len(inputDirs) == 0 {
		// По умолчанию просматриваем текущую директорию
		showListElems(".", *recursiveFlag, *allFlag, *longListingFlag, *reverseFlag, *humanReadableFlag)
		return
	} else {
		// Для множества указанных директорий
		for i, dir := range inputDirs {
			if len(inputDirs) > 1 {
				if i > 0 {
					fmt.Println()
				}
				fmt.Printf("%s:\n", dir)
			}
			showListElems(dir, *recursiveFlag, *allFlag, *longListingFlag, *reverseFlag, *humanReadableFlag)
		}
	}
}

func showListElems(path string, recursive, all, longListing, reverse, humanReadable bool) {
	if recursive {
		// Рекурсивный режим с комбинацией флагов
		showListElemsRecursive(path, all, longListing, reverse, humanReadable)
	} else {
		// Обычный режим
		showSingleDir(path, all, longListing, reverse, humanReadable, true)
	}
}

func showListElemsRecursive(root string, all, longListing, reverse, humanReadable bool) {
	// Собираем все директории для обхода
	var dirs []string
	
	// Начинаем с корневой директории
	dirs = append(dirs, root)
	
	// Используем WalkDir для сбора всех директорий
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Пропускаем ошибки
		}
		
		if info.IsDir() && path != root {
			// Пропускаем . и .. даже если all=true (как в оригинальном ls)
			if info.Name() == "." || info.Name() == ".." {
				return nil
			}
			
			// Проверяем скрытые директории
			if !all && isHidden(info.Name()) {
				return filepath.SkipDir
			}
			dirs = append(dirs, path)
		}
		return nil
	})
	
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	
	// Сортируем директории для правильного порядка
	sort.Strings(dirs)
	if reverse {
		for i, j := 0, len(dirs)-1; i < j; i, j = i+1, j-1 {
			dirs[i], dirs[j] = dirs[j], dirs[i]
		}
	}
	
	// Обрабатываем каждую директорию
	for i, dir := range dirs {
		if i > 0 {
			fmt.Println() // Пустая строка между директориями
		}
		fmt.Printf("%s:\n", dir)
		showSingleDir(dir, all, longListing, reverse, humanReadable, false)
	}
}

func showSingleDir(path string, all, longListing, reverse, humanReadable, showHeader bool) {
	// Чтение содержимого директории
	lst, err := ioutil.ReadDir(path)
	if err != nil {
		fmt.Printf("ls: cannot access '%s': %v\n", path, err)
		return
	}

	// Фильтрация скрытых файлов
	var entries []os.FileInfo
	for _, entry := range lst {
		if all || !isHidden(entry.Name()) {
			entries = append(entries, entry)
		}
	}

	// Если включен флаг -a, добавляем записи для . и ..
	if all {
		// Добавляем запись для текущей директории (.)
		if dirInfo, err := os.Stat(path); err == nil {
			dotEntry := &dotDirInfo{
				name:    ".",
				size:    dirInfo.Size(),
				mode:    dirInfo.Mode(),
				modTime: dirInfo.ModTime(),
				sys:     dirInfo.Sys(),
			}
			entries = append([]os.FileInfo{dotEntry}, entries...)
		}

		// Добавляем запись для родительской директории (..)
		parentPath := filepath.Dir(path)
		if parentInfo, err := os.Stat(parentPath); err == nil {
			dotDotEntry := &dotDirInfo{
				name:    "..",
				size:    parentInfo.Size(),
				mode:    parentInfo.Mode(),
				modTime: parentInfo.ModTime(),
				sys:     parentInfo.Sys(),
			}
			entries = append([]os.FileInfo{dotDotEntry}, entries...)
		}
	}

	// Сортировка по имени
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	// Разворот порядка если нужно
	if reverse {
		for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
			entries[i], entries[j] = entries[j], entries[i]
		}
	}

	if longListing {
		// Длинный формат как в ls -l
		showLongFormat(entries, path, humanReadable)
	} else {
		// Обычный формат вывода
		showSimpleFormat(entries)
	}
}

// Структура для представления . и .. директорий
type dotDirInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	sys     interface{}
}

func (d *dotDirInfo) Name() string       { return d.name }
func (d *dotDirInfo) Size() int64        { return d.size }
func (d *dotDirInfo) Mode() os.FileMode  { return d.mode }
func (d *dotDirInfo) ModTime() time.Time { return d.modTime }
func (d *dotDirInfo) IsDir() bool        { return true }
func (d *dotDirInfo) Sys() interface{}   { return d.sys }

func showSimpleFormat(entries []os.FileInfo) {
	if len(entries) == 0 {
		return
	}

	// Выводим записи
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name = name + "/"
		}
		fmt.Println(name)
	}
}

func showLongFormat(entries []os.FileInfo, path string, humanReadable bool) {
	// Общее количество блоков по 1K (1024 байта), как в ls
	var totalBlocks int64
	for _, entry := range entries {
		// Получаем информацию о файле через stat
		if stat, ok := entry.Sys().(*syscall.Stat_t); ok {
			// stat.Blocks возвращает количество блоков по 512 байт
			// ls делит это на 2 чтобы получить блоки по 1K
			totalBlocks += stat.Blocks / 2
		} else {
			// Fallback: вычисляем блоки по 1024 байта
			totalBlocks += (entry.Size() + 1023) / 1024
		}
	}
	
	if len(entries) > 0 {
		fmt.Printf("total %d\n", totalBlocks)
	}

	// Находим максимальную длину для разных полей
	maxSizeLen := 0
	maxLinksLen := 0
	maxUserLen := 0
	maxGroupLen := 0
	
	for _, entry := range entries {
		// Длина для размера файла
		var sizeStr string
		if humanReadable {
			sizeStr = formatSizeHumanReadable(getFileSize(entry))
		} else {
			sizeStr = fmt.Sprintf("%d", getFileSize(entry))
		}
		if len(sizeStr) > maxSizeLen {
			maxSizeLen = len(sizeStr)
		}
		
		// Длина для количества ссылок
		links := fmt.Sprintf("%d", getFileLinks(entry))
		if len(links) > maxLinksLen {
			maxLinksLen = len(links)
		}
		
		// Длина для имени пользователя и группы
		uid, gid := getFileOwner(entry)
		user := getUserName(uid)
		group := getGroupName(gid)
		
		if len(user) > maxUserLen {
			maxUserLen = len(user)
		}
		if len(group) > maxGroupLen {
			maxGroupLen = len(group)
		}
	}

	for _, entry := range entries {
		// Права доступа
		mode := formatFileMode(entry.Mode())

		// Количество ссылок
		nlink := getFileLinks(entry)

		// Владелец и группа
		uid, gid := getFileOwner(entry)
		user := getUserName(uid)
		group := getGroupName(gid)

		// Размер файла
		fileSize := getFileSize(entry)
		var sizeStr string
		if humanReadable {
			sizeStr = formatSizeHumanReadable(fileSize)
		} else {
			sizeStr = fmt.Sprintf("%d", fileSize)
		}

		// Время изменения
		modTime := formatModTime(entry.ModTime())

		// Имя файла
		name := formatFileName(entry)

		// Форматированный вывод
		fmt.Printf("%s %*d %-*s %-*s %*s %s %s\n", 
			mode, 
			maxLinksLen, nlink,
			maxUserLen, user,
			maxGroupLen, group,
			maxSizeLen, sizeStr,
			modTime, name)
	}
}

// getFileSize возвращает размер файла в байтах
func getFileSize(entry os.FileInfo) int64 {
	if stat, ok := entry.Sys().(*syscall.Stat_t); ok {
		return stat.Size
	}
	return entry.Size()
}

// getFileLinks возвращает количество жестких ссылок
func getFileLinks(entry os.FileInfo) int64 {
	if stat, ok := entry.Sys().(*syscall.Stat_t); ok {
		return int64(stat.Nlink)
	}
	return 1
}

// getFileOwner возвращает UID и GID владельца файла
func getFileOwner(entry os.FileInfo) (uint32, uint32) {
	if stat, ok := entry.Sys().(*syscall.Stat_t); ok {
		return stat.Uid, stat.Gid
	}
	return 1000, 1000 // Default for user
}

// getUserName возвращает имя пользователя по UID
func getUserName(uid uint32) string {
	// Здесь должна быть реализация получения имени из /etc/passwd
	// Для простоты возвращаем числовой идентификатор
	return fmt.Sprintf("%d", uid)
}

// getGroupName возвращает имя группы по GID
func getGroupName(gid uint32) string {
	// Здесь должна быть реализация получения имени из /etc/group
	// Для простоты возвращаем числовой идентификатор
	return fmt.Sprintf("%d", gid)
}

// formatFileMode форматирует права доступа в стиле ls -l
func formatFileMode(mode os.FileMode) string {
	str := mode.String()
	
	// Преобразуем в формат ls (10 символов)
	if len(str) > 10 {
		return str[:10]
	}
	return str
}

// formatModTime форматирует время изменения
func formatModTime(t time.Time) string {
	now := time.Now()
	sixMonthsAgo := now.AddDate(0, -6, 0)
	
	if t.Before(sixMonthsAgo) {
		// Более 6 месяцев назад - показываем год
		return t.Format("Jan _2  2006")
	}
	// В последние 6 месяцев - показываем время
	return t.Format("Jan _2 15:04")
}

func formatFileName(entry os.FileInfo) string {
	name := entry.Name()
	
	// Не добавляем / для . и .. (как в оригинальном ls)
	if name == "." || name == ".." {
		return name
	}
	
	// Добавляем / для остальных директорий
	if entry.IsDir() {
		return name + "/"
	}
	
	return name
}

// formatSizeHumanReadable преобразует размер в байтах в человеко-читаемый формат
// formatSizeHumanReadable преобразует размер в байтах в человеко-читаемый формат
// formatSizeHumanReadable преобразует размер в байтах в человеко-читаемый формат
func formatSizeHumanReadable(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d", size)
	}
	
	units := []string{"K", "M", "G", "T", "P", "E"}
	
	div := unit
	exp := 0
	for n := size / unit; n >= unit && exp < len(units); n /= unit {
		div *= unit
		exp++
	}
	
	if exp == 0 {
		// Размер между 1K и 1023K
		return fmt.Sprintf("%.0fK", float64(size)/float64(unit))
	}
	
	value := float64(size) / float64(div)
	if value < 10 {
		return fmt.Sprintf("%.1f%s", value, units[exp-1])
	}
	return fmt.Sprintf("%.0f%s", value, units[exp-1])
}


func isHidden(path string) bool {
	if len(path) == 0 {
		return false
	}
	return path[0] == dotCharacter
}
