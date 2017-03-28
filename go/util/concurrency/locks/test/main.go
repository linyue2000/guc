/**
 * Created by linyue on 17/3/17.
 */

package main

import (
	"git-pd.megvii-inc.com/linyue/guc/go/util/concurrency/locks"
	"time"
	"fmt"
	"sync"
)

func main() {
	//TestParker()
	TestChannel()
	//TestMap()
	//TestWaitGroup()
}

func TestWaitGroup(){
	wg := &sync.WaitGroup{}
	wg.Wait()
	fmt.Println("done")
}

func TestMap() {
	m := map[int]int{}
	m[8] = 8
	m[2] = 2
	m[3] = 3
	m[4] = 4
	m[5] = 5
	m[6] = 6
	for k, _ := range m {
		//fmt.Println(k, v)
		delete(m, k)
	}
	for k, v := range m {
		fmt.Println(k, v)
	}
}

func TestChannel() {
	opened := make(chan int, 3)
	//opened <- 1
	//opened <- 2
	//opened <- 3
	go func() {
		time.Sleep(time.Second * 3)
		close(opened)
	}()
	//close(opened)
	v, ok := <-opened
	fmt.Println(v, ok)
	v, ok = <-opened
	fmt.Println(v, ok)
	v, ok = <-opened
	fmt.Println(v, ok)
	v, ok = <-opened
	fmt.Println(v, ok)
}

func TestParker() {
	blocker := time.Now()
	parker := locks.NewParker()
	parker.Unpark(&blocker)
	go func(parker *locks.Parker, blocker interface{}) {
		time.Sleep(2 * time.Second)
		parker.Unpark(blocker)
	}(parker, &blocker)
	fmt.Println("waiting")
	parker.Park(&blocker)
	fmt.Println("notified 1")
	parker.Park(&blocker)
	fmt.Println("notified 2")
}