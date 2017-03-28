/**
 Created by linyue on 17/3/8.
 */

package concurrency

import (
	"time"
	"sync/atomic"
	"unsafe"
	"runtime"
	"git-pd.megvii-inc.com/linyue/guc/go/util/concurrency/locks"
)

const (
	FTS_NEW int32 = 0
	FTS_COMPLETING int32 = 1
	FTS_NORMAL int32 = 2
	FTS_EXCEPTIONAL int32 = 3
	FTS_CANCELLED int32 = 4
	FTS_INTERRUPTING int32 = 5
	FTS_INTERRUPTED int32 = 6
)

type WaitNode struct {
	next    *WaitNode
	blocker interface{}
}

type Blocker struct {

}

func NewWaitNode() *WaitNode {
	return &WaitNode{
		next: nil,
		blocker: &Blocker{},
	}
}

//implements Future, Callable
type FutureTask struct {
	/*The underlying callable; niled out after running */
	Callable Callable
	/**
	The run state of this task, initially NEW.  The run state
	transitions to a terminal state only in methods set,
	setException, and cancel.  During completion, state may take on
	transient values of COMPLETING (while outcome is being set) or
	INTERRUPTING (only while interrupting the runner to satisfy a
	cancel(true)). Transitions from these intermediate to final
	states use cheaper ordered/lazy writes because values are unique
	and cannot be further modified.

	Possible state transitions:
	NEW -> COMPLETING -> NORMAL
	NEW -> COMPLETING -> EXCEPTIONAL
	NEW -> CANCELLED
	NEW -> INTERRUPTING -> INTERRUPTED
       */
	State    int32
	/*The result to return or exception to throw from get() */
	Outcome  interface{}
	/*Treiber stack of waiting goroutine */
	waiters  *WaitNode
	/* 0 - not running; 1 - running */
	running  int32
	/* Parker for block and single */
	parker   *locks.Parker
}

func NewFutureTask(callable Callable) (*FutureTask, error) {
	if callable == nil {
		return nil, &NullPointerError{}
	}
	task := &FutureTask{
		Callable: callable,
		State: FTS_NEW,
		Outcome: nil,
		waiters: nil,
		running: 0,
		parker: locks.NewParker(),
	}
	return task, nil
}

//Future
func (this *FutureTask) Cancel(mayInterruptIfRunning bool) bool {
	if this.State != FTS_NEW {
		return false
	}
	if mayInterruptIfRunning {
		if ok := atomic.CompareAndSwapInt32(&this.State, FTS_NEW, FTS_INTERRUPTING); !ok {
			return false
		}
		this.Callable.Interrupt()
		atomic.StoreInt32(&this.State, FTS_INTERRUPTED)
	} else if ok := atomic.CompareAndSwapInt32(&this.State, FTS_NEW, FTS_CANCELLED); !ok {
		return false
	}
	this.finishCompletion()
	return true
}

func (this *FutureTask) IsCancelled() bool {
	return this.State >= FTS_CANCELLED
}

func (this *FutureTask) IsDone() bool {
	return this.State != FTS_NEW
}

func (this *FutureTask) Get() (interface{}, error) {
	s := this.State
	if s <= FTS_COMPLETING {
		s = this.awaitDone(false, 0)
	}
	return this.report(s)
}

func (this *FutureTask) Get2(timeout time.Duration) (interface{}, error) {
	s := this.State
	if s <= FTS_COMPLETING {
		s = this.awaitDone(true, timeout)
		if s <= FTS_COMPLETING {
			return nil, &TimeoutError{}
		}
	}
	return this.report(s)
}

//Callable
func (this *FutureTask) Call() interface{} {
	//avoid concurrent call
	if this.State != FTS_NEW || !atomic.CompareAndSwapInt32(&this.running, 0, 1) {
		//TODO return this.Outcome or nil ?
		return nil
	}
	if this.State == FTS_NEW {
		defer func() {
			err := recover()
			if err != nil {
				//TODO probably not an error interface{} ?
				this.setError(err.(error))
			}
			atomic.StoreInt32(&this.running, 0)
			s := this.State
			if s >= FTS_INTERRUPTING {
				this.handlePossibleCancellationInterrupt(s)
			}
		}()
		result := this.Callable.Call()
		//consider all interruptions as errors (unlike java here, if task in java caught interrupt exception, it can outcome a normal result)
		if this.Callable.IsInterrupted() {
			this.setError(&InterruptError{})
		} else {
			this.set(result)
		}
	}
	return this.Outcome
}

