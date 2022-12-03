package main

import (
	"time"

	"github.com/emirpasic/gods/maps/linkedhashmap"
)

type LoginQueue struct {
	request chan string
	leave   chan string

	ticketQueue *linkedhashmap.Map
}

type Ticket struct {
	clientId     string
	isActive     bool
	createTime   time.Time
	inactiveTime time.Time
}

func (t *Ticket) IsStale() bool {
	return !t.isActive &&
		!t.inactiveTime.IsZero() &&
		t.inactiveTime.Before(time.Now().Add(-time.Second*30))
}

func NewLoginQueue() *LoginQueue {
	return &LoginQueue{
		request:     make(chan string), // need buffer?
		leave:       make(chan string),
		ticketQueue: linkedhashmap.New(),
	}
}

func (l *LoginQueue) Run() {
	ticker := time.NewTicker(pingPeriod)

	defer ticker.Stop()

	// if cannot scale, we don't need redis... scaling is way harder.
	// have to consider multiple login queue worker is reading redis
	// queue.

	// Don't need lock on ticket and queue since we're in same goroutine.
	for {
		select {
		case clientId := <-l.request:
			var ticket *Ticket
			if value, doesExist := l.ticketQueue.Get(clientId); doesExist {
				// Skip for ticket that's already in queue. Remove it
				// if it's stale, so new ticket can be inserted into
				// start of the queue.
				ticket = value.(*Ticket)
				if !ticket.IsStale() {
					ticket.isActive = true
					continue
				}
				l.ticketQueue.Remove(ticket.clientId)
			}

			// If not exist, create a ticket.
			ticket = &Ticket{
				clientId:   clientId,
				isActive:   true,
				createTime: time.Now(),
			}
			l.ticketQueue.Put(clientId, ticket)

		case clientId := <-l.leave:
			var ticket *Ticket
			if value, doesExist := l.ticketQueue.Get(clientId); doesExist {
				ticket = value.(*Ticket)
				ticket.isActive = false
				ticket.inactiveTime = time.Now()
			}

		case <-ticker.C:
			// Ask main server how many ticket can go. Dequeue the first n
			// tickets that is active, skip inactive.
			// A fucking case: if client is inactive and not stale, should we wait for him to come back or just ignore him.
			// Maybe we will just skip him until next ticker.
			// slots := 100
			it := l.ticketQueue.Iterator()
			for it.End(); it.Prev(); {
				clientId, ticket := it.Key(), it.Value().(Ticket)
				logger.Debugf("clientId[%v] ticket[%+v]", clientId, ticket)
				if !ticket.isActive {
					continue
				}
			}

			// Remove staled ticket from pool
			for it.Begin(); it.Next(); {
				clientId, ticket := it.Key(), it.Value().(Ticket)
				if ticket.IsStale() {
					logger.Infof("Remove stale ticket clientId[%v] ticket[%+v]", clientId, ticket)
					l.ticketQueue.Remove(clientId)
				}
			}
		}
	}
}
