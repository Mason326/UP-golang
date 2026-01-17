# Educational practice Golang ![gopher favicon](https://upload.wikimedia.org/wikipedia/commons/2/2d/Go_gopher_favicon.svg) (System Programming)

During our educational practice, we had to implement (write an analogue of) a certain number of utilities for *Linux* in the **Golang** programming language.

>[!IMPORTANT]
> To receive credit for the educational practice, you must complete the requirements and tasks listed below.

- Using an integrated development environment (IDE), develop analogs of system utilities in accordance with the corresponding assessment task for the GNU Linux OS.
- Only the Golang programming language is permitted for development. Use of any other programming language is prohibited.
- Each command must be implemented in a separate executable file.
- Comments must be included in the implementation code for each project command. Implement the use of various Golang data structures.
- Each utility must implement its own set of three input arguments. The -h argument must be implemented in each command; it is mandatory.
- Using any third-party external libraries is prohibited.
- Using the exec function from the Golang library to call a similar system utility is prohibited.
- It is imperative to provide validation of initial/input values ​​(e.g., out-of-range values) using an exception handling mechanism. If an application/team suddenly crashes or freezes during project delivery, exactly 1 point will be deducted from the overall score!


### Table of implemented utilities:

| Name | Description | Keys (Options) | 
|--------------------|---------------------------|-----------------|
|ls|Displays information about files (by default in the current directory)|-a, -l, -r, -R|
|pwd|Displays the full path of the current working directory|-L, -P|
|cd|Changes the current working directory, replacing the current shell|-,~,~username, -P|
|mkdir|Creates the specified directories|-m, -p, -v|
|rmdir|Removes empty directories|-ignore-fail-on-non-empty, -p, -v|
|cat|Concatenates files and prints their contents to standard output. If FILE is omitted or specified as -, reads standard input|-A, -b, -E, -n|
|file|Defines the file type|-b, -i, -z|
|nl|Numbers the lines of files|-a, -i, -s|
|zip|Creates a ZIP archive from the specified files|-r, -q, -e|
|unzip|Unpacks a ZIP archive|-l, -q, -d|
|tar|A utility for creating, extracting, and viewing tar archives.|-c, -x, -t, -f, -v|
|!!|Repeat last command with optional arguments|-|
|!n|Execute command from history|-|
|touch|Changes access and modification times of files|-a, -c, -r, -t|
|free|Displays information about memory usage|-b, -k, -m, -g, -s|
|exit|Terminates current shell session|-|
|rm|Remove (unlink) the FILE(s)|-d, -f, -i, -r, -v|
|head|Prints the first 10 lines of each FILE to standard output. If FILE is not specified, reads standard input|-c, -n, -q, -v|
|date|Shows the system date and time|-d, -r, -u|
|arch|Prints the machine architecture name|-|
|clear|Clearing the terminal screen|-T, -V, -x|
|ps|Displays information about processes|-A, -a, -f, -u|
|df|Show filesystem disk space usage|-a, -h, -B|
|wc|Word, line, character, and byte count|-c, -m, -l, -w|
|tail|Displays the last lines of files (default 10)|-f, -n, -q|
|uname|Print certain system information|-a, -m, -n, -p|
|cp|Copy SOURCE to DEST, or multiple SOURCE(s) to DIRECTORY|-f, -i, -r, -v|
|history|Displays the command history from the history files|-n, -d, -c|

## How to launch THIS project ?

1. You need a Linux distribution installed (preferably RED OS 8.0.2 x86_64).
Link to the official website: https://redos.red-soft.ru/product/downloads/
2. You need to update packages using: <br>
```bash
 sudo dnf update
```
3. You need to have the Golang compiler (if it isn't installed after dnf update):<br>
```bash
 sudo dnf install go
```
4. You need Git to clone the repository:<br>
```git
 git clone https://github.com/Mason326/UP-golang.git
``` 
   or simply download the ZIP archive from the [link](https://github.com/Mason326/UP-golang/archive/refs/heads/main.zip)

5. Once the project is installed on your computer, navigate to the main project directory: <br>
```bash
    cd UP-golang/
```
6. Run the script Build executable utility files:
```bash
    ./build-script.sh scriptDir/
```
7. Change to the directory with the executable files
```bash
    cd ./scriptDir
```
8. Finally, you can work with the Linux utility analogs
```bash
    ./date
```
<pre>
Output:
    Sun Jan 18 00:50:29 MSK 2026