package msg

type EventCode uint

const (
	LoginCode       EventCode = 0
	FinishQueueCode EventCode = 1
	NoQueueCode     EventCode = 2
	QueueStatsCode  EventCode = 3
	TicketCode      EventCode = 4
)

type LoginClientEvent struct {
	Type  string `json:"type"` // enum?
	Token string `json:"token"`
}

type LoginServerEvent struct {
	Jwt string `json:"jwt"`
}

type QueueStatsServerEvent struct {
	ActiveTickets int32 `json:"ActiveTickets"`
	HeadPosition  int32 `json:"headPosition"`
	TailPosition  int32 `json:"tailPosition"`
}

type TicketServerEvent struct {
	TicketId string `json:"ticketId"`
	Position int32  `json:"position"`
}
