/**
 * Created by linyue on 17/3/8.
 */

package concurrency

import "time"

//extends ExecutorBus
type ExecutorService interface {
	/**
	ExecutorBus
	 */
	Shutdown()
	ShutdownNow() []Callable
	IsShutdown() bool
	IsTerminated() bool
	AwaitTermination(timeout time.Duration) bool

	//self
	Submit(task Callable) (Future, error)
	InvokeAll(tasks []Callable) ([]Future, error)
	InvokeAll2(tasks []Callable, timeout time.Duration) ([]Future, error)
	//InvokeAny(tasks []Callable) interface{}
	//InvokeAny2(tasks []Callable, timeout time.Duration) interface{}
}