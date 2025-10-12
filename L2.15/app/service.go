package app

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

var (
	currentMu sync.Mutex // currentMu защищает доступ к массиву current (чтобы не было гонок при сигнале SIGINT)
	// current — список всех внешних (exec.Command) процессов, которые запущены в данный момент. Нужен, чтобы можно было послать им сигнал прерывания при ctrl+c
	current []*exec.Cmd
)

// при получении ctrl+c отправляет SIGINT всем текущим процессам
func SigCancel(sigc chan os.Signal) {
	for range sigc {
		currentMu.Lock()
		for _, c := range current {
			if c != nil && c.Process != nil {
				err := c.Process.Signal(syscall.SIGINT)
				if err != nil {
					fmt.Fprintln(os.Stderr, "err:", err)
				}
			}
		}
		currentMu.Unlock()
	}
}

// проверяет, запущена ли оболочка в интерактивном режиме нужно для показа приглашения к вводу
func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

/*
Главный цикл командной оболочки читает ввод, разбирает команды, выполняет их с учётом пайпов, логических операторов (&&, ||) и встроенных команд
*/
func UnixShell() {
	in := bufio.NewReader(os.Stdin)
	interactive := isInteractive()

	for {
		// Показываем приглашение к вводу только если интерактивная сессия
		if interactive {
			fmt.Print("> ")
		}

		// Читаем строку до перевода строки
		line, err := in.ReadString('\n')
		if err != nil {
			// Ctrl+D (EOF) — выходим из программы
			if err == io.EOF {
				fmt.Println()
				return
			}
			// Любая другая ошибка чтения
			fmt.Fprintln(os.Stderr, "err:", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Подставляем переменные окружения вида $HOME, ${USER}, и т.д
		line = os.ExpandEnv(line)

		// Разбиваем строку по логическим операторам && и ||
		commands := splitLogical(line)

		// Выполняем команды последовательно, с учётом логических связей
		for idx, cmdLine := range commands {
			err := runLogical(cmdLine.cmd)
			// Проверяем, нужно ли выполнять следующую команду:
			if idx+1 < len(commands) {
				nextOp := commands[idx+1].op

				if err != nil && nextOp == "&&" { // если текущая завершилась с ошибкой и дальше стоит "&&" — прекращаем
					break
				}
				if err == nil && nextOp == "||" { // если завершилась успешно, а дальше стоит "||" — тоже прекращаем
					break
				}
			}
		}
	}
}

// logicalCmd — одна команда с указанием, каким логическим оператором она связана с предыдущей (&& или ||)
type logicalCmd struct {
	cmd string // сама команда, например: "echo hi"
	op  string // оператор перед ней ("" для первой, "&&" или "||" для остальных)
}

// разбивает строку на список логических команд
func splitLogical(line string) []logicalCmd {
	tokens := strings.Fields(line)
	var result []logicalCmd
	current := ""
	lastOp := ""

	for _, t := range tokens {
		if t == "&&" || t == "||" {
			// Сохраняем накопленную команду
			result = append(result, logicalCmd{cmd: strings.TrimSpace(current), op: lastOp})
			lastOp = t
			current = ""
		} else {
			current += t + " "
		}
	}

	//Добавляем последнюю команду
	if strings.TrimSpace(current) != "" {
		result = append(result, logicalCmd{cmd: strings.TrimSpace(current), op: lastOp})
	}
	return result
}

// runLogical выполняет одну логическую команду
func runLogical(line string) error {
	lines := strings.Split(line, "|")
	var stages [][]string

	// Разбиваем строку пайпа на части (по командам между '|')
	for _, l := range lines {
		fields := strings.Fields(l)
		if len(fields) == 0 {
			continue
		}
		stages = append(stages, fields)
	}

	if len(stages) == 0 {
		return nil
	}

	// Если одна команда и это встроенная — выполняем без создания процесса
	if len(stages) == 1 && isBuiltIn(stages[0][0]) {
		clean, _, outFile, err := parseRedirections(stages[0])
		if err != nil {
			return err
		}
		if f, ok := outFile.(*os.File); ok && f != os.Stdout {
			defer func() {
				err := f.Close()
				if err != nil {
					fmt.Fprintln(os.Stderr, "err:", err)
				}
			}() // закрываем файл, если был редирект
		}
		stages[0] = clean
		return runBuiltIn(stages[0], outFile)
	}

	// Иначе — полноценный пайплайн
	return runPipeline(stages)
}

// isBuiltIn проверяет, является ли команда встроенной.
func isBuiltIn(cmd string) bool {
	switch cmd {
	case "cd", "pwd", "echo", "kill", "ps":
		return true
	default:
		return false
	}
}

// runBuiltIn выполняет одну встроенную команду (cd, pwd, echo, kill, ps)
// out — это поток вывода (stdout или файл при перенаправлении)
func runBuiltIn(stages []string, out io.Writer) error {
	switch stages[0] {
	case "cd":
		// cd без аргументов → переход в домашний каталог
		if len(stages) < 2 {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			return os.Chdir(home)
		}
		return os.Chdir(stages[1])

	case "pwd":
		dir, err := os.Getwd()
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(out, dir)
		return err

	case "echo":
		_, err := fmt.Fprintln(out, strings.Join(stages[1:], " "))
		return err

	case "kill":
		if len(stages) < 2 {
			return fmt.Errorf("kill: требуется PID")
		}
		pid, err := strconv.Atoi(stages[1])
		if err != nil {
			return err
		}
		proc, err := os.FindProcess(pid)
		if err != nil {
			return err
		}
		return proc.Signal(syscall.SIGTERM)

	case "ps":
		cmd := exec.Command("ps")
		cmd.Stderr = os.Stderr
		cmd.Stdout = out
		return cmd.Run()

	default:
		return fmt.Errorf("неизвестная встроенная команда")
	}
}

// runPipeline выполняет последовательность команд, соединённых пайпами (|)
func runPipeline(stages [][]string) error {
	n := len(stages)
	if n == 0 {
		return nil
	}

	type pipeEnds struct {
		r *os.File // чтение
		w *os.File // запись
	}

	// Создаём пайпы между командами
	pipes := make([]pipeEnds, 0, n-1)
	for i := 0; i < n-1; i++ {
		r, w, err := os.Pipe()
		if err != nil {
			return err
		}
		pipes = append(pipes, pipeEnds{r: r, w: w})
	}

	cmds := make([]*exec.Cmd, n)
	inFiles := make([]io.Reader, n)
	outFiles := make([]io.Writer, n)

	// Разбор редиректов для каждой стадии пайпа
	for idx, stage := range stages {
		cleanStage, inFile, outFile, err := parseRedirections(stage)
		if err != nil {
			return err
		}
		stages[idx] = cleanStage
		inFiles[idx] = inFile
		outFiles[idx] = outFile

		if isBuiltIn(cleanStage[0]) {
			cmds[idx] = nil // встроенные выполняются внутри Go, без отдельного процесса
		} else {
			cmds[idx] = exec.Command(cleanStage[0], cleanStage[1:]...)
		}
	}

	// Сбрасываем список текущих процессов
	currentMu.Lock()
	current = []*exec.Cmd{}
	currentMu.Unlock()

	var wg sync.WaitGroup

	// Настраиваем ввод/вывод каждой команды в пайплайне
	for i := 0; i < n; i++ {
		var stdin io.Reader = os.Stdin
		var stdout io.Writer = os.Stdout

		if i > 0 {
			stdin = pipes[i-1].r // вход = выход предыдущей
		}
		if i < n-1 {
			stdout = pipes[i].w // выход = вход следующей
		}

		// Применяем перенаправления (< >)
		if inFiles[i] != nil && inFiles[i] != os.Stdin {
			stdin = inFiles[i]
		}
		if outFiles[i] != nil && outFiles[i] != os.Stdout {
			stdout = outFiles[i]
		}

		// Встроенные команды
		if isBuiltIn(stages[i][0]) {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				err := runBuiltIn(stages[i], stdout)
				if err != nil {
					fmt.Fprintln(os.Stderr, "err:", err)
				}
				if i < len(pipes) {
					err := pipes[i].w.Close()
					if err != nil {
						fmt.Fprintln(os.Stderr, "err:", err)
					}
				}
				if i > 0 {
					err := pipes[i-1].r.Close()
					if err != nil {
						fmt.Fprintln(os.Stderr, "err:", err)
					}
				}
			}(i)
			continue
		}

		// Внешние команды
		cmd := cmds[i]
		cmd.Stdin = stdin
		cmd.Stdout = stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			for _, p := range pipes {
				err := p.r.Close()
				if err != nil {
					fmt.Fprintln(os.Stderr, "err:", err)
				}
				err = p.w.Close()
				if err != nil {
					fmt.Fprintln(os.Stderr, "err:", err)
				}
			}
			return err
		}

		currentMu.Lock()
		current = append(current, cmd)
		currentMu.Unlock()

		if i < len(pipes) {
			err := pipes[i].w.Close()
			if err != nil {
				fmt.Fprintln(os.Stderr, "err:", err)
			}
		}
		if i > 0 {
			err := pipes[i-1].r.Close()
			if err != nil {
				fmt.Fprintln(os.Stderr, "err:", err)
			}
		}
	}

	wg.Wait()

	// Ожидаем завершения всех внешних процессов
	var lastErr error
	for _, c := range cmds {
		if c == nil {
			continue
		}
		if err := c.Wait(); err != nil {
			lastErr = err
		}
	}

	// Закрываем все оставшиеся открытые файлы и пайпы
	for _, in := range inFiles {
		if f, ok := in.(*os.File); ok && f != os.Stdin {
			_ = f.Close()
		}
	}
	for _, out := range outFiles {
		if f, ok := out.(*os.File); ok && f != os.Stdout {
			_ = f.Close()
		}
	}
	for _, p := range pipes {
		_ = p.r.Close()
		_ = p.w.Close()
	}

	currentMu.Lock()
	current = nil
	currentMu.Unlock()

	return lastErr
}

/*
parseRedirections разбирает аргументы команды и возвращает:
очищенный список аргументов (без < и >),
входной поток (stdin),
выходной поток (stdout),
возможную ошибку
*/
func parseRedirections(args []string) ([]string, io.Reader, io.Writer, error) {
	cleanArgs := []string{}
	stdin := os.Stdin
	stdout := os.Stdout

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case ">":
			if i+1 >= len(args) {
				return nil, nil, nil, fmt.Errorf("ошибка синтаксиса: ожидается файл после '>'")
			}
			fname := args[i+1]
			f, e := os.Create(fname)
			if e != nil {
				return nil, nil, nil, e
			}
			stdout = f
			i++ // пропускаем имя файла

		case "<":
			if i+1 >= len(args) {
				return nil, nil, nil, fmt.Errorf("ошибка синтаксиса: ожидается файл после '<'")
			}
			fname := args[i+1]
			f, e := os.Open(fname)
			if e != nil {
				return nil, nil, nil, e
			}
			stdin = f
			i++ // пропускаем имя файла

		default:
			cleanArgs = append(cleanArgs, arg)
		}
	}
	return cleanArgs, stdin, stdout, nil
}
