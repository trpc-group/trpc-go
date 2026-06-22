# HTTPS example

This example demonstrates HTTPS Standard Service in tRPC-Go and provides two
ways to run the client:

- Pure code example (see `client/`).
- YAML-only example (see `clientyaml/`).

The server listens on `9443` with a self-signed certificate (demo only).

## Layout

```
examples/features/https/
├── server/               # Server (9443, https_no_protocol)
├── client/               # Pure code: enable TLS via WithTarget/WithTLS
└── clientyaml/           # YAML-only: enable TLS via YAML
```

## Start the server (9443)

```bash
cd server
go run main.go -conf trpc_go.yaml
```

## Client options

Pick either the pure code client or the YAML-only client.

### Pure code ([client/](./client/))

Explicitly set backend target and TLS in code:

- Case A: `https + ca_cert:none` (insecure).
- Case B: `http + ca_cert:none` (enable TLS via `ca_cert:none`).

Run:

```bash
cd ../client
# Code uses ip://127.0.0.1:9443 + WithTLS(..., "none", ...)
# You will see three logs: A success, B failure (no ca), C success (http+none)
go run main.go
```

Notes (code):

- `protocol: https` alone does not enable TLS; you need a TLS signal:
  - `WithTLS("", "", "none", "localhost")`, or
  - YAML provides `ca_cert: "none"`/real CA, or
  - Port inference (e.g., direct `:9443`).

### YAML-only ([clientyaml/](./clientyaml/))

Three YAML files show common cases:

- `trpc_go.yaml`: `protocol: https` + `ca_cert: "none"`.
- `trpc_go_no_ca.yaml`: `protocol: https`, no `ca_cert`, target port `9443`.
- `trpc_go_http_ca.yaml`: `protocol: http` + `ca_cert: "none"`.

Run:

```bash
cd ../clientyaml
# Case A: protocol:https, no ca_cert (port 9443)
go run main.go -conf trpc_go_no_ca.yaml
# Case B: protocol:http, ca_cert:"none" (port 9443)
go run main.go -conf trpc_go_http_ca.yaml
# Case C: protocol:https, ca_cert:"none" (port 9443)
go run main.go -conf trpc_go.yaml
```

Notes (YAML):

- `dns://host` without port defaults to 80 (no TLS inference).
  - Append `:443` (or your TLS port), or
  - Provide `ca_cert: "none"`/real CA in YAML.
- Ensure the config is loaded:
  - In-service: `trpc.NewServer()` loads automatically.
  - Tools: see `clientyaml/main.go` using `LoadGlobalConfig` + `Setup`.

## Expected

- Case A (https, no ca_cert, 9443): success, body `https response body`,
  header `reply: https-response-head`.
- Case B (http, ca_cert:none, 9443): success (TLS enabled by `ca_cert:none`).

## FAQ

- Why is `protocol: https` not enough by itself?
  - TLS requires a signal (port, CA, or code `WithTLS`).
- Do I have to use 443?
  - No. 9443 works in this demo; the key is the target is a TLS service.
- Production tips?
  - Provide real `ca_cert` and `tls_server_name` instead of `none`.
