package main

import (
	"fmt"
	"os"
	"runtime"
)

func main() {
	args := os.Args[1:]
	
	if len(args) == 0 {
		// По умолчанию - имя ядра
		fmt.Println(runtime.GOOS)
		return
	}
	
	// Обработка флагов
	showAll := false
	showMachine := false
	showHostname := false
	showProcessor := false
	showHelp := false
	
	for _, arg := range args {
		switch arg {
		case "-a":
			showAll = true
		case "-m":
			showMachine = true
		case "-n":
			showHostname = true
		case "-p":
			showProcessor = true
		case "--help":
			showHelp = true
		default:
			fmt.Printf("uname: invalid option: %s\n", arg)
			os.Exit(1)
		}
	}
	
	if showHelp {
		fmt.Println("Usage: uname [-amnph]")
		fmt.Println("Options:")
		fmt.Println("  -a    Show all information")
		fmt.Println("  -m    Show machine type")
		fmt.Println("  -n    Show hostname")
		fmt.Println("  -p    Show processor type")
		fmt.Println("  -h, --help  Show this help")
		return
	}
	
	// Получаем значения
	kernelName := runtime.GOOS
	hostname, _ := os.Hostname()
	machine := getArchName()
	processor := machine // упрощенно
	
	// Формируем вывод
	if showAll {
		fmt.Printf("%s %s %s %s\n", kernelName, hostname, machine, processor)
	} else {
		first := true
		if showMachine {
			if !first {
				fmt.Print(" ")
			}
			fmt.Print(machine)
			first = false
		}
		if showHostname {
			if !first {
				fmt.Print(" ")
			}
			fmt.Print(hostname)
			first = false
		}
		if showProcessor {
			if !first {
				fmt.Print(" ")
			}
			fmt.Print(processor)
			first = false
		}
		// Если ничего не выбрано, но флаги были - показываем имя ядра
		if first && len(args) > 0 {
			fmt.Print(kernelName)
		}
		if !first || len(args) > 0 {
			fmt.Println()
		}
	}
}

func getArchName() string {
	switch runtime.GOARCH {
	case "386":
		return "i686"
	case "amd64":
		return "x86_64"
	case "arm":
		return "arm"
	case "arm64":
		return "arm64"
	default:
		return runtime.GOARCH
	}
}
