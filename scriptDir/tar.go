package main

import (
	"archive/tar"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	// Основные флаги
	create := flag.Bool("c", false, "создать новый архив")
	extract := flag.Bool("x", false, "извлечь файлы из архива")
	list := flag.Bool("t", false, "вывести список файлов в архиве")
	verbose := flag.Bool("v", false, "подробный вывод (verbose)")
	gzipFlag := flag.Bool("z", false, "использовать сжатие gzip")
	file := flag.String("f", "", "имя архивного файла (обязательный параметр)")
	help := flag.Bool("help", false, "показать справку")

	// Настраиваем вывод справки
	flag.Usage = func() {
		printHelp()
	}

	flag.Parse()

	// Обрабатываем флаг help
	if *help {
		printHelp()
		return
	}

	// Проверяем обязательный параметр -f
	if *file == "" {
		fmt.Fprintln(os.Stderr, "Ошибка: необходимо указать имя архивного файла с помощью -f")
		fmt.Fprintln(os.Stderr, "Используйте 'tar --help' для получения дополнительной информации.")
		os.Exit(1)
	}

	// Проверяем, что указана ровно одна операция
	ops := 0
	if *create {
		ops++
	}
	if *extract {
		ops++
	}
	if *list {
		ops++
	}

	if ops != 1 {
		fmt.Fprintln(os.Stderr, "Ошибка: необходимо указать ровно одну операцию: -c, -x или -t")
		fmt.Fprintln(os.Stderr, "Используйте 'tar --help' для получения дополнительной информации.")
		os.Exit(1)
	}

	// Получаем список файлов для операций
	var files []string
	if *create {
		files = flag.Args()
		if len(files) == 0 {
			files = []string{"."} // По умолчанию архивируем текущую директорию
		}
	}

	// Выполняем операцию
	var err error
	if *create {
		err = createArchive(*file, files, *verbose, *gzipFlag)
	} else if *extract {
		err = extractArchive(*file, *verbose, *gzipFlag)
	} else if *list {
		err = listArchive(*file, *verbose, *gzipFlag)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}
}

func createArchive(filename string, files []string, verbose, gzipCompress bool) error {
	// Создаем выходной файл
	out, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("не удалось создать файл %s: %v", filename, err)
	}
	defer out.Close()

	var writer io.Writer = out

	// Добавляем gzip сжатие если нужно
	if gzipCompress {
		gzWriter := gzip.NewWriter(writer)
		defer gzWriter.Close()
		writer = gzWriter
		if verbose {
			fmt.Println("Используется сжатие gzip")
		}
	}

	tw := tar.NewWriter(writer)
	defer tw.Close()

	if verbose {
		fmt.Printf("Создание архива: %s\n", filename)
		fmt.Println("Добавляемые файлы:")
	}

	addedCount := 0
	totalSize := int64(0)

	// Обрабатываем каждый файл/шаблон
	for _, pattern := range files {
		// Разворачиваем шаблоны
		matches, err := filepath.Glob(pattern)
		if err != nil {
			// Если не шаблон, то используем как есть
			matches = []string{pattern}
		}

		for _, filePath := range matches {
			if err := addToArchive(tw, filePath, "", verbose, &addedCount, &totalSize); err != nil {
				return fmt.Errorf("ошибка при добавлении %s: %v", filePath, err)
			}
		}
	}

	if verbose {
		fmt.Printf("\nАрхив создан: %s\n", filename)
		fmt.Printf("Добавлено файлов: %d\n", addedCount)
		fmt.Printf("Общий размер: %s\n", formatBytes(totalSize))
	} else {
		fmt.Printf("Архив создан: %s\n", filename)
	}

	return nil
}

func addToArchive(tw *tar.Writer, path, basePath string, verbose bool, count *int, totalSize *int64) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}

	// Определяем имя в архиве
	nameInArchive := path
	if basePath != "" {
		rel, err := filepath.Rel(basePath, path)
		if err == nil {
			nameInArchive = rel
		}
	}

	// Создаем header
	var header *tar.Header
	var linkTarget string

	if info.Mode()&os.ModeSymlink != 0 {
		// Для символических ссылок получаем цель
		linkTarget, err = os.Readlink(path)
		if err != nil {
			return err
		}
		header, err = tar.FileInfoHeader(info, linkTarget)
	} else {
		header, err = tar.FileInfoHeader(info, "")
	}
	
	if err != nil {
		return err
	}

	// Используем относительный путь в архиве
	header.Name = nameInArchive

	// Записываем header
	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	// Выводим информацию если нужно
	if verbose {
		modeStr := info.Mode().String()
		size := info.Size()
		if info.IsDir() {
			size = 0
		}
		timeStr := info.ModTime().Format("2006-01-02 15:04")
		
		fmt.Printf("%10s %8d %s %s\n", modeStr, size, timeStr, nameInArchive)
	}

	// Если это не обычный файл, не пишем данные
	if !info.Mode().IsRegular() {
		if info.IsDir() {
			// Рекурсивно обрабатываем содержимое директории
			dir, err := os.Open(path)
			if err != nil {
				return err
			}
			defer dir.Close()

			entries, err := dir.Readdir(0)
			if err != nil {
				return err
			}

			for _, entry := range entries {
				fullPath := filepath.Join(path, entry.Name())
				if err := addToArchive(tw, fullPath, basePath, verbose, count, totalSize); err != nil {
					return err
				}
			}
		}
		*count++
		return nil
	}

	// Открываем файл для чтения
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Копируем содержимое
	written, err := io.Copy(tw, file)
	if err != nil {
		return err
	}

	*totalSize += written
	*count++
	return nil
}

