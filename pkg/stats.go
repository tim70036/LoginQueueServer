package main

import (
	"time"

	"github.com/emirpasic/gods/queues/linkedlistqueue"
)

const (
	initAvgWaitDuration = 3 * time.Minute
	avgWaitWindowSize   = 50
)

type Stats struct {
	// Number of tickets that are active in the queue.
	activeTickets int32

	// Queue index, used for each ticket to deduct how many tickets
	// are in front/back of it.
	headPosition int32
	tailPosition int32

	// Avg wait time for a ticket since it was inserted into the
	// queue. Calculated by a fixed size sliding window.
	avgWaitDuration   time.Duration
	waitDurationQueue *linkedlistqueue.Queue
}

func ProvideStats() *Stats {
	return &Stats{
		avgWaitDuration:   initAvgWaitDuration,
		waitDurationQueue: linkedlistqueue.New(),
	}
}

func (s *Stats) updateAvgWait() {
	if s.waitDurationQueue.Size() <= 0 {
		return
	}

	it := s.waitDurationQueue.Iterator()
	var totalWaitDuration time.Duration
	for it.Next() {
		_, waitDuration := it.Index(), it.Value().(time.Duration)
		totalWaitDuration += waitDuration
	}

	s.avgWaitDuration = totalWaitDuration / time.Duration(s.waitDurationQueue.Size())
	logger.Infof("updated avgWaitDuration[%v]", s.avgWaitDuration)
}
