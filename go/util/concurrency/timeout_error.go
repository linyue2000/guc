/**
 * Created by linyue on 17/3/16.
 */

package concurrency

type TimeoutError struct {
	Message string
}

func (this *TimeoutError) Error() string {
	return "[TimeoutError] " + this.Message
}

