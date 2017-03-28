/**
 * Created by linyue on 17/3/21.
 */

package tasks

import (
	"time"
	"fmt"
	"sync/atomic"
)

type SampleTask struct {
	id          string
	counter     int32
	done        chan bool
	interrupted int32
}

func NewSampleTask(id string) *SampleTask {
	return &SampleTask{
		id: id,
		counter: 0,
		done: make(chan bool, 1),
		interrupted: 0,
	}
}

func (this *SampleTask) Call() interface{} {
	ticker := time.Tick(time.Millisecond * 10)
	after := time.After(time.Second * 5)
	for {
		select {
		case <-ticker:
			if atomic.LoadInt32(&this.interrupted) == 0 {
				fmt.Println("[" + this.id + "] counter", this.counter)
				this.counter++
			}
		case <-after:
			if atomic.LoadInt32(&this.interrupted) == 0 {
				this.clearOnClose()
				result := "[" + this.id + "] completed"
				fmt.Println(result)
				return "normal exit " + this.id
			}
		case <-this.done:
			this.clearOnClose()
			result := "[" + this.id + "] interrupted"
			fmt.Println(result)
			return result
		}
	}
}

func (this *SampleTask) Interrupt() {
	atomic.CompareAndSwapInt32(&this.interrupted, 0, 1)
	select {
	case this.done <- true:
	default:
		return
	}
}

func (this *SampleTask) IsInterrupted() bool {
	return this.interrupted == 1
}

func (this *SampleTask) clearOnClose() {
	close(this.done)
}