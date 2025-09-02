package main

type customError struct {
  msg string
}

func (e *customError) Error() string {
  return e.msg
}

func test() *customError {
  // ... do something
  return nil
}

func main() {
  var err error
  err = test()
  if err != nil {
    println("error")
    return
  }
  println("ok")
}

/*
Здесь работает по такому же принципу, что и в l2.3
Функция тест возвращает указатель на структуру customError. А err это интерфейс error. customError удовлетворяет интерфейсу error. 
Соответственно err не может быть равен нил, так как он указывает на структуру customError. 
err в данном случае будет равен адресу структуры customError и реализацией интерфейса error.
*/