func (this *FutureTask) Interrupt() {
	this.Callable.Interrupt()
}

func (this *FutureTask) IsInterrupted() bool {
	return this.Callable.IsInterrupted()
}

//self
func (this *FutureTask) set(v interface{}) {
	if atomic.CompareAndSwapInt32(&this.State, FTS_NEW, FTS_COMPLETING) {
		this.Outcome = v
		atomic.StoreInt32(&this.State, FTS_NORMAL)
		this.finishCompletion()
	}
}

func (this *FutureTask) setError(t error) {
	if atomic.CompareAndSwapInt32(&this.State, FTS_NEW, FTS_COMPLETING) {
		this.Outcome = t
		atomic.StoreInt32(&this.State, FTS_EXCEPTIONAL)
		this.finishCompletion()
	}
}

func (this *FutureTask) finishCompletion() {
	for {
		var q *WaitNode = this.waiters
		if q == nil {
			break
		}
		if this.compareAndSwapWaitNode(&this.waiters, q, nil) {
			for {
				t := q.blocker
				if t != nil {
					q.blocker = nil
					this.parker.Unpark(t)
				}
				next := q.next
				if next == nil {
					break
				}
				q.next = nil // unlink to help gc
				q = next
			}
			break
		}
	}
	//TODO to add callback (go does not support abstract method)
	//this.done();
}

/**
Awaits completion or aborts on interrupt or timeout.
 */
func (this *FutureTask) awaitDone(timed bool, timeout time.Duration) int32 {
	deadline := time.Now().Add(timeout)
	var q *WaitNode = nil
	queued := false
	for {
		if this.Callable.IsInterrupted() {
			this.removeWaiter(q)
			return this.State
		}

		s := this.State
		if s > FTS_COMPLETING {
			if q != nil {
				q.blocker = nil
			}
			return s
		} else if s == FTS_COMPLETING {
			// cannot time out yet
			runtime.Gosched()
		} else if q == nil {
			q = NewWaitNode()
		} else if !queued {
			q.next = this.waiters
			queued = this.compareAndSwapWaitNode(&this.waiters, q.next, q)
		} else if timed {
			if time.Now().After(deadline) {
				this.removeWaiter(q)
				return this.State
			}
			this.parker.ParkWithTimeout(q.blocker, deadline.Sub(time.Now()))
		} else {
			this.parker.Park(q.blocker)
		}
	}
}

/**
Tries to unlink a timed-out or interrupted wait node to avoid
accumulating garbage.  Internal nodes are simply unspliced
without CAS since it is harmless if they are traversed anyway
by releasers.  To avoid effects of unsplicing from already
removed nodes, the list is retraversed in case of an apparent
race.  This is slow when there are a lot of nodes, but we don't
expect lists to be long enough to outweigh higher-overhead
schemes.
*/
func (this *FutureTask) removeWaiter(node *WaitNode) {
	if (node != nil) {
		node.blocker = nil
		retry:
		// restart on removeWaiter race
		for {
			var pred, q, s *WaitNode = nil, this.waiters, nil
			for ; q != nil; q = s {
				s = q.next
				if q.blocker != nil {
					pred = q
				} else if pred != nil {
					pred.next = s
					// check for race
					if pred.blocker == nil {
						continue retry
					}
				} else if !this.compareAndSwapWaitNode(&this.waiters, q, s) {
					continue retry
				}
			}
			break
		}
	}
}

func (this *FutureTask) handlePossibleCancellationInterrupt(s int32) {
	if (s == FTS_INTERRUPTING) {
		for {
			if this.State != FTS_INTERRUPTING {
				return
			}
			runtime.Gosched() // wait out pending interrupt
		}
	}
}

/**
Returns result or throws exception for completed task.
 */
func (this *FutureTask) report(s int32) (interface{}, error) {
	x := this.Outcome
	if s == FTS_NORMAL {
		return x, nil
	}
	if s >= FTS_CANCELLED {
		return nil, &CancellationError{}
	}
	return nil, &ExecutionError{
		Cause: x.(error),
	}
}

func (this *FutureTask) compareAndSwapWaitNode(dest **WaitNode, old, new *WaitNode) bool {
	udest := (*unsafe.Pointer)(unsafe.Pointer(dest))
	return atomic.CompareAndSwapPointer(udest,
		unsafe.Pointer(old),
		unsafe.Pointer(new),
	)
}
