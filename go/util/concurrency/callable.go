/**
 * Created by linyue on 17/3/8.
 */

package concurrency

type Callable interface {
	//!!!只允许是能为nil的interface{}
	Call() interface{}
	Interrupt()
	IsInterrupted() bool
}
