package server

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	uuid "github.com/satori/go.uuid"

	"github.com/joesonw/drlee/proto"
)

func (s *Server) RPCDebug(ctx context.Context, req *proto.DebugRequest) (res *proto.DebugResponse, err error) {
	switch req.Name {
	case "reload":
		dur := time.Nanosecond * time.Duration(binary.LittleEndian.Uint64(req.Body))
		err = s.StopLua(dur)
		if err != nil {
			return
		}

		err = s.LoadLua(ctx, s.luaScript)
		if err != nil {
			return
		}

		res = &proto.DebugResponse{Body: []byte("reloaded")}
	case "call":
		{
			call := &proto.CallRequest{}
			err = json.Unmarshal(req.Body, call)
			if err != nil {
				return
			}

			var result []byte
			result, err = s.callLuaRPCMethod(ctx, &RPCRequest{
				ID:         uuid.NewV4().String(),
				Name:       call.Name,
				Body:       call.Body,
				Timestamp:  time.Now(),
				IsLoopBack: true,
			})
			if err != nil {
				return
			}

			res = &proto.DebugResponse{Body: result}
		}
	default:
		res = &proto.DebugResponse{Body: []byte(fmt.Sprintf("command '%s' not found", req.Name))}
	}
	return res, err
}

func (s *Server) RPCDebugStream(req *proto.DebugRequest, stream proto.RPC_RPCDebugStreamServer) error {
	return nil
}
