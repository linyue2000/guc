/**
 * Created by linyue on 17/3/17.
 */

package locks

import (
	"time"
	"sync"
)

/**
the same as LockSupport.park() and LockSupport.unpark() in java
 */
type Parker struct {
	lock            *sync.Mutex
	blockerChannels map[interface{}]chan bool
}

func NewParker() *Parker {
	return &Parker{
		lock: &sync.Mutex{},
		blockerChannels: map[interface{}]chan bool{},
	}
}

/**
alert! multi goroutines call Park() with one blocker at one time is not permitted! otherwise some go routines will be blocked forever
 */
func (this *Parker) Park(blocker interface{}) error {
	this.lock.Lock()
	blockerChannel := this.getOrCreateBlockerChannel(blocker)
	this.lock.Unlock()
	select {
	case <-blockerChannel:
		this.lock.Lock()
		this.removeBlockerChannel(blocker)
		this.lock.Unlock()
	}
	return nil
}

func (this *Parker) Unpark(blocker interface{}) {
	this.lock.Lock()
	defer this.lock.Unlock()
	blockerChannel := this.getOrCreateBlockerChannel(blocker)
	select {
	case blockerChannel <- true:
		return
	default:
		return
	}
}

/**
one blocker can only park once
 */
func (this *Parker) ParkWithTimeout(blocker interface{}, timeout time.Duration) {
	this.lock.Lock()
	blockerChannel := this.getOrCreateBlockerChannel(blocker)
	this.lock.Unlock()
	after := time.After(timeout)
	select {
	case <-blockerChannel:
		this.lock.Lock()
		this.removeBlockerChannel(blocker)
		this.lock.Unlock()
	case <-after:
		this.lock.Lock()
		this.removeBlockerChannel(blocker)
		this.lock.Unlock()
	}
}

func (this *Parker) getOrCreateBlockerChannel(blocker interface{}) chan bool {
	blockerChannel, ok := this.blockerChannels[blocker]
	if !ok {
		blockerChannel = make(chan bool, 1)
		this.blockerChannels[blocker] = blockerChannel
	}
	return blockerChannel
}

func (this *Parker) removeBlockerChannel(blocker interface{}) {
	if _, ok := this.blockerChannels[blocker]; ok {
		delete(this.blockerChannels, blocker)
	}

}

type MultiParkError struct {

}

func (this *MultiParkError) Error() string {
	return ""
}