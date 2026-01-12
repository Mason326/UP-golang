directory="$1"
if [ -n "$directory" ] && [ -d "$directory" ]
then
	targetScripts=`ls "$directory"`
	echo ""
	for file in $targetScripts
	do
		go build "$file"
		echo "Builded $file"
	done
	
	echo ""
	echo "Build successful"
	echo ""	
	exit 0
else
	echo "You must to provide a directory name"
	exit 1
fi