func extractArchive(filename string, verbose, gzipDecompress bool) error {
	// Открываем архивный файл
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("не удалось открыть архив %s: %v", filename, err)
	}
	defer file.Close()

	var reader io.Reader = file

	// Добавляем gzip декомпрессию если нужно
	if gzipDecompress {
		gzReader, err := gzip.NewReader(reader)
		if err != nil {
			return fmt.Errorf("ошибка при распаковке gzip: %v", err)
		}
		defer gzReader.Close()
		reader = gzReader
		if verbose {
			fmt.Println("Используется распаковка gzip")
		}
	}

	tr := tar.NewReader(reader)

	if verbose {
		fmt.Printf("Извлечение архива: %s\n", filename)
	}

	extractedCount := 0
	totalSize := int64(0)

	// Извлекаем файлы
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("ошибка чтения архива: %v", err)
		}

		// Извлекаем файл
		size, err := extractFile(tr, header, verbose)
		if err != nil {
			return fmt.Errorf("ошибка при извлечении %s: %v", header.Name, err)
		}

		extractedCount++
		totalSize += size
	}

	if verbose {
		fmt.Printf("\nАрхив извлечен: %s\n", filename)
		fmt.Printf("Извлечено файлов: %d\n", extractedCount)
		fmt.Printf("Общий размер: %s\n", formatBytes(totalSize))
	} else {
		fmt.Printf("Архив извлечен: %s\n", filename)
	}

	return nil
}

func extractFile(tr *tar.Reader, header *tar.Header, verbose bool) (int64, error) {
	// Проверяем безопасность пути
	if !isSafePath(header.Name) {
		return 0, fmt.Errorf("небезопасный путь: %s", header.Name)
	}

	// Создаем все родительские директории
	dir := filepath.Dir(header.Name)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return 0, err
		}
	}

	var size int64

	// Обрабатываем в зависимости от типа файла
	switch header.Typeflag {
	case tar.TypeDir:
		// Создаем директорию
		if err := os.MkdirAll(header.Name, os.FileMode(header.Mode)); err != nil {
			return 0, err
		}
		if verbose {
			fmt.Printf("Создана директория: %s/\n", header.Name)
		}

	case tar.TypeReg, tar.TypeRegA:
		// Создаем обычный файл
		file, err := os.OpenFile(header.Name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
		if err != nil {
			return 0, err
		}
		defer file.Close()

		// Копируем содержимое
		written, err := io.Copy(file, tr)
		if err != nil {
			return 0, err
		}
		size = written

		if verbose {
			fmt.Printf("Извлечен файл: %s (%s)\n", header.Name, formatBytes(size))
		}

	case tar.TypeSymlink:
		// Создаем символическую ссылку
		if err := os.Symlink(header.Linkname, header.Name); err != nil {
			return 0, err
		}
		if verbose {
			fmt.Printf("Создана ссылка: %s -> %s\n", header.Name, header.Linkname)
		}

	case tar.TypeLink:
		// Создаем жесткую ссылку
		if err := os.Link(header.Linkname, header.Name); err != nil {
			return 0, err
		}
		if verbose {
			fmt.Printf("Создана жесткая ссылка: %s -> %s\n", header.Name, header.Linkname)
		}

	default:
		if verbose {
			fmt.Printf("Пропущен: %s (тип: %c)\n", header.Name, header.Typeflag)
		}
	}

	// Устанавливаем время модификации
	if !header.ModTime.IsZero() {
		os.Chtimes(header.Name, time.Now(), header.ModTime)
	}

	return size, nil
}

func isSafePath(path string) bool {
	// Проверяем, что путь не содержит опасных элементов
	cleaned := filepath.Clean(path)
	
	// Запрещаем абсолютные пути и переходы наверх
	if filepath.IsAbs(path) || strings.HasPrefix(cleaned, "..") {
		return false
	}
	
	// Проверяем каждый компонент пути
	parts := strings.Split(cleaned, string(filepath.Separator))
	for _, part := range parts {
		if part == ".." {
			return false
		}
	}
	
	return true
}

