package msg

import "encoding/json"

type WsMessage struct {
	EventCode uint            `json:"eventCode`
	EventData json.RawMessage `json:"eventData`
}
