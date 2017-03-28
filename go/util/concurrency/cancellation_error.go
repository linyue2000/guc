/**
 * Created by linyue on 17/3/16.
 */

package concurrency

type CancellationError struct {
	Message string
}

func (this *CancellationError) Error() string {
	return "[CancellationError] " + this.Message
}

