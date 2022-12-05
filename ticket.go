package main

import "time"

type Ticket struct {
	ticketId     string
	isActive     bool
	createTime   time.Time
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
