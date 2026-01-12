#!/bin/bash
# Аргумент с названием директории
directory="$1"
if [ -n "$directory" ] && [ -d "$directory" ]
then
	cd "$directory"
	targetScripts=`ls | grep -E "*go"`
	# Создаем go.mod если его нет
	if [ ! -f "go.mod" ]; then
 	   	go mod init myapp
	fi

	# Собираем каждый .go файл отдельно
	for gofile in $targetScripts
	do
		# Отделяем расширение от имени файла
        	exename="${gofile%.go}"
        	echo "Building $exename"
        	go build -o "$exename" "$gofile"
	done

	echo ""
	echo "Build successful"
	echo ""
	exit 0
else
	echo "You must to provide a directory name"
	exit 1
fi
