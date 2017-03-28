/**
 * Created by linyue on 17/3/17.
 */

package main

import (
	"git-pd.megvii-inc.com/linyue/guc/go/util/concurrency"
	"fmt"
	"time"
	"git-pd.megvii-inc.com/linyue/guc/go/util/concurrency/test/tasks"
	"strconv"
)

func main() {
	TestInvokeAll()
}

func TestInvokeAll() {
	executorService := concurrency.NewThreadPoolExecutor(3, 100000)
	allTask := []concurrency.Callable{}
	for i := 0; i < 5; i++ {
		task := tasks.NewSampleTask(strconv.FormatInt(int64(i), 10))
		allTask = append(allTask, task)
	}
	_, err := executorService.InvokeAll2(allTask, time.Second * 3)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("submitting gone")
	goneFuture, err := executorService.Submit(tasks.NewSampleTask("gone"))
	if err != nil {
		fmt.Println(err)
	}
	goneFuture.Get()
}

func Test() {
	executorService := concurrency.NewThreadPoolExecutor(3, 100000)
	futures := []concurrency.Future{}
	for i := 0; i < 10; i++ {
		task := tasks.NewSampleTask(strconv.FormatInt(int64(i), 10))
		future, err := executorService.Submit(task)
		if err != nil {
			fmt.Println(err)
		} else {
			futures = append(futures, future)
		}
	}
	go func() {
		time.Sleep(3 * time.Second)
		tasksLeave := executorService.ShutdownNow()
		fmt.Println(tasksLeave)
	}()
	for _, future := range futures {
		r, err := future.Get()
		if err != nil {
			fmt.Println("task error", err)
		} else {
			fmt.Println("task result", r)
		}
	}
	executorService.AwaitTermination(60 * time.Second)
	fmt.Println("!!!!!!!!!!!!!!!!")
	//for {
	//	time.Sleep(time.Second)
	//}
}