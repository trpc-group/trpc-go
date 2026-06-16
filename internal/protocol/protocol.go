// Package protocol provides internal protocol and network name constants.
package protocol

const (
	// HTTP is the HTTP RPC protocol name.
	HTTP = "http"
	// HTTP2 is the HTTP/2 RPC protocol name.
	HTTP2 = "http2"
	// HTTPS is the HTTPS RPC protocol name.
	HTTPS = "https"
	// HTTPNoProtocol is the standard HTTP service protocol name.
	HTTPNoProtocol = "http_no_protocol"
	// HTTP2NoProtocol is the standard HTTP/2 service protocol name.
	HTTP2NoProtocol = "http2_no_protocol"
	// HTTPSNoProtocol is the standard HTTPS service protocol name.
	HTTPSNoProtocol = "https_no_protocol"
	// FastHTTP is the FastHTTP RPC protocol name.
	FastHTTP = "fasthttp"
	// FastHTTPNoProtocol is the standard FastHTTP service protocol name.
	FastHTTPNoProtocol = "fasthttp_no_protocol"
	// TRPC is the tRPC protocol name.
	TRPC = "trpc"
	// TNET is the tnet transport name.
	TNET = "tnet"
)

const (
	// TCP is the TCP network name.
	TCP = "tcp"
	// TCP4 is the TCP over IPv4 network name.
	TCP4 = "tcp4"
	// TCP6 is the TCP over IPv6 network name.
	TCP6 = "tcp6"
	// UDP is the UDP network name.
	UDP = "udp"
	// UDP4 is the UDP over IPv4 network name.
	UDP4 = "udp4"
	// UDP6 is the UDP over IPv6 network name.
	UDP6 = "udp6"
	// UNIX is the Unix domain socket network name.
	UNIX = "unix"
)
