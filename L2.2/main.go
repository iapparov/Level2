package main

import "fmt"

func test() (x int) { 
  defer func() {
    x++
  }()
  x = 1
  return
}

func anotherTest() int {
  var x int
  defer func() {
    x++
  }()
  x = 1
  return x
}

func main() {
  fmt.Println(test()) // 2
  fmt.Println(anotherTest()) //1
}

/* 
  В случае test() x иницилаизируется в стэке функции. Но из за того что у нас именнованый возврат, x фактически хранит резульат выполнения функции и после ретерна +1 прибавляется
  к именнованому возврату. 
  В случае anotherTest() x так же инициализируется в стэке. Но вернет 1, так как на момент return x = 1. Хотя на самом деле если мы внутри defer после x++ выполним fmt.Println(x) то выдаст тоже 2, 
  просто это будет локальный x из стэка функции.
*/