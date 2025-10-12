#!/usr/bin/env bash
set -u

SHELL_BIN=${SHELL_BIN:-./myshell}

# Если бинарник не найден — пытаемся собрать
if [ ! -x "$SHELL_BIN" ]; then
  echo "Собираем бинарник оболочки как $SHELL_BIN ..."
  if ! go build -o "$SHELL_BIN" ../cmd/main.go; then
    echo "Ошибка сборки. Укажите SHELL_BIN — путь к существующему бинарнику оболочки."
    exit 2
  fi
fi

# Временная директория для тестов
tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

# Вспомогательная функция для форматирования вывода команд

# Убираем приглашение '> ' и пробелы в конце строк
FMT_CMD_OUTPUT() {
  sed 's/^> //g' | sed -e 's/\r$//' -e 's/[[:space:]]*$//'
}

# Запуск одного теста
run_test() {
  local name="$1"; shift
  local input="$1"; shift
  local expected="${1:-}"

  echo "---- Тест: $name ----"

  # Передаём ввод в оболочку
  out=$(printf "%s\n" "$input" | "$SHELL_BIN" 2>&1 | FMT_CMD_OUTPUT)
  status=$?

  if [ -z "$expected" ]; then
    # Проверяем только код возврата
    if [ $status -eq 0 ]; then
      echo "PASS (код возврата 0)"
    else
      echo "FAIL (код возврата $status)"
      echo "Вывод:"
      echo "$out"
      return 1
    fi
  else
    # Нормализуем пробелы перед сравнением
    out_norm=$(printf "%s\n" "$out" | sed '/^$/d')
    exp_norm=$(printf "%s\n" "$expected" | sed '/^$/d')
    if [ "$out_norm" = "$exp_norm" ]; then
      echo "PASS"
    else
      echo "FAIL"
      echo "Ожидалось:"
      printf '%s\n' "$expected"
      echo "Получено:"
      printf '%s\n' "$out"
      return 1
    fi
  fi
  return 0
}

# Тесты
fails=0

run_test "echo simple"          "echo hello"                "hello" || fails=$((fails+1))
run_test "cd then pwd"          $'cd /tmp\npwd'             "/private/tmp" || fails=$((fails+1))
run_test "cd home then pwd"     $'cd\npwd'                  "$(printf "%s" "$HOME")" || fails=$((fails+1))
run_test "echo args"            "echo a b c"                "a b c" || fails=$((fails+1))
run_test "pipeline wc words"    'echo "one two three" | wc -w' "       3" || fails=$((fails+1))

f="$tmpdir/test7.txt"
run_test "redirect > and <"     $'echo hello > '"$f"$'\ncat < '"$f" "hello" || fails=$((fails+1))

f8="$tmpdir/test8.txt"
printf 'foo\nbar\nfoo\n' > "$f8"
run_test "grep < file | wc -l"  'grep foo < '"$f8"' | wc -l' "       2" || fails=$((fails+1))

run_test "&& true"              'true && echo ok'           "ok" || fails=$((fails+1))
run_test "&& false"             'false && echo ok'          "" || fails=$((fails+1))
run_test "|| false"             'false || echo fallback'    "fallback" || fails=$((fails+1))
run_test "|| true"              'true || echo no'           "" || fails=$((fails+1))
run_test "combined || &&"       'false || echo a && echo b' $'a\nb' || fails=$((fails+1))

# Проверка встроенной команды kill
sleep_pid_file="$tmpdir/sleep.pid"
( sleep 60 & echo $! > "$sleep_pid_file" )
sleep_pid=$(cat "$sleep_pid_file")

if kill -0 "$sleep_pid" 2>/dev/null; then
  printf "kill %s\n" "$sleep_pid" | "$SHELL_BIN" >/dev/null 2>&1
  sleep 0.2
  if kill -0 "$sleep_pid" 2>/dev/null; then
    kill -9 "$sleep_pid" 2>/dev/null || true
    echo "---- Тест: kill builtin FAIL (процесс всё ещё жив)"
    fails=$((fails+1))
  else
    echo "---- Тест: kill builtin PASS"
  fi
else
  echo "---- Тест: kill builtin SKIP (не удалось запустить sleep)"
fi

# Проверка встроенной команды ps
out16=$(printf "ps\n" | "$SHELL_BIN" 2>&1 | FMT_CMD_OUTPUT)
if [ -n "$out16" ]; then
  echo "---- Тест: ps builtin PASS (непустой вывод)"
else
  echo "---- Тест: ps builtin FAIL"
  fails=$((fails+1))
fi

# Перезапись при перенаправлении
f17="$tmpdir/test17.txt"
run_test "overwrite redirect" $'echo one > '"$f17"$'\necho two > '"$f17"$'\ncat < '"$f17" "two" || fails=$((fails+1))

# Пайплайн: встроенная -> внешняя команда
run_test "builtin -> external" 'echo hello | wc -c' "       6" || fails=$((fails+1))

# Несколько команд в одном вводе
run_test "multiple lines stream" $'echo first\necho second' $'first\nsecond' || fails=$((fails+1))

# Перенаправление в пайплайне
f22="$tmpdir/test22.txt"
out22=$(printf "echo hi > $f22 | wc -c\ncat < $f22\n" | "$SHELL_BIN" 2>&1 | FMT_CMD_OUTPUT)
filecont=$(cat "$f22")
if [ "$filecont" = "hi" ]; then
  echo "---- Тест: redirect in pipeline PASS"
else
  echo "---- Тест: redirect in pipeline FAIL (в файле: $filecont)"
  fails=$((fails+1))
fi

# Ошибка cd и проверка поведения &&
run_test "cd fail &&" 'cd /no/such/dir && echo ok' "" || fails=$((fails+1))

# Переменная окружения в фигурных скобках
run_test "env braces" 'echo ${HOME}' "$HOME" || fails=$((fails+1))


echo "---------------------------------"
if [ $fails -eq 0 ]; then
  echo "ВСЕ ТЕСТЫ ПРОЙДЕНЫ"
else
  echo "НЕУСПЕШНО: $fails из 25"
  exit 1
fi

rm -rf "$SHELL_BIN"