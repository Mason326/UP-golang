package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	args := os.Args[1:]
	
	if len(args) == 0 {
		runBangBang(nil)
		return
	}
	
	// Проверяем специальные флаги
	switch args[0] {
	case "-h", "--help":
		printSimpleHelp()
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
			printSimpleHelp()
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
	fmt.Printf("!!: %s\n", cmd)
	executeCommand(cmd)
}

func runBangBangN(n int, extraArgs []string) {
	cmd := getNthCommand(n)
	if cmd == "" {
		fmt.Fprintf(os.Stderr, "Command %d not found\n", n)
		os.Exit(1)
	}
	
	cmd = modifyCommand(cmd, extraArgs)
	fmt.Printf("!!%d: %s\n", n, cmd)
	executeCommand(cmd)
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
	
	historyFile := filepath.Join(home, ".bash_history")
	file, err := os.Open(historyFile)
	if err != nil {
		return []string{}
	}
	defer file.Close()
	
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
		
		commands = append(commands, line)
	}
	
	return commands
}

func executeCommand(cmd string) {
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

func printSimpleHelp() {
	fmt.Println("Usage: !! [!!n] [args...]")
	fmt.Println()
	fmt.Println("Repeat last command with optional arguments.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  !!          # repeat last command")
	fmt.Println("  !! -la      # add -la to last command")
	fmt.Println("  !! /var     # replace last argument with /var")
	fmt.Println("  !!2         # repeat second last command")
	fmt.Println("  !! --help   # show this help")
}
