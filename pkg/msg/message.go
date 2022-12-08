package msg

import "encoding/json"

type WsMessage struct {
	EventCode EventCode       `json:"eventCode"`
	EventData json.RawMessage `json:"eventData"`
}
