# tRPC-Go Network Transport Layer

trpc-go/transport is a pluggable network transport layer, on which tRPC-Go unifies storages, message queues, timer, etc.

Transport is only responsible for basic binary data transmission, and has no business logic. We provide two default
implementations, TCP and UDP, users may implement their own one.
