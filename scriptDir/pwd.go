package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	// Определяем флаги - делаем оба false по умолчанию
	logicalFlag := flag.Bool("L", false, "use PWD from environment, even if it contains symlinks")
	physicalFlag := flag.Bool("P", false, "avoid all symlinks")
	helpFlag := flag.Bool("help", false, "display this help and exit")
	
	// Добавляем поддержку --help
	flag.CommandLine.Init("pwd", flag.ContinueOnError)
	
	// Кастомное использование
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Использование: %s [КЛЮЧ]...\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Выводит полное имя текущей рабочей директории.\n\n")
		fmt.Fprintf(os.Stderr, "Ключи:\n")
		fmt.Fprintf(os.Stderr, "  -L, --logical    использовать PWD из окружения, даже если содержит симлинки\n")
		fmt.Fprintf(os.Stderr, "  -P, --physical   избегать всех симлинков\n")
		fmt.Fprintf(os.Stderr, "      --help       показать эту справку и выйти\n")
	}
	
	// Парсим аргументы с учетом длинных опций
	// Проверяем наличие --help до парсинга флагов
	for _, arg := range os.Args[1:] {
		if arg == "--help" {
			flag.Usage()
			return
		}
		if arg == "--logical" {
			*logicalFlag = true
		}
		if arg == "--physical" {
			*physicalFlag = true
		}
	}
	
	// Парсим обычные флаги
	flag.Parse()
	
	// Проверяем флаг help
	if *helpFlag {
		flag.Usage()
		return
	}
	
	// Проверяем конфликтующие флаги
	if *logicalFlag && *physicalFlag {
		fmt.Fprintf(os.Stderr, "%s: невозможно использовать одновременно -L и -P\n", os.Args[0])
		os.Exit(1)
	}
	
	// Определяем, какой метод использовать
	// По умолчанию ведем себя как -L в GNU и как -P в POSIX режиме
	useLogical := true // По умолчанию как -L
	
	if *physicalFlag {
		useLogical = false
	} else if *logicalFlag {
		useLogical = true
	} else {
		// Если флаги не указаны вообще, проверяем POSIXLY_CORRECT
		if os.Getenv("POSIXLY_CORRECT") != "" {
			useLogical = false // По умолчанию для POSIX используется -P
		} else {
			useLogical = true // По умолчанию для GNU используется -L
		}
	}
	
	// Получаем путь
	var dir string
	var err error
	
	if useLogical {
		// Логический путь (может содержать симлинки)
		dir, err = getLogicalPath()
	} else {
		// Физический путь (без симлинков)
		dir, err = getPhysicalPath()
	}
	
	if err != nil {
		fmt.Fprintf(os.Stderr, "pwd: %v\n", err)
		os.Exit(1)
	}
	
	// Выводим путь
	fmt.Println(dir)
}

// getLogicalPath возвращает логический путь (может содержать симлинки)
func getLogicalPath() (string, error) {
	// Сначала пытаемся получить из переменной окружения PWD
	if pwd := os.Getenv("PWD"); pwd != "" {
		// Проверяем, что PWD является абсолютным путем
		if filepath.IsAbs(pwd) {
			// Проверяем, что PWD существует
			if _, err := os.Stat(pwd); err == nil {
				// Сравниваем с текущим рабочим каталогом
				if isSameDir(pwd) {
					return filepath.Clean(pwd), nil
				}
			}
		}
	}
	
	// Если PWD не подходит, используем os.Getwd
	return os.Getwd()
}

// getPhysicalPath возвращает физический путь (разрешает все симлинки)
func getPhysicalPath() (string, error) {
	// Получаем абсолютный путь, разрешая все симлинки
	absPath, err := filepath.Abs(".")
	if err != nil {
		return "", err
	}
	
	// Разрешаем все симлинки в пути
	physicalPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", err
	}
	
	// Очищаем путь (убираем . и ..)
	return filepath.Clean(physicalPath), nil
}

// isSameDir проверяет, указывает ли путь на ту же директорию
func isSameDir(path string) bool {
	absPath, err1 := filepath.Abs(path)
	currentDir, err2 := os.Getwd()
	
	if err1 != nil || err2 != nil {
		return false
	}

	// Разрешаем симлинки для сравнения
	absPathResolved, err1 := filepath.EvalSymlinks(absPath)
	currentDirResolved, err2 := filepath.EvalSymlinks(currentDir)
	
	if err1 != nil || err2 != nil {
		return false
	}

	// Приводим пути к каноническому виду
	absPathResolved = filepath.Clean(absPathResolved)
	currentDirResolved = filepath.Clean(currentDirResolved)
	
	// Сравниваем
	return absPathResolved == currentDirResolved
}
