# Changelog

This changelog records notable user-facing changes. Entries are grouped by
release and include the merge date and public pull request for traceability.

## [Unreleased]

## [1.1.0] - 2026-07-27

### Compatibility

- 2026-07-27 [CHANGE] Restored compatibility of the `server.Service`
  interface by removing its `ServiceName` requirement. Code that used the
  short-lived method through the interface can use a local interface assertion
  instead. ([#247](https://github.com/trpc-group/trpc-go/pull/247))
- 2026-06-16 [CHANGE] Adding overload-control configuration changed the exact
  anonymous type of `Config.Server`. Normal field access and YAML
  configuration remain compatible, but whole-struct assignments using an
  identical anonymous type need to include the new field.
  ([#230](https://github.com/trpc-group/trpc-go/pull/230))
- 2026-06-16 [CHANGE] `transport.ListenServeOptions` is no longer comparable
  after adding keep-order callbacks. Code using the entire value as a map key
  or comparing it with `==` needs to compare individual fields instead.
  ([#229](https://github.com/trpc-group/trpc-go/pull/229))
- 2026-06-15 [CHANGE] `http.ClientRspHeader` is no longer comparable after
  adding the SSE receive callback. Code using the entire value as a map key or
  comparing it with `==` needs to compare individual fields instead.
  ([#223](https://github.com/trpc-group/trpc-go/pull/223))
- 2026-06-14 [CHANGE] The REST router now gives static segments precedence
  over parameters, and parameters precedence over wildcards. Applications
  relying on ambiguous route registration order should verify matching
  behavior. ([#217](https://github.com/trpc-group/trpc-go/pull/217))

### Added

- 2026-06-16 [FEATURE] Added overload-control configuration, server hooks, and
  reusable overload-control helpers.
  ([#230](https://github.com/trpc-group/trpc-go/pull/230))
- 2026-06-16 [FEATURE] Added keep-order request support to client and server
  transports. ([#229](https://github.com/trpc-group/trpc-go/pull/229))
- 2026-06-16 [FEATURE] Added server precool checks for graceful readiness
  transitions. ([#228](https://github.com/trpc-group/trpc-go/pull/228))
- 2026-06-16 [FEATURE] Added client connection prewarming.
  ([#227](https://github.com/trpc-group/trpc-go/pull/227))
- 2026-06-16 [FEATURE] Added FastHTTP client and server transports.
  ([#225](https://github.com/trpc-group/trpc-go/pull/225))
- 2026-06-15 [FEATURE] Added HTTP response-header helpers, HTTP decorators,
  server-sent events, stream initialization timeouts, and stream flow-control
  options. ([#223](https://github.com/trpc-group/trpc-go/pull/223))
- 2026-06-15 [FEATURE] Added `log.DisableTrace` for disabling trace metadata
  injection. ([#222](https://github.com/trpc-group/trpc-go/pull/222))
- 2026-06-14 [FEATURE] Added LZ4 request and response compression.
  ([#216](https://github.com/trpc-group/trpc-go/pull/216))
- 2026-06-14 [FEATURE] Added stream close-frame metadata, client-stream node
  selection, custom HTTP TLS certificate providers, and HTTP/2 configuration.
  ([#215](https://github.com/trpc-group/trpc-go/pull/215))
- 2025-09-23 [FEATURE] Expanded the admin service with runtime diagnostics and
  stability improvements.
  ([#206](https://github.com/trpc-group/trpc-go/pull/206))

### Changed

- 2026-06-18 [CHANGE] Updated the tnet transport dependency to v1.1.0.
  ([#236](https://github.com/trpc-group/trpc-go/pull/236))
- 2026-06-16 [CHANGE] Exported the default HTTP serialization tag for
  integration reuse. ([#233](https://github.com/trpc-group/trpc-go/pull/233))
- 2026-06-16 [CHANGE] Centralized protocol names in internal constants and
  refreshed protocol dependencies without exposing internal-only APIs.
  ([#226](https://github.com/trpc-group/trpc-go/pull/226))

### Fixed

- 2026-06-18 [BUGFIX] Applied server filters to custom REST routes.
  ([#237](https://github.com/trpc-group/trpc-go/pull/237))
- 2026-06-16 [BUGFIX] Limited retained REST request and response body buffers
  to avoid keeping oversized allocations in pools.
  ([#234](https://github.com/trpc-group/trpc-go/pull/234))
- 2026-06-16 [BUGFIX] Preserved the configured cookie jar when an HTTP client
  enables TLS. ([#231](https://github.com/trpc-group/trpc-go/pull/231))
- 2026-06-16 [BUGFIX] Reported partial-read timeouts as read failures instead
  of successful transport reads.
  ([#224](https://github.com/trpc-group/trpc-go/pull/224))
- 2026-06-15 [BUGFIX] Fixed top-level YAML key handling, server log context,
  and wrapped errors in TCP multiplexed transports.
  ([#221](https://github.com/trpc-group/trpc-go/pull/221))
- 2026-06-15 [BUGFIX] Prevented late multiplexed responses from being
  dispatched after their request had closed.
  ([#220](https://github.com/trpc-group/trpc-go/pull/220))
- 2026-06-15 [BUGFIX] Fixed codec metadata cloning, close-frame business-error
  handling, and concurrent metrics access.
  ([#219](https://github.com/trpc-group/trpc-go/pull/219))
- 2026-06-15 [BUGFIX] Fixed unwatched configuration providers, propagated the
  latest HTTP context, and completed REST message lifecycle handling.
  ([#218](https://github.com/trpc-group/trpc-go/pull/218))
- 2026-06-12 [BUGFIX] Fixed transport half-close handling, HTTP edge cases,
  selector behavior, multiplexed connection pools, metrics, and configuration
  stability. ([#214](https://github.com/trpc-group/trpc-go/pull/214))

### Documentation

- 2026-06-26 [DOCS] Added examples for HTTP/2, custom TLS certificate
  providers, HTTP decorators, response headers, serialization, and routing.
  ([#241](https://github.com/trpc-group/trpc-go/pull/241))
- 2026-06-24 [DOCS] Added connection-prewarming and server-sent event examples.
  ([#246](https://github.com/trpc-group/trpc-go/pull/246))
- 2026-06-24 [DOCS] Expanded timeout examples for client, server, stream, and
  transport scenarios.
  ([#245](https://github.com/trpc-group/trpc-go/pull/245))
- 2026-06-24 [DOCS] Added TLS transport examples.
  ([#244](https://github.com/trpc-group/trpc-go/pull/244))
- 2026-06-24 [DOCS] Added keep-order transport examples.
  ([#243](https://github.com/trpc-group/trpc-go/pull/243))
- 2026-06-24 [DOCS] Added RPC lifecycle examples for stream metadata,
  overload control, and graceful precooling.
  ([#242](https://github.com/trpc-group/trpc-go/pull/242))
- 2026-06-22 [DOCS] Added FastHTTP and server-precool examples.
  ([#240](https://github.com/trpc-group/trpc-go/pull/240))
- 2026-06-22 [DOCS] Clarified HTTP route and client usage documentation.
  ([#238](https://github.com/trpc-group/trpc-go/pull/238))

[Unreleased]: https://github.com/trpc-group/trpc-go/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/trpc-group/trpc-go/compare/v1.0.3...v1.1.0
