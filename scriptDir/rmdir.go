package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

func main() {
	// Определяем флаги
	parentsFlag := flag.Bool("p", false, "remove DIRECTORY and its ancestors")
	verboseFlag := flag.Bool("v", false, "output a diagnostic for every directory processed")
	ignoreNonEmptyFlag := flag.Bool("ignore-fail-on-non-empty", false,
		"ignore each failure that is solely because a directory is non-empty")
	helpFlag := flag.Bool("help", false, "display this help and exit")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Использование: %s [КЛЮЧ]... ДИРЕКТОРИЯ...\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Удаляет пустые директории.\n\n")
		fmt.Fprintf(os.Stderr, "Ключи:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "%s: отсутствует операнд\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Попробуйте '%s --help' для получения дополнительной информации.\n", os.Args[0])
		os.Exit(1)
	}

	exitCode := 0
	for _, dir := range args {
		err := removeDirectory(dir, *parentsFlag, *verboseFlag, *ignoreNonEmptyFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s: %v\n", os.Args[0], dir, err)
			exitCode = 1
		}
	}

	os.Exit(exitCode)
}

// removeDirectory удаляет директорию с учетом флагов
func removeDirectory(path string, parents, verbose, ignoreNonEmpty bool) error {
	// Нормализуем путь
	path = filepath.Clean(path)

	if parents {
		return removeWithParentsSimple(path, verbose, ignoreNonEmpty)
	}

	return removeSingleDir(path, verbose, ignoreNonEmpty)
}

// removeSingleDir удаляет одну директорию
func removeSingleDir(path string, verbose, ignoreNonEmpty bool) error {
	// Проверяем существование
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("Not a directory")
	}

	// Проверяем, пуста ли директория
	if !isEmptyDir(path) {
		if ignoreNonEmpty {
			if verbose {
				fmt.Printf("%s: пропущена непустая директория\n", path)
			}
			return nil
		}
		return fmt.Errorf("Directory not empty")
	}

	if verbose {
		fmt.Printf("%s: удаляется директория\n", path)
	}

	return os.Remove(path)
}

// removeWithParents удаляет директорию и ее пустых предков
func removeWithParents(path string, verbose, ignoreNonEmpty bool) error {
	path = filepath.Clean(path)
	
	// Сначала убедимся, что сама директория существует и является директорией
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("Not a directory")
	}

	// Создаем стек директорий для удаления (от самой глубокой к корневой)
	var dirsToRemove []string
	
	// Начинаем с указанной директории
	current := path
	
	for {
		// Проверяем существование текущей директории
		if _, err := os.Stat(current); os.IsNotExist(err) {
			break
		}
		
		// Проверяем, является ли она директорией
		info, err := os.Stat(current)
		if err != nil || !info.IsDir() {
			break
		}
		
		// Проверяем, пуста ли директория
		if !isEmptyDir(current) {
			// Если это исходная директория, возвращаем ошибку
			if current == path {
				if ignoreNonEmpty {
					if verbose {
						fmt.Printf("%s: пропущена непустая директория\n", current)
					}
					return nil
				}
				return fmt.Errorf("Directory not empty")
			}
			// Если это не исходная директория, прерываем цепочку
			break
		}
		
		// Добавляем директорию в начало списка (чтобы получить порядок от глубокой к корневой)
		// ВАЖНО: добавляем в начало, чтобы потом просто идти по порядку
		dirsToRemove = append([]string{current}, dirsToRemove...)
		
		// Получаем родительскую директорию
		parent := filepath.Dir(current)
		
		// Если достигли корневой директории, останавливаемся
		if parent == current || isRootDir(parent) {
			break
		}
		
		current = parent
	}
	
	// Теперь удаляем директории в порядке от самой глубокой к самой верхней
	// Теперь dirsToRemove уже содержит правильный порядок
	for _, dir := range dirsToRemove {
		if verbose {
			fmt.Printf("%s: удаляется директория\n", dir)
		}
		
		err := os.Remove(dir)
		if err != nil {
			// Если не удалось удалить, возвращаем ошибку
			return err
		}
	}
	
	return nil
}

// Альтернативная, более простая версия removeWithParents
func removeWithParentsSimple(path string, verbose, ignoreNonEmpty bool) error {
	path = filepath.Clean(path)
	
	// Сначала удаляем указанную директорию
	err := removeSingleDir(path, verbose, ignoreNonEmpty)
	if err != nil {
		return err
	}
	
	// Теперь пытаемся удалить родительские директории
	parent := filepath.Dir(path)
	
	for parent != "." && parent != "/" && parent != "" && parent != path {
		// Пропускаем, если директории не существует
		if _, err := os.Stat(parent); os.IsNotExist(err) {
			break
		}
		
		// Пытаемся удалить родительскую директорию
		err := removeSingleDir(parent, verbose, ignoreNonEmpty)
		if err != nil {
			// Если не удалось удалить (скорее всего не пуста), выходим
			break
		}
		
		// Переходим к следующему родителю
		parent = filepath.Dir(parent)
	}
	
	return nil
}

// isRootDir проверяет, является ли директория корневой
func isRootDir(path string) bool {
	// Для Unix/Linux
	if path == "/" {
		return true
	}
	
	// Для Windows (C:\, D:\ и т.д.)
	if len(path) == 3 && path[1] == ':' && (path[2] == '\\' || path[2] == '/') {
		return true
	}
	
	// Для Windows без слеша (C:)
	if len(path) == 2 && path[1] == ':' {
		return true
	}
	
	return false
}

// isEmptyDir проверяет, пуста ли директория
func isEmptyDir(path string) bool {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return false
	}
	return len(files) == 0
}