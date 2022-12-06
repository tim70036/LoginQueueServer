package msg

type WsEvent interface {
	EventCode() uint
}

func (e *EnterRequestClientEvent) EventCode() uint { return 1 }
func (e *FinishQueueServerEvent) EventCode() uint  { return 2 }
func (e *QueueStatusServerEvent) EventCode() uint  { return 3 }

type EnterRequestClientEvent struct {
	Platform   string `json:"platform` // enum?
	Credential string `json:"credential`
}

type FinishQueueServerEvent struct {
	Dummy string
}

type QueueStatusServerEvent struct {
}
