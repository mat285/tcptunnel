package protocol

import (
	"encoding/binary"
	"fmt"
	"net"
)

const (
	MinSecretLength = 16
	MaxSecretLength = 32
)

const (
	ClientHelloTypeCommand = 1
	ClientHelloTypeData    = 2

	TypeServerHello = 3
)

type ClientHello struct {
	Type   byte
	ID     uint64
	Port   uint16
	Secret []byte
}

type ServerHello struct {
	Type   byte
	ID     uint64
	Port   uint16
	Secret []byte
}

func ParseClientHello(conn net.Conn) (*ClientHello, error) {
	buf := make([]byte, 1+8+2+MaxSecretLength)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}
	if n < MinSecretLength+1+8+2 {
		return nil, fmt.Errorf("Malformed client hello")
	}

	hello := ClientHello{
		Type:   buf[0],
		ID:     binary.BigEndian.Uint64(buf[1:9]),
		Port:   binary.BigEndian.Uint16(buf[9:11]),
		Secret: buf[11:n],
	}
	return &hello, nil
}

func ParseServerHello(conn net.Conn) (*ServerHello, error) {
	buf := make([]byte, 1+2+8+MaxSecretLength)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}
	if n < MinSecretLength+1+2+8 {
		return nil, fmt.Errorf("Malformed server hello")
	}
	hello := ServerHello{
		Type:   buf[0],
		ID:     binary.BigEndian.Uint64(buf[1 : 1+8]),
		Port:   binary.BigEndian.Uint16(buf[1+8 : 1+8+2]),
		Secret: buf[1+2+8 : n],
	}
	return &hello, nil
}

func ParseServerHelloBytes(buf []byte) (*ServerHello, error) {
	if len(buf) < MinSecretLength+1+2+8 {
		return nil, fmt.Errorf("Malformed server hello")
	}
	hello := ServerHello{
		Type:   buf[0],
		ID:     binary.BigEndian.Uint64(buf[1 : 1+8]),
		Port:   binary.BigEndian.Uint16(buf[1+8 : 1+8+2]),
		Secret: buf[1+2+8:],
	}
	return &hello, nil
}

func (c ClientHello) Serialize() []byte {
	ret := []byte{c.Type}
	id := make([]byte, 8)
	port := make([]byte, 2)
	binary.BigEndian.PutUint64(id, c.ID)
	binary.BigEndian.PutUint16(port, c.Port)
	return append(ret, append(id, append(port, c.Secret...)...)...)
}

func (s ServerHello) Serialize() []byte {
	ret := []byte{s.Type}
	id := make([]byte, 8)
	port := make([]byte, 2)
	binary.BigEndian.PutUint64(id, s.ID)
	binary.BigEndian.PutUint16(port, s.Port)
	return append(ret, append(id, append(port, s.Secret...)...)...)
}
