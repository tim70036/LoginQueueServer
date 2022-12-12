package queue

import (
	"fmt"
	"game-soul-technology/joker/joker-login-queue-server/pkg/config"
	"game-soul-technology/joker/joker-login-queue-server/pkg/infra"
	"time"

	"github.com/emirpasic/gods/maps/linkedhashmap"
	"go.uber.org/zap"
)

const (
	notifyStatsInterval = 5 * time.Second
	dequeueInterval     = 15 * time.Second
)

type Queue struct {
	// Enter queue request for a ticket from hub.
	Enter chan TicketId

	// Leave queue request for a ticket from hub. Will set ticket in
	// queue to inactive.
	Leave chan TicketId

	// Notify hub that a ticket is done queueing.
	NotifyFinish chan TicketId

	// Notify a ticket has changed its status.
	NotifyDirtyTicket chan *Ticket

	// Notify current stats of the queue.
	NotifyStats chan *Stats

	// A queue of tickets. A ticket can be active or inactive in
	// queue. Only active tickets can be dequeued, inactive tickets is
	// left in it. If an inactive ticket stays inactive for too long,
	// it will be viewed as stale and removed from the queue. It's
	// implemented as linkedhasmap since we want to find ticket
	// frequntly through ticketId, but at the same time we want to
	// record the insert order of the ticket so we can correctly
	// dequeue. Key value: ticketId -> ticket.
	ticketQueue *linkedhashmap.Map

	stats *Stats

	config *config.Config

	logger *zap.SugaredLogger
}

func ProvideQueue(stats *Stats, config *config.Config, loggerFactory *infra.LoggerFactory) *Queue {
	return &Queue{
		Enter:             make(chan TicketId, 1024),
		Leave:             make(chan TicketId, 1024),
		NotifyFinish:      make(chan TicketId, 1024),
		NotifyDirtyTicket: make(chan *Ticket, 1024),
		NotifyStats:       make(chan *Stats, 1024),
		ticketQueue:       linkedhashmap.New(),

		stats:  stats,
		config: config,
		logger: loggerFactory.Create("Queue").Sugar(),
	}
}

func (q *Queue) Run() {
	go q.queueWorker()
	go q.statsWorker()
}

// Don't need lock on ticket and queue since only have 1 goroutine
// that will access them. Scaling is way harder. If use redis, have to
// consider multiple login queue worker is reading redis queue.
func (q *Queue) queueWorker() {
	ticker := time.NewTicker(dequeueInterval)
	defer ticker.Stop()

	for {
		select {
		case ticketId := <-q.Enter:
			q.logger.Debugf("enter ticketId[%+v]", ticketId)
			var ticket *Ticket
			if value, doesExist := q.ticketQueue.Get(ticketId); doesExist {
				// Skip for ticket that's already in queue. Remove it
				// if it's stale, so new ticket can be inserted into
				// start of the queue.
				ticket = value.(*Ticket)
				if !ticket.IsStale() {
					ticket.isActive = true
					ticket.isDirty = true
					q.logger.Infof("set back to active ticket[%+v]", ticket)
					continue
				}
				q.ticketQueue.Remove(ticket.TicketId)
				q.logger.Infof("removed stale ticket[%+v]", ticket)
			}
			q.push(ticketId)

		case ticketId := <-q.Leave:
			q.logger.Debugf("leave ticketId[%+v]", ticketId)
			value, ok := q.ticketQueue.Get(ticketId)
			if !ok {
				continue
			}

			ticket := value.(*Ticket)
			ticket.isActive = false
			ticket.inactiveTime = time.Now()
			q.logger.Infof("set inactive ticket[%+v]", ticket)

		case <-ticker.C:
			// Dequeue the first n tickets that is active, skip
			// inactive. If client is inactive and not stale, we will
			// just skip him until next ticker. If he never comes
			// back, will be removed due to stale.
			slots := q.config.GetFreeSlots()
			q.logger.Infof("dequeueing with slots[%v]", slots)

			it := q.ticketQueue.Iterator()
			var waitDurations []time.Duration
			for it.Begin(); it.Next() && slots > 0; {
				ticketId, ticket := it.Key().(TicketId), it.Value().(*Ticket)
				if !ticket.isActive {
					continue
				}

				q.pop(ticketId)
				q.NotifyFinish <- ticketId
				slots--

				waitDuration := time.Since(ticket.createTime)
				waitDurations = append(waitDurations, waitDuration)

				q.logger.Infof("dequeue ticket[%+v] waitDuration[%v]", ticket, waitDuration)
			}

			// Remove staled ticket from pool
			q.logger.Infof("removing stale tickets")
			for it.Begin(); it.Next(); {
				ticketId, ticket := it.Key().(TicketId), it.Value().(*Ticket)
				if !ticket.IsStale() {
					continue
				}

				q.pop(ticketId)
				q.logger.Infof("removed stale ticket[%+v]", ticket)
			}

			// Update stats.
			q.stats.resetHeadPosition(q.ticketQueue)
			q.stats.updateAvgWait(waitDurations)
		}
	}
}

func (q *Queue) statsWorker() {
	ticker := time.NewTicker(notifyStatsInterval)
	defer ticker.Stop()

	for ; true; <-ticker.C {
		q.NotifyStats <- q.stats

		it := q.ticketQueue.Iterator()
		for it.Begin(); it.Next(); {
			_, ticket := it.Key().(TicketId), it.Value().(*Ticket)
			if ticket.isDirty {
				ticket.isDirty = false
				q.NotifyDirtyTicket <- ticket
			}
		}
	}
}

func (q *Queue) push(ticketId TicketId) {
	q.stats.incrTailPosition()

	ticket := &Ticket{
		TicketId:   ticketId,
		Position:   q.stats.TailPosition,
		isActive:   true,
		isDirty:    true,
		createTime: time.Now(),
	}
	q.ticketQueue.Put(ticketId, ticket)

	q.logger.Infof("inserted new ticket[%+v]", ticket)
}

func (q *Queue) pop(ticketId TicketId) {
	q.ticketQueue.Remove(ticketId)
}

func (q *Queue) dumpQueue() {
	var ticketData string
	it := q.ticketQueue.Iterator()
	for it.Begin(); it.Next(); {
		_, ticket := it.Key(), it.Value().(*Ticket)
		ticketData = ticketData + fmt.Sprintf("ticket[%+v]\n", ticket)
	}
	q.logger.Debugf("ticketQueue:\n\n" + ticketData + "\n\n")
}
