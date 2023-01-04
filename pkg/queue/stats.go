package queue

import (
	"game-soul-technology/joker/joker-login-queue-server/pkg/infra"
	"math"
	"time"

	"github.com/emirpasic/gods/maps/linkedhashmap"
	"github.com/emirpasic/gods/queues/linkedlistqueue"
	"go.uber.org/zap"
)

const (
	// Initial default value of wait duration.
	initAvgWaitDuration = 3 * time.Minute

	// The size of sliding window for calculating average wait time of
	// a ticket.
	avgWaitWindowSize = 50
)

type Stats struct {
	// Used for each ticket to deduct how many tickets are in front of
	// it (ticket.position - HeadPosition).
	HeadPosition int32

	// Used for each ticket to deduct how many tickets are in back of
	// it (TailPosition - ticket.position).
	TailPosition int32

	// Avg wait time for a ticket since it was inserted into the
	// queue. Calculated by a fixed size sliding window.
	AvgWaitDuration time.Duration

	// A fixed size sliding window for calculating average wait time.
	waitDurationQueue *linkedlistqueue.Queue

	logger *zap.SugaredLogger
}

func ProvideStats(loggerFactory *infra.LoggerFactory) *Stats {
	return &Stats{
		HeadPosition: 0,
		TailPosition: 0,

		AvgWaitDuration:   initAvgWaitDuration,
		waitDurationQueue: linkedlistqueue.New(),

		logger: loggerFactory.Create("Stats").Sugar(),
	}
}

func (s *Stats) incrTailPosition() {
	if s.TailPosition < math.MaxInt32 {
		s.TailPosition += 1
	} else {
		s.TailPosition = 1
	}
}

func (s *Stats) resetHeadPosition(queue *linkedhashmap.Map) {
	if queue.Size() <= 0 {
		s.HeadPosition = s.TailPosition
		return
	}

	it := queue.Iterator()
	it.Begin()
	it.Next()
	firstTicket := it.Value().(*Ticket)
	s.HeadPosition = firstTicket.Position
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

	s.AvgWaitDuration = totalWaitDuration / time.Duration(s.waitDurationQueue.Size())
	s.logger.Infof("updated avgWaitDuration[%v]", s.AvgWaitDuration)
}
