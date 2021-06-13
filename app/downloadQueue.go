package app

import (
	"container/heap"
	"errors"
	"log"
	"sync"
)

type downloadQueue []*Piece

func (p *downloadQueue) RemoveAt(i int,mutex *sync.Mutex) (*Piece, error) {
	mutex.Lock()
	if i > len(*p) {
		log.Fatal(2323)
		return nil, errors.New("i > len(x)")
	}
	item := heap.Remove(p,i)

	mutex.Unlock()
	return item.(*Piece), nil
}

func (p downloadQueue) Len() int {
	return len(p)
}

func (p downloadQueue) Less(i, j int) bool {
	return p[i].Availability < p[j].Availability
}

func (p downloadQueue) Swap(i, j int) {

	p[i], p[j] = p[j], p[i]
	p[i].QueueIndex = i
	p[j].QueueIndex = j
}

func (p *downloadQueue) Push(x interface{}) {
	piece := x.(*Piece)
	piece.QueueIndex = len(*p)
	*p = append(*p, piece)
}

func (p *downloadQueue) Pop() interface{} {
	l := len(*p)
	if l == 0 {
		return nil
	}
	old := *p
	item := old[l-1]
	old[l-1] = nil
	item.QueueIndex = -1
	*p = old[0 : l-1]
	return item
}
/*
	Fix the download queue when a piece's availability has been changed
*/
func (p *downloadQueue) fixQueue(i int) {
	heap.Fix(p, i)
}