package msg

type EventCode uint

const (
	NoQueueCode    EventCode = 0
	LoginCode      EventCode = 1
	QueueStatsCode EventCode = 2
	TicketCode     EventCode = 3
)

type LoginTypeCode uint

const (
	AppleLogin    LoginTypeCode = 1000
	DeviceLogin   LoginTypeCode = 1001
	FacebookLogin LoginTypeCode = 1002
	GoogleLogin   LoginTypeCode = 1003
	LineLogin     LoginTypeCode = 1004
)

type LoginClientEvent struct {
	Type  LoginTypeCode `json:"type"`
	Token string        `json:"token"`
}

type LoginServerEvent struct {
	Jwt string `json:"jwt"`
}

type QueueStatsServerEvent struct {
	HeadPosition int32 `json:"headPosition"`
	TailPosition int32 `json:"tailPosition"`
	AvgWaitMsec  int64 `json:"avgWaitMsec"`
}

type TicketServerEvent struct {
	TicketId string `json:"ticketId"`
	Position int32  `json:"position"`
}
