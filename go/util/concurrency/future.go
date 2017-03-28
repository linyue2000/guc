/**
 * Created by linyue on 17/3/8.
 */

package concurrency

import "time"

type Future interface {
	Cancel(mayInterruptIfRunning bool) bool
	IsCancelled() bool
	IsDone() bool
	Get() (interface{}, error)
	Get2(timeout time.Duration) (interface{}, error)
}
