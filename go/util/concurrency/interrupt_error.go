/**
 * Created by linyue on 17/3/17.
 */

package concurrency

type InterruptError struct {
	Message string
}

func (this *InterruptError) Error() string {
	return "[InterruptError] " + this.Message
}


