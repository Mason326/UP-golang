package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	args := os.Args[1:]
	
	if len(args) == 0 {
		// !! без аргументов
		runBangBang(nil)
		return
	}
	
	// Проверяем специальные флаги
	switch args[0] {
	case "-h", "--help":
		printHelp()
		return
	case "-v", "--version":
		fmt.Println("!! 1.0")
		return
	}
	
	// Проверяем первый аргумент
	firstArg := args[0]
	
	if firstArg == "!!" {
		runBangBang(args[1:])
	} else if strings.HasPrefix(firstArg, "!!") && len(firstArg) > 2 {
		numStr := firstArg[2:]
		n, err := strconv.Atoi(numStr)
		if err != nil || n <= 0 {
			fmt.Fprintf(os.Stderr, "Invalid number: %s\n", numStr)
			printHelp()
			os.Exit(1)
		}
		runBangBangN(n, args[1:])
	} else {
		// Все аргументы - дополнительные к последней команде
		runBangBang(args)
	}
}

func runBangBang(extraArgs []string) {
	cmd := getLastCommand()
	if cmd == "" {
		fmt.Fprintln(os.Stderr, "No previous command found")
		os.Exit(1)
	}
	
	cmd = modifyCommand(cmd, extraArgs)
	
	// Проверяем, является ли команда cd
	if isCdCommand(cmd) {
		fmt.Printf("!!: %s\n", cmd)
		executeCdCommand(cmd)
	} else {
		fmt.Printf("!!: %s\n", cmd)
		executeRegularCommand(cmd)
	}
}

func runBangBangN(n int, extraArgs []string) {
	cmd := getNthCommand(n)
	if cmd == "" {
		fmt.Fprintf(os.Stderr, "Command %d not found\n", n)
		os.Exit(1)
	}
	
	cmd = modifyCommand(cmd, extraArgs)
	
	if isCdCommand(cmd) {
		fmt.Printf("!!%d: %s\n", n, cmd)
		executeCdCommand(cmd)
	} else {
		fmt.Printf("!!%d: %s\n", n, cmd)
		executeRegularCommand(cmd)
	}
}

func isCdCommand(cmd string) bool {
	parts := strings.Fields(cmd)
	return len(parts) > 0 && parts[0] == "cd"
}

func executeCdCommand(cmd string) {
	parts := strings.Fields(cmd)
	if len(parts) < 2 {
		// cd без аргументов - домашняя директория
		executeCdToHome()
		return
	}
	
	target := parts[1]
	
	// Обрабатываем специальные символы
	resolvedPath, err := resolveCdPath(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cd: %v\n", err)
		os.Exit(1)
	}
	
	// Меняем директорию для текущего процесса
	if err := os.Chdir(resolvedPath); err != nil {
		fmt.Fprintf(os.Stderr, "cd: %v\n", err)
		os.Exit(1)
	}
	
	// Обновляем переменные окружения
	updatePwdEnvironment()
	
	// Запускаем новый shell в новой директории
	launchShellInNewDirectory()
}

func resolveCdPath(target string) (string, error) {
	// Обработка специальных символов
	
	// ~ - домашняя директория
	if target == "~" {
		return getHomeDir(), nil
	}
	
	// ~/path
	if strings.HasPrefix(target, "~/") {
		return filepath.Join(getHomeDir(), target[2:]), nil
	}
	
	// - - предыдущая директория
	if target == "-" {
		oldpwd := os.Getenv("OLDPWD")
		if oldpwd == "" {
			return "", fmt.Errorf("OLDPWD not set")
		}
		return oldpwd, nil
	}
	
	// .. - родительская директория
	if target == ".." {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return filepath.Dir(cwd), nil
	}
	
	// . - текущая директория
	if target == "." {
		return os.Getwd()
	}
	
	// Относительный путь
	if !filepath.IsAbs(target) {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return filepath.Join(cwd, target), nil
	}
	
	// Абсолютный путь
	return target, nil
}

