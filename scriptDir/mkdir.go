package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

func main() {
	// Определяем флаги
	parentsFlag := flag.Bool("p", false, "no error if existing, make parent directories as needed")
	modeFlag := flag.String("m", "", "set file mode (as in chmod), not a=rwx - umask")
	verboseFlag := flag.Bool("v", false, "print a message for each created directory")
	helpFlag := flag.Bool("help", false, "display this help and exit")
	
	// Кастомное использование
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Использование: %s [КЛЮЧ]... ДИРЕКТОРИЯ...\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Создает указанные директории.\n\n")
		fmt.Fprintf(os.Stderr, "Ключи:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nПо умолчанию используется режим 0755 (rwxr-xr-x).\n")
	}
	
	flag.Parse()
	
	// Проверяем флаг help
	if *helpFlag {
		flag.Usage()
		return
	}
	
	// Проверяем, указаны ли директории
	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "%s: отсутствует операнд\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Попробуйте '%s --help' для получения дополнительной информации.\n", os.Args[0])
		os.Exit(1)
	}
	
	// Определяем режим доступа
	mode := os.ModeDir | 0755 // Режим по умолчанию
	if *modeFlag != "" {
		// Парсим режим (восьмеричное число)
		modeValue, err := strconv.ParseUint(*modeFlag, 8, 32)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: неверный режим '%s'\n", os.Args[0], *modeFlag)
			os.Exit(1)
		}
		mode = os.ModeDir | os.FileMode(modeValue)
	}
	
	// Создаем директории
	exitCode := 0
	for _, dir := range args {
		err := createDirectory(dir, *parentsFlag, mode, *verboseFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: невозможно создать директорию '%s': %v\n", 
				os.Args[0], dir, err)
			exitCode = 1
		}
	}
	
	os.Exit(exitCode)
}

// createDirectory создает директорию с учетом флагов
func createDirectory(path string, parents bool, mode os.FileMode, verbose bool) error {
	if parents {
		// Создаем все родительские директории при необходимости
		err := os.MkdirAll(path, mode)
		if err == nil && verbose {
			fmt.Printf("mkdir: создана директория '%s'\n", path)
		}
		return err
	} else {
		// Создаем только указанную директорию
		err := os.Mkdir(path, mode)
		if err == nil && verbose {
			fmt.Printf("mkdir: создана директория '%s'\n", path)
		}
		return err
	}
}
