/**
 * Created by linyue on 17/3/20.
 */

package concurrency

type RejectExecutionError struct {
	Message string
}

func (this *RejectExecutionError)Error() string {
	return "[RejectExecutionError] " + this.Message
}
