# tRPC-Go error code definition

| Error code | Error message |
| :----: | :---- |
| 0 | Success |
| 1 | Server decoding error |
| 2 | Server encoding error |
| 11 | The server did not call the corresponding service implementation |
| 12 | The server does not call the corresponding interface implementation |
| 21 | The request timed out in the server queue |
| 22 | The request is overloaded on the server |
| 23 | Server Current Limiting |
| 31 | Server system error |
| 41 | Authentication failed |
| 51 | Automatic verification of request parameters failed |
| 101 | Request timed out on client call |
| 111 | Client connection error |
| 121 | Client encoding error |
| 122 | Client decoding error |
| 123 | Client current limit |
| 131 | The client selects the IP route incorrectly |
| 141 | Client network error |
| 151 | Automatic verification of response parameters failed |
| 161 | The upstream caller canceled the request early |
| 201 | Client streaming queue full |
| 999 | Unspecified error |

## Error code usage example
- The trpc framework supports multiple languages, and use two fields (error codes and error information) as error uniformly.
- The error returned by the handler is the standard library error, so you need to use the trpc errs module to generate the error, otherwise an unknown error code will be returned.
- The demo is as follows:
```golang
func (s *GreeterServerImpl) SayHello(ctx context.Context, req *pb.HelloRequest, rsp *pb.HelloReply) (err error) {
    //.........
    return errs.New(1111, "business error")
}
```