/**
 * Created by linyue on 17/3/20.
 */

package concurrency

import "time"

type ExecutorBus interface {
	Shutdown()
	ShutdownNow() []Callable
	IsShutdown() bool
	IsTerminated() bool
	AwaitTermination(timeout time.Duration) bool
}