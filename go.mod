module trpc.group/trpc-go/trpc-go

go 1.18

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/cespare/xxhash v1.1.0
	github.com/fsnotify/fsnotify v1.7.0
	github.com/go-playground/form/v4 v4.2.1
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.3
	github.com/golang/snappy v0.0.4
	github.com/google/flatbuffers v24.3.25+incompatible
	github.com/google/go-cmp v0.6.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/json-iterator/go v1.1.12
	github.com/lestrrat-go/strftime v1.0.6
	github.com/mitchellh/mapstructure v1.5.0
	github.com/panjf2000/ants/v2 v2.10.0
	github.com/spaolacci/murmur3 v1.1.0
	github.com/spf13/cast v1.6.0
	github.com/stretchr/testify v1.9.0
	github.com/valyala/fasthttp v1.52.0
	go.uber.org/automaxprocs v1.5.4-0.20240213192314-8553d3bb2149
	go.uber.org/zap v1.24.0
	golang.org/x/net v0.23.0
	golang.org/x/sync v0.7.0
	golang.org/x/sys v0.22.0
	google.golang.org/protobuf v1.34.2
	gopkg.in/yaml.v3 v3.0.1
	trpc.group/trpc-go/tnet v1.0.2-0.20250605025854-7d3ff1be9972
)

require (
	github.com/fasthttp/router v1.5.0
	github.com/google/pprof v0.0.0-20240722153945-304e4f0156b8
	github.com/jinzhu/copier v0.4.0
	github.com/pierrec/lz4/v4 v4.1.21
	github.com/r3labs/sse/v2 v2.10.0
	go.uber.org/atomic v1.11.0
)

require (
	github.com/andybalholm/brotli v1.1.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/kavu/go_reuseport v1.5.0 // indirect
	github.com/klauspost/compress v1.17.6 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/savsgio/gotils v0.0.0-20240704082632-aef3928b8a38 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	gopkg.in/cenkalti/backoff.v1 v1.1.0 // indirect
)

// The hash of current code of v0.11.0 does not match with
// the hash stored in sumdb.
retract v0.11.0

// Retract all versions between v0.17.0 and v0.17.2
// The trpc-go server implementation of these versions will return error code
// 171 for trpc-go client, and 141 for trpc-cpp client occasionally due to the
// changes introduced by !2139.
// This issue has been resolved in merge request !2292 and the fix is available
// in versions >=v0.17.3.
// https://go.dev/ref/mod#go-mod-file-retract
retract [v0.17.0, v0.17.2]

// v0.18.0 includes a critical bug that was introduced by !2231.
// This issue has been resolved in merge request !2321 and
// the fix is available in versions >=v0.18.1.
//
// Details:
//
// The reconstruction of the YAML nodes used for loop variables, resulting in
// plugins of the same type all sharing the configuration corresponding to the
// last name. This caused the issue of the default log output file being
// #937.
retract v0.18.0

replace trpc.group/trpc/trpc-protocol/pb/go/trpc => github.com/hyprh/trpc/pb/go/trpc v1.0.1-0.20251010083826-35ec3b4cd2b3
