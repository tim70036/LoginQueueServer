package msg

type EventCode uint

const (
	ShouldQueueCode EventCode = 1000
	LoginCode       EventCode = 1001
	QueueStatsCode  EventCode = 1002
	TicketCode      EventCode = 1003
)

type LoginTypeCode uint

const (
	FacebookLogin LoginTypeCode = 0
	GoogleLogin   LoginTypeCode = 1
	AppleLogin    LoginTypeCode = 2
	LineLogin     LoginTypeCode = 3
	DeviceLogin   LoginTypeCode = 4
)

type ShouldQueueEvent struct {
	ShouldQueue bool `json:"shouldQueue"`
}

type LoginClientEvent struct {
	Type      LoginTypeCode `json:"type"`
	Token     string        `json:"token"`
	DeviceId  string        `json:"deviceId"`
	SessionId string        `json:"sessionId"`
}

type LoginServerEvent struct {
	StatusCode int    `json:"statusCode"`
	Jwt        string `json:"jwt"`
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
