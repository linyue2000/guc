/**
 * Created by linyue on 17/3/16.
 */

package concurrency

import (
	"time"
)

//implements ExecutorService
type AbstractExecutorService struct {
	Executor    Executor
	ExecutorBus ExecutorBus
}

func NewAbstractExecutorService(executor Executor, executorBus ExecutorBus) *AbstractExecutorService {
	return &AbstractExecutorService{
		Executor: executor,
		ExecutorBus: executorBus,
	}
}

//ExecutorBus
func (this *AbstractExecutorService) Shutdown() {
	this.ExecutorBus.Shutdown()
}

func (this *AbstractExecutorService) ShutdownNow() []Callable {
	return this.ExecutorBus.ShutdownNow()
}

func (this *AbstractExecutorService) IsShutdown() bool {
	return this.ExecutorBus.IsShutdown()
}

func (this *AbstractExecutorService) IsTerminated() bool {
	return this.ExecutorBus.IsTerminated()
}

func (this *AbstractExecutorService) AwaitTermination(timeout time.Duration) bool {
	return this.ExecutorBus.AwaitTermination(timeout)
}

//self
func (this *AbstractExecutorService) Submit(task Callable) (Future, error) {
	ftask, err := NewFutureTask(task)
	if err != nil {
		return ftask, err
	}
	err = this.Executor.Execute(ftask)
	return ftask, err
}

func (this *AbstractExecutorService) InvokeAll(tasks []Callable) ([]Future, error) {
	if tasks == nil {
		return nil, &NullPointerError{}
	}
	futures := []Future{}
	defer func() {
		for _, future := range futures {
			future.Cancel(true)
		}
	}()
	var lastError error = nil
	for _, task := range tasks {
		future, err := this.Submit(task)
		if err != nil {
			lastError = err
		}
		futures = append(futures, future)
	}
	for _, future := range futures {
		if !future.IsDone() {
			future.Get()
		}
	}
	return futures, lastError
}

func (this *AbstractExecutorService) InvokeAll2(tasks []Callable, timeout time.Duration) ([]Future, error) {
	if tasks == nil {
		return nil, &NullPointerError{}
	}
	futures := []Future{}
	defer func() {
		for _, future := range futures {
			future.Cancel(true)
		}
	}()
	startTime := time.Now()
	endTime := startTime.Add(timeout)
	expired := false
	var lastError error = nil
	for _, task := range tasks {
		var future Future = nil
		var err error = nil
		if !expired {
			future, err = this.Submit(task)
			if err != nil {
				lastError = err
			}
			if time.Now().Sub(startTime) > timeout {
				expired = true
			}
		} else {
			future, err = NewFutureTask(task)
			if err != nil {
				lastError = err
			}
		}
		futures = append(futures, future)
	}
	for _, future := range futures {
		if expired {
			break
		}
		if !future.IsDone() {
			future.Get2(endTime.Sub(time.Now()))
			if time.Now().Sub(startTime) > timeout {
				expired = true
			}
		}
	}
	return futures, lastError
}