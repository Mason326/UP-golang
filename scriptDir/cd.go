package main

import (
	"flag"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

func main() {
	// Флаги
	printPath := flag.Bool("P", false, "print physical path")
	evalMode := flag.Bool("eval", false, "output shell eval commands")
	help := flag.Bool("help", false, "show help")
	version := flag.Bool("version", false, "show version")
	
	flag.Parse()
	
	if *help {
		printFullHelp()
		return
	}
	
	if *version {
		fmt.Println("cd-go 1.0")
		return
	}
	
	// Получаем целевой каталог
	var target string
	args := flag.Args()
	
	if len(args) > 0 {
		target = args[0]
	} else {
		target = "~"
	}
	
	// Разрешаем путь
	newDir, oldDir, err := resolveDirectory(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cd: %v\n", err)
		os.Exit(1)
	}
	
	// Режим eval - выводим команды для shell
	if *evalMode {
		outputEvalCommands(oldDir, newDir, *printPath)
		return
	}
	
	// Обычный режим - просто выводим путь
	if *printPath || target == "-" {
		fmt.Println(newDir)
	} else {
		// В обычном режиме просто выводим путь
		// Shell функция будет использовать его
		fmt.Println(newDir)
	}
}

func resolveDirectory(target string) (newDir, oldDir string, err error) {
	// Получаем текущую директорию
	oldDir, _ = os.Getwd()
	
	// Разрешаем специальные символы
	resolved, err := expandPath(target)
	if err != nil {
		return "", oldDir, err
	}
	
	// Получаем абсолютный путь
	absPath, err := filepath.Abs(resolved)
	if err != nil {
		return "", oldDir, fmt.Errorf("%s: %v", target, err)
	}
	
	// Проверяем директорию
	if err := verifyDirectory(absPath); err != nil {
		return "", oldDir, fmt.Errorf("%s: %v", target, err)
	}
	
	return absPath, oldDir, nil
}

func expandPath(path string) (string, error) {
	// ~
	if path == "~" || path == "" {
		return getHomeDir(), nil
	}
	
	// ~/path
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(getHomeDir(), path[2:]), nil
	}
	
	// ~username/path
	if strings.HasPrefix(path, "~") {
		parts := strings.SplitN(path[1:], "/", 2)
		username := parts[0]
		
		usr, err := user.Lookup(username)
		if err != nil {
			return "", fmt.Errorf("%s: no such user", username)
		}
		
		if len(parts) > 1 {
			return filepath.Join(usr.HomeDir, parts[1]), nil
		}
		return usr.HomeDir, nil
	}
	
	// -
	if path == "-" {
		oldpwd := os.Getenv("OLDPWD")
		if oldpwd == "" {
			return "", fmt.Errorf("OLDPWD not set")
		}
		return oldpwd, nil
	}
	
	return path, nil
}

func getHomeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	
	return "/"
}

func verifyDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("No such file or directory")
		}
		return err
	}
	
	if !info.IsDir() {
		return fmt.Errorf("Not a directory")
	}
	
	// Проверяем права на выполнение
	if info.Mode().Perm()&0111 == 0 {
		return fmt.Errorf("Permission denied")
	}
	
	return nil
}

func outputEvalCommands(oldDir, newDir string, printPath bool) {
	// Выводим команды которые можно выполнить через eval
	commands := []string{
		fmt.Sprintf("export OLDPWD='%s'", oldDir),
		fmt.Sprintf("export PWD='%s'", newDir),
		fmt.Sprintf("builtin cd '%s'", newDir),
	}
	
	if printPath {
		commands = append([]string{fmt.Sprintf("echo '%s'", newDir)}, commands...)
	}
	
	fmt.Println(strings.Join(commands, "; "))
}

func printFullHelp() {
	fmt.Println()
	fmt.Println("Usage: cd [OPTIONS] [DIRECTORY]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -P        Print the physical directory")
	fmt.Println("  -help     Show this help")
	fmt.Println("  -version  Show version")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ./cd /tmp                 # Output directory path")
	fmt.Println("  ./cd -P ~                 # Print physical home directory")
	fmt.Println()
}
