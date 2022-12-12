package main

import (
	"math"
	"time"

	"github.com/emirpasic/gods/maps/linkedhashmap"
	"github.com/emirpasic/gods/queues/linkedlistqueue"
)

const (
	initAvgWaitDuration = 3 * time.Minute
	avgWaitWindowSize   = 50
)

type Stats struct {
	// Used for each ticket to deduct how many tickets are in front of
	// it (ticket.position - headPosition).
	headPosition int32

	// Used for each ticket to deduct how many tickets are in back of
	// it (tailPosition - ticket.position).
	tailPosition int32

	// Avg wait time for a ticket since it was inserted into the
	// queue. Calculated by a fixed size sliding window.
	avgWaitDuration time.Duration

	// A fixed size sliding window for calculating average wait time.
	waitDurationQueue *linkedlistqueue.Queue
}

func ProvideStats() *Stats {
	return &Stats{
		headPosition: 0,
		tailPosition: 0,

		avgWaitDuration:   initAvgWaitDuration,
		waitDurationQueue: linkedlistqueue.New(),
	}
}

func (s *Stats) incrTailPosition() {
	if s.tailPosition < math.MaxInt32 {
		s.tailPosition += 1
	} else {
		s.tailPosition = 1
	}
}

func (s *Stats) resetHeadPosition(queue *linkedhashmap.Map) {
	if queue.Size() <= 0 {
		s.headPosition = s.tailPosition
		return
	}

	it := queue.Iterator()
	it.Begin()
	it.Next()
	firstTicket := it.Value().(*Ticket)
	s.headPosition = firstTicket.position
}

func (s *Stats) updateAvgWait(waitDurations []time.Duration) {
	if waitDurations == nil {
		return
	}

	for _, value := range waitDurations {
		if s.waitDurationQueue.Size() >= avgWaitWindowSize {
			s.waitDurationQueue.Dequeue()
		}
		s.waitDurationQueue.Enqueue(value)
	}

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
