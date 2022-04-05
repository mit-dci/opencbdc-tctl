package wire

import "reflect"

type MessageType int16

// TypeToMessageTypeMap helps translate from type (reflect.Type) to MessageType
// (int16)
var TypeToMessageTypeMap = map[reflect.Type]MessageType{
	reflect.TypeOf(&HelloMsg{}):                     MessageType(1),
	reflect.TypeOf(&AckMsg{}):                       MessageType(2),
	reflect.TypeOf(&ErrorMsg{}):                     MessageType(3),
	reflect.TypeOf(&PrepareEnvironmentRequestMsg{}): MessageType(4),
	reflect.TypeOf(&PrepareEnvironmentReplyMsg{}):   MessageType(5),
	reflect.TypeOf(&DestroyEnvironmentMsg{}):        MessageType(6),
	reflect.TypeOf(&DeployFileRequestMsg{}):         MessageType(7),
	reflect.TypeOf(&DeployFileResponseMsg{}):        MessageType(8),
	reflect.TypeOf(&ExecuteCommandRequestMsg{}):     MessageType(9),
	reflect.TypeOf(&ExecuteCommandResponseMsg{}):    MessageType(10),
	reflect.TypeOf(&ExecuteCommandStatusMsg{}):      MessageType(11),
	reflect.TypeOf(&UpdateSystemInfoMsg{}):          MessageType(12),
	reflect.TypeOf(&PingMsg{}):                      MessageType(13),
	reflect.TypeOf(&BreakCommandRequestMsg{}):       MessageType(15),
	reflect.TypeOf(&TerminateCommandRequestMsg{}):   MessageType(16),
	reflect.TypeOf(&HelloResponseMsg{}):             MessageType(19),
	reflect.TypeOf(&DeployFileFromS3RequestMsg{}):   MessageType(20),
	reflect.TypeOf(&DeployFileFromS3ResponseMsg{}):  MessageType(21),
	reflect.TypeOf(&RenameFileRequestMsg{}):         MessageType(22),
	reflect.TypeOf(&RenameFileResponseMsg{}):        MessageType(23),
	reflect.TypeOf(&UploadFileToS3RequestMsg{}):     MessageType(24),
	reflect.TypeOf(&UploadFileToS3ResponseMsg{}):    MessageType(25),
}

// MessageTypeToTypeMap is the reverse of TypeToMessageTypeMap to translate in
// reverse, and is generated at runtime from the TypeToMessageTypeMap
var MessageTypeToTypeMap = map[MessageType]reflect.Type{}

func init() {
	// Build the reverse map
	for k, v := range TypeToMessageTypeMap {
		MessageTypeToTypeMap[v] = k
	}
}
