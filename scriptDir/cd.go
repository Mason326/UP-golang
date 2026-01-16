package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

func main() {
	// Определяем флаги
	printHelpFlag := flag.Bool("help", false, "display this help and exit")
	printVersionFlag := flag.Bool("version", false, "output version information and exit")
	noPrintFlag := flag.Bool("P", false, "do not print the directory name")
	
	flag.Usage = func() {
		printHelp()
	}
	
	// Проверяем наличие --help до парсинга флагов
	for _, arg := range os.Args[1:] {
		if arg == "--help" {
			printHelp()
			return
		}
		if arg == "--version" {
			printVersion()
			return
		}
	}
	
	flag.Parse()
	
	if *printHelpFlag {
		printHelp()
		return
	}
	
	if *printVersionFlag {
		printVersion()
		return
	}
	
	// Получаем аргументы
	args := flag.Args()
	
	// Определяем целевой путь
	var target string
	if len(args) == 0 {
		// Без аргументов - домашняя директория
		target = "~"
	} else {
		target = args[0]
	}
	
	// Выполняем смену директории
	if err := changeDirectory(target, *noPrintFlag); err != nil {
		fmt.Fprintf(os.Stderr, "cd: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Printf(`Использование: %s [ДИРЕКТОРИЯ]

Изменяет текущую рабочую директорию, заменяя текущий shell.

Если ДИРЕКТОРИЯ не указана, переходит в домашнюю директорию.

Специальные значения:
  -              перейти в предыдущую рабочую директорию
  ~              домашняя директория текущего пользователя
  ~username      домашняя директория пользователя username

Ключи:
  -P             не выводить имя директории (для -)
      --help     показать эту справку и выйти
      --version  показать информацию о версии и выйти

Примеры:
  %s /tmp        # перейти в /tmp
  %s ~          # перейти в домашнюю директорию
  %s -          # вернуться в предыдущую директорию
`, os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}

func printVersion() {
	fmt.Printf("%s (go-cd) 1.0\n", os.Args[0])
	fmt.Println("Реализация cd с использованием exec")
}

func changeDirectory(target string, noPrint bool) error {
	// Раскрываем тильду
	path, err := expandPath(target)
	if err != nil {
		return err
	}
	
	// Получаем абсолютный путь
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	
	// Проверяем существование
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("%s: %v", target, err)
	}
	
	if !info.IsDir() {
		return fmt.Errorf("%s: не является директорией", target)
	}
	
	// Специальный случай для "-" (предыдущая директория)
	if target == "-" {
		oldpwd := os.Getenv("OLDPWD")
		if oldpwd == "" {
			return fmt.Errorf("OLDPWD не установлена")
		}
		absPath = oldpwd
		if !noPrint {
			fmt.Println(absPath)
		}
	}
	
	// Получаем текущую директорию
	currentDir, err := os.Getwd()
	if err != nil {
		currentDir = os.Getenv("PWD")
	}
	
	// Устанавливаем OLDPWD в окружении
	os.Setenv("OLDPWD", currentDir)
	
	// Меняем директорию в текущем процессе
	if err := os.Chdir(absPath); err != nil {
		return fmt.Errorf("не удалось сменить директорию: %v", err)
	}
	
	// Устанавливаем PWD в окружении
	os.Setenv("PWD", absPath)
	
	// Получаем текущий shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}
	
	// Используем exec для замены текущего процесса новым shell
	// с сохранением всех переменных окружения
	cmd := exec.Command(shell)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	// Устанавливаем директорию для новой команды
	cmd.Dir = absPath
	
	// Копируем все переменные окружения
	cmd.Env = os.Environ()
	
	// Заменяем текущий процесс новым shell
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка запуска shell: %v", err)
	}
	
	// Этот код никогда не выполнится из-за exec
	return nil
}

func expandPath(path string) (string, error) {
	// Специальный случай для "-"
	if path == "-" {
		oldpwd := os.Getenv("OLDPWD")
		if oldpwd == "" {
			return "", fmt.Errorf("OLDPWD не установлена")
		}
		return oldpwd, nil
	}
	
	// Раскрываем тильду
	if strings.HasPrefix(path, "~") {
		return expandTilde(path)
	}
	
	return path, nil
}

func expandTilde(path string) (string, error) {
	// ~
	if path == "~" {
		home := os.Getenv("HOME")
		if home == "" {
			if u, err := user.Current(); err == nil {
				return u.HomeDir, nil
			}
			return "", fmt.Errorf("HOME не установлена")
		}
		return home, nil
	}
	
	// ~/something
	if strings.HasPrefix(path, "~/") {
		home := os.Getenv("HOME")
		if home == "" {
			if u, err := user.Current(); err == nil {
				home = u.HomeDir
			} else {
				return "", fmt.Errorf("HOME не установлена")
			}
		}
		return filepath.Join(home, path[2:]), nil
	}
	
	// ~username
	if strings.HasPrefix(path, "~") && !strings.HasPrefix(path, "~/") {
		username := path[1:]
		if strings.Contains(username, "/") {
			username = strings.Split(username, "/")[0]
		}
		
		// Ищем пользователя
		u, err := user.Lookup(username)
		if err != nil {
			return "", fmt.Errorf("пользователь %s не найден", username)
		}
		
		// Если путь содержит поддиректорию
		if strings.Contains(path, "/") {
			return filepath.Join(u.HomeDir, strings.SplitN(path, "/", 2)[1]), nil
		}
		
		return u.HomeDir, nil
	}
	
	return path, nil
}
