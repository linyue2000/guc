/**
 * Created by linyue on 17/3/8.
 */

package concurrency

import (
	"sync/atomic"
	"sync"
	"time"
)

const (
	/**
	RUNNING:  Accept new tasks and process queued tasks
	SHUTDOWN: Don't accept new tasks, but process queued tasks
	STOP:     Don't accept new tasks, don't process queued tasks, and interrupt in-progress tasks
	TIDYING:  All tasks have terminated, workerCount is zero, the thread transitioning to state TIDYING will run the terminated() hook method
	TERMINATED: terminated() has completed
	*/
	TPE_NOT_START = 0
	TPE_RUNNING int32 = 1
	TPE_SHUTDOWN int32 = 2
	TPE_STOP int32 = 3
	TPE_TIDYING int32 = 4
	TPE_TERMINATED int32 = 5
)

//extends AbstractExecutorService, implements Executor
type ThreadPoolExecutor struct {
	abstractExecutorService *AbstractExecutorService
	size                    int64
	workers                 []*Worker
	aliveWorkers            int64
	taskQueue               chan Callable
	state                   int32
	mainLock                *sync.Mutex
	termination             *sync.WaitGroup
}

//self
func (this *ThreadPoolExecutor) Start() {
	this.mainLock.Lock()
	defer this.mainLock.Unlock()
	this.workers = make([]*Worker, this.size, this.size)
	for i := int64(0); i < this.size; i++ {
		this.workers[i] = newWorker(this.taskQueue, this)
		this.workers[i].Start()
		this.aliveWorkers++
	}
	this.termination.Add(1)
	this.AdvanceRunState(TPE_RUNNING)
}

func (this *ThreadPoolExecutor) tryTerminate() {
	for {
		s := this.state
		//fmt.Println("tryTerminate, s =", s)
		if this.isRunning(s) ||
			s >= TPE_TIDYING ||
			s == TPE_SHUTDOWN && len(this.taskQueue) > 0 {
			//fmt.Println("tryTerminate, return")
			return
		}
		if this.aliveWorkers > 0 {
			//fmt.Println("alive worker", this.aliveWorkers)
			this.terminateIdleWorkers(true)
			//fmt.Println("alive worker", this.aliveWorkers)
			if this.aliveWorkers > 0 {
				return
			}
		}
		needReturn := func() bool {
			this.mainLock.Lock()
			defer this.mainLock.Unlock()
			//fmt.Println("needReturn, s =", s)
			if atomic.CompareAndSwapInt32(&this.state, s, TPE_TERMINATED) {
				this.terminateWorkers()
				//termination.signallAll() for awaitTermination
				this.termination.Done()
				return true
			} else {
				return false
			}
		}()
		if needReturn {
			return
		}
	}
}

func (this *ThreadPoolExecutor) isRunning(s int32) bool {
	return s == TPE_RUNNING
}

func (this *ThreadPoolExecutor) terminateWorkers() {
	for _, worker := range this.workers {
		if worker.Shutdown() {
			this.aliveWorkers--
		}
	}
}

func (this *ThreadPoolExecutor) terminateIdleWorkers(onlyOne bool) {
	for _, worker := range this.workers {
		shut := worker.ShutdownOnlyIfIdle()
		if shut {
			this.aliveWorkers--
			if onlyOne {
				return
			}
		}
	}
}

func (this *ThreadPoolExecutor) AdvanceRunState(targetState int32) {
	for {
		s := this.state
		if (s >= targetState) ||
			atomic.CompareAndSwapInt32(&this.state, s, targetState) {
			break
		}
	}
}

//WorkerObserver
func (this *ThreadPoolExecutor) OnTaskCompleted() {
	this.tryTerminate()
}

//AbstractExecutorService
/**
     * Initiates an orderly shutdown in which previously submitted
     * tasks are executed, but no new tasks will be accepted.
     * Invocation has no additional effect if already shut down.
     *
     * <p>This method does not wait for previously submitted tasks to
     * complete execution.  Use {@link #awaitTermination awaitTermination}
     * to do that.
     *
     */
func (this *ThreadPoolExecutor) Shutdown() {
	func() {
		this.mainLock.Lock()
		defer this.mainLock.Unlock()
		if this.state <= TPE_RUNNING {
			this.AdvanceRunState(TPE_SHUTDOWN)
			this.terminateIdleWorkers(false)
			close(this.taskQueue)
		}
	}()
	this.tryTerminate()
}

/**
     * Attempts to stop all actively executing tasks, halts the
     * processing of waiting tasks, and returns a list of the tasks
     * that were awaiting execution.
     *
     * <p>This method does not wait for actively executing tasks to
     * terminate.  Use {@link #awaitTermination awaitTermination} to
     * do that.
     *
     * <p>There are no guarantees beyond best-effort attempts to stop
     * processing actively executing tasks.  For example, typical
     * implementations will cancel via {@link Thread#interrupt}, so any
     * task that fails to respond to interrupts may never terminate.
     *
     * @return list of tasks that never commenced execution
     */
