package core

import (
	"container/heap"
	"errors"
	"log"
	"sync"
)

type downloadQueue struct {
	queue []*Piece
	mutex *sync.Mutex
}

func (p *downloadQueue) RemoveAt(i int) (*Piece, error) {
	if i > len(p.queue) {
		log.Fatal(2323)
		return nil, errors.New("i > len(x)")
	}
	item := heap.Remove(p,i)

	return item.(*Piece), nil
}

func (p *downloadQueue) Len() int {
	return len(p.queue)
}

func (p downloadQueue) Less(i, j int) bool {
	return p.queue[i].Availability < p.queue[j].Availability
}

func (p downloadQueue) Swap(i, j int) {

	p.queue[i], p.queue[j] = p.queue[j], p.queue[i]
	p.queue[i].QueueIndex = i
	p.queue[j].QueueIndex = j
}

func (p *downloadQueue) Push(x interface{}) {
	p.mutex.Lock()
	piece := x.(*Piece)
	piece.QueueIndex = len(p.queue)
	p.queue = append(p.queue, piece)
	p.fixQueue(piece.QueueIndex)
	p.mutex.Unlock()

}

func (p *downloadQueue) Pop() interface{} {
	p.mutex.Lock()
	l := len(p.queue)
	if l == 0 {
		return nil
	}
	old := p.queue
	item := old[l-1]
	old[l-1] = nil
	item.QueueIndex = -1
	p.queue = old[0 : l-1]
	p.mutex.Unlock()
	return item
}
/*
	Fix the download queue when a piece's availability has been changed
*/
func (p *downloadQueue) fixQueue(i int) {
	heap.Fix(p, i)
}