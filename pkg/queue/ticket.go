package queue

import "time"

type TicketId string

type Ticket struct {
	// BJ4
	TicketId TicketId

	// Position in queue. Through this, we can know how many tickets
	// are in the front and the back of this ticket in queue.
	Position int32

	// True if client ws connection is still open. otherwise, false.
	isActive bool

	// The time when ticket is created.
	createTime time.Time

	// The most recent time the ticket starts become inactive. If it's
	// default value, then it means this ticket has never been
	// inactive.
	inactiveTime time.Time
}
