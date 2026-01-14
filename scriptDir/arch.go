package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
)

func main() {
	// Определение флагов
	helpFlag := flag.Bool("help", false, "показать справку")
	flag.Parse()

	if *helpFlag {
		printHelp()
		return
	}


	// Выводим архитектуру
	fmt.Println(getArchitecture())
}

func getArchitecture() string {
	// Получаем архитектуру из runtime
	goArch := runtime.GOARCH
	
	// Преобразуем архитектуру Go в стандартные имена архитектур Linux
	switch goArch {
	case "386":
		return "i386"
	case "amd64":
		return "x86_64"
	case "arm":
		// Для ARM нужно определить версию
		return getArmArchitecture()
	case "arm64":
		return "aarch64"
	case "mips", "mipsle":
		return "mips"
	case "mips64", "mips64le":
		return "mips64"
	case "ppc64":
		return "ppc64"
	case "ppc64le":
		return "ppc64le"
	case "riscv64":
		return "riscv64"
	case "s390x":
		return "s390x"
	default:
		return goArch
	}
}

func getArmArchitecture() string {
	// Проверяем, есть ли информация в runtime
	if runtime.GOOS == "linux" {
		// Пробуем прочитать /proc/cpuinfo для Linux
		data, err := os.ReadFile("/proc/cpuinfo")
		if err == nil {
			content := string(data)
			if strings.Contains(content, "ARMv7") {
				return "armv7l"
			} else if strings.Contains(content, "ARMv8") || strings.Contains(content, "AArch64") {
				// На самом деле arm64, но на всякий случай
				return "armv8l"
			} else if strings.Contains(content, "ARMv6") {
				return "armv6l"
			} else if strings.Contains(content, "ARMv5") {
				return "armv5tel"
			}
		}
	}
	
	// Значение по умолчанию для ARM
	return "armv7l"
}

func printHelp() {
	helpText := `Использование: arch [ПАРАМЕТР]...
Печатает имя архитектуры машины.

Параметры:
      --help     показать эту справку и выйти

Описание:
  Команда arch выводит архитектуру процессора текущей машины
  в стандартный вывод. Это полезно в скриптах для определения
  типа процессора.


Примеры:
  arch          Вывести архитектуру процессора
  arch --help   Показать справку
`

	fmt.Println(helpText)
}