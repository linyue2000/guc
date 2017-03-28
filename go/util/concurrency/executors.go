/**
 * Created by linyue on 17/3/8.
 */

package concurrency

import "sync"

func NewThreadPoolExecutor(nThreads int64, taskQueueSize int64) ExecutorService {
	executor := &ThreadPoolExecutor{
		size: nThreads,
		workers: []*Worker{},
		aliveWorkers: 0,
		taskQueue: make(chan Callable, taskQueueSize),
		state: TPE_NOT_START,
		mainLock: &sync.Mutex{},
		termination: &sync.WaitGroup{},
	}
	executor.abstractExecutorService = NewAbstractExecutorService(executor, executor)
	executor.Start()
	return executor
}
