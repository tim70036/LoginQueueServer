package main

import "time"

type TicketId string

type Ticket struct {
	ticketId TicketId

	// True if client ws connection is still open. otherwise, false.
	isActive bool

	position int32

	// The time when ticket is created.
	createTime time.Time

	// The most recent time the ticket starts become inactive. If it's
	// default value, then it means this ticket has never been
	// inactive.
	inactiveTime time.Time
}

const (
	// After client is inactive for this period, ticket is viewed as
	// stale and can be removed (not immediately removed). If client
	// come back, he will have to wait from the start of the queue.
	ticketStalePeriod = 30 * time.Second
)

func (t *Ticket) IsStale() bool {
	return !t.isActive &&
		!t.inactiveTime.IsZero() &&
		t.inactiveTime.Before(time.Now().Add(-ticketStalePeriod))
}
