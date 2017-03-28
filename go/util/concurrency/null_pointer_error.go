/**
 * Created by linyue on 17/3/21.
 */

package concurrency

type NullPointerError struct {

}

func (this *NullPointerError) Error() string {
	return "[NullPointerError]"
}