func getHomeDir() string {
	home := os.Getenv("HOME")
	if home != "" {
		return home
	}
	
	usr, err := user.Current()
	if err != nil {
		return "/"
	}
	
	return usr.HomeDir
}

func executeCdToHome() {
	home := getHomeDir()
	if err := os.Chdir(home); err != nil {
		fmt.Fprintf(os.Stderr, "cd: %v\n", err)
		os.Exit(1)
	}
	updatePwdEnvironment()
	launchShellInNewDirectory()
}

func updatePwdEnvironment() {
	// Сохраняем старую PWD
	oldPwd := os.Getenv("PWD")
	
	// Получаем новую PWD
	newPwd, err := os.Getwd()
	if err != nil {
		return
	}
	
	// Устанавливаем переменные окружения
	if oldPwd != "" {
		os.Setenv("OLDPWD", oldPwd)
	}
	os.Setenv("PWD", newPwd)
}

func launchShellInNewDirectory() {
	// Получаем текущий shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}
	
	// Запускаем интерактивный shell
	cmd := exec.Command(shell, "-i")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	// Устанавливаем текущую директорию
	cmd.Dir, _ = os.Getwd()
	
	// Запускаем
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Error launching shell: %v\n", err)
		os.Exit(1)
	}
	
	// Выходим после завершения shell
	os.Exit(0)
}

func executeRegularCommand(cmd string) {
	shell := "/bin/bash"
	execCmd := exec.Command(shell, "-c", cmd)
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	
	err := execCmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func modifyCommand(originalCmd string, extraArgs []string) string {
	if len(extraArgs) == 0 {
		return originalCmd
	}
	
	// Простая логика: если один аргумент и он не флаг, заменяем последний
	if len(extraArgs) == 1 && len(originalCmd) > 0 {
		arg := extraArgs[0]
		parts := strings.Fields(originalCmd)
		
		if len(parts) > 1 && !strings.HasPrefix(arg, "-") {
			// Заменяем последнюю часть
			parts[len(parts)-1] = arg
			return strings.Join(parts, " ")
		}
	}
	
	// Иначе добавляем в конец
	return originalCmd + " " + strings.Join(extraArgs, " ")
}

func getLastCommand() string {
	return getNthCommand(1)
}

func getNthCommand(n int) string {
	history := readHistory()
	if len(history) == 0 {
		return ""
	}
	
	if n > len(history) {
		return ""
	}
	
	return history[len(history)-n]
}

func readHistory() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return []string{}
	}
	
	// Пробуем разные файлы истории
	files := []string{
		filepath.Join(home, ".bash_history"),
		filepath.Join(home, ".zsh_history"),
	}
	
	for _, historyFile := range files {
		file, err := os.Open(historyFile)
		if err != nil {
			continue
		}
		
		var commands []string
		scanner := bufio.NewScanner(file)
		
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			
			// Пропускаем таймстемпы
			if strings.HasPrefix(line, "#") && len(line) > 1 {
				if _, err := strconv.ParseInt(line[1:], 10, 64); err == nil {
					continue
				}
			}
			
			// Парсим zsh формат
			if strings.HasPrefix(line, ": ") {
				parts := strings.SplitN(line, ";", 2)
				if len(parts) == 2 {
					line = parts[1]
				}
			}
			
			commands = append(commands, line)
		}
		
		file.Close()
		
		if len(commands) > 0 {
			return commands
		}
	}
	
	return []string{}
}

func printHelp() {
	fmt.Println("Usage: !! [!!n] [args...]")
	fmt.Println()
	fmt.Println("Repeat last command with optional arguments.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  !!          # repeat last command")
	fmt.Println("  !! -la      # add -la to last command")
	fmt.Println("  !! /var     # replace last argument with /var")
	fmt.Println("  !! --help   # show this help")
	fmt.Println()
}
