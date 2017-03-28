/**
 * Created by linyue on 17/3/16.
 */

package concurrency

type ExecutionError struct {
	Message string
	Cause   error
}

func (this *ExecutionError) Error() string {
	return "[ExecutionError] " + this.Message + " [caused by] " + this.Cause.Error()
}


