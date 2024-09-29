package main

import (
	"fmt"
	"sync"
)

type Unacknowledged struct {
	message  int
	neighbor string
}

type queue struct {
	mutex sync.Mutex
	queue []*Unacknowledged
}

func NewQueue() *queue {
	return &queue{
		queue: []*Unacknowledged{},
	}
}

// Returns the first matching message from the queue
func (q *queue) Dequeue(u *Unacknowledged) (*Unacknowledged, error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.IsEmpty() {
		return nil, fmt.Errorf("cannot dequeue empty queue")
	}

	// dequeue the first member if argument is nil
	if u == nil {
		m, _ := q.Peek()
		q.queue = q.queue[1:]
		return m, nil
	}

	// find the given member
	for i, m := range q.queue {
		if m.message == u.message && m.neighbor == u.neighbor {
			q.queue = append(q.queue[:i], q.queue[i+1:]...)
			return m, nil
		}
	}

	return nil, fmt.Errorf("member not found in queue")
}

// Adds a message to the end of the queue
func (q *queue) Enqueue(u *Unacknowledged) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	q.queue = append(q.queue, u)
}

func (q *queue) IsEmpty() bool {
	return len(q.queue) == 0
}

func (q *queue) Peek() (*Unacknowledged, error) {
	if q.IsEmpty() {
		return nil, fmt.Errorf("cannot peek empty queue")
	}
	return q.queue[0], nil
}
