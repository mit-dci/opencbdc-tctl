package wire

import (
	binser "github.com/kelindar/binary"
)

// msgFromBytes translates a byteslice to a Msg of the passed type
func msgFromBytes(mt MessageType, payload []byte) (Msg, error) {
	msg := NewMessage(mt)
	err := binser.Unmarshal(payload, msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// msgBytes translates a Msg (and any of its underlying types) into a byteslice
// using binser
func msgToBytes(msg Msg) ([]byte, error) {
	return binser.Marshal(msg)
}