func (this *ThreadPoolExecutor) ShutdownNow() []Callable {
	result := func() []Callable {
		this.mainLock.Lock()
		defer this.mainLock.Unlock()
		result := []Callable{}
		if this.state <= TPE_RUNNING {
			this.AdvanceRunState(TPE_STOP)
			close(this.taskQueue)
			this.terminateWorkers()
			for task := range this.taskQueue {
				result = append(result, task)
			}
		}
		return result
	}()
	this.tryTerminate()
	return result
}

/**
     * Returns <tt>true</tt> if this executor has been shut down.
     *
     * @return <tt>true</tt> if this executor has been shut down
     */
func (this *ThreadPoolExecutor) IsShutdown() bool {
	return this.state >= TPE_SHUTDOWN
}

/**
     * Returns <tt>true</tt> if all tasks have completed following shut down.
     * Note that <tt>isTerminated</tt> is never <tt>true</tt> unless
     * either <tt>shutdown</tt> or <tt>shutdownNow</tt> was called first.
     *
     * @return <tt>true</tt> if all tasks have completed following shut down
     */
func (this *ThreadPoolExecutor) IsTerminated() bool {
	return this.state == TPE_TERMINATED
}

/**
     * Blocks until all tasks have completed execution after a shutdown
     * request, or the timeout occurs, or the current thread is
     * interrupted, whichever happens first.
     *
     * @param timeout the maximum time to wait
     * @return <tt>true</tt> if this executor terminated and
     *         <tt>false</tt> if the timeout elapsed before termination
     */
func (this *ThreadPoolExecutor) AwaitTermination(timeout time.Duration) bool {
	awaitDone := make(chan bool)
	go func() {
		this.termination.Wait()
		close(awaitDone)
	}()
	select {
	case <-awaitDone:
		return true
	case <-time.After(timeout):
		return false
	}
}

/**
     * Submits a value-returning task for execution and returns a
     * Future representing the pending results of the task. The
     * Future's <tt>get</tt> method will return the task's result upon
     * successful completion.
     *
     * <p>
     * If you would like to immediately block waiting
     * for a task, you can use constructions of the form
     * <tt>result = exec.submit(aCallable).get();</tt>
     *
     * <p> Note: The {@link Executors} class includes a set of methods
     * that can convert some other common closure-like objects,
     * for example, {@link java.security.PrivilegedAction} to
     * {@link Callable} form so they can be submitted.
     *
     * @param task the task to submit
     * @return a Future representing pending completion of the task
     */
func (this *ThreadPoolExecutor) Submit(task Callable) (Future, error) {
	return this.abstractExecutorService.Submit(task)
}

/**
     * Executes the given tasks, returning a list of Futures holding
     * their status and results when all complete.
     * {@link Future#isDone} is <tt>true</tt> for each
     * element of the returned list.
     * Note that a <em>completed</em> task could have
     * terminated either normally or by throwing an exception.
     * The results of this method are undefined if the given
     * collection is modified while this operation is in progress.
     *
     * @param tasks the collection of tasks
     * @return A list of Futures representing the tasks, in the same
     *         sequential order as produced by the iterator for the
     *         given task list, each of which has completed.
     */
func (this *ThreadPoolExecutor) InvokeAll(tasks []Callable) ([]Future, error) {
	return this.abstractExecutorService.InvokeAll(tasks)
}

/**
     * Executes the given tasks, returning a list of Futures holding
     * their status and results
     * when all complete or the timeout expires, whichever happens first.
     * {@link Future#isDone} is <tt>true</tt> for each
     * element of the returned list.
     * Upon return, tasks that have not completed are cancelled.
     * Note that a <em>completed</em> task could have
     * terminated either normally or by throwing an exception.
     * The results of this method are undefined if the given
     * collection is modified while this operation is in progress.
     *
     * @param tasks the collection of tasks
     * @param timeout the maximum time to wait
     * @param unit the time unit of the timeout argument
     * @return a list of Futures representing the tasks, in the same
     *         sequential order as produced by the iterator for the
     *         given task list. If the operation did not time out,
     *         each task will have completed. If it did time out, some
     *         of these tasks will not have completed.
     */
func (this *ThreadPoolExecutor) InvokeAll2(tasks []Callable, timeout time.Duration) ([]Future, error) {
	return this.abstractExecutorService.InvokeAll2(tasks, timeout)
}

//Executor
/**
never called directly by user
 */
func (this *ThreadPoolExecutor) Execute(task Callable) error {
	if this.isRunning(this.state) {
		this.mainLock.Lock()
		defer this.mainLock.Unlock()
		//recheck
		if !this.isRunning(this.state) {
			return &RejectExecutionError{"executor has been shut down"}
		}
		select {
		case this.taskQueue <- task:
		default:
		//queue full
			return &RejectExecutionError{"task queue is full"}
		}
	} else if this.state == TPE_NOT_START {
		return &RejectExecutionError{"executor has not been started"}
	} else {
		return &RejectExecutionError{"executor has been shut down"}
	}
	return nil
}