package main

import(
	"time"
	"fmt"
)

var or func(channels ...<-chan interface{}) <-chan interface{}

func main(){
	sig := func(after time.Duration) <-chan interface{} {
		c := make(chan interface{})
		go func() {
			defer close(c)
			time.Sleep(after)
		}()
		return c
	}

	or = func(channels ...<-chan interface{}) <-chan interface{}{
		switch len(channels){ // в случае если каналов 0 или 1, то либо возвращаем nil, либо сам канал
			case 0:
				return nil
			case 1:
				return channels[0]
		}

		out := make(chan interface{})
		go func(){
			defer close(out)
			switch len(channels){
				case 2: // если каналов 2, то просто ждем любой из них
					select{
					case <-channels[0]:
					case <-channels[1]:
					}
				default:
					select{ // иначе ждем первый из первых 3 каналов, и рекурсивно вызываем or для остальных
						case <-channels[0]:
						case <-channels[1]:
						case <-channels[2]:
						case <-or(append(channels[3:], out)...): // добавляем out в конец, чтобы гарантировать, что горутина не утечет (если какая-то из первых 3 каналов закроется, то out закроется в любом случае)
					}
			}
		}()
		return out
	}

	start := time.Now()
	<-or(
	sig(2*time.Hour),
	sig(5*time.Minute),
	sig(10*time.Second),
	sig(7*time.Second),
	sig(1*time.Hour),
	sig(1*time.Minute),
	)
	fmt.Printf("done after %v", time.Since(start))
}

/* 
вообще кусок с горутиной в функции or можно реализовать как 
		go func(){
			defer close(out)
				select{ // 
					case <-channels[0]:
					case <-or(append(channels[1:], out)...): // добавляем out в конец, чтобы гарантировать, что горутина не утечет (если какая-то из первых 3 каналов закроется, то out закроется в любом случае)
				}
		}()
Но в этом случае глубина рекурсии кратно увеличится, количество горутин возрастет и производительность уменьшится
*/