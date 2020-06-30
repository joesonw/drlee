package server

import "encoding/binary"

type Meta struct {
	RPCPort int32
}

var metaEndian = binary.LittleEndian

func DecodeMeta(b []byte) Meta {
	rpcPort := metaEndian.Uint32(b[0:4])
	return Meta{
		RPCPort: int32(rpcPort),
	}
}

func (m Meta) Encode() []byte {
	b := make([]byte, 4)
	metaEndian.PutUint32(b[0:4], uint32(m.RPCPort))
	return b
}
