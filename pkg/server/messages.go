package server

import (
	"bytes"
	"encoding/json"
	"time"
)

type MessageType byte

const (
	TypeRegistryBroadcast MessageType = 'r'
)

func marshalMessage(typ MessageType, in interface{}) []byte {
	b := bytes.NewBuffer([]byte{byte(typ)})
	if err := json.NewEncoder(b).Encode(in); err != nil {
		panic(err)
	}
	return b.Bytes()
}

func unmarshalMessage(b []byte, in interface{}) error {
	return json.Unmarshal(b[1:], in)
}

type RegistryBroadcast struct {
	NodeName  string    `json:"NodeName,omitempty"`
	Timestamp time.Time `json:"Timestamp,omitempty"`
	Name      string    `json:"Name,omitempty"`
	Weight    float64   `json:"Weight,omitempty"`
	IsDeleted bool      `json:"IsDeleted,omitempty"`
}

func (b *RegistryBroadcast) Message() []byte {
	return marshalMessage(TypeRegistryBroadcast, b)
}
