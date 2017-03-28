/**
 * Created by linyue on 17/3/8.
 */

package concurrency

type Executor interface {
	Execute(task Callable) error
}