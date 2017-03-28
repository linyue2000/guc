/**
 * Created by linyue on 17/3/8.
 */

package concurrency

import (
	"sync/atomic"
)

//
type Worker struct {
	taskQueue      chan Callable
	observer       WorkerObserver
	done           chan bool
	//0 - not terminated, 1 - terminated
	terminated     int32
	//0 - not idle, 1 - idle
	idle           int32
	completedTasks int64
	currentTask    Callable
}

type WorkerObserver interface {
	OnTaskCompleted()
}

func newWorker(taskQueue chan Callable, observer WorkerObserver) *Worker {
	worker := &Worker{
		taskQueue: taskQueue,
		observer: observer,
		done: make(chan bool, 1),
		terminated: 0,
		idle: 1,
		completedTasks: 0,
		currentTask: nil,
	}
	return worker
}

func (this *Worker) Start() {
	go func() {
		this.run()
	}()
}

func (this *Worker) run() {
	for {
		select {
		case task, ok := <-this.taskQueue:
			if ok {
				//TODO 这里可能在ShutdownOnlyIfIdle的时候丢掉一个task。不想加个锁的原因是怕observer回调外面的函数时候出现死锁
				//TODO 想想怎么办，其实想要不出现死锁，只要observer的实现保证require lock的顺序就行了
				if atomic.CompareAndSwapInt32(&this.idle, 1, 0) {
					//fmt.Println("worker get task")
					this.currentTask = task
					task.Call()
					this.currentTask = nil
					this.completedTasks += 1
					atomic.CompareAndSwapInt32(&this.idle, 0, 1)
					if this.observer != nil {
						this.observer.OnTaskCompleted()
					}
				}
			}
		case <-this.done:
		//fmt.Println("worker shutdown")
			return
		}
	}
}

func (this *Worker) Shutdown() bool {
	if !atomic.CompareAndSwapInt32(&this.terminated, 0, 1) {
		return false
	}
	c := this.currentTask
	if c != nil {
		go c.Interrupt()
	}
	go func() {
		select {
		case this.done <- true:
		default:
		}
	}()
	return true
}

func (this *Worker) ShutdownOnlyIfIdle() bool {
	if atomic.CompareAndSwapInt32(&this.idle, 1, 0) {
		return this.Shutdown()
	}
	return false
}
