package msg

type EventCode uint

const (
	LoginCode       EventCode = 0
	FinishQueueCode EventCode = 1
	QueueStatusCode EventCode = 2
	NoQueueCode     EventCode = 3
)

type LoginClientEvent struct {
	Platform   string `json:"platform"` // enum?
	Credential string `json:"credential"`
}

type LoginServerEvent struct {
	Jwt string `json:"jwt"`
}

type QueueStatusServerEvent struct {
}

type NoQueueServerEvent struct {
}
