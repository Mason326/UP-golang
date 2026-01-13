package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
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

Примеры:
  %s -la               Подробный список всех файлов
  %s -lh *.go          Go файлы с читаемыми размерами
  %s -R /var/log       Рекурсивно показать содержимое /var/log
  %s --help            Показать эту справку
`, programName, programName, programName, programName, programName)
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
	var totalBlocks int64
	for _, entry := range entries {
		totalBlocks += entry.Size() / 512 // Блоки по 512 байт
	}
	if len(entries) > 0 {
		fmt.Printf("total %d\n", totalBlocks)
	}

	// Находим максимальную длину строки размера для выравнивания
	maxSizeLen := 0
	for _, entry := range entries {
		var sizeStr string
		if humanReadable {
			sizeStr = formatSizeHumanReadable(entry.Size())
		} else {
			sizeStr = fmt.Sprintf("%d", entry.Size())
		}
		if len(sizeStr) > maxSizeLen {
			maxSizeLen = len(sizeStr)
		}
	}

	for _, entry := range entries {
		// Права доступа
		mode := entry.Mode().String()

		// Время изменения (формат как в ls)
		modTime := entry.ModTime().Format("Jan _2 15:04")

		// Размер файла в нужном формате
		var sizeStr string
		if humanReadable {
			sizeStr = formatSizeHumanReadable(entry.Size())
		} else {
			sizeStr = fmt.Sprintf("%d", entry.Size())
		}

		// Имя файла/директории
		name := entry.Name()
		if entry.IsDir() {
			name = name + "/"
		}

		// Форматированный вывод с выравниванием
		fmt.Printf("%s %*s %s %s\n", mode, maxSizeLen, sizeStr, modTime, name)
	}
}

// formatSizeHumanReadable преобразует размер в байтах в человеко-читаемый формат
func formatSizeHumanReadable(size int64) string {
	if size == 0 {
		return "0"
	}

	// Единицы измерения
	units := []string{"B", "K", "M", "G", "T", "P", "E"}
	
	// Определяем основание (1024 для двоичных префиксов, как в ls -h)
	base := 1024.0
	
	// Если размер меньше 1K, показываем в байтах
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	}
	
	// Находим подходящую единицу измерения
	exp := 0
	value := float64(size)
	for value >= base && exp < len(units)-1 {
		value /= base
		exp++
	}
	
	// Форматируем вывод
	// Если значение целое, показываем без десятичной части
	if value == float64(int64(value)) {
		return fmt.Sprintf("%.0f%s", value, units[exp])
	}
	
	// Для значений больше 10 показываем без десятичной части
	if value >= 10 {
		return fmt.Sprintf("%.0f%s", value, units[exp])
	}
	
	// Для значений меньше 10 показываем с одной десятичной цифрой
	return fmt.Sprintf("%.1f%s", value, units[exp])
}

func isHidden(path string) bool {
	if len(path) == 0 {
		return false
	}
	return path[0] == dotCharacter
}
