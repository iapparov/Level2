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

	or = func(channels ...<-chan interface{}) <-chan interface{} {
		switch len(channels){
			case 0:
				return nil
			case 1:
				return channels[0]
		}

		out := make(chan interface{})
		go func(){
			defer close(out)
			switch len(channels){
				case 2:
					select{
					case <-channels[0]:
					case <-channels[1]:
					}
				default:
					select{
						case <-channels[0]:
						case <-channels[1]:
						case <-channels[2]:
						case <-or(append(channels[3:], out)...):
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