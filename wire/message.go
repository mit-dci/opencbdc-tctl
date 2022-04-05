package wire

import (
	"log"
	"reflect"
)

type Msg interface {
}

type MsgHeader struct {
	ID     int
	YourID int
}

// SetMessageHeaderID uses reflection to set the message ID in the header,
// regardless of the underlying type of message. If the underlying type does not
// have a header, nothing happens
func SetMessageHeaderID(m Msg, fieldName string, id int) {
	val := reflect.ValueOf(m)
	if !val.IsValid() {
		return
	}
	if val.IsZero() {
		return
	}

	if val.Elem().Kind() != reflect.Struct {
		return
	}

	hdr := val.Elem().FieldByName("Header")
	idField := hdr.FieldByName(fieldName)
	if idField.CanSet() {
		idField.SetInt(int64(id))
	}
}

// GetMessageHeaderID returns the message ID in the header, if the underlying
// type has a header and a value for the message ID. If it doesn't have a header
// , or the ID is not set, this method will return 0
func GetMessageHeaderID(m Msg, fieldName string) int {
	val := reflect.ValueOf(m)
	if !val.IsValid() {
		return 0
	}
	if val.IsZero() {
		return 0
	}

	if val.Elem().Kind() != reflect.Struct {
		return 0
	}

	hdr := val.Elem().FieldByName("Header")

	id := hdr.FieldByName(fieldName)
	idInt, ok := id.Interface().(int)
	if !ok {
		return 0
	}
	return idInt
}

// NewMessage instantiates a new message of the passed MessageType (int16)
func NewMessage(mt MessageType) Msg {
	newType, ok := MessageTypeToTypeMap[mt]
	if !ok {
		log.Printf("Unable to create message of type %d", mt)
		return nil
	}

	return reflect.New(newType.Elem()).Interface()
}

// GetMessageType inspects a message to determine its type and returns a
// MessageType (int16)
func GetMessageType(m Msg) MessageType {
	mt, ok := TypeToMessageTypeMap[reflect.TypeOf(m)]
	if !ok {
		log.Printf("Unable to detect type of message %T", m)
		return MessageType(0)
	}
	return mt
}
