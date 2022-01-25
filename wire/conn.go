package wire

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
)

// Conn describes a connection between agent and coordinator
type Conn struct {
	conn          net.Conn
	connLock      sync.Mutex
	nextMessageID int32
	Tag           string
}

// Listener describes a server that can accept new connections
type Listener struct {
	listener net.Listener
}

// NewServer will return a new instance of Listener, which is listening on the
// port given in the `port` parameter
func NewServer(port int) (*Listener, error) {
	l, err := net.Listen("tcp4", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}

	return &Listener{
		listener: l,
	}, nil
}

// Accept will block until a new client connects, and returns an instance of
// Conn for that connection
func (l *Listener) Accept() (*Conn, error) {
	c, err := l.listener.Accept()
	if err != nil {
		return nil, err
	}
	return &Conn{
		conn:     c,
		connLock: sync.Mutex{},
	}, nil
}

// NewClient will open a TCP connection to the given host and port and return an
// instance of Conn for the connection that's open
func NewClient(host string, port int) (*Conn, error) {
	c, err := net.Dial("tcp4", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, fmt.Errorf("could not dial: %v", err)
	}

	return &Conn{
		conn:     c,
		connLock: sync.Mutex{},
	}, nil
}

// Close will close the connection
func (c *Conn) Close() error {
	return c.conn.Close()
}

// readBytes will try to read exactly the passed number of bytes from the
// connection - blocking until the bytes are all received, or fail when it is
// unable to read
func (c *Conn) readBytes(size int) ([]byte, error) {
	b := make([]byte, size)
	if n, err := io.ReadFull(c.conn, b); err != nil || n != size {
		return nil, fmt.Errorf("could not read element (%d): %v", n, err)
	}

	return b, nil
}

// SetMessageID uses an atomic message ID counter kept in the connection to
// assign a sequential message ID to the instance of a Msg, which is required to
// have a MsgHeader
func (c *Conn) SetMessageID(msg Msg) {
	id := atomic.AddInt32(&c.nextMessageID, 1)
	SetMessageHeaderID(msg, "ID", int(id))
}

// Recv will try to read a Msg from the wire connection or return an error if it
// is unable to do so. The method call will block until either a message or
// error is available
func (c *Conn) Recv() (Msg, error) {
	// First read the message type (uint16)
	var mti16 int16
	err := binary.Read(c.conn, binary.BigEndian, &mti16)
	if err != nil {
		return nil, err
	}
	mt := MessageType(mti16)

	// Then read the message length (int32)
	var msglen int32
	err = binary.Read(c.conn, binary.BigEndian, &msglen)
	if err != nil {
		return nil, err
	}

	// Read the whole message payload from the wire or die trying
	b, err := c.readBytes(int(msglen))
	if err != nil {
		return nil, err
	}

	// Once the whole message is read, decode it and return it
	msg, err := msgFromBytes(mt, b)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

// Send will encode a message into bytes and send it over the wire
func (c *Conn) Send(msg Msg) error {
	if msg == nil {
		return errors.New("cannot send nil message")
	}
	// Set an incremental message ID if one has not been set by the caller
	if GetMessageHeaderID(msg, "ID") == 0 {
		c.SetMessageID(msg)
	}

	// Encode the message into a byte array
	msgb, err := msgToBytes(msg)
	if err != nil {
		return err
	}

	// Make sure no two callers submit over the wire at the same time by locking
	// the connLock mutex
	c.connLock.Lock()
	defer c.connLock.Unlock()

	// Write the message type
	err = binary.Write(c.conn, binary.BigEndian, int16(GetMessageType(msg)))
	if err != nil {
		return err
	}

	// Write the message length
	err = binary.Write(c.conn, binary.BigEndian, int32(len(msgb)))
	if err != nil {
		return err
	}

	// Write the message to the wire
	sent := 0
	for sent < len(msgb) {
		n, err := c.conn.Write(msgb[sent:])
		sent += n
		// If err is nil, but we didn't write all bytes for some reason, we just
		// go another cycle to retry.
		if sent != len(msgb) && err != nil {
			return fmt.Errorf(
				"not all bytes sent: %d vs %d - %v - MessageType: %d",
				n,
				len(msgb),
				err,
				GetMessageType(msg),
			)
		}
	}

	// All good - it's sent!
	return nil
}
