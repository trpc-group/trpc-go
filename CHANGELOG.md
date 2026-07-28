# Changelog

## [Unreleased]

## [1.1.0] - 2026-07-28

### Compatibility

- The version component constants `MajorVersion`, `MinorVersion`,
  `PatchVersion`, and `VersionSuffix` are no longer exported. Use
  `trpc.Version()` to obtain the complete version string.
- Restored compatibility of the `server.Service` interface by removing its
  `ServiceName` requirement. Code written specifically against the v1.0.4
  interface can use a local interface assertion to access the optional method.
  ([#247](https://github.com/trpc-group/trpc-go/pull/247))
- Adding overload-control configuration changed the exact anonymous type of
  `Config.Server`. Normal field access and YAML configuration remain
  compatible, but whole-struct assignments using an identical anonymous type
  need to include the new field.
  ([#230](https://github.com/trpc-group/trpc-go/pull/230))
- `transport.ListenServeOptions` is no longer comparable after adding
  keep-order callbacks. Code using the entire value as a map key or comparing
  it with `==` needs to compare individual fields instead.
  ([#229](https://github.com/trpc-group/trpc-go/pull/229))
- `http.ClientRspHeader` is no longer comparable after adding the SSE receive
  callback. Code using the entire value as a map key or comparing it with `==`
  needs to compare individual fields instead.
  ([#223](https://github.com/trpc-group/trpc-go/pull/223))

### Added

- Added overload-control configuration, server hooks, and reusable
  overload-control helpers.
  ([#230](https://github.com/trpc-group/trpc-go/pull/230))
- Added keep-order request support to client and server transports.
  ([#229](https://github.com/trpc-group/trpc-go/pull/229))
- Added server precool checks for graceful readiness transitions.
  ([#228](https://github.com/trpc-group/trpc-go/pull/228))
- Added client connection prewarming.
  ([#227](https://github.com/trpc-group/trpc-go/pull/227))
- Added FastHTTP client and server transports.
  ([#225](https://github.com/trpc-group/trpc-go/pull/225))
- Added HTTP response-header helpers, HTTP decorators, server-sent events,
  stream initialization timeouts, and stream flow-control options.
  ([#223](https://github.com/trpc-group/trpc-go/pull/223))
- Added `log.DisableTrace` for disabling trace metadata injection.
  ([#222](https://github.com/trpc-group/trpc-go/pull/222))
- Exported the default HTTP serialization tag for integration reuse.
  ([#233](https://github.com/trpc-group/trpc-go/pull/233))

### Changed

- Updated the tnet transport dependency to v1.1.0.
  ([#236](https://github.com/trpc-group/trpc-go/pull/236))
- Centralized protocol names in internal constants and refreshed protocol
  dependencies without exposing internal-only APIs.
  ([#226](https://github.com/trpc-group/trpc-go/pull/226))

### Fixed

- Applied server filters to custom REST routes.
  ([#237](https://github.com/trpc-group/trpc-go/pull/237))
- Limited retained REST request and response body buffers to avoid keeping
  oversized allocations in pools.
  ([#234](https://github.com/trpc-group/trpc-go/pull/234))
- Preserved the configured cookie jar when an HTTP client enables TLS.
  ([#231](https://github.com/trpc-group/trpc-go/pull/231))
- Reported partial-read timeouts as read failures instead of successful
  transport reads.
  ([#224](https://github.com/trpc-group/trpc-go/pull/224))
- Fixed top-level YAML key handling, server log context, and wrapped errors in
  TCP multiplexed transports.
  ([#221](https://github.com/trpc-group/trpc-go/pull/221))
- Prevented late multiplexed responses from being dispatched after their
  request had closed.
  ([#220](https://github.com/trpc-group/trpc-go/pull/220))
- Fixed codec metadata cloning, close-frame business-error handling, and
  concurrent metrics access.
  ([#219](https://github.com/trpc-group/trpc-go/pull/219))
- Fixed unwatched configuration providers, propagated the latest HTTP context,
  and completed REST message lifecycle handling.
  ([#218](https://github.com/trpc-group/trpc-go/pull/218))

### Documentation

- Clarified HTTP route and client usage documentation.
  ([#238](https://github.com/trpc-group/trpc-go/pull/238))
- Added examples for FastHTTP, precooling, HTTP features, RPC lifecycles,
  keep-order transports, TLS transports, timeout scenarios, connection
  prewarming, and server-sent events.
  ([#240](https://github.com/trpc-group/trpc-go/pull/240),
  [#241](https://github.com/trpc-group/trpc-go/pull/241),
  [#242](https://github.com/trpc-group/trpc-go/pull/242),
  [#243](https://github.com/trpc-group/trpc-go/pull/243),
  [#244](https://github.com/trpc-group/trpc-go/pull/244),
  [#245](https://github.com/trpc-group/trpc-go/pull/245),
  [#246](https://github.com/trpc-group/trpc-go/pull/246))

## [1.0.4] - 2026-06-14

The v1.0.4 version was published as a Git tag without a separate GitHub
Release.

### Compatibility

- Replaced the linear REST router with a trie-backed router that gives static
  segments precedence over parameters, and parameters precedence over
  wildcards. Applications relying on ambiguous route registration order should
  verify matching behavior.
  ([#217](https://github.com/trpc-group/trpc-go/pull/217))
- Added `ServiceName` to the `server.Service` interface. This requirement was
  removed in v1.1.0 while retaining optional service-name support on built-in
  services. ([#215](https://github.com/trpc-group/trpc-go/pull/215),
  [#247](https://github.com/trpc-group/trpc-go/pull/247))

### Added

- Added stream close-frame metadata, client-stream node selection, custom HTTP
  TLS certificate providers, HTTP/2 configuration, and server service names.
  ([#215](https://github.com/trpc-group/trpc-go/pull/215))
- Added LZ4 request and response compression.
  ([#216](https://github.com/trpc-group/trpc-go/pull/216))

### Fixed

- Hardened admin pprof route handling and graceful shutdown behavior.
  ([#206](https://github.com/trpc-group/trpc-go/pull/206))
- Fixed transport half-close handling, HTTP edge cases, selector behavior,
  multiplexed connection pools, metrics, and configuration stability.
  ([#214](https://github.com/trpc-group/trpc-go/pull/214))
- Removed duplicate admin handler calls.
  ([#193](https://github.com/trpc-group/trpc-go/pull/193))

## [1.0.3] - 2024-05-16

### Added

- Added registration for custom log format encoders.
  ([#146](https://github.com/trpc-group/trpc-go/pull/146))
- Allowed selectors to provide a custom `net.Addr` parser.
  ([#176](https://github.com/trpc-group/trpc-go/pull/176))

### Changed

- Updated tnet to handle negative connection idle timeouts.
  ([#169](https://github.com/trpc-group/trpc-go/pull/169))
- Updated `google.golang.org/protobuf` from v1.30.0 to v1.33.0.
  ([#171](https://github.com/trpc-group/trpc-go/pull/171))

### Fixed

- Synchronized a collection of client, codec, configuration, logging, metrics,
  naming, server, stream, and transport fixes.
  ([#161](https://github.com/trpc-group/trpc-go/pull/161))

### Documentation

- Clarified service listeners, JSON API configuration, server and client idle
  timeouts, and related usage guidance.
  ([#144](https://github.com/trpc-group/trpc-go/pull/144),
  [#160](https://github.com/trpc-group/trpc-go/pull/160),
  [#166](https://github.com/trpc-group/trpc-go/pull/166),
  [#170](https://github.com/trpc-group/trpc-go/pull/170),
  [#172](https://github.com/trpc-group/trpc-go/pull/172))

## [1.0.2] - 2023-12-05

### Added

- Restored the default HTTP server transport.
  ([#140](https://github.com/trpc-group/trpc-go/pull/140))

### Changed

- Avoided mutating `registry.Node` in `LoadNodeConfig` to prevent data races
  during node selection.
  ([#138](https://github.com/trpc-group/trpc-go/pull/138))
- Replaced unnecessary `fmt.Errorf` calls with `errors.New`.
  ([#127](https://github.com/trpc-group/trpc-go/pull/127))

### Fixed

- Fixed connection overwriting when a stream client reused the same local port.
  ([#131](https://github.com/trpc-group/trpc-go/pull/131))
- Fixed plugin and metadata examples.
  ([#128](https://github.com/trpc-group/trpc-go/pull/128))

### Documentation

- Expanded contribution guidance, API testing documentation, test
  documentation, and server routine-limit guidance.
  ([#125](https://github.com/trpc-group/trpc-go/pull/125),
  [#129](https://github.com/trpc-group/trpc-go/pull/129),
  [#133](https://github.com/trpc-group/trpc-go/pull/133),
  [#134](https://github.com/trpc-group/trpc-go/pull/134),
  [#137](https://github.com/trpc-group/trpc-go/pull/137))

## [1.0.1] - 2023-10-20

### Changed

- Updated module dependencies for the v1.0 release line and upgraded
  `golang.org/x/net`.
  ([#98](https://github.com/trpc-group/trpc-go/pull/98),
  [#106](https://github.com/trpc-group/trpc-go/pull/106))
- Removed internal-only information from the client package.
  ([#103](https://github.com/trpc-group/trpc-go/pull/103))
- Improved stream examples and normalized YAML import aliases.
  ([#110](https://github.com/trpc-group/trpc-go/pull/110),
  [#112](https://github.com/trpc-group/trpc-go/pull/112))

### Fixed

- Registered the stream server transport when using the Go network transport.
  ([#101](https://github.com/trpc-group/trpc-go/pull/101))
- Corrected module definitions in examples.
  ([#105](https://github.com/trpc-group/trpc-go/pull/105))

### Documentation

- Reworked package comments and repaired links across the README, quick start,
  server guide, examples, and benchmark documentation.
  ([#92](https://github.com/trpc-group/trpc-go/pull/92),
  [#93](https://github.com/trpc-group/trpc-go/pull/93),
  [#94](https://github.com/trpc-group/trpc-go/pull/94),
  [#96](https://github.com/trpc-group/trpc-go/pull/96),
  [#97](https://github.com/trpc-group/trpc-go/pull/97),
  [#109](https://github.com/trpc-group/trpc-go/pull/109),
  [#113](https://github.com/trpc-group/trpc-go/pull/113),
  [#115](https://github.com/trpc-group/trpc-go/pull/115))

## [1.0.0] - 2023-10-17

### Added

- Published the first stable tRPC-Go release.

[Unreleased]: https://github.com/trpc-group/trpc-go/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/trpc-group/trpc-go/compare/v1.0.4...v1.1.0
[1.0.4]: https://github.com/trpc-group/trpc-go/compare/v1.0.3...v1.0.4
[1.0.3]: https://github.com/trpc-group/trpc-go/compare/v1.0.2...v1.0.3
[1.0.2]: https://github.com/trpc-group/trpc-go/compare/v1.0.1...v1.0.2
[1.0.1]: https://github.com/trpc-group/trpc-go/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/trpc-group/trpc-go/releases/tag/v1.0.0
