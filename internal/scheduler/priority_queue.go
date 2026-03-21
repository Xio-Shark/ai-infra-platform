package scheduler

import (
	"container/heap"

	"ai-infra-platform/internal/model"
)

type PriorityQueue []model.Job

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	if pq[i].Priority == pq[j].Priority {
		return pq[i].CreatedAt.Before(pq[j].CreatedAt)
	}
	return pq[i].Priority > pq[j].Priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *PriorityQueue) Push(x any) {
	*pq = append(*pq, x.(model.Job))
}

func (pq *PriorityQueue) Pop() any {
	old := *pq
	last := len(old) - 1
	item := old[last]
	*pq = old[:last]
	return item
}

func NewPriorityQueue(items []model.Job) *PriorityQueue {
	queue := PriorityQueue(items)
	heap.Init(&queue)
	return &queue
}
