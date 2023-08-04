// Package main is the server main package for admin demo.
package main

import (
	"fmt"
	"net/http"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/admin"
	"trpc.group/trpc-go/trpc-go/examples/features/common"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

// testCmds defines a custom admin command
func testCmds(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("test cmds"))
}

// init registers routes for custom admin commands.
func init() {
	// Register custom handler.
	admin.HandleFunc("/testCmds", testCmds)
}

func main() {
	// Init server.
	s := trpc.NewServer()

	// Register service.
	pb.RegisterGreeterService(s, &common.GreeterServerImpl{})

	// Serve and listen.
	if err := s.Serve(); err != nil {
		fmt.Println(err)
	}
}
