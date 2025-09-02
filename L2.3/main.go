package main

import (
  "fmt"
  "os"
)

func Foo() error {
  var err *os.PathError = nil
  return err
}

func main() {
  err := Foo()
  fmt.Println(err)
  fmt.Println(err == nil)
}

/*
Дело в том, что error – это интерфейс, включающий метод Error() string.
Структура os.PathError удовлетворяет этому интерфейсу.
В Foo() мы возвращаем интерефейс error, который реализует структура os.PathError.
Так как при инициализации мы ей присвоили nil, то fmt.Println(err) будет выводить nil (указывая, что структура содержит nil)
Однако с fmt.Println(err == nil) вернет false потому что интерфейс err содержит тип os.PathError и 
указывает на nil значение этого типа. Это значит, что интерфейс не равен nil.

В Go интерфейс считается равным nil, если и тип, и значение, на которое он указывает, оба равны nil. 
*/