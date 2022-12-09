package msg

type EventCode uint

const (
	LoginCode       EventCode = 0
	FinishQueueCode EventCode = 1
	QueueStatusCode EventCode = 2
	NoQueueCode     EventCode = 3
)

type LoginClientEvent struct {
	Type  string `json:"type"` // enum?
	Token string `json:"token"`
}

type LoginServerEvent struct {
	Jwt string `json:"jwt"`
}

type QueueStatusServerEvent struct {
}
