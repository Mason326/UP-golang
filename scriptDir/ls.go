package main

import (
	"flag"
	"fmt"
	"io/ioutil"
)

const dotCharacter = 46

func main() {
	// Варианты флагов (ключей для передачи скрипту)
	//recursiveFlag := flag.Bool("R", false, "List subdirectories recursively")
	//allFlag := flag.Bool("a", false, "Do not ignore entries starting with .")
	//longListingFlag := flag.Bool("l", false, "Use a long listing format")
	//reverseFlag := flag.Bool("r", false, "Reverse order while sorting")
	//helpFlag := flag.Bool("h", false, "With -l and -s, print sizes like 1K 234M 2G etc.")

	// Парсинг флагов
	flag.Parse()

	// Получаем массив введенных директорий
	inputDirs := flag.Args()

	if len(inputDirs) == 0 {
		// По умолчанию просматриваем текущую директорию
		showListElems(".")
		return
	} else {
		// Для множества указанных директорий
		for _, dir := range inputDirs {
			fmt.Printf("%s:\n", dir)
			showListElems(dir)
			fmt.Println()
		}
	}

}

func showListElems(path string) {
	// Чтение содержимого директории
	lst, err := ioutil.ReadDir(path)
	if err != nil {
		fmt.Printf("Error while read a directory. Is %s a directory?\n", path)
		return
	}
	for i := 0; i < len(lst); i++ {
		element := lst[i]
		// Пропускаем скрытые файлы и папки
		if isHidden(element.Name()) {
			continue
		}
		// Отдельный вывод для директорий
		if element.IsDir() {
			fmt.Printf("%s/\n", element.Name())
		} else {
			// Отдельный вывод для файлов
			fmt.Println(element.Name())
		}
	}
}

func isHidden(path string) bool {
	return path[0] == dotCharacter
}