func listArchive(filename string, verbose, gzipDecompress bool) error {
	// Открываем архивный файл
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("не удалось открыть архив %s: %v", filename, err)
	}
	defer file.Close()

	var reader io.Reader = file

	// Добавляем gzip декомпрессию если нужно
	if gzipDecompress {
		gzReader, err := gzip.NewReader(reader)
		if err != nil {
			return fmt.Errorf("ошибка при распаковке gzip: %v", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	tr := tar.NewReader(reader)

	fmt.Printf("Содержимое архива: %s\n", filename)
	fmt.Println()

	// Заголовок таблицы
	if verbose {
		fmt.Println("Права      Владелец Группа    Размер     Дата                Имя")
		fmt.Println("---------  -------  -------  ----------  ------------------  ----")
	} else {
		fmt.Println("Тип  Размер     Дата                Имя")
		fmt.Println("---  ----------  ------------------  ----")
	}

	totalFiles := 0
	totalSize := int64(0)

	// Выводим информацию о файлах
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("ошибка чтения архива: %v", err)
		}

		// Выводим информацию о файле
		printFileInfo(header, verbose)

		totalFiles++
		if header.Typeflag == tar.TypeReg || header.Typeflag == tar.TypeRegA {
			totalSize += header.Size
		}

		// Пропускаем содержимое файла
		if header.Typeflag == tar.TypeReg || header.Typeflag == tar.TypeRegA {
			io.Copy(io.Discard, tr)
		}
	}

	fmt.Println()
	fmt.Printf("Всего файлов: %d\n", totalFiles)
	fmt.Printf("Общий размер: %s\n", formatBytes(totalSize))

	return nil
}

func printFileInfo(header *tar.Header, verbose bool) {
	// Определяем тип файла
	var typeChar string
	switch header.Typeflag {
	case tar.TypeDir:
		typeChar = "d"
	case tar.TypeReg, tar.TypeRegA:
		typeChar = "-"
	case tar.TypeSymlink:
		typeChar = "l"
	case tar.TypeLink:
		typeChar = "h"
	case tar.TypeChar:
		typeChar = "c"
	case tar.TypeBlock:
		typeChar = "b"
	case tar.TypeFifo:
		typeChar = "p"
	default:
		typeChar = "?"
	}

	if verbose {
		// Подробный вывод
		mode := fmt.Sprintf("%s%s", typeChar, formatMode(header.Mode))
		owner := fmt.Sprintf("%d", header.Uid)
		group := fmt.Sprintf("%d", header.Gid)
		size := formatBytes(header.Size)
		date := header.ModTime.Format("2006-01-02 15:04")
		name := header.Name

		// Добавляем информацию о ссылках
		if header.Typeflag == tar.TypeSymlink {
			name = fmt.Sprintf("%s -> %s", name, header.Linkname)
		} else if header.Typeflag == tar.TypeDir {
			name = name + "/"
		}

		fmt.Printf("%-10s %7s %7s %10s  %s  %s\n", 
			mode, owner, group, size, date, name)
	} else {
		// Простой вывод
		var typeStr string
		switch header.Typeflag {
		case tar.TypeDir:
			typeStr = "DIR"
		case tar.TypeReg, tar.TypeRegA:
			typeStr = "FILE"
		case tar.TypeSymlink:
			typeStr = "LINK"
		case tar.TypeLink:
			typeStr = "HLNK"
		default:
			typeStr = "????"
		}

		size := formatBytes(header.Size)
		date := header.ModTime.Format("2006-01-02 15:04")
		name := header.Name

		// Добавляем информацию о ссылках
		if header.Typeflag == tar.TypeSymlink {
			name = fmt.Sprintf("%s -> %s", name, header.Linkname)
		} else if header.Typeflag == tar.TypeDir {
			name = name + "/"
		}

		fmt.Printf("%-4s %10s  %s  %s\n", typeStr, size, date, name)
	}
}

func formatMode(mode int64) string {
	perms := []byte("rwxrwxrwx")
	for i := range perms {
		if mode&(1<<uint(8-i)) == 0 {
			perms[i] = '-'
		}
	}
	return string(perms)
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func printHelp() {
	fmt.Println("Использование: tar [ОПЦИИ] АРХИВ [ФАЙЛЫ...]")
	fmt.Println()
	fmt.Println("Утилита для создания, извлечения и просмотра tar архивов")
	fmt.Println()
	fmt.Println("Основные режимы работы (укажите один):")
	fmt.Println("  -c, --create    Создать новый архив")
	fmt.Println("  -x, --extract   Извлечь файлы из архива")
	fmt.Println("  -t, --list      Вывести список файлов в архиве")
	fmt.Println()
	fmt.Println("Опции:")
	fmt.Println("  -f, --file=АРХИВ  Использовать архивный файл АРХИВ (обязательно)")
	fmt.Println("  -v, --verbose     Подробный вывод обрабатываемых файлов")
	fmt.Println("      --help        Показать эту справку и выйти")
	fmt.Println()
	fmt.Println("Аргументы:")
	fmt.Println("  АРХИВ    Имя архивного файла")
	fmt.Println("  ФАЙЛЫ    Файлы и директории для архивации (только с -c)")
	fmt.Println("           Поддерживаются шаблоны (wildcards): *.txt, dir/* и т.д.")
	fmt.Println()
}
