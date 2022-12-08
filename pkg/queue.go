package main

import (
	"fmt"
	. "game-soul-technology/joker/joker-login-queue-server/pkg/infra"
	"time"

	"github.com/emirpasic/gods/maps/linkedhashmap"
)

type Queue struct {
	// Enter queue request for a ticket from hub.
	enter chan TicketId

	// Leave queue request for a ticket from hub. Will set ticket in
	// queue to inactive.
	leave chan TicketId

	// A queue of tickets. A ticket can be active or inactive in
	// queue. Only active tickets can be dequeued, inactive tickets is
	// left in it. If an inactive ticket stays inactive for too long,
	// it will be viewed as stale and removed from the queue. It's
	// implemented as linkedhasmap since we want to find ticket
	// frequntly through ticketId, but at the same time we want to
	// record the insert order of the ticket so we can correctly
	// dequeue. Key value: ticketId -> ticket.
	ticketQueue *linkedhashmap.Map
	stats       *Stats
}

var (
	queue = NewQueue()
)

const (
	statsUpdateInterval = 10 * time.Second

	dequeueInterval = 10 * time.Second
)

type Stats struct {
	activeTickets uint
	// avgWaitDuration      time.Duration
}

func NewQueue() *Queue {
	return &Queue{
		enter:       make(chan TicketId, 1024), // need buffer?
		leave:       make(chan TicketId, 1024),
		ticketQueue: linkedhashmap.New(),
		stats:       &Stats{},
	}
}

func (q *Queue) Run() {
	go q.QueueWorker()
	go q.StatsWorker()
}

func (q *Queue) QueueWorker() {
	ticker := time.NewTicker(dequeueInterval)
	defer ticker.Stop()

	// if cannot scale, we don't need redis... scaling is way harder.
	// have to consider multiple login queue worker is reading redis
	// queue.

	// Don't need lock on ticket and queue since we're in same goroutine.
	for {
		select {
		case ticketId := <-q.enter:
			Logger.Debugf("enter ticketId[%+v]", ticketId)
			var ticket *Ticket
			if value, doesExist := q.ticketQueue.Get(ticketId); doesExist {
				// Skip for ticket that's already in queue. Remove it
				// if it's stale, so new ticket can be inserted into
				// start of the queue.
				ticket = value.(*Ticket)
				if !ticket.IsStale() {
					ticket.isActive = true
					Logger.Infof("set back to active ticket[%+v]", ticket)
					continue
				}
				q.ticketQueue.Remove(ticket.ticketId)
				Logger.Infof("removed stale ticket[%+v]", ticket)
			}

			// If not exist, create a ticket.
			ticket = &Ticket{
				ticketId:   ticketId,
				isActive:   true,
				createTime: time.Now(),
			}
			q.ticketQueue.Put(ticketId, ticket)
			Logger.Infof("inserted new ticket[%+v]", ticket)

		case ticketId := <-q.leave:
			Logger.Debugf("leave ticketId[%+v]", ticketId)
			value, ok := q.ticketQueue.Get(ticketId)
			if !ok {
				continue
			}

			ticket := value.(*Ticket)
			ticket.isActive = false
			ticket.inactiveTime = time.Now()
			Logger.Infof("set inactive ticket[%+v]", ticket)

		case <-ticker.C:
			// Dequeue the first n tickets that is active, skip
			// inactive. A fucking case: if client is inactive and not
			// stale, should we wait for him to come back or just
			// ignore him. Maybe we will just skip him until next
			// ticker.
			Logger.Debugf("dequeueing")
			slots := cfg.GetFreeSlots()

			it := q.ticketQueue.Iterator()
			for it.Begin(); it.Next() && slots > 0; {
				ticketId, ticket := it.Key().(TicketId), it.Value().(*Ticket)
				if !ticket.isActive {
					continue
				}

				Logger.Infof("dequeue slots[%+v] ticket[%+v]", slots, ticket)
				q.ticketQueue.Remove(ticketId)
				hub.finishQueue <- ticketId
				slots--
			}

			// Remove staled ticket from pool
			Logger.Debugf("removing stale ticket")
			for it.Begin(); it.Next(); {
				ticketId, ticket := it.Key(), it.Value().(*Ticket)
				if ticket.IsStale() {
					q.ticketQueue.Remove(ticketId) // TODO: will this change data structure??
					Logger.Infof("removed stale ticket[%+v]", ticket)
				}
			}
		}
	}
}

func (q *Queue) StatsWorker() {
	// TODO: for testing
	time.Sleep(3 * time.Second)

	ticker := time.NewTicker(statsUpdateInterval)
	defer ticker.Stop()

	// Ask main server how many ticket can go.
	for {
		select {
		case <-ticker.C:
			q.stats.activeTickets = 0
			for _, value := range q.ticketQueue.Values() {
				ticket := value.(*Ticket)
				if ticket.isActive {
					q.stats.activeTickets++
				}
			}

			Logger.Infof("stats updated [%+v]", q.stats)

			// TODO: remove this.
			var ticketData string
			it := q.ticketQueue.Iterator()
			for it.Begin(); it.Next(); {
				_, ticket := it.Key(), it.Value().(*Ticket)
				ticketData = ticketData + fmt.Sprintf("ticket[%+v]\n", ticket)
			}
			Logger.Debugf("ticketQueue:\n\n" + ticketData + "\n\n")
		}
	}
}
