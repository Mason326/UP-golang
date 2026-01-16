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
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	arg := os.Args[1]
	
	if arg == "-h" || arg == "--help" {
		printHelp()
		return
	}
	
	if arg == "-v" || arg == "--version" {
		fmt.Println("!n 1.0")
		return
	}
	
	extraArgs := os.Args[2:]
	
	// Получаем команду из истории
	cmd, err := getHistoryCommand(arg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "!n: %v\n", err)
		os.Exit(1)
	}
	
	// Модифицируем команду
	finalCmd := modifyCommand(cmd, extraArgs)
	
	// Показываем команду
	fmt.Printf("%s\n", finalCmd)
	
	// Определяем тип команды
	if isCdCommand(finalCmd) {
		executeCdAndLaunchShell(finalCmd)
	} else {
		executeRegularCommand(finalCmd)
	}
}

func isCdCommand(cmd string) bool {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return false
	}
	return parts[0] == "cd"
}

func executeCdAndLaunchShell(cmd string) {
	// Разбираем команду cd
	parts := strings.Fields(cmd)
	if len(parts) < 2 {
		// cd без аргументов - домашняя директория
		executeCdAndLaunchShellToHome()
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
	
	// Обновляем переменные окружения PWD и OLDPWD
	oldPwd := os.Getenv("PWD")
	newPwd := resolvedPath
	
	// Запускаем новый shell в новой директории
	launchShellInDirectory(newPwd, oldPwd)
}

func executeCdAndLaunchShellToHome() {
	home := getHomeDir()
	
	// Меняем директорию
	if err := os.Chdir(home); err != nil {
		fmt.Fprintf(os.Stderr, "cd: %v\n", err)
		os.Exit(1)
	}
	
	oldPwd := os.Getenv("PWD")
	
	// Запускаем новый shell в домашней директории
	launchShellInDirectory(home, oldPwd)
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

func launchShellInDirectory(newPwd, oldPwd string) {
	// Получаем текущий shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}
	
	// Подготавливаем окружение
	env := prepareEnvironment(newPwd, oldPwd)
	
	// Запускаем shell в новой директории
	cmd := exec.Command(shell, "-i") // -i для интерактивного режима
	cmd.Dir = newPwd
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	// Запускаем и ждем завершения
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Error launching shell: %v\n", err)
		os.Exit(1)
	}
	
	// Если shell завершился, выходим
	os.Exit(0)
}

func prepareEnvironment(newPwd, oldPwd string) []string {
	// Копируем текущее окружение
	env := os.Environ()
	
	// Обновляем PWD и OLDPWD
	var newEnv []string
	
	for _, e := range env {
		// Пропускаем старые PWD и OLDPWD
		if !strings.HasPrefix(e, "PWD=") && !strings.HasPrefix(e, "OLDPWD=") {
			newEnv = append(newEnv, e)
		}
	}
	
	// Добавляем обновленные переменные
	newEnv = append(newEnv, fmt.Sprintf("OLDPWD=%s", oldPwd))
	newEnv = append(newEnv, fmt.Sprintf("PWD=%s", newPwd))
	
	return newEnv
}

func executeRegularCommand(cmd string) {
	// Для не-cd команд используем shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}
	
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

// Остальные функции остаются без изменений...
func getHistoryCommand(arg string) (string, error) {
	if strings.HasPrefix(arg, "!") {
		return resolveBangCommand(arg[1:])
	}
	
	num, err := strconv.Atoi(arg)
	if err != nil {
		return "", fmt.Errorf("invalid command: %s", arg)
	}
	
	return getCommandByNumber(num)
}

func resolveBangCommand(bang string) (string, error) {
	// !n
	if num, err := strconv.Atoi(bang); err == nil {
		return getCommandByNumber(num)
	}
	
	// !-n
	if strings.HasPrefix(bang, "-") {
		if num, err := strconv.Atoi(bang); err == nil {
			return getCommandFromEnd(-num)
		}
		return "", fmt.Errorf("invalid number: %s", bang)
	}
	
	// !string
	return getCommandByPrefix(bang)
}

func getCommandByNumber(num int) (string, error) {
	history, err := readHistory()
	if err != nil {
		return "", err
	}
	
	if num <= 0 || num > len(history) {
		return "", fmt.Errorf("command %d not found", num)
	}
	
	return history[num-1], nil
}

func getCommandFromEnd(offset int) (string, error) {
	history, err := readHistory()
	if err != nil {
		return "", err
	}
	
	if offset <= 0 || offset > len(history) {
		return "", fmt.Errorf("command -%d not found", offset)
	}
	
	return history[len(history)-offset], nil
}

func getCommandByPrefix(prefix string) (string, error) {
	history, err := readHistory()
	if err != nil {
		return "", err
	}
	
	for i := len(history) - 1; i >= 0; i-- {
		if strings.HasPrefix(history[i], prefix) {
			return history[i], nil
		}
	}
	
	return "", fmt.Errorf("no command starting with '%s'", prefix)
}

func modifyCommand(cmd string, extraArgs []string) string {
	if len(extraArgs) == 0 {
		return cmd
	}
	
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return cmd
	}
	
	if len(extraArgs) == 1 && len(parts) > 1 {
		arg := extraArgs[0]
		if !strings.HasPrefix(arg, "-") {
			parts[len(parts)-1] = arg
			return strings.Join(parts, " ")
		}
	}
	
	return cmd + " " + strings.Join(extraArgs, " ")
}

func readHistory() ([]string, error) {
	path := findHistoryFile()
	if path == "" {
		return nil, fmt.Errorf("history file not found")
	}
	
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	var commands []string
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		
		line = cleanHistoryLine(line)
		if line != "" {
			commands = append(commands, line)
		}
	}
	
	return commands, nil
}

func findHistoryFile() string {
	if histfile := os.Getenv("HISTFILE"); histfile != "" {
		return histfile
	}
	
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	
	// Пробуем разные файлы
	files := []string{
		filepath.Join(home, ".bash_history"),
		filepath.Join(home, ".zsh_history"),
	}
	
	for _, file := range files {
		if _, err := os.Stat(file); err == nil {
			return file
		}
	}
	
	return filepath.Join(home, ".bash_history")
}

func cleanHistoryLine(line string) string {
	// Bash таймстемпы
	if strings.HasPrefix(line, "#") && len(line) > 1 {
		if _, err := strconv.ParseInt(line[1:], 10, 64); err == nil {
			return ""
		}
	}
	
	// Zsh формат
	if strings.HasPrefix(line, ": ") {
		parts := strings.SplitN(line, ";", 2)
		if len(parts) == 2 {
			return parts[1]
		}
	}
	
	return line
}

func printHelp() {
	fmt.Println("!n - Execute command from history")
	fmt.Println("================================")
	fmt.Println()
	fmt.Println("Usage: !n <spec> [args...]")
	fmt.Println()
	fmt.Println("Command specifications:")
	fmt.Println("  n           Execute command number n")
	fmt.Println("  !n          Execute command number n")
	fmt.Println("  !-n         Execute command n from end")
	fmt.Println("  !string     Execute last command starting with string")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  !n 42          Execute command 42")
	fmt.Println("  !n !-1         Execute last command")
	fmt.Println("  !n !cd         Execute last cd command (launches new shell in directory)")
	fmt.Println()
	fmt.Println("Note: cd command launches a new shell in the target directory")
}
