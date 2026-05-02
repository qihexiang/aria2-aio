package ws

import (
	"encoding/json"
)

type Message struct {
	Type       string `json:"type"`
	InstanceID string `json:"instance_id,omitempty"`
	Data       any    `json:"data,omitempty"`
}

func (m Message) ToJSON() []byte {
	b, err := json.Marshal(m)
	if err != nil {
		return []byte(`{"type":"error","data":"marshal failed"}`)
	}
	return b
}