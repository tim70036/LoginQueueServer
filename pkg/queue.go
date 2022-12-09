package main

import (
	"fmt"
	"math"
	"time"

	"github.com/emirpasic/gods/maps/linkedhashmap"
)

type Queue struct {
	// Enter queue request for a ticket from hub.
	enter chan TicketId

	// Leave queue request for a ticket from hub. Will set ticket in
	// queue to inactive.
	leave chan TicketId

	// Notify hub that a ticket is done queueing.
	notifyFinish chan TicketId

	// Notify a ticket has changed its status.
	notifyDirtyTicket chan *Ticket

	// Notify current stats of the queue.
	notifyStats chan *Stats

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

	config *Config
}

const (
	notifyStatsInterval = 5 * time.Second
	dequeueInterval     = 15 * time.Second
)

type Stats struct {
	activeTickets int32
	headPosition  int32
	tailPosition  int32

	// TODO
	// avgWaitDuration      time.Duration
}

func ProvideQueue(config *Config) *Queue {
	return &Queue{
		enter:             make(chan TicketId, 1024),
		leave:             make(chan TicketId, 1024),
		notifyFinish:      make(chan TicketId, 1024),
		notifyDirtyTicket: make(chan *Ticket, 1024),
		notifyStats:       make(chan *Stats, 1024),
		ticketQueue:       linkedhashmap.New(),
		stats:             &Stats{},

		config: config,
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
		case ticketId := <-q.enter:
			logger.Debugf("enter ticketId[%+v]", ticketId)
			var ticket *Ticket
			if value, doesExist := q.ticketQueue.Get(ticketId); doesExist {
				// Skip for ticket that's already in queue. Remove it
				// if it's stale, so new ticket can be inserted into
				// start of the queue.
				ticket = value.(*Ticket)
				if !ticket.IsStale() {
					ticket.isActive = true
					ticket.isDirty = true
					logger.Infof("set back to active ticket[%+v]", ticket)
					continue
				}
				q.ticketQueue.Remove(ticket.ticketId)
				logger.Infof("removed stale ticket[%+v]", ticket)
			}
			q.push(ticketId)
			logger.Infof("inserted new ticket[%+v]", ticket)

		case ticketId := <-q.leave:
			logger.Debugf("leave ticketId[%+v]", ticketId)
			value, ok := q.ticketQueue.Get(ticketId)
			if !ok {
				continue
			}

			ticket := value.(*Ticket)
			ticket.isActive = false
			ticket.inactiveTime = time.Now()
			logger.Infof("set inactive ticket[%+v]", ticket)

		case <-ticker.C:
			// Dequeue the first n tickets that is active, skip
			// inactive. If client is inactive and not stale, we will
			// just skip him until next ticker. If he never comes
			// back, will be removed due to stale.
			slots := q.config.GetFreeSlots()
			logger.Infof("dequeueing with slots[%v]", slots)

			it := q.ticketQueue.Iterator()
			for it.Begin(); it.Next() && slots > 0; {
				ticketId, ticket := it.Key().(TicketId), it.Value().(*Ticket)
				if !ticket.isActive {
					continue
				}

				q.pop(ticketId)
				q.notifyFinish <- ticketId
				slots--
				logger.Infof("dequeue ticket[%+v]", ticket)
			}

			// Remove staled ticket from pool
			logger.Infof("removing stale tickets")
			for it.Begin(); it.Next(); {
				ticketId, ticket := it.Key().(TicketId), it.Value().(*Ticket)
				if !ticket.IsStale() {
					continue
				}

				q.pop(ticketId)
				logger.Infof("removed stale ticket[%+v]", ticket)
			}

			// TODO: remove this.
			// q.dumpQueue()

			// TODO: update wait time?
		}
	}
}

func (q *Queue) statsWorker() {
	ticker := time.NewTicker(notifyStatsInterval)
	defer ticker.Stop()

	for ; true; <-ticker.C {
		q.stats.activeTickets = 0
		it := q.ticketQueue.Iterator()
		for it.Begin(); it.Next(); {
			_, ticket := it.Key().(TicketId), it.Value().(*Ticket)
			if ticket.isActive {
				q.stats.activeTickets++
			}
		}

		q.notifyStats <- q.stats

		for it.Begin(); it.Next(); {
			_, ticket := it.Key().(TicketId), it.Value().(*Ticket)
			if ticket.isDirty {
				ticket.isDirty = false
				q.notifyDirtyTicket <- ticket
			}
		}
	}
}

func (q *Queue) push(ticketId TicketId) {
	if q.stats.tailPosition < math.MaxInt32 {
		q.stats.tailPosition += 1
	} else {
		q.stats.tailPosition = 1
	}

	ticket := &Ticket{
		ticketId:   ticketId,
		isActive:   true,
		isDirty:    true,
		position:   q.stats.tailPosition,
		createTime: time.Now(),
	}
	q.ticketQueue.Put(ticketId, ticket)
}

func (q *Queue) pop(ticketId TicketId) {
	q.ticketQueue.Remove(ticketId)

	if q.stats.headPosition < math.MaxInt32 {
		q.stats.headPosition += 1
	} else {
		q.stats.headPosition = 1
	}
}

func (q *Queue) dumpQueue() {
	var ticketData string
	it := q.ticketQueue.Iterator()
	for it.Begin(); it.Next(); {
		_, ticket := it.Key(), it.Value().(*Ticket)
		ticketData = ticketData + fmt.Sprintf("ticket[%+v]\n", ticket)
	}
	logger.Debugf("ticketQueue:\n\n" + ticketData + "\n\n")
}
