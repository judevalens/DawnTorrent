package PeerProtocol

import (
	"container/list"
	"fmt"
	"sync"
)

type Queue struct {
	queue           *list.List
	queueMutex      *sync.RWMutex
	queueEmptyMutex *sync.Mutex
	queueEmptyCond  *sync.Cond
	Worker          Worker
}

type TestQueue struct {
	Q *Queue
}

func (testQueue *TestQueue) Worker(q *Queue, id int) {
	i := 0
	fmt.Printf("\n running \n")
	for {
		fmt.Printf("step %v\n", i)

		item := q.Pop()
		data := item.(int)
		fmt.Printf("woker # %v step %v item %v\n", id, i, data)
		i++
	}
}

func (testQueue *TestQueue) wosrker(q *Queue) {
	i := 0
	fmt.Printf("running\n")
	for {
		fmt.Printf("step %v\n", i)
		//item := q.Pop()
		//data := item.(int)
		//fmt.Printf("step %v item %v\n", i,data)
		i++
	}
}

func NewQueue(worker Worker) *Queue {
	newQueue := new(Queue)
	newQueue.queue = new(list.List)
	newQueue.queueMutex = new(sync.RWMutex)
	newQueue.queueEmptyMutex = new(sync.Mutex)
	newQueue.queueEmptyCond = sync.NewCond(newQueue.queueEmptyMutex)
	newQueue.Worker = worker
	return newQueue
}

type Worker interface {
	Worker(q *Queue, id int)
}

func (q *Queue) Add(item interface{}) {
	q.queueMutex.Lock()
	q.queue.PushBack(item)
	q.queueEmptyCond.Broadcast()
	q.queueMutex.Unlock()

}

func (q *Queue) Pop() interface{} {
	var item *list.Element
	var value interface{}

	q.queueMutex.Lock()
	item = q.queue.Front()
	if item != nil {
		value = q.queue.Remove(item)
	}
	println("popped")
	q.queueMutex.Unlock()
	q.queueEmptyCond.L.Lock()
	for item == nil {
		q.queueEmptyCond.Wait()

		q.queueMutex.Lock()
		item = q.queue.Front()
		if item != nil {
			value = q.queue.Remove(item)
		}
		q.queueMutex.Unlock()

		///os.Exit(223)
	}
	q.queueEmptyCond.L.Unlock()

	return value
}

func (q *Queue) Run(i int) {
	fmt.Printf("running %v workers\n", i)
	for x := 0; x < i; x++ {
		fmt.Printf("start %v\n", x)

		go q.Worker.Worker(q, x)

		fmt.Printf("launched %v\n", x)

	}
}
