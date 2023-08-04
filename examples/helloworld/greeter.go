// Package main is the main package.
package main

import (
	"context"

	"trpc.group/trpc-go/trpc-go/log"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

var greeter = &greeterServiceImpl{
	proxy: pb.NewGreeterClientProxy(),
}

// greeterServiceImpl implements greeter service.
type greeterServiceImpl struct {
	proxy pb.GreeterClientProxy
}

// SayHello says hello request.
// trpc-cli -func "/trpc.test.helloworld.Greeter/SayHello" -target "ip://127.0.0.1:8000" -body '{"msg":"hellotrpc"}'
// curl -X POST -d '{"msg":"hellopost"}' -H "Content-Type:application/json"
// http://127.0.0.1:8080/trpc.test.helloworld.Greeter/SayHello
// curl http://127.0.0.1:8080/trpc.test.helloworld.Greeter/SayHello?msg=helloget
func (s *greeterServiceImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Debugf("SayHello recv req:%s", req)

	hi, err := s.proxy.SayHi(ctx, req)
	if err != nil {
		log.Errorf("say hi fail:%v", err)
		return nil, err
	}

	rsp := &pb.HelloReply{
		Msg: "Hello " + hi.Msg,
	}
	return rsp, nil
}

// SayHi says hi request.
func (s *greeterServiceImpl) SayHi(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Debugf("SayHi recv req:%s", req)

	rsp := &pb.HelloReply{
		Msg: "Hi " + req.Msg,
	}
	return rsp, nil
}
