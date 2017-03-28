# GUC (go.util.concurrency)
====

#### GUC 是一个go语言的并发框架，接口完全参照java 1.7的java.util.concurrency包

#### 目前实现了
1、协程池ThreadPoolExecutor

2、一个用于阻塞、唤醒的工具（park unpark）

#### Example:

[ThreadPoolExecutor]

	//协程数量100，任务队列最大长度100000
	executorService := concurrency.NewThreadPoolExecutor(100, 100000)
	
	
	//提交任务
	task := &SampleTask{}
	future, err := executorService.Submit(task)
	if err != nil {
	    ...
	}
	
	
	//获取结果，阻塞到任务结束
	//（注意，如果用ShutdownNow()关闭了协程池，且这个任务还在队列中没进入执行，则这里会一直阻塞，
	//除非这个任务被提交到其他协程池中运行完成）
	result, err := future.Get()
	//获取结果，阻塞到任务结束或timeout
	result, err := future.Get2(time.Second)
	
	
	//立即关闭协程池，不接收新的任务，并且interrupt所有worker中正在执行的任务
	//返回值是所有没有开始执行的正在排队的任务
	notExecutedTasks := executorService.ShutdownNow()
	
	
	//关闭协程池，不接收新的任务，但是会把队列里的任务都执行完
	executorService.Shutdown()
	
	
	
	//阻塞等待协程池关闭完毕（两种关闭模式不一样，Shutdown模式会阻塞到所有任务完成）
	executorService.AwaitTermination(60 * time.Second)


[SampleTask]

	//继承Callable，给一个例子。如果不想支持打断，可以实现空的Interrupt()和IsInterrupted()，只实现Call()
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
		ticker := time.Tick(time.Millisecond * 1000)
		after := time.After(time.Second * 5)
		for {
			select {
			case <-ticker:
				fmt.Println("[" + this.id + "] counter", this.counter)
				this.counter++
			case <-after:
				this.clearOnClose()
				result := "[" + this.id + "] completed"
				fmt.Println(result)
				return "normal exit " + this.id
			case <-this.done:
				this.clearOnClose()
				this.interrupted = 1
				result := "[" + this.id + "] interrupted"
				fmt.Println(result)
				return result
			}
		}
	}
	
	func (this *SampleTask) Interrupt() {
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