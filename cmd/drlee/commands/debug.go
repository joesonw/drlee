package commands

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"os"
	"time"

	"github.com/joesonw/drlee/proto"
	"google.golang.org/grpc"

	"github.com/spf13/cobra"
	ishell "gopkg.in/abiosoft/ishell.v2"
)

type DebugCommand struct {
}

func NewDebugCommand() *DebugCommand {
	return &DebugCommand{}
}

func (d *DebugCommand) Build(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug",
		Short: "debug over rpc port",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				println("usage: drlee debug <remote rpc addrress> ")
				os.Exit(1)
			}

			cc, err := grpc.Dial(args[0], grpc.WithInsecure())
			if err != nil {
				println("unable to connect to remote rpc: " + err.Error())
				os.Exit(1)
			}

			rpc := proto.NewRPCClient(cc)

			shell := ishell.New()
			shell.AddCmd(&ishell.Cmd{
				Name: "reload",
				Func: func(ctx *ishell.Context) {
					if len(ctx.Args) < 1 {
						shell.Println("call timeout")
						return
					}

					dur, err := time.ParseDuration(ctx.Args[0])
					if err != nil {
						shell.Println(err.Error())
						return
					}

					durBytes := make([]byte, 8)
					binary.LittleEndian.PutUint64(durBytes, uint64(dur.Nanoseconds()))
					res, err := rpc.RPCDebug(context.TODO(), &proto.DebugRequest{
						Name: "reload",
						Body: durBytes,
					})
					if err != nil {
						shell.Println("remote: " + err.Error())
						return
					}
					shell.Println(string(res.Body))
				},
				Help: "reload lua script",
			})

			shell.AddCmd(&ishell.Cmd{
				Name: "call",
				Func: func(ctx *ishell.Context) {
					if len(ctx.Args) < 2 {
						shell.Println("call <name> <body>")
						return
					}
					call := &proto.CallRequest{
						Name: ctx.Args[0],
						Body: []byte(ctx.Args[1]),
					}
					b, err := json.Marshal(call)
					if err != nil {
						shell.Println(err.Error())
						return
					}
					res, err := rpc.RPCDebug(context.TODO(), &proto.DebugRequest{
						Name: "call",
						Body: b,
					})
					if err != nil {
						shell.Println("remote: " + err.Error())
						return
					}
					shell.Println(string(res.Body))
				},
				Help: "call lua rpc method directly",
			})
			shell.Run()
			os.Exit(0)
		},
	}
	return cmd
}
