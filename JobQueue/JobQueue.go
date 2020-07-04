package JobQueue

import "fmt"

type QueueWorker interface {
	 Worker(id int)
}

type WorkerPool struct {
	nWorker  int
	JobQueue chan interface{}
	Worker   QueueWorker

}

type job struct {
	data interface{}
}


func NewWorkerPool(worker QueueWorker,nWorker int) *WorkerPool {
	workerPool := new(WorkerPool)
	workerPool.JobQueue = make(chan interface{},nWorker)
	workerPool.nWorker = nWorker
	workerPool.Worker = worker
	return workerPool
}


/*func worker(workerPool *WorkerPool, id int ){
	for j := range  workerPool.jobQueue{
		ans := j * 5
		fmt.Printf("job done by woker #%v, result is : %v\n",id, ans)
	}
}
*/
func (workerPool *WorkerPool)Start(){
	for k:= 0; k < workerPool.nWorker; k++ {
		fmt.Printf("launched worker #%v\n",k)
		go workerPool.Worker.Worker(k)
	}
}


func (workerPool *WorkerPool) AddJob(j interface{}){

	workerPool.JobQueue <- j
}