# Change Log

### **注：** v0.18.x 为 tRPC-Go 的 LTS (Long Term Support, 长期维护) 版本，不会引入新特性，但会长期backport bug fixes。

> 变更记录格式示例及说明：
> 
> ````markdown
> ## [v0.18.5](https://git.woa.com/trpc-go/trpc-go/tree/v0.18.5) (2024-09-18)
> 
> ### Bug Fixes
> 
> - **http:** fix handleSSE might return token too long error (!2537) (v0.18.0)
>   - HTTP SSE 在处理时使用 trpc.DefaultMaxFrameSize（对应 10MB）而非 codec.DefaultReaderSize（4KB），以避免出现 "token too long" 错误
> 
> ### Features
> 
> - **client:** add tag to provide users with fine-grained routing (!2610)
>   - 添加标签机制，支持用户配置更细粒度的路由策略
> 
> ### Enhancements
> 
> - **test:** add test cases for UDP graceful restart (!2702)
>   - 为 UDP 优雅重启功能添加完整的测试用例，提升功能稳定性
> 
> ### Refactor
> 
> - **codec:** refactor codec package to improve readability (!2699)
>   - 重构 codec 包以提升代码可读性，优化了包的整体结构、命名规范和文档说明
> 
> ### Documentation
> 
> - **docs:** update tnet plugin support details (!2707)
>   - 更新 tnet 插件支持的详细信息，包括功能特性和使用说明
> ````
> 
> 变更记录主要包含以下几个类别：
> 
> * Bug Fixes - 问题修复：
>   * 每条记录为对应变更 MR 的标题，其中 `(!2537)` 表示变更对应的 MR 编号
>   * 最后标注的版本号如 `(v0.18.0)` 表示该 bug 的影响范围，从该版本到当前版本 `v0.18.5`
>   * 注意：仅 Bug Fixes 类别会标明版本号影响范围
>   * 每条记录下方会附带中文说明，详细描述修复内容
> 
> * Features - 新增功能：
>   * 每条记录为对应变更 MR 的标题，其中 `(!2610)` 表示变更对应的 MR 编号
>   * 包含新增的功能、接口、协议等重要更新
>   * 如果存在不兼容变更，会在说明中特别标注
> 
> * Enhancements - 功能增强：
>   * 每条记录为对应变更 MR 的标题，其中 `(!2544)` 表示变更对应的 MR 编号
>   * 包含性能优化、测试覆盖率提升等改进内容
> 
> * Refactor - 代码重构：
>   * 每条记录为对应变更 MR 的标题，其中 `(!2699)` 表示变更对应的 MR 编号
>   * 包含代码重构、包结构优化、可读性提升等工程改进
> 
> * Documentation - 文档更新：
>   * 每条记录为对应变更 MR 的标题，其中 `(!2707)` 表示变更对应的 MR 编号
>   * 包含文档的新增、更新、优化等内容
>   * 重点关注用户指南、API 文档等使用说明的完善

## [v0.19.3](https://git.woa.com/trpc-go/trpc-go/tree/v0.19.3) (2025-02-27)

### Features

- **pool/connpool:** add additional health checker to use tnet isactive (!2862)
  - 为 tnet client transport 添加额外的健康检查器，额外使用 tnet 的 isactive 方法检查连接池中的连接，避免出现 111 错误
- **pool/connpool:** allow multiple health checker to be added (!2865)
  - 允许添加多个健康检查器，提升连接池的健康检查能力（和 !2862 是一块的）


## [v0.19.2](https://git.woa.com/trpc-go/trpc-go/tree/v0.19.2) (2025-02-18)

### Features

- **lsc**: move thrift from trpc-go to trpc-codec (!2815)

### Bugfixes

- **transport**: reduce goroutine number using reflect select cases (!2750) (v0.17.0)
- **transport**: revise lifecycle manager select cases (!2840)  (v0.19.1)
- **http**: optimize value detached context scavenger with limit control (!2835) (v0.19.1)
- **codec**: allow empty pb header during decoding (!2831) (v0.1.0)

### Tests

- **transport**: fix flaky test with bigger timeout (!2772)
- **test**: add e2e test for fasthttp server without pre-configured listener (!2769)
- **transport**: fix 32-bit test (!2765)
- **test**: cleanup streaming server (!2845)
- **test**: fix graceful restart test (!2849)

### Enhancements

- **codec**: add raw frame head info into decoding error (!2827)
- **codec**: fix ReaderSize comment bit => bytes (!2820)
- **http**: add capacity shrinking for value detached context scavenger (!2792)
- **codec**: remove codec register log (!2829)
- **restful**: improve error details in HeaderMatcher error handling ( !2832)

## [v0.19.1](https://git.woa.com/trpc-go/trpc-go/tree/v0.19.1) (2024-12-17)

### Bug Fixes

- **http:** fix fasthttp transport serve listener unwrap issue (!2756) (v0.19.0)
  - 修复 fasthttp transport serve 时 listener unwrap 的问题，确保服务正常启动
- **go.mod:** fix wrong retract define (!2753) (v0.19.0-beta)
  - 修复 go.mod 中错误的 retract 定义，确保版本管理正确性
- **transport:** implement fallback mechanism for hot restart (!2729) (v0.19.0-beta)
  - 假如热重启的新进程存在新服务，则直接创建新的 listener
- **transport:** optimize UDP listeners with maxproc (!2738) (v0.1.0)
  - 使用 maxproc 替代 cpunum 优化 UDP 监听器数量

### Features

- **log:** add new file name format setting (!2617)
  - 添加新的日志文件名格式设置，方便用户自定义日志文件名

### Enhancements

- **log:** add comprehensive log documentation (!2740)
  - 添加完整的日志功能说明文档，方便用户使用
- **transport:** enhance error handling for keep-order (!2687)
  - 优化保序功能的错误处理，提升消息处理可靠性
- **http:** refactor http client transport roundtrip (!2744)
  - 修复并重构 HTTP 客户端传输层 roundtrip 实现，提升稳定性
- **{codec,transport}:** enhance protocol registry logging (!2746)
  - 为协议注册添加更详细的日志记录，提升问题诊断能力

### Documentation

- **docs:** update LTS version information (!2749, !2736)
  - 更新 LTS 版本说明，提供最新的版本支持信息
- **docs:** enhance configuration documentation (!2742, !2741)
  - 完善配置相关文档，包括过载保护策略和服务路由说明
- **docs:** improve HTTP RPC and graceful restart documentation (!2737, !2731)
  - 优化 HTTP RPC 和优雅重启相关文档，提供更详细的使用说明
- **docs:** improve documentation standardization (!2728)
  - 优化文档和代码的标准化，修复拼写错误，提升文档质量

## [v0.19.0](https://git.woa.com/trpc-go/trpc-go/tree/v0.19.0) (2024-12-04)

### Bug Fixes
- **{transport, server}:** improve rpcz Handler span management (!2698) (v0.16.0)
  - 修复 rpcz Handler span 的生命周期管理问题，确保 ender 只被调用一次，避免重复操作
- **pool/multiplexed:** fix concurrent map read/write and the fragile test cases (!2706) (v0.19.0-beta)
  - 修复并发读写 map 导致的竞态问题，同时优化了不稳定的测试用例
- **http:** fix bug in encoding map for form (!2507) (v0.13.0)
  - 修复表单编码时处理 map 类型数据的 bug，确保数据正确序列化
- **internal/graceful:** fix UDP fds loss after graceful restart (!2701) (v0.19.0-beta)
  - 修复优雅重启后 UDP 文件描述符丢失的问题，保证服务正常运行
- **server:** replace windows.SIG with syscall.SIG (!2686) (v0.15.0)
  - 将 windows.SIG 替换为 syscall.SIG，提升跨平台兼容性

### Features

- **server:** provide local server for client to call in local scope (!2638)
  - 为客户端提供本地服务调用功能，方便业务合并微服务
- **client:** implement client-side keep-order (!2627)
  - 实现客户端端保序功能，确保请求按序发送
- **transport:** implement server side keep-order interface for tnet (!2681)
  - 为 tnet 实现服务端保序接口，支持消息顺序处理
- **client:** add tag to provide users with fine-grained routing (!2610)
  - 添加标签机制，支持用户配置更细粒度的路由策略
- **{client, transport, trpc}:** add LocalAddr for msg when invoking rpc (!2696)
  - 调用 RPC 时为消息添加本地地址信息，便于问题定位和监控
- **{internal/graceful, http}:** use socketPair for gracefulRestart (!2702)
  - HTTP 优雅重启使用 socketPair 替代环境变量，提升重启可靠性
- **server:** enable DisableGracefulRestart (!2685)
  - 支持禁用优雅重启功能，满足特殊场景需求

### Enhancements

- **transport:** restore UDPServerTransportJobQueueFullFail reporting (!2708) 
  - 恢复 UDP 服务端传输层任务队列满时的失败上报，提升问题诊断能力
- **pool/connpool:** create token channel with make (!2675)
  - 使用 make 创建令牌通道以提升性能，优化内存分配
- **test:** clear metric sink to avoid verbose info (!2719)
  - 清理 metric sink 以避免冗余日志输出

### Documentation

- **docs:** optimize the fasthttp documentation (!2712)
  - 优化 fasthttp 相关文档，提供更详细的使用说明
- **docs:** update tnet plugin support details (!2707)
  - 更新 tnet 插件支持的详细信息，包括功能特性和使用说明
- **docs:** add iwiki link for noconfig (!2705)
  - 添加 noconfig 模式的 iwiki 链接，方便用户查阅
- **docs:** emphasize the importance of callee (!2688)
  - 强调 callee 配置的重要性，避免常见错误
- **docs:** provide doc for circuit breaker and rate limiting (!2682)
  - 提供熔断和限流功能的详细文档说明
- **docs:** add notes on server config path (!2674)
  - 补充服务端配置路径相关说明
- **docs:** add user guide for thrift (!2665)
  - 新增 thrift 协议支持的用户指南
- **docs:** update method timeout notes (!2659)
  - 更新方法超时配置的注意事项说明

## [v0.19.0-beta](https://git.woa.com/trpc-go/trpc-go/tree/v0.19.0-beta) (2024-10-21)
## [v0.19.0-beta](https://git.woa.com/trpc-go/trpc-go/tree/v0.19.0-beta) (2024-10-21)

### Bug Fixes

- **client:** fix missing filter name when repairing default selector filter (!2353) (v0.10.0)
- **client:** fix RequestTime with value 0 (!2467) (v0.8.2)
- **client:** assert nil err to avoid panic when err points to nil *errs.Error (!2388) (v0.8.2)
- **http:** fix bug for WithTarget in stdHTTPClient and add info for header mismatch (!2536) (v0.17.0)
- **http:** fix handleSSE might return token too long error (!2537) (v0.18.0)
- **http:** fix gzip error for http request (!2548) (v0.1.0)
- **log:** fix log level by adding CoreLevelNewer (!2351) (v0.18.0)
- **plugin:** avoid using reference to loop iterator variable (!2321) (v0.18.0)
- **pool/multiplexed:** fix unnecessary reconnections for multiplexed (!2640) (v0.6.3)
- **transport:** fix rpcz ender logic for server transport (!2376) (v0.17.0)
- **transport/tnet:** fix the priority between custom and default options (!2311) (v0.15.0)

### Features

- **{admin, server, stream}:** support profiler tag (!2369)
- **{client, http, pool, transport}:** support http pool (!2526)
- **{client, internal, naming}:** implement the broadcast call feature by modifying the stub code (!2577)
- **client:** support client configuration of caller namespace/env/set (!2614)
- **client:** support setting of caller metadata through client config (!2328)
- **codec:** support codec for thrift protocol (!2490)
- **http:** add an option to support case sensitivity selection for GetSerialization (!2391)
- **http:** add local addr in message when successful connection is obtained (!2340)
- **http:** enable fasthttp (!2452)
- **{http, restful}:** add trpc-caller-method and remote addr to msg to help report metric in plugin (!2487)
- **internal/atomic:** add comprehensive atomic implementation (!2383)
- **{internal/graceful, transport}:** support perfect graceful udp (!2462)
- **{internal/tls, transport}:** support multiple TLS certs and keys (!2629)
- **log:** add a name to zaplog to distinguish different loggers (!2504)
- **log:** desensitize address (!2335)
- **overloadctrl:** add server global config for overloadctrl (!2544)
- **overloadctrl:** add priority overload control plugin (!2294)
- **plugin:** add "type-*" match for DependsOn and FlexDependsOn (!2636)
- **pool/multiplexed:** support multiplexing reconnect configuration (!2405)
- **{reflection, errs, server}:** support server reflection (!2121)
- **restful:** support disable request timeout (!2569)
- **robust:** add report_enabled flag to configuration (!2338)
- **robust:** add robust server metrics aggregator (!2337)
- **server:** support callback before restart (!2377)
- **server:** support service option by service name (!2341)
- **transport:** add port reuse for server-side TCP transport (!2469)
- **transport:** support alloc exact buffer size for tnet udp (!2436)
- **transport:** support tnet udp transport (!2410)
- **trpc:** add CloneContextWithTimeout to preserve the timeout (!2384)
- **trpc.go:** add default values for CloseWaitTime and MaxCloseWaitTime (!2453)
- **{trpc,docs}:** add option to round up cpu quota (!2605)

### Enhancements

- **overloadctrl:** calculate client overload control information per node (!2389)
- **admin:** check error before setting header in test (!2348)
- **admin:** reset default serve mux to remove pprof registration (!2489)
- **all:** using protocol enumeration members instead of string magic literals (!2530)
- **changelog:** mark v0.18.0 as deprecated version (!2450)
- **ci:** remove ci build.yml (!2414)
- **client:** ignore conn_type configuration for http protocol (!2362)
- **client:** make configuration struct members exportable (!2421)
- **codec:** refactor message_impl.go for clarity (!2552)
- **codec:** refine magic number mismatch error (!2352)
- **codec:** message should be put back to pool (!2312)
- **{codec, server}:** assert Sizer interface for attachment to avoid extra copy (!2478)
- **codec:** use separate key for robust priority passed back (!2399)
- **config:** provide client global set name (!2545)
- **config:** rename variable to avoid confusion (!2451)
- **config:** use type definitions instead of anonymous structs (!2305)
- **errs:** check nil error to avoid panic (!2484)
- **{errs,stream}:** wrapped as trpc err when new stream (!2538)
- **example:** add example for fasthttp mux (!2616)
- **example:** add example for tnet udp and update robust import (!2492)
- **example:** fix incorrect input/output for http example (!2543)
- **examples:** use new package for robust (!2622)
- **examples:** add noconfig client configuration example (!2447)
- **examples/httprpc:** add "how to use custom field json alias in proto file" (!2342)
- **{example, test}:** add example and e2e test for tnet (!2473)
- **{example, test, testdata}:** update pile code with trpc-cmdline v2.6.1 (!2440)
- **go.mod:** retract v0.18.0 and add changelog for v0.18.1 (!2323)
- **go.mod:** remove git.woa.com/trpc/trpc-protocol/pb/go/trpc (!2533)
- **go.mod:** retract v0.17.0-v0.17.2 (!2344)
- **go.mod:** add go mod for .resources (!2524)
- **http:** capitalize the proper name ID (!2553)
- **http:** fix TestHTTPGotConnectionRemoteAddr (!2430)
- **http:** implement context scavenger to reduce goroutine number (!2403)
- **http:** optimize for code readability (!2483, !2470)
- **http:** optimize test case for readability (!2481)
- **http:** provide decodeErrHandler for user-defined error handling in ClientCodec.Decode() (!2566)
- **{http,server,transport}:** remove unused code in graceful restart feature (!2441)
- **{http,test}:** improve comment and skip multiplexed for http test cases (!2522)
- **http:** use HandleFunc to eliminate duplicated code (!2468)
- **http:** wrap error into return message (!2459)
- **internal:** define and internalize protocol name constants (!2366)
- **internal:** fix flaky test in fasttime (!2418)
- **log:** update WithContextFields comment (!2431)
- **lsc:** translate omitted comments into English (!2635)
- **multiplexed:** lower log level of invalid streamID (!2620)
- **naming:** avoid unnecessary recursion and optimize the execution logic (!2573)
- **overloadcrtl:** fix flaky test in robust server (!2419)
- **overloadctrl/robust:** fix fragile test TestCPUUsageIsWorking (!2324)
- **plugin:** refine deprecation note (!2318)
- **pool/connpool:** better check func for getDialCtx (!2330)
- **pool/connpool:** Decrement ConnectionPool.used when Conn acquisition fails (!2415)
- **pool:** enable reconnectResetInterval config for multiplexed (!2571)
- **pool:** optimize struct field layout and optimize some logic for connpool (!2557)
- **pool:** switch sync.Map to native map and optimize struct field layout for multiplexed (!2564)
- **pool:** update comment for VirtualConnection Close Method (!2406)
- **reflection:** listen on automatically selected port (!2424)
- **restful:** align the serializer names for RESTful and HTTP (!2370)
- **restful:** provide [FastHTTP]RespSerializerGetter options for user-specified Serializer (!2365)
- **restful:** provide UnquoteString to allow unquote (!2633)
- **restful:** utilize 'RawPath' for pattern matching upon 'Path' failure (!2317)
- **revert "config:** provide client global env name !2545" (!2562)
- **robust:** export getters for aggregate reporter (!2465)
- **robust:** fix fragile strategy server interceptor test (!2372)
- **robust:** fix TestAllow test case (!2378)
- **robust:** fix TestStrategyClientInterceptor test case (!2379)
- **robust:** improve dagor client/server algorithm (!2444)
- **robust:** refine robust configuration (!2332)
- **robust:** reject probability should be reverted (!2331)
- **robust:** remove robust logic from main repo (!2472)
- **server:** check context deadline to generate server timeout error (!2375)
- **server/log:** global regex pattern for desensitize (!2349)
- **{server, stream}:** append reason when return RetServerNoFunc error (!2361)
- **stream:** only wraps io.EOF when stream is already closed (!2449)
- **stream:** remove punctuation mark from err msg at client invoke (!2540)
- **test:** add e2e test cases for gzip compression cases (!2558)
- **test:** add e2e test to validate no Read Frame Fail error (!2480)
- **test:** add reuseport e2e test (!2479)
- **test:** adjust the priority of opts passed by the server to the highest for TRPCServer and StreamingServer (!2637)
- **test:** assert additional error code for e2e test (!2327)
- **test:** checks for errors before retrieving the value (!2626)
- **test:** enhance the extensibility of tests and fix errors (!2494)
- **test:** fix test cases and add test cases for protocol mismatch (!2532)
- **{test, http}:** fix some comments for cr (!2541)
- **test:** make http_test.go compatible with the upcoming fasthttp_test.go (!2529)
- **test:** test for multi plugins with same type (!2336)
- **transport:** optimized the order of NewClientStreamTransport (!2385)
- **{transport, test}:** fix tnet udp transport concurrent safe of remoteAddr (!2550)
- **transport/tnet:** do not use tnet's idle timer (!2319)
- **transport/tnet:** use udpTaskPool rather than taskPool for udp (!2556)
- **transport:** update log message when serve stream exit (!2615)
- **transport:** use shorter update interval for setting of read timeout (!2454)
- **trpc:** remove periodicallyUpdateGOMAXPROCS func (!2310)
- **trpc:** synchronize the serialization types (!2493)
- **trpc/filter:** delete useless filtername fixTimeout (!2360)

### Documentations

- **admin:** fix port incomplete typo in readme (!2359)
- **client:** update WithCalleeMetadata and WithCallerMetadata options comments (!2386)
- **{client, docs}:** provide yaml config for multiplexed and improve the docs (!2520)
- **changelog:** add "Breaking Changes" for v0.7.3 to help users upgrade (!2623)
- **codec:** fix format rendering error in README (!2407)
- **codec:** fix the broken git code links that cannot be found (!2363)
- **codec:** update README (!2401)
- **docs:** add default priority strategy for overloadctrl (!2567)
- **docs:** add description of token expired time in knocknock (!2371)
- **docs:** add doc for start_reject_grace_period and quiescent_period (!2563)
- **docs:** add graceful exit documentation (!2534)
- **docs:** add handle error logic when using config package (!2373)
- **docs:** add "how to find server owner in knocknock" (!2339)
- **docs:** add http server configuration notes (!2412)
- **docs:** add instructions for bu_id apply (!2503)
- **docs:** add instructions for overload_control's flags (!2346)
- **docs:** add introduction for ResetInterval and fix mistake for the docs of routing (!2602)
- **docs:** add limiting rules are based on the configuration of the callee service on polaris (!2354)
- **docs:** add links to overview of overload control (!2554)
- **docs:** add note and example on setting of max frame size (!2630)
- **docs:** add note on before filter for trpc-robust (!2628)
- **docs:** add notes on close wait time configuration for graceful restart (!2381)
- **docs:** add notes on fallback logic for overload control (!2329)
- **docs:** add notes on graceful stop (!2531)
- **docs:** add notes on method name config (!2429)
- **docs:** add notes on multiple body read for http rpc (!2320)
- **docs:** add notice on conn_type configuration (!2598)
- **docs:** add usage of non retry errors/fatal errors in hedge/retry documentation (!2445)
- **docs:** add usage of non retry errors/fatal errors in slime/retry documentation (!2443)
- **docs:** add validation v3 quickstart (!2578)
- **docs:** change version requirement of slime (!2448)
- **docs:** require latest version of overloadctrl and runtime-metrics (!2488)
- **docs/config:** add "how to use rainbow to manage client configuration" (!2542)
- **docs:** delete faq directory and optimize showcase directory structure (!2506)
- **docs:** emphasize name finding for http rpc service (!2502)
- **{docs, errs}:** move error code FAQ to err's README (!2466)
- **{docs, errs}:** remove unused docs and fix some links (!2496)
- **docs/faq:** update docs about generating stream stub (!2432)
- **docs:** fix readme typo (!2343)
- **docs:** fix the duplicate ports (!2358)
- **docs:** fix the timing of the execution of shutdown hooks (!2535)
- **docs:** fix typo (!2347)
- **docs:** fix typo (!2374)
- **docs:** fix typo in flatbuffers (!2528)
- **docs:** fix typo in readme (!2364)
- **docs:** fix typos and grammar mistakes for md (!2501)
- **docs:** fix version info for MaxCloseWaitTime and CloseWaitTime defaults (!2631)
- **docs/kk:** update faq about 141 error from http transport client (!2475)
- **docs/knocknock:** update "verify ip failed" error faq for TKEx-TEG platform (!2565)
- **{docs, metrics}:** move routing and monitor FAQ to corresponding chapter (!2477)
- **docs:** move authentication_authorization.zh_CN.md to knocknock/auth-center repo (!2600)
- **docs:** move client and http FAQ to corresponding chapter (!2455)
- **docs:** move code FAQs to server overview (!2491)
- **docs:** move config FAQ to corresponding chapter (!2439)
- **docs:** move environment FAQ to corresponding chapter (!2426)
- **docs:** move server FAQ to corresponding chapter (!2446)
- **docs:** optimize comments in client config doc (!2397)
- **docs:** optimize the resources directory structure and fix the relative path of the images in the documents (!2525)
- **{docs, pool}:** fix typos (!2505)
- **docs:** provide command-line parameters documentation (!2551)
- **docs:** provide example of building a frontend service using the trpc-go (!2624)
- **docs:** provide filter examples on setting priority (!2539)
- **docs:** provide trpc-robust documentation (!2521)
- **docs:** quote URLs in curl commands and add code block languages (!2350)
- **docs:** reduce overloadctrl configs (!2495)
- **docs:** remend changelog (!2308)
- **docs/showcase:** update how to use canary routing in non-123 platform (!2474)
- **docs:** specify that version is trpc framework version (!2304)
- **docs:** sync the code_interoperability doc with the actual code (!2400)
- **docs:** update async call example by using CloneContextWithTimeout (!2527)
- **docs:** update config notes on overload control (!2482)
- **docs:** update CONTRIBUTING.md (!2367)
- **docs:** update env setup on v1 and v2 (!2333)
- **docs:** update explanation of cpu smoothing (!2576)
- **docs:** update idletime to 60000 to avoid confusion (!2568)
- **docs:** update links of code_interoperability doc and fix typo (!2402)
- **docs:** update overloadctrl version requirements (!2476)
- **docs:** update server timeout notes (!2396)
- **docs:** update slime doc on setting of retriable errors (!2422)
- **docs:** update tnet client configuration (!2394)
- **docs:** update trpc-robust docs on filter position and degrade strategy (!2575)
- **docs:** update v0.18.5 changelog (!2609)
- **docs/user_guide/graceful_restart:** add version information in docs. (!2427)
- **docs/user_guide/server/overview:** add authentication method description (!2428)
- **doc:** update CONTRIBUTING.md to add replace method (!2390)
- **doc:** use relative path for readme (!2413)
- **examples:** add mtls demo with doc (!2420)
- **{examples, docs}:** fix typos and grammar mistakes for md (!2499)
- **{http, examples, docs}:** improve docs and add examples for better sse (!2471)
- **{http, examples, docs}:** support APIs that might return SSE and non-SSE response (!2486)
- **log:** add description of the effectiveness of configuration fields (!2334)
- **log:** add unit description for max_age (!2625)
- **lsc:** fix some typos and grammar errors (!2382, !2387, !2409, !2411, !2555)
- **lsc:** fix typo for absent spaces after colon (!2497)
- **overloadctrl:** remove client overloadctrl doc (!2559)
- **restful:** supplement some documentation related to restful options (!2500)
- **robust:** add doc on how to set priority (!2380)
- **robust:** fix typo in readme (!2368)
- **server:** fix some typos (!2604)
- **{transport, restful, codec}:** fix some typos and grammar errors (!2442)

## [v0.18.7](https://git.woa.com/trpc-go/trpc-go/tree/v0.18.7) (2024-12-05)

### Bug Fixes
- **{transport, server}:** improve rpcz Handler span management (!2698) (v0.16.0)
  - 修复 rpcz Handler span 的生命周期管理问题，确保 ender 只被调用一次，避免重复操作
- **http:** fix bug in encoding map for form (!2507) (v0.13.0)
  - 修复表单编码时处理 map 类型数据的 bug，确保数据正确序列化
- **server:** replace windows.SIG with syscall.SIG (!2686) (v0.15.0)
  - 将 windows.SIG 替换为 syscall.SIG，提升跨平台兼容性

## [v0.18.6](https://git.woa.com/trpc-go/trpc-go/tree/v0.18.6) (2024-10-22)

### Bug Fixes

- **restful:** provide UnquoteString to allow unquote (!2633) (v0.6.4)
  - 在 restful 中，当用户将响应结构体的某个字符串字段映射到 `response_body` 上时，字符串会带双引号，而非原始格式，!2633 提供了 UnquoteString 字段来解除双引号
- **pool/multiplexed:** fix unnecessary reconnections for multiplexed (!2640) (v0.12.0)
  - 识别多路复用重连时的错误，如果是 io.EOF 则不再重连，从而避免一直重连
    - 参考 #990, #991
- **multiplexed:** lower log level of invalid streamID (!2620) (v0.17.0)
  - v0.17.0 在客户端多路复用收到无法识别的 streamID 时会打印一条错误日志，但是这里在大部分情况下是正常的，不需要有错误日志惊扰，!2620 降低这条日志的级别到 trace
    - 参考 #1013

## [v0.18.5](https://git.woa.com/trpc-go/trpc-go/tree/v0.18.5) (2024-09-18)

### Bug Fixes

- http: fix handleSSE might return token too long error (!2537) (v0.18.0)
  - HTTP SSE 在处理时使用 trpc.DefaultMaxFrameSize（对应 10MB）而非 codec.DefaultReaderSize（4KB）以避免 "token too long" 错误
- http: wrap error into return message (!2459) (v0.18.0)
  - 将 HTTP SSEHandler 的错误做正确的 wrap 返回，在之前的版本这个错误被遗漏掉了
- {http, examples, docs}: improve docs and add examples for better sse (!2471) (v0.18.0)
  - 添加了 WriteSSE 能力，!2537 依赖了 !2471，因此也 pick 出来
- {codec, server}: assert Sizer interface for attachment to avoid extra copy (!2478) (v0.15.0)
  - 避免 attachment 频繁拷贝问题
    - 参考 #983
- admin: reset default serve mux to remove pprof registration (!2489) (v0.5.2)
  - 新建并覆盖 http.DefaultServerMux 以成功移除 pprof 注册
    - 参考 #912
- {errs, stream}: wrapped as trpc err when new stream (!2538) (v0.4.0)
  - 将新建流式返回的错误包装为框架错误以方便处理
    - 参考 #999
- http: fix gzip error for http request (!2548) (v0.1.0)
  - 当 HTTP 客户端收到的回包中没有 Content-Encoding 信息时，不要使用任何解压缩方式（之前默认是使用客户端发包的压缩方式，假如客户端发包采用 gzip 压缩，但是回包本身没有压缩并且不带 Content-Encoding 信息时，客户端就会解包失败）
- restful: support disable request timeout (!2569) (v0.6.4)
  - 支持 restful 禁用全链路超时，在之前的实现中，restful 模式下全链路超时始终会生效
- {client, docs}: provide yaml config for multiplexed and improve the docs (!2520) (v0.12.0)
- pool: enable reconnectResetInterval config for multiplexed (!2571) (v0.12.0)
  - 为多路复用提供 `initial_backoff`,`max_reconnect_count`,`reconnect_count_reset_interval` 等配置支持用户自定义重连的策略以避免一些情况下的永久重连
    - 参考 #990, #991
- transport/tnet: do not use tnet's idle timer (!2319) (v0.11.0)
  - trpc-go 框架对于连接池本身有健康检查机制以实现空闲连接超时能力，但是 tnet 自己也存在空闲超时能力，并完全独立于框架的逻辑，会导致 tnet 在触发空闲连接关闭时，trpc-go 框架的连接池仍然认为该连接是健康的，拿出来后用户做读写会发现 connection is closed 的错误，修复后，tnet 连接池将不再使用 tnet 本身的空闲超时能力，而是只依赖通用的连接池健康检查机制对应的空闲超时能力
    - 触发条件：在客户端使用了 trpc-go 框架的 tnet 连接池
    - 参考：<https://mk.woa.com/q/294084>
- trpc: remove periodicallyUpdateGOMAXPROCS func (!2310) (v0.3.2)
- {trpc,docs}: add option to round up cpu quota (!2605) (v0.3.2)
  - 支持通过配置 `round_up_cpu_quota: true` 来对非整数核做向上取整以设置 maxprocs，避免向下取整导致垂直扩容无法触发
    - 触发条件：使用容器环境，并且容器分配的核数为非整数核，并期望能够触发垂直扩容能力
    - 参考：#995

## [v0.18.4](https://git.woa.com/trpc-go/trpc-go/tree/v0.18.4) (2024-08-08)

### Bug Fixes

- pool/multiplexed: support multiplexing reconnect configuration to disable reconnect (!2405)
- client: make configuration struct members exportable (!2421)
- stream: only wraps io.EOF when stream is already closed (!2449)
- errs: check nil error to avoid panic (!2484)
- client: fix RequestTime with value 0 (!2467)
- {http, restful}: add trpc-caller-method and remote addr to msg to help report metric in plugin (!2487)
- transport: add port reuse for server-side TCP transport (!2469)
- http: fix TestHTTPGotConnectionRemoteAddr (!2430)

## [v0.18.3](https://git.woa.com/trpc-go/trpc-go/tree/v0.18.3) (2024-07-18)

### Bug Fixes

- transport: use shorter update interval for setting of read timeout (!2454)
- trpc.go: add default values for CloseWaitTime and MaxCloseWaitTime (!2453)

## [v0.18.2](https://git.woa.com/trpc-go/trpc-go/tree/v0.18.2) (2024-07-11)

### Bug Fixes

- pool/connpool: Decrement ConnectionPool.used when Conn acquisition fails (!2415)
- log: fix log level by adding CoreLevelNewer (!2351)

## [v0.18.1](https://git.woa.com/trpc-go/trpc-go/tree/v0.18.1) (2024-05-13)

### Bug Fixes

- plugin: avoid using reference to loop iterator variable (!2321)

**Attention:** !2321 fixes a critical bug introduced in !2231.

> The reconstruction of the YAML nodes used for loop variables, resulting in
> plugins of the same type all sharing the configuration corresponding to the
> last name. This caused the issue of the default log output file being
> incorrect, as reported in <https://mk.woa.com/q/294169>, and also fostered
> #937.

## [v0.18.0](https://git.woa.com/trpc-go/trpc-go/tree/v0.18.0) (2024-05-07)

<span style="color: red;">v0.18.0 有严重 bug，已废弃，请使用当前 latest 版本</span>

⚠️⚠️⚠️ trpc-go v0.18.0 bug 版本禁用公告 ⚠️⚠️⚠️

警告⚠️！trpc-go v0.18.0 有严重 bug（会导致配置中的同类型插件只有第一个配置会生效！），不要使用！

请使用最新版本（latest，当前是 v0.18.2），该 bug 的修复见 changelog：

<https://git.woa.com/trpc-go/trpc-go/blob/master/CHANGELOG.md#bug-fixes>

以及 MR <https://git.woa.com/trpc-go/trpc-go/merge_requests/2321>

### Documentations

- **docs:** add explanation for overload control metrics (!2259)
- **docs:** add notes on cpu_threshold for overloadctrl (!2291)
- **docs:** add notes on dry_run flag (!2290)
- **docs:** add server timeout performance issue for restful (!2297)
- **docs:** add showcase (!2257)
- **docs:** add trpc create note on restful project (!2284)
- **docs:** change absolute path to relative path for oc doc (!2272)
- **docs:** fix uncorrected description caller Service => caller Server (!2270)
- **docs:** revise knocknock document (!2246)
- **docs:** update code example for client-only loading configuration (!2262)
- **docs:** update overload control doc on implementation (!2285)
- **log:** remove "modifying the log level of the sub logger" from readme (!2301)
- **restful:** add notes on restful server transport register (!2277)
- **docs:** specify that version is trpc framework version (!2304)

### Bug Fixes

- **codec:** change priority prefix to trpc (!2300)
- **log:** revert multi level logger implementation for log.With (revert !2017,!2204) (!2276)
- **restful:** check result of response type assertion to avoid panic (!2286)
- **http:** revert !2263 (!2269)

### Features

- **client:** add cannot be reused comment for WithSelectorNode option (!2268)
- **http:** support sse through ClientRspHeader.SSEHandler (!2217)
- **server:** add read timeout option explicitly (!2292)
- **server:** skip calls of address string when oc is noop (!2253)
- **{trpc,plugin}:** support starting server without configuration (!2231)
- **codec:** provide priority based overload control api (!2243)
- **http:** pass options to restful server transport (!2274)
- **restful:** provide stdjson for better performance (!2296)
- **http:** provide fastop for canonical header operations (!2249)
- **config:** use type definitions instead of anonymous structs (!2305)

### Enhancements

- **docs:** update overload control user case for testing (!2248)
- **http:** allocate new slice for appending options (!2275)
- **http:** body can still be nil for GET method (!2267)
- **http:** only decorate with cancel when manual read body is true (!2266)
- **http:** replace the old http.Request to ensure context modified by filters is embedded (!2263)
- **metrics:** update description about Counter (!2293)
- **naming/selector:** defaults to vanilla ip selector to reduce overhead (!2265)
- **pool:** improve stability of reconnect tests (!2288)
- **restful:** change header set operation to fastop (!2271)
- **restful:** enable timeout for fasthttp based router (!2295)
- **restful:** reduce redundant strings split (!2299)
- **restful:** try using accept as response serializer first (!2298)
- **server:** sleep extra time for server test (!2289)
- **test:** increase delta of time interval in tests (!2287)
- **{trpc, examples, test}:** bump golang.org/x/net from 0.17.0 to 0.23.0 (!2279)
- **{client,http,pool}:** add inet parse address to avoid performance overhead (!2264)

## [v0.17.3](https://git.woa.com/trpc-go/trpc-go/tree/v0.17.3) (2024-05-21)

### Note

All versions between v0.17.0 and v0.17.2 have the following bug:

The trpc-go server implementation of these versions will return error code
171 for trpc-go client, and 141 for trpc-cpp client occasionally due to the
changes introduced by !2139.

This issue has been resolved in merge request !2292 and the fix is available
in versions >=v0.17.3.

### Bug Fixes

- **server:** add read timeout option explicitly to avoid 171/141 error codes (!2292)
- **transport/tnet:** do not use tnet's idle timer (!2319)

## [v0.17.2](https://git.woa.com/trpc-go/trpc-go/tree/v0.17.2) (2024-04-08)

### Bug Fixes

- **stream:** fix stream server panic caused by mismatch rpc name (!2256)

### Enhancements

- **graceful:** remove build tag unix to provide compatibility (!2255)
- **client:** ensure remote addr msg right after get (!2254)

## [v0.17.1](https://git.woa.com/trpc-go/trpc-go/tree/v0.17.1) (2024-04-08)

### Documentations

- **doc:** update overload control notes (!2241)

### Enhancements

- **config:** fix TestWatchWithError (!2244)
- **example:** complete the stub code for httprpc case (!2235)
- **go.mod:** back to go1.18 (!2245) (v0.17.0)
- **pool/multiplexed:** fix unstable test cases (!2242)
- **server:** invite back try close in non-graceful restart mode (!2247) (v0.17.0)

## [v0.17.0](https://git.woa.com/trpc-go/trpc-go/tree/v0.17.0) (2024-03-28)

### Breaking Changes

- **trpc:** update min go version to v1.20 (!2139)  
  Context.Cause is used by graceful restart and was introduced in Go 1.20. Go lower than 1.20 would result in compilation error.

### Bug Fixes

- **naming:** fix some bugs in circuit breaker (!2164) (v0.17.0-dev)
- **gomod:** fix CVE-2024-24786 (!2225) (v0.1.0)

### Features

- **client:** add GetConfig API to obtain client.Config (!2232)
- **config:** add callback function with error returned for data provider (!2224)
- **errs:** provide NewCalleeFrameError function to create callee frame errors (!2222)
- **log:** support customizing the time format in the log file name (!2181)
- **naming:** circuit breaker in direct ip selector supports all die all alive (!2165)
- **naming:** support circuit breaker for direct IP selector (!2111)
- **restful:** support additional customized route rules for fasthttp (!2128)

### Enhancements

- **admin:** add warn log when unregister pprof handlers failed (!2229)
- **admin:** handle not allowed http method (!2206)
- **all:** allow asynchronous mode when stream and unary coexist (!2190)
- **all:** pre-allocating slice memory whenever possible (!2212)
- **codec:** provide build tag optimization for performance (!2159)
- **errs:** deprecate Cause method (!2167)
- **examples/httprpc:** move "HTTP RPC Service" example from http to httprpc (!2205)
- **http:** bypass naming process for std http client (!2207)
- **http:** expose current type for wrong client header (!2211)
- **http:** support connContext in http transport (!2230)
- **log:** use core rather than level for enable check (!2204)
- **multiplexed:** log error when invalid streamID is received (!2215)
- **multiplexed:** ensure that the remote address of the virtual connection is not empty (!2180)
- **rpcz:** provide rpczenable flag to avoid unnecessary overhead (!2136)
- **server:** implement perfect graceful stop (!2135)
- **server:** perfect server graceful restart (!2139)
- **stream:** avoid using plain addr in tests (!2189)
- **stream:** postpone nil frame head setting for error frame (!2216)
- **test:** fix fragile TestGo (!2200)
- **test:** fix TestHTTPGotConnectionRemoteAddr report 'more than once' error (!2199)

### Refactors

- **restful:** url.Values => map[string][]string for form (!2197)
- **server:** refactor Register function (!2191)

### Documentations

- **all:** sync docs from git to iwiki (!2158)
- **{config, log, plugin}:** fix hyperlink in readme (!2178)
- **codec:** fix some typo and grammar error (!2233)
- **docs:** add upgrade guide document (!2214)
- **docs:** add unit testing, and integration testing (!2183)
- **docs:** add versatile pure client example (!2184)
- **docs:** refine polaris setup for pure client (!2185)
- **docs:** add an introduction of stream filter in stream documents (!2192)
- **docs:** add authentication_authorization.zh_CN.md (!2186)
- **docs:** add code_interoperability.zh_CN.md (!2221)
- **docs:** add contacts (!2227)
- **docs:** add data_validation.zh_CN.md (!2213)
- **docs:** add description for the service name and services (!2202)
- **docs:** add environment_setup.md (!2203)
- **docs:** add FAQ about "StreamTransport is not implemented" error (!2198)
- **docs:** add faq for error code 4 and 5 in knocknock (!2210)
- **docs:** add FAQs (!2187)
- **docs:** add notes for unix domain socket (!2194)
- **docs:** add performance data (!2228)
- **docs:** add server side file upload for http (!2201)
- **docs:** add specific version information for filter and codec plugin (!2218)
- **docs:** fix iwikiPageID (!2170)
- **docs:** fix png (!2173)
- **docs:** fix relative file path (!2172)
- **docs:** quickly creates a tRPC environment with one click on cloud (!2226)
- **docs:** revise documentation of overload control (!2193)
- **docs:** update http rpc doc link for error code mapping (!2174)
- **docs/knocknock:** add description that the server has multiple service names (!2219)
- **filter:** update how to develop stream filter (!2196)
- **http:** add new NewRESTServerTransport based on fast http or standard http (!2122)
- **http:** change stream read version requirement to v0.15.0 (!2208)
- **http:** enhance distinction between two options for urlencoded (!2175)
- **http:** add notes on client sending arbitrary content type (!2182)
- **http:** add notes on host setting (!2179)
- **http:** add notes on noop serialization for form encoding (!2176)
- **{http,restful}:** document separate service approach for coexistence of http and restful (!2143)
- **log:** add the FAQ section in the readme file (!2177)
- **pool/connpool:** add doc on put connections back into the pool (!2188)
- **stream:** add doc for the concurrent-unsafe stream method (!2209)

## [0.16.2](https://git.woa.com/trpc-go/trpc-go/tree/v0.16.2) (2024-01-24)

### Bug Fixes

- **stream:** fix goroutine leak for unawakened reading when close multiplexed virtual connection (!2157) (v0.16.0)
- **stream:** fix 'uninitialized meta' error caused by receiving feedback frames during the server-side stream closure phase (!2138) (v0.5.2)
- **stream:** fix client blocking when using tnet (!2160) (v0.16.0)

### Enhancements

- **test:** add TestCalleeMethod for using trpc.alias in protobuf (!2156)
- **test:** add tests for tnet stream (!2161)
- **{codec,test}:** add more detail error message for errFrameTooLarge (!2162)

### Documentations

- **{http,restful}:** document separate service approach for coexistence of http and restful (!2143)
- **readme:** fix API Docs badge (!2163)

## [0.16.1](https://git.woa.com/trpc-go/trpc-go/tree/v0.16.1) (2024-01-09)

### Bug Fixes

- **client:** fix client wildcard match for config (!2140) (v0.1.0)
- **codec:** revert !2059 "optimize performance of extracting method name out of rpc name" (!2150) (v0.16.0)
- **http:** fix form serialization panicking for compatibility (!2144) (v0.16.0)

### Enhancements

- **{log, plugin}:** add newline character to fmt.Printf message (!2133)

### Documentations

- **all:** quote URLs in curl commands to avoid "zsh: no matches found" error (!2129)
- **all:** add may not panic comment for MustRegisterXXX due to the unpredictable execution order of init functions (!2132)

## [0.16.0](https://git.woa.com/trpc-go/trpc-go/tree/v0.16.0) (2023-12-21)

### Breaking Changes

- **trpc:** update min go version to v1.18 (!2094)  
  Some tRPC-Go packages is refactored by generics. Go lower than 1.18 would result in build error.
- **http:** get serializer should also be able to unmarshal nested structure (!2044) (v0.3.1)
  It changes how HTTP GET query parameters are processed (Refer to #921 for details):
  - < v0.16.0: the query parameters are case-insensitive
  - >= v0.16.0: the query parameters are case-sensitive

### Features

- **server:** provide configurations for serialization and compression (!1995)
- **all:** Add a Must method to the Register function (!2016)
- **tnet/multiplex:** support tls (!2000)
- **rpcz:** provide AND, OR, NOT logic operation in record when config (!1967)
- **plugin:** provide setup hook (!2021,!2057,!2077)
- **log:** add `multiLevelCore` to make `With` return a new child logger whose level can be altered by `SetLevel` locally (!2017,!2060)
- **server:** add `MustService` and `NoopService` to avoid annoying error check (!2051)
- **log:** add `RegisterCoreNewer` `GetCoreNewer` and deprecate `RegisterWriter` `GetWriter` (!2062)
- **selector:** allow net.Addr parser for selector to avoid unnecessary dns lookup in trpc-database (!2023)
- **codec:** support lz4 compression (!2082)
- **trpc:** support periodically update GOMAXPROCS (!2085)
- **http:** provide `CacheRequestBody` flag to disable caching of the request body (!2087)
- **log:** pick zap field to enable customized zap field (!2120)
- **http:** introduce DecorateRequest to enable modification of http.Request (!2123)

### Bug Fixes

- **log:** log.Info("a","b") print "a b" instead of "ab" (!1969) (v0.1.0)
- **stream:** return an error when receiving an unexpected frame type (!2022) (v0.5.2)
- **stream:** ensure server returns an error when connection is closed (!2046) (v0.4.0)
- **stream:** fix connection overwriting when a client uses the same port to connect. (!2073,!2103) (v0.4.0)
- **stream:** fix client's compression type setting not working (!2078) (v0.4.0)
- **stream:** notify server when client context canceled (!2097) (v0.4.0)
- **client:** remove the write operation on *registry.Node in LoadNodeConfig to avoid data race during selecting Node (!2055) (v0.6.0)
- **config:** re-enable Config.Global.LocalIP to perfect !1936 (!2024) (v0.15.0)
- **http:** get serializer should also be able to unmarshal nested structure (!2044) (v0.3.1)
- **client:** fix possible nil method timeout (!2070) (v0.15.0)
- **http:** check type of url.Values for form serialization (!2084) (v0.3.1)
- **http:** expose possible io.Writer interface for http response body (!2089) (v0.15.0)
- **{restful}:** continue to handle even if transcodeRequest failed (!2113) (v0.7.0)

### Enhancements

- **test:** fix data race in e2e test cases of close wait time. (!2009)
- **admin:** log when admin server starts (!2014)
- **test:** fix broken test in go1.21.0 (!2010)
- **config:** add API promise comment for *TrpcConfig.Unmarshal (!2007)
- **test:** add restful deep wildcard(`**`) match test case. (!2005)
- **codec:** unwrap err from server rsp error (!1999)
- **{test,example}:** update tnet to v0.0.15 (!2019, !2029)
- **http:** add client-side proxy example (!2031)
- **config:** provide a full trpc_go.yaml example (!2027)
- **restful:** fix typo (!2025)
- **config:** let kvConfigs and codecs use different RWMutex (!2040)
- **config:** fix lint warning "G601: Implicit memory aliasing in for loop." (!2036)
- **test:** add full test message for err (!2050)
- **test:** relax error checking of plugin setup timeout (!2049)
- **client:** add more comments for WithCurrentSerializationType and WithCurrentCompressType (!2069)
- **test-http:** add HandleErrServerNoResponse test case (!2075)
- **tnet:** accent on internal error for multiplex (!2048)
- **plugin:** log normal plugin setup time (!2056)
- **attachment:** avoid memory allocation while getting or setting empty attachment (!2058)
- **http:** add comments for map allocation (!2080)
- **http:** add POSTOnly option to restrict method used in HTTP RPC (!2067)
- **codec:** optimize performance of extracting method name out of rpc name (!2059)
- **config:** update fsnotify from v1.4.9 to v1.7.0 (!2083)
- **config:** explain more about MaxRoutines option (!2081)
- **test/transport:** add listener closed test case (!2076)
- **codec:** explicitly check noop compression (!2066)
- **test:** fix unstable e2e tests (!2088)
- **changelog:** update and reformat Bug Fixes from v0.10.0 to v0.15.1 (!2091,!2104)
- **errs:** fix ErrorTypeCalleeFramework comment (!2105)
- **http:** provide nop closer buffer pool for request body (!2086)
- **go.mod:** update golang.org/x/net from v0.5.0 to v0.19.0 to fix vulnerability scanned by osv-scanner tool (!2108)
- **http:** create the header just in time to prevent any potential trampling (!2106)
- **config:** lower the log level from debug to trace (!2095)
- **examples/http:** add more http examples (!2096)
- **codec:** include the type of the value that failed to jce serialize in error message (!2112)
- **{rpcz,server}:** add docs about how to inject root span for custom transport (!2110)
- **{config,client,log}:** add `omitempty` tag for yaml configuration (!2092)
- **http:** support explicit https protocol (!2107)

### Refactors

- **restful:** deduplicate get listener (!2026)
- **{errs,log}:** refactor the code to avoid using `fallthrough` in switch clause (!2020)
- **log:** refactor some logic about WithFields and With to improve readability (!2018)
- **http:** replace raw strings with pre-defined constants. (!2042)
- **codec:** eliminate map access for compression and serialization (!2068)
- **metrics:** avoid allocating metrics if sinks have a size of zero (!2065)
- **internal/codec:** add inline directive (!2061)
- **server:** remove handlerSet field from Options (!2109)
- **restful:** refactor transcode into transcodeRequest, handle, and transcodeResponse (!2117)
- **all:** use generics to refactor internal Ring, Stack and Queue (!2116)

### Documentations

- **log:** merge and update readme (!1196,!2035)
- **rpcz:** move readme from trpc-wiki to trpc-go (!2015)
- **restful:** move readme from trpc-wiki to trpc-go (!1993)
- **metrics:** rewrite readme (!2028)
- **config:** update readme (!2043)
- **http:** emphasize the significance of ca_cert in HTTPS (!2039)
- **plugin:** update readme (!2038)
- **http:** add possible causes of empty rsp to faq (!2054)
- **http:** refine formdata send and read example (!2052)
- **http:** no fullstop in heading (!2053)
- **http:** add doc for https dns target (!2072)
- **http:** provide examples to report req rsp using filters (!2030)
- **http:** add timeout handler example (!2102)
- **http:** add example for sse content type (!2101,!2114)

## [0.15.1](https://git.woa.com/trpc-go/trpc-go/tree/v0.15.1) (2023-08-17)

### Bug Fixes

- **server:** do not close old listener immediately after hot restart (!1998) (v0.11.0)
- **config:** promise that dst of codec.Unmarshal is always map[string]interface{} (!1989) (v0.15.0)
- **restful:** fix that deep wildcard matches in reverse order (!2003) (v0.6.4)
- **transport:** ensure that the timeout for UDP dialing takes effect (!1988) (v0.1.0)

### Enhancements

- **transport/test:** remove the unix socket files after test (!1997)

## [0.15.0](https://git.woa.com/trpc-go/trpc-go/tree/v0.15.0) (2023-08-04)

### Breaking Changes

- `RoundTripOptions.Multiplexed` changed from struct `*multiplexed.Multiplexed` to interface `multiplexed.Pool` (!1624)
  The following codes may not work anymore:

  ```go
  var m *multiplexed.Multiplexed
  var o RoundTripOptions
  m = o.Multiplexed // This will report a type error. You can not assign interface to a concrete struct.
  ```

### Features

- **{client, server}:** support method timeout (!1897)
- **{client, stream, transport, tnet}:** support tnet client stream transport (!1957)
- **server:** provide on response obsoleted option (!1976)
- **tnet:** support multiplexed (!1707)
- **http:** support to customize std http client transport (!1965)
- **http:** attach body to error message for status code >= 300 (!1864)
- **tls:** support client certificate without server verification (!1959, !1968)
- **transport:** support with udp listener options (!1952)
- **log:** enable colorful output (!1866)
- **log:** support removing certain field through config (!1929)
- **config:** expands env like trpc_go.yaml (!1921)
- **config:** provide configuration for max frame size (!1918)
- **config:** provide configuration for plugin setup timeout (!1945)
- **config:** add watchHook option to get notice when provider triggers watch events (!1904)
- **config:** provide enable multiplexed configuration (!1950)

### Bug Fixes

- **attachment:** fix possible uint32 overflows (!1854) (v0.14.0)
- **attachment:** copy attachment size if user provides their own rsp head (!1887) (v0.14.0)
- **stream:** fix the memory leak issue that occurs when stream.NewStream fails (!1899, !1930) (v0.5.2)
- **errs:** Msg should unwrap the inner trpc error (!1892) (v0.1.0)
- **http:** use GotConn for obtaining remote addr in connection reuse case (!1901) (v0.6.0)
- **http:** http trace should not replace req ctx with transport ctx (!1955) (v0.6.0)
- **http:** do not ignore server no response error (!1948) (v0.3.1)
- **restful:** fix timeout does not take effect which is introduced in !1461 (!1896) (v0.9.5)
- **log:** skip buffer and write directly when data packet exceeds expected size (!1923) (v0.4.0)
- **config:** set empty ip to default 0.0.0.0 to avoid graceful restart error (!1936) (v0.1.0)
- **config:** fix watch callback leak when call TrpcConfigLoader.Load multiple times (!1904) (v0.1.0)
- **server** fix unaligned 64-bit atomic words at linux-386 (!1938) (v0.10.0)
- **server:** don't wait entire service timeout, but close quickly on no active request (!1970) (v0.10.0)

### Deprecates

- **config:** config.Loader interface is deprecated (!1869)

### Enhancements

- **errs:** print error even if Msg is empty (!1830)
- **log:** add flag to handle rollwriter close correctly (!1835)
- **http:** return RetClientConnectFail when init tls failed (!1849)
- **http:** add example of sending and receiving different content type (!1878)
- **http:** add client sending form data example (!1860)
- **{http, trpc}:** time.Duration/time.Millisecond => time.Duration.Milliseconds() (!1920)
- **transport:** return an error when set deadline failed (!1793)
- **transport:** log non context done, non temporary network error of Accept (!1855, !1882)
- **transport:** elevate the priority of the protocol over the transport priority (!1951)
- **transport:** remove unused constants (!1949)
- **transport/tnet:** lower the log level when switch to gonet default transport (!1960)
- **config:** wrap original error (!1874)
- **config:** use path, provider name, decoder name, expanEnv and watched to identify a TrpcConfig instead of a single path (!1904)
- **admin:** remove jsoniter dependency and replace global variable (!1837)
- **admin:** cleanup test listener and add err log (!1913)
- **admin:** ignore normal listener close error (!1935)
- **{admin, codec}:** minimize error scope (!1947)
- **client:** return swallowed RegisterConfig error from LoadClientConfig (!1942)
- **{client, log, config}:** add missing yaml tags (!1872)
- **{client, stream, transport, transport/tnet, admin, test}:** add rpcz span (!1900, !1964, !1966)
- **stream:** wrap more information into error to improve debuggability (!1962)
- **all:** replace obsoleted golang.org/x/net/context with std context (!1853)
- **restful:** avoid unsafe conversion from []byte to string (!1881)
- **trpc:** add warning message for NewAsyncGoer (!1883)
- **trpc:** fix TestGo testcase (!1958)
- **{trpc, rpcz}:** replace math/rand with internal/rand to prevent sharing globalRand of `math/rand` with other packages (!1954)
- **metrics:** guard `metricsSinks` with read lock to avoid data race (!1886)
- **server:** change service encode error level to trace (!1931)
- **{server, pool/connpool}:** replace syscall with golang.org/x/sys/unix partially (!1946)
- **multiplexed:** add multiplexed.Pool interface (!1624)
- **multiplexed:** enhance readability (!1953)
- **examples:**
  - add log (!1893)
  - add http (!1894)
  - add errs (!1895)
  - add rpcz (!1912)
  - add admin (!1910)
  - add config (!1884)
  - add plugin (!1815)
  - add filter (!1839)
  - add stream (!1842)
  - add timeout (!1814)
  - add restful (!1819)
  - add selector (!1857)
  - add metadata (!1821)
  - add discovery (!1907, !1911)
  - add attachment (!1863)
  - add healthcheck (!1873)
  - add compression (!1856)
  - add load balance (!1909)
  - add client cancel (!1852)

### Documentations

- **client:** explain callee and name in README (!1867)
- **http:** translate missing content to English (!1861)
- **http:** add doc for multipart/form-data (!1926)
- **http:** method must be specified when using custom client req head (!1944)
- **trpc-go:** add tencent opensource statement (!1898)
- **restful:** add docs for fasthttp (!1934)
- **restful:** more docs about how to extract http head from context when enabling fasthttp (!1972)
- **admin:** update README (!1932)
- **admin:** update pprof/{profile,trace} readme of write_timeout (!1941)
- **{log, admin}:** add readme for enabling trace level (!1943)
- **rpcz:** use trpc-wiki as the only one link for trpc-go and wiki (!1956)

### Refactors

- **codec:** refactor tests (!1841)
- **config:** package config is refactored (!1904)
- **transport:** move `wrapNetError` to internal package (!1817)
- **naming:** refactor tests (!1876)
- **log:** abstract `filterByXxx` with `PartitionXxx` (!1925)
- **multiplexed:** refine `filterOutConnection` to follow single responsibility (!1924)

### Integration Tests

- **plugin:** add dependency tests (!1831)
- **plugin:** add tests for FinishNotifier (!1848)
- **http:** add patch tests (!1827)
- **{http, codec}:** fix e2e pipeline (!1902)
- **{codec, http, trpc}:** add some abnormal tests (!1859)
- **transport:** extract the common dial codes for gonet and tnet (!1793)
- **attachment:** add tests for very large attachment (!1868)
- **server:** add WritevOption test (!1906)
- **test:** remove unused gracefulrestart directory (!1937)
- **client:** fix case TestClientConfigLoadWrongServiceName (!1974)

## [0.14.0](https://git.woa.com/trpc-go/trpc-go/tree/v0.14.0) (2023-05-11)

### Features

- **{codec, trpc, test}:** added support for attachment feature (!1745)
- **log:** now uses logger inside msg on panic (!1813)
- **{rpcz, http}:** added support for rpcz on http-server (!1808)

### Bug Fixes

- **config:** fixed service.timeout setting to take effect (service.timeout > defaultIdleTimeout) (!1782) (v0.7.3)
- **http:** fixed post and patch typos (!1818) (v0.13.0)
- **log:** added a mapping from "trace" to zapcore.DebugLevel (!1786) (v0.1.0)
- **rpcz:** returns RPCZ itself from its NewChild method (!1811) (v0.11.0)
- **server:** lowered server encode log level to debug (!1809) (v0.13.0)

### Enhancements

- **test:** added proxy test (!1807)
- **config:** lowered log level of search to debug (!1810)

### Documentations

- **examples:** added feature requirements (!1812)
- **http:** provided client/server http chunked examples (!1783)

### Refactors

- **http:** replaced getTLSConfig with internal/tls.GetClientConfig (!1803)

## [0.13.1](https://git.woa.com/trpc-go/trpc-go/tree/v0.13.1) (2023-04-27)

### Features

- **naming:** provide service router option (!1785)

### Bug Fixes

- **config:** change type of unmarshalledData from `map[string]interface{}` to `interface{}` to fix type incompatible problem while unmarshalling introduced by !1732 (!1801) (v0.13.0)
- **rpcz:** use `comma, ok` assert interface type to avoid panic (!1802) (v0.11.0)

### Enhancements

- **log:** validate WriteMode config parameter (!1798)
- **test:** add e2e-testing for trpc-util (!1795)
- **pool/multiplexed:** fix multiplexed server test panic (!1791)
- **trpc:** fix unstable test due to inaccurate timer under windows (!1787)

### Deprecated

- **log:** deprecate CustomTimeFormat, DefaultTimeFormat (!1784)

### Documentation

- **http:** provide http sse example (!1800)
- **http:** add HTTPS, chunked, stream send/read examples to README (!1797)

### Refactors

- **lsc:** improve code comments and reduce duplication (!1790)

## [0.13.0](https://git.woa.com/trpc-go/trpc-go/tree/v0.13.0) (2023-04-18)

### Features

- **http:** support disabling keep-alives (!1746)
- **http:** replace the old http.Request to ensure the inner context is embedded. The user can then get the the request body in the ErrHandler (!1749)
- **http:** enable passed listener to use tls for restful transport (!1767)
- **http:** provide io.Reader as body to enable stream client send (!1762)
- **http:** provide ManualReadBody flag to enable stream client read (!1766)
- **http:** reset request body in server decoder to allow multiple reads (!1776)
- **log:** provide option logger (!1736)
- **metrics:** add GetMetricsSink function (!1737)
- **rpcz:** add SpanExporter interface to allow user export spans (!1756)
- **server:** log service handle error to ensure that the server side gets the error message (!1731)
- **test:** add and improve the integration test cases for `pool`, `stream` (!1738, !1747)

### Bug Fixes

- **client:** set "container name" and "set name" even though timeout is reached (!1688) (v0.8.2)
- **connpool:** set the default PoolIdleTimeout for the connection pool to ensure that connections will eventually be cleaned up by timeout (!1764) (v0.11.0)
- **http:** form serializer should be able to unmarshal nested structure (!1725) (v0.3.3)
- **metrics:** fix ConsoleSink print format (!1768) (v0.2.0)
- **multiplex:** fix concurrent map read and write when close UDP connections (!1739) (v0.9.5)
- **multiplex:** fix concurrent read/write on message's meta data (!1761) (v0.4.0)

### Enhancements

- **test:** fix `multiplex` unstable unit test `TestMultiplexedServerFail` (!1711)
- **typo:** fix go meta linter errors (!1741)

### Documentation

- **changelog:** add warning for v0.11.1 (!1744)
- **example:** update README (!1752)
- **http:** remove unimplemented function used in README (!1765)

### Refactors

- **admin:** remove global variable `admin.ro` and refactor test case (!1723)
- **all:** rename {reqbuf, rspbuf, reqbody, rspbody, reqbodybuf, rspbodybuf} (!1763)
- **client:** improve readability of package client and its test (!1773)
- **config:** refactor the code related cast.ToXXX operation (!1732)
- **example:** move package examples to a new module (!1778)
- **http:** improve readability of package http and its test (!1770, !1771)
- **metrics:** refactor to improve readability (!1772)
- **trpc:** refactor unary codec (!1748)

## [0.12.0](https://git.woa.com/trpc-go/trpc-go/tree/v0.12.0) (2023-03-13)

### Features

- **client:** set address info into node and message for stream (!1696)
- **codec:** add IsValidCompress(Serialization)Type function (!1680)
- **log/rollwriter:** use symlink for rollwriter on windows to ensure successful renaming (!1670)
- **test:** add restfulServerEnv to test fast-http and async options (!1668)

### Bug Fixes

- **connpool:** do not free extra token when pool connection has been closed (!1695) (v0.11.0)
- **go.mod:** compilation failure on 32 bit architecture of tnet (!1718) (v0.11.0)
- **http:** fix panic of nested map in http form (!1697) (v0.3.1)
- **multiplexed:** fix goroutine leak caused by destroy (!1687) (v0.5.0)
- **stream:** client should receive a non-io.EOF error when the server crashes (!1701) (v0.4.0)
- **stream:** fix client gets stuck while sending data (!1690) (v0.4.0)
- **transport:** save raw tcp listener to prevent failure of tls fd retrieval (!1703) (v0.3.0)
- **transport:** use syscall.Conn to retrieve fd to prevent indefinite hangs (!1671) (v0.4.0)

### Enhancements

- **all:** eliminate "deprecated" warning and gofmt/lint/vet/imports errors (!1681,!1667,!1669)
- **go.mod:** upgrade strftime(v1.0.3 => v1.0.6) (!1709)
- **go.mod:** retract v0.11.0 (!1708)
- **go.mod:** remove go.uber.org/atomic direct dependency (!1693)
- **go.mod:** update go directive from 1.13 to 1.17 (!1679)

### Refactors

- **err:** refactor err package to improve readability (!1685)

### Documentation

- **http:** update readme for usage of standard http and rpc http (!1700)
- **naming/selector:** improve comments (!1673,!1672)
- **rpcz/readme:** fix typo(?span_id => /spans/) (!1686)

### Uint Tests and Integration Tests

- **go.mod:** remove testing frameworks except testify and testing packages (!1689)
- **log/rollwriter:** fix occasional failure of roll_by_time test (!1691)
- **log/rollwriter:** remove benchmarks of third package (!1710)
- **http:** improve stability of test on value detached transport (!1666)
- **multiplexed:** fix unstable test case (!1714,!1678)
- **restful/dat:** fix "dependent test" problem (!1692)
- **restful/dat:** add some white-box test cases and refactor slightly (!1702)
- **test:** fix unstable test case (!1713,!1675,!1674)
- **test:** update dependencies introduced by mr-1697 accordingly (!1699)
- **test:** update tnet version to be consistent with the trpc-go repository (!1682)

## [0.11.1](https://git.woa.com/trpc-go/trpc-go/tree/v0.11.1) (2023-01-12)

本版本 bug 原因：[https://mk.woa.com/q/287484](https://mk.woa.com/q/287484) 及 [fix](!1695)

触发条件及现象：

- 使用了 `connpool.WithMaxActive` 设置了最大活跃连接数 (必要条件)
- 下游节点高负载，重连失败会导致阻塞

详细描述：

- 正常连接到下游节点 A 后，由于某些原因 (比如下游节点因为空闲超时主动关闭连接，或者下游节点故障等) 该连接上在读取数据时返回错误
- 下次再尝试重新连接这个下游节点 A, 此时连接失败返回错误 => 在 v0.11.1 版本下会造成永远阻塞

根本原因：

- 连接读取数据返回错误时会在多个地方调用 `put` 方法，该方法中会将连接进行以下操作：
  - 关闭连接
  - 标志 closed 为 true
  - 释放 token
- 其中释放 token 这一步操作的是一个固定大小的 channel, 每个连接严格对应一个 token, 因此每个连接只能释放一个 token, 多次释放会导致阻塞
- v0.11.1 在 `put` 方法开始时未检查 closed 标志，导致一个连接会重复释放多次 token
- 此处会造成已有连接出错时的阻塞，更严重的问题在于假如下游节点再也无法连接 (dial) 成功，那么任何获取该节点上连接的操作都会被阻塞住 (而不仅仅是已存在的连接)
  - 原因是连接池的 `get` 方法在 `dial` 出错时会手动释放 token, 而这个释放 token 的操作会因为之前其他连接的多次释放而阻塞住

对应到 [https://mk.woa.com/q/287484](https://mk.woa.com/q/287484) 即为：

- 用户对下游节点 A 的连接在读取数据时出错 => 多次释放 token => 再次连接下游 dial 失败 => 无法释放 token 阻塞
- 其中多次连接到下游节点 A 的失败会造成该节点的北极星熔断

### Bug Fixes

- **connpool:** do not free extra token when pool connection has been closed (!1695) (v0.11.0)
- **admin:** 修复优雅重启时出现 panic 问题 (!1643) (v0.11.0)
- **tnet:** 修复在 Windows 系统下编译失败问题 (!1644) (v0.11.0)

### Enhancements

- **server:** 调整服务启动出错时打印的日志级别为 `error` (!1646)

### Documentation

- **docs:** 修正文档中出现的 `code.oa` 的 URL，改为 `woa` (!1642)
- **typo:** 修正拼写问题 (!1652)

### Uint Tests and Integration Tests

- **test:** 修复不稳定集成测试和单元测试用例 (!1640, !1647, !1648, !1649)

## [0.11.0](https://git.woa.com/trpc-go/trpc-go/tree/v0.11.0) (2023-01-05)

### Features

- **admin:** 用户自定义的 `admin.HandleFunc` 执行出现 panic 时，打日志并上报监控 (!1583)
- **connpool:** 添加 `PushIdleConnToTail` 参数，支持自定义连接回收方式，可将连接回收到队列尾部（默认是回收到队列头部），它可以更好地保证各个连接上的负载是均衡的，但是牺牲了空间局部性 (!1582)
- **connpool:** 添加 `CustomReader` 参数，支持自定义连接的是否封装 Buffer (!1565)
- **filter:** 支持 `CopyTo` 接口方法，用户可以自定义 `copyRsp` 过程 (!1579)
- **http:** 支持关闭对透传数据的 `Base64` 编码行为 (!1611)
- **rpcz:** 增加 `rpcz` 功能，以帮助监控框架中 RPC 过程的运行状态 (!1576, !1618, !1621, !1622, !1625)
- **transport:** 支持 `tnet` 网络库 (!1596)
- **test:** 增加及完善 `server`，`client`，`transport`，`stream` 等组件的集成测试用例 (!1571, !1577, !1578, !1580, !1597, !1599, !1605, !1606, !1616, !1627, !1631, !1632)

### Bug Fixes

- **codec:** 修复 `CalleeMethod` 取值不兼容问题，在 `rpc name` 不为 `tRPC` 协议格式的情况下，先判断当前 `method name` 是否已经设置，如果没有设置，则将 `method name` 设置为完整的 `rpc name` (!1629) (v0.9.1)
- **config:** 修复解析参数问题，保证在还没解析进程参数时，才解析 `conf` 参数 (!1587) (v0.5.2)
- **connpool:** 修复协程泄漏，健康检查协程永远不会退出问题 (!1623) (~v0.6.5)
- **connpool:** 修复获取连接时偶现 `connection pool limit` 问题 (!1626) (v0.1.0)
- **http:** 修复传输大文件场景下，生成 `multipart` 临时文件没被删除问题 (!1603) (v0.2.8)
- **http:** 修复 `env transinfo` 没有清理问题，导致 http 请求的 `disable_servicerouter` 无效 (!1628) (v0.2.8)
- **multiplexed:** 修复多路复用模式下 `slime` 不生效问题，在 v0.8.2 引入的 bug (!1630) (v0.8.2)
- **multiplexed:** 修复无限重建连接问题，在建立连接成功但读包失败场景引发的无限重连 (!1633) (v0.7.3)
- **restful:** 修复 panic 问题，在同时使用新生成的 `xxx.trpc.go` 和 `trpc-filter/cors` 时会发生 panic (!1607) (v0.6.6)
- **server:** 修复热重启时旧的 `listener` 没被关闭问题，导致旧进程会接收到新的请求 (!1609) (v0.4.0)

### Regression

- **admin:** 当设置了 `skipServe=true` 时，不初始化 `Router`，当用户没有启用 `admin` 时， `pprof` 保持关闭 (!1573)

### Refactors

- **stream:** 重构相关代码，提高代码可读性 (!1610)
- **transport:** 重构相关代码，提高代码可读性 (!1548, !1598, !1613)

### Uint Tests and Integration Tests

- **test:** 补充以及修复部分单元测试 (!1587, !1588, !1589, !1601, !1604, !1608, !1620)

## [0.10.0](https://git.woa.com/trpc-go/trpc-go/tree/v0.10.0) (2022-11-03)

### Features

- **admin:** 提供 `Watch` 功能，当服务的健康状态发生变化时，会进行通知；为保证 `admin` 模块一直可用，当没有配置 `admin` 端口的时候，也会实例化 `admin`，但是不会启动 `adminService` 进行端口监听；为 `admin` 模块添加一个是否启动 `service` 的开关 (!1531)
- **healthcheck:** 在服务注册完成时，触发健康检查的 `OnStatusChanged(Unknown)` 回调函数 (!1559)
- **admin:** `listener` 默认启用端口复用的功能 (!1543)
- **admin:** 支持优雅关闭，保证 `listenfd` 在热重启时进行传递，防止新进程启动失败 (!1556)
- **http:** 客户端支持透传环境信息 (!1524)
- **http:** 增加 `server` 启动异常能够打印错误日志的功能，当在传入错误的 TLS 配置文件时，服务启动不会返回错误，也不会打印错误日志，导致外部无法观测到服务的异常行为 (!1554)
- **server:** 热重启子进程就绪后，通知父进程就绪；父进程支持处理完请求后再退出 (!1523)
- **plugin:** 插件支持在服务关闭时，执行自定义关闭操作 (!1570)
- **client:** 使用 `callee` 和 `service name` 一起作为 `key` 来索引配置 (!1535)
- **test:** 增加及完善 `http`，`trpc`，`restful` 等组件的集成测试用例 (!1517, !1522, !1525, !1528, !1539, !1539,!1540, !1552, !1558, !1561, !1562, !1564)
- **test:** 开启 `CI`，新增代码合入时做代码扫描 (!1541)

### Bug Fixes

- **stream:** 修复服务端会向不支持流控的客户端发送流控帧，导致连接断开问题  码客：<https://mk.woa.com/q/285369?ADTAG=rb_c_a> (!1537) (v0.5.2)
- **restful:** 修复 `WithRESTOptions` 会覆盖原有值的问题；返回的错误应该由用户自己设置的 `ErrorHandler` 处理 (!1563) (v0.9.0)
- **restful:** 修复注册多个 `service`，只会保存最后注册的路由问题  码客：<https://mk.woa.com/q/285716> (!1551) (v0.6.6)
- **restful:** 修复设置 `FieldMask` panic 问题 (!1566) (v0.7.0)
- **filter:** 修复 `LoadClientFilterConfig` 方法，需要对 `selector` 过滤器作特殊判断 (!1547) (v0.8.3)

### Enhancements

- **filter:** 使用 `jsonpb` 替换标准库 `json` 进行序列化，修复 `string(json) -> int64(go struct)` 的解析失败的问题 (!1544)
- **transport:** 重构断言 `Listener` 逻辑，去除重复断言 (!1533)
- **test:** 补充以及修复部分单元测试 (!1532, !1560, !1567)
- **typo:** 修复拼写问题 (!1557, !1572d)

### Regression

- **errs:** `Error.Error()` 在缺失 `msg` 的时候应该返回空字符串 (!1521)
- **log:** 重新启用被废弃的 `log.WithContextFields`(!1514)

## [0.9.5](https://git.woa.com/trpc-go/trpc-go/tree/v0.9.5) (2022-09-08)

### Features

- 支持全链路超时
- 支持健康检查机制
- 支持结构化 Error 和 Error chain，错误记录更详细信息
- 支持 Snappy  Block 模式压缩
- 增加对 Unix Domain Socket 的支持
- RESTful 兼容指定 Content-Type 的编码类型为 UTF-8
- RESTful 支持多环境路由
- RESTful 兼容 HTTP Header 映射 Metadata
- HTTP Server 的 Request 存入 context 用于后续的 ErrHandler
- HTTP Transport 支持传入 Listener
- HTTP 没有必要校验 URL，去除校验，提高性能
- HTTP 错误码与 TRPC 状态码映射补充
- 更新 jsonpb 版本，支持 EscapeHTML
- Selector 获取的实例标签信息，需要反馈给上下文，指标上报的时候需要获取标签信息
- 提供配置选项，由用户控制上报哪些错误到 Selector
- Client 支持配置用于名字服务的 callee_metadata
- 增加集成测试的框架代码，增加 Config 和 Naming 的集成测试用例
- 完善注释，翻译剩余的中文注释

### Bug Fixes

- 修复 log AsyncRollWriter 单测偶发失败
- 修复 HeaderLen 作为 slice index 时，转换为 uint16 后再与其他值相加导致溢出问题
- 当服务端编码出现包体过大错误时，只返回 rspHeader 不返回 rspBody，保证错误信息的能被客户端接收
- 修复 Target、ServiceName 生效顺序不符合预期问题
- 修复多路复用的连接在重连后，返回的错误不符合预期问题
- 修复多路复用在建立连接失败的时候偶发出现 Read 卡住问题
- 修复 CopyCommonMessage 时 ServerMetaData 未拷贝问题
- 修复使用 WithTarget 时 callee_metadata 无效问题
- 修复拼写问题和代码规范问题
- 修复设置 DefaultMaxFrameSize 后报错信息不准确的问题
- 修复 UDP 读帧失败后没有返回错误的问题
- 解决负载均衡的哈希冲突问题

## [0.9.4](https://git.woa.com/trpc-go/trpc-go/tree/v0.9.4) (2022-06-16)

### Features

- 在 snappy compressor 中使用对象池以提升性能
- 将 multiplex 池的缓存改成队列，容量可自动分配
- 使 WithServiceName 兼容后端 calleeName 为通配符 * 的情况
- trpc http service 支持 h2c
- 使 restful jsonpb serializer 支持对 nil 选项的反序列化

### Bug Fixes

- 修复 AsyncRollWriter 的 Sync 方法为可重入的
- 修复 AsyncRollWriter 在 Close 时未释放 ticker 资源的问题
- 修复 roll_writer_test 测试用例，日志关闭时增加延迟，解决单测偶现失败问题
- 修复 restful 测试用例中的 reuseport 设置，解决单测偶现失败问题
- 修复 client transport 测试用例，解决 context 未 timeout 导致单测失败的问题
- 修复 multiplex 偶发 panic 的问题
- 修复客户端流式 RecvMsg 应等待服务端处理函数退出后再返回
- 修复检查进程状态返回值顺序错误的问题
- 修复连接池并发 Dial 会发生 Data Race，导致 DialTimeout 出错的问题
- 修复 UDP Dial 时出现错误未将错误值返回的问题
- 解决 trpc.Go 可测试性问题

## [0.9.3](https://git.woa.com/trpc-go/trpc-go/tree/v0.9.3) (2022-05-11)

### Bug Fixes

- 修复 server filter rsp 覆盖问题

## [0.9.2](https://git.woa.com/trpc-go/trpc-go/tree/v0.9.2) (2022-05-07)

### Features

- 调整流式客户端 context 拷贝，使得 trace 拦截器中能获取到 span context

### Bug Fixes

- 修复 restful 老桩代码兼容问题

## [0.9.1](https://git.woa.com/trpc-go/trpc-go/tree/v0.9.1) (2022-05-06)

### Features

- 支持自定义 UDP 链接 buffer 大小
- 支持 http 禁用连接池选项

### Bug Fixes

- 修复无协议多次回包问题
- 修正 app server service 切割方式
- 修正日志 Sync 同步等待问题

## [0.9.0](https://git.woa.com/trpc-go/trpc-go/tree/v0.9.0) (2022-04-18)

### Features

- server filter 和 rpc 入口函数 rsp 改成返回值
- http 同时支持 application/xml 和 text/xml
- 流式拦截器支持配置
- client 暴露获取 Options 方法
- 支持配置文件切换 transport
- 翻译注释

### Bug Fixes

- 升级 json-iterator，兼容 go1.18
- 修复流式 metadata 透传每次设置问题
- 修复 http client dial 失败时无法获取 ip 问题
- 修复 http client req header request 复用导致请求失败问题
- 修复流控内存泄漏问题
- 解决热重启父进程取消注册问题
- 解决 log data race 问题

### Breaking Changes

- 新版本框架可以同时支持新老不同函数签名的 server filter 插件，老版本格式的 server filter rsp 会传入 nil，所以新版本框架不允许在 server filter 里面操作 rsp 用于篡改回包数据，必须改成新版本函数签名格式

老版本格式：

```golang
func ServerFilter(ctx, req, rsp, next) error { // 新版本框架也可以支持这种格式的拦截器插件，不过此时传入的 rsp 是空指针
    // 前置逻辑，这里的 rsp 是 nil
    err := next(ctx, req, rsp)
    // 后置逻辑，这里不能操作 rsp，会触发空指针 panic，或者断言失败
}
```

新版本格式：

```golang
func ServerFilter(ctx, req, next) (rsp, error) { // 后续所有拦截器插件最好都慢慢改成这种格式
    // 前置逻辑
    rsp, err := next(ctx, req)
    // 后置逻辑，这里可以随意更改 rsp，甚至返回一个新的 rsp 结构体
}
```

## [0.8.6](https://git.woa.com/trpc-go/trpc-go/tree/v0.8.6) (2022-03-10)

### Bug Fixes

- 删除 gomonkey 单测代码

## [0.8.5](https://git.woa.com/trpc-go/trpc-go/tree/v0.8.5) (2022-03-04)

### Bug Fixes

- 解决 options overload ctrl 空指针问题
- 解决默认 client config 覆盖问题

## [0.8.4](https://git.woa.com/trpc-go/trpc-go/tree/v0.8.4) (2022-03-03)

### Features

- 插件提供加载完成回调通知
- 流式支持拦截器
- 流式支持单个连接最大并发流数量
- restful 支持 httprule 指定 body

### Bug Fixes

- 升级 gomonkey 依赖版本，解决 Apple M1 编译失败问题
- 修复流控帧卡死问题
- 修复 rand 输入参数错误导致死锁问题

## [0.8.3](https://git.woa.com/trpc-go/trpc-go/tree/v0.8.3) (2022-02-22)

### Bug Fixes

- 保留 options.LoadClientConfig，兼容历史问题

## [0.8.2](https://git.woa.com/trpc-go/trpc-go/tree/v0.8.2) (2022-02-22)

### Features

- restful 支持请求超时配置
- http 库支持返回标准库 http.Client
- http client 支持返回错误时，多次读取 body
- log.With fields 支持任意类型数据
- client selector 改成 filter 模式，支持寻址逻辑配置成任意执行顺序
- 流式支持一个连接多个流

### Bug Fixes

- 修复 http client 自动解压缩两次问题

## [0.8.1](https://git.woa.com/trpc-go/trpc-go/tree/v0.8.1) (2022-01-14)

### Features

- 支持用户注册 services 关闭前后的回调函数，在服务关闭时执行
- 连接池支持区分网络和协议类型
- 重构了 filter chain 实现，优化性能
- 新增过载保护相关错误码
- http 报错时相关的 header 现在可导出
- 新增 server 层级 timeout 配置
- 支持在 log format_config 中设置 function_key
- 可读性优化，注释优化

### Bug Fixes

- 修复流式相关 bugs
- 修复过载保护 marshalling/unmarshalling 相关问题
- 解决多路复用单测偶现失败问题
- 修复 admin 单测生成多余文件问题
- 修复 errs 包中 Newf 函数直接调用 New 函数导致 caller 多一层问题

## [0.8.0](https://git.woa.com/trpc-go/trpc-go/tree/v0.8.0) (2021-11-16)

### Breaking Changes

- 依赖模块 jce 和 reuseport 的 module 名从 git.code.oa.com 切换为 woa.com 域名 (!1253)
如果你项目中的 jce 或 reuseport 依赖的 module 名仍使用原来的 git.code.oa.com 可能会出现编译错误或运行时错误，可以考虑升级到使用 woa.com 域名的版本

### Features

- 新增 server client 过载保护模块
- udp service 支持协程池
- udp client transport 支持 buffer 池
- 优化 metrics histogram

### Bug Fixes

- 解决日志模块 race 问题
- 解决弱依赖插件 bug
- 解决 compress type copy 问题
- 解决无断言单测问题
- 解决 restful 单测偶现失败问题
- 解决 stream client 覆盖 transport 问题

## [0.7.3](https://git.woa.com/trpc-go/trpc-go/tree/v0.7.3) (2021-10-14)

### Breaking Changes

server 包中内置的 trpc.service 结构体实现的 server.Register 接口时，对于 serviceDesc 中重复注册的方法名将实现逻辑从覆盖变成直接报错 (!1220)

```go
type Service interface {
    // Register registers a proto service.
    Register(serviceDesc interface{}, serviceImpl interface{})
}
```

如果调用 Register 方法，但是忽略了错误，则可能会导致根据方法名路由失败的问题，例如使用了 thttp 包中 RegisterDefaultService。

### Features

- NoopSerialization Body 支持接口
- server 端空闲时间支持框架配置 server.service.idletime
- 优化连接复用逻辑
- errs 包支持设置跳过堆栈帧数
- 添加日志写入量属性监控 trpc.LogWriteSize
- 添加 trpc.Go(ctx, timeout, handler) 工具函数，方便用户启动异步任务，减少 ctx 相关 bug

### Bug Fixes

- restful 回包没有设置 Content-Type
- plugin 包内的 Config 结构体去除全局变量依赖
- go.mod 去除插件依赖
- 解决单测偶现失败问题
- 解决 http client 没有设置染色消息类型问题

## [0.7.2](https://git.woa.com/trpc-go/trpc-go/tree/v0.7.2) (2021-09-06)

### Features

- 支持 flatbuffers
- 连接池支持最小空闲连接数
- restful 支持跨域
- 客户端支持 WithDialTimeout
- RESTful 性能优化并支持设置默认的 Serializer
- 提供公共的安全随机函数可支持多模块调用
- 添加 panic buffer 长度定义
- 添加两个新的框架错误码
  - 23  被服务端限流
  - 123 被客户端限流

### Bug Fixes

- 将多路复用每个连接的队列长度默认值从 100k 改为 1024
- 在 copyCommonMessage 中加上对 commonMeta 和 CompressType 的拷贝
- 多路复用可以正确地返回客户端超时 (101) 和用户取消 (161) 两种错误
- 框架 udp 增加 context check
- 修复 m007 上报 RemoteAddr 为空

## [0.7.1](https://git.woa.com/trpc-go/trpc-go/tree/v0.7.1) (2021-08-03)

### Features

- 连接池支持初始化连接数
- client 支持 WithLocalAddr Option

### Bug Fixes

- 修复 restful 协议定义绝对路径时的空指针问题
- 修改超时控制有歧义注释
- 修复 msg resetDefault 时没将 callType 重置回默认值的问题
- 一些 typo 修改

## [0.7.0](https://git.woa.com/trpc-go/trpc-go/tree/v0.7.0) (2021-07-16)

### Features

- 支持 tRPC RESTful，pb option 注解生成 restful 接口
- 支持服务端过载保护
- config 接口提供 gomock 能力
- 支持 WriteV 系统调用，提升发包效率
- 支持采集上报服务端和客户端包大小

### Bug Fixes

- 修复 http 客户端无法从错误码判断是否超时
- 修复 admin 包 unregisterHandlers 的数组越界问题
- 修复 udp FramerBuilder 为 nil 错误
- 优化相同配置文件变更事件只触发一次
- 修复流式服务端 error 没有返回给客户端
- admin 调整为 service 实现，避免独立客户端无法开启 pprof 问题
- 修复多路复用重复 close 导致 err 变更问题

## [0.6.6](https://git.woa.com/trpc-go/trpc-go/tree/v0.6.6) (2021-06-25)

### Features

- 性能优化
- 支持只发不收
- 更新 godoc 到 pkg.woa.com

### Bug Fixes

- 解决连接泄露问题
- 解决内存占用大问题
- 解决 rand.Seed 干扰问题

## [0.6.5](https://git.woa.com/trpc-go/trpc-go/tree/v0.6.5) (2021-05-27)

### Features

- 性能优化：slice 预分配内存
- 提升连接池空闲状态检查时效性
- udp client 校验 framer
- 插件支持弱依赖关系
- errs 堆栈支持过滤能力
- http client 支持 patch 方法

### Bug Fixes

- 解决单测偶现失败问题
- 解决 http transinfo env-key base64 问题
- 解决 client stream data race 问题

## [0.6.4](https://git.woa.com/trpc-go/trpc-go/tree/v0.6.4) (2021-05-13)

### Bug Fixes

- 解决 registry 检查失败问题
- 流式关闭连接导致 decode 错误

### Features

- restful: 实现 double array trie, 用于过滤已被 httprule 引用的字段 (!1033)

### Enhancements

- internal: 支持对 pb Option: trpc.http.api 的解析; pattern: 支持对 http 请求 url path 的匹配 (!1033)

## [0.6.3](https://git.woa.com/trpc-go/trpc-go/tree/v0.6.3) (2021-05-12)

### Features

- 性能优化：协程池改为开源 ants 实现
- http status code 支持 2xx 成功返回码

### Bug Fixes

- udp 解包失败直接丢包，解决 udp server 和 dns server 冲突问题
- http transinfo env-key base64 编码
- selector options loadbalancer 拼写错误问题
- 多路复用失败重连

## [0.6.2](https://git.woa.com/trpc-go/trpc-go/tree/v0.6.2) (2021-04-26)

### Features

- 支持 http post multipart/form-data
- 尽早设置 http client rsp header

### Bug Fixes

- 解决包长度溢出 bug
- 解决单测偶发失败问题
- 解决代码规范问题
- 修复向已关闭的 stream 流写入时不会返回 err

## [0.6.1](https://git.woa.com/trpc-go/trpc-go/tree/v0.6.1) (2021-04-16)

### Bug Fixes

- 解决 http clent request content-length 为 0 问题

## [0.6.0](https://git.woa.com/trpc-go/trpc-go/tree/v0.6.0) (2021-04-14)

### Features

- 支持 udp client transport io 复用
- 支持服务无损更新
- 支持 http/https 客户端链接参数设置
- client 在拦截器之前设置超时时间
- 性能优化

### Bug Fixes

- 解决 http client 大包内存泄露问题
- 解决代码重复问题
- 解决流式无法获取 metadata 问题
- 解决单测偶现失败问题

## [0.5.2](https://git.woa.com/trpc-go/trpc-go/tree/v0.5.2) (2021-02-26)

### Features

- 统一收拢 trpc 工具类函数到 trpc_util.go 文件
- 统一收拢环境变量 key 到 internal/env/env.go 文件
- 统一收拢监控上报 key 到 internal/report/metrics_report.go 文件
- 去除重定向 std log 到日志文件逻辑，提供`log.RedirectStdLog`函数供用户调用
- 流控功能实现完成
- 支持动态设置虚拟节点数
- 支持同时使用有协议和无协议 http 服务
- admin 使用`net/http/pprof`，支持分析 cpu，内存
- 支持配置`network: tcp,udp`同时监听 tcp 和 udp
- http 支持`application/xml`

### Bug Fixes

- 解决 client.DefaultClientConfig 并发问题
- 解决 http env 多环境透传问题
- 解决创建日志实例失败导致 panic 问题
- 解决 client target 非域名解析卡顿问题
- 解决 io 复用内存泄露问题
- 禁用服务路由时清空多环境透传信息
- 解决 client 后端拦截器并发问题

## [0.5.1](https://git.woa.com/trpc-go/trpc-go/tree/v0.5.1) (2021-01-08)

### Features

- 增加 trpc.CloneContext 接口，方便异步处理
- 增加 client.WithMultiplexedPool 接口，方便用户自定义 io 复用连接参数
- 增加 config.Reload 接口

### Bug Fixes

- 日志按时间滚动也限制大小，异步满丢弃上报监控
- 优化大包时，内存使用率过高问题
- 解决圈复杂度超标问题
- 修复 DataFrameType 字段错误问题

## [0.5.0](https://git.woa.com/trpc-go/trpc-go/tree/v0.5.0) (2020-12-28)

### Features

- 支持 client 重试策略
- 性能优化：支持协程池
- 性能优化：gzip 压缩缓存
- 性能优化：io 复用支持多连接
- 支持 http application/x-protobuf Content-Type
- WithTarget 支持负载均衡方式
- http client transport 支持配置最大空闲连接数
- selector 支持传入 context

### Bug Fixes

- 日志模式默认极速写：日志异步写队列，队列满则丢弃
- 修复 client filter 获取不到请求 header 问题
- 解决代码规范问题，圈复杂度超标问题
- 更新覆盖率图标到 https 链接，解决 chrome mixed-content 问题
- 解决 filter 非并发安全问题

## [0.4.2](https://git.woa.com/trpc-go/trpc-go/tree/v0.4.2) (2020-11-26)

### Bug Fixes

- 框架配置解析环境变量，只解析${var}不解析$var，解决 redis 密码包含$字符问题

## [0.4.1](https://git.woa.com/trpc-go/trpc-go/tree/v0.4.1) (2020-11-24)

### Bug Fixes

- 修复 kafka 等自定义协议没配置 ip 的情况

## [0.4.0](https://git.woa.com/trpc-go/trpc-go/tree/v0.4.0) (2020-11-24)

### Features

- 支持流式
- 客户端连接模式支持 IO 复用
- 单测覆盖率提升到 87% 以上
- Config 接口支持 toml 格式
- Config 支持填写默认值
- client 寻址逻辑移到拦截器内部
- 框架配置支持环境变量占位符

### Bug Fixes

- admin 模块去掉 net/http/pprof 依赖，解决安全问题
- 修复。code.yml 问题
- 修复 client 配置 timeout 不生效问题
- 解决代码规范问题，圈复杂度过高问题
- 解决框架配置 nic 填错没有阻止启动问题
- http 响应没有返回透传字段 trpc-trans-info

## [0.3.7](https://git.woa.com/trpc-go/trpc-go/tree/v0.3.7) (2020-09-22)

### Features

- errs 增加 WithStack 携带调用栈信息
- 热重启信号量更改为变量允许用户自己修改
- 服务端默认异步 server_async

### Bug Fixes

- 解决热重启问题
- 解决 http response error msg 错误问题
- noresponse 不关闭连接

## [0.3.6](https://git.woa.com/trpc-go/trpc-go/tree/v0.3.6) (2020-07-29)

### Features

- http client method 支持 option 参数
- 框架自身监控上报属性加上 trpc. 前缀
- 支持单个 client 配置 set_name env_name disable_servicerouter

### Bug Fixes

- 解决连接池复用 bug，导致串包问题
- 解决 log 删除多余备份失效问题
- 解决 http rpcname invalid 问题
- 解决多维监控无法设置 name 问题

## [0.3.5](https://git.woa.com/trpc-go/trpc-go/tree/v0.3.5) (2020-07-27)

### Bug Fixes

- 解决框架 SetGlobalConfig 后移导致插件启动失败问题
- 修复 client namespace 为空问题

## [0.3.4](https://git.woa.com/trpc-go/trpc-go/tree/v0.3.4) (2020-07-24)

### Features

- rpc invalid 时，添加当前服务 service name，方便排查问题
- 提高单测覆盖率
- http 端口 443 时默认设置 scheme 为 https
- 将开源 lumberjack 日志切换为内置 rollwriter 日志，提高打日志性能
- 解决圈复杂度问题，每个函数尽量控制到 5 以内
- 对端口复用的 httpserver 添加热重启时停止接收新请求

### Bug Fixes

- 解决动态设置日志等级无效问题
- 修复同一 server 使用多个证书时缓存冲突问题
- 修复 http client 连接失败上报问题
- 解决 server write 错误导致死循环问题
- 解决 server 代理透传二进制问题
- 解决 http get 请求无法解析二进制字段问题
- 解决框架启动调用两次 SetGlobalConfig 问题

## [0.3.3](https://git.woa.com/trpc-go/trpc-go/tree/v0.3.3) (2020-07-01)

### Features

- http default transport 使用原生标准库的 default transport
- 支持 client 短连接模式
- 支持设置自定义连接池
- 日志 key 字段支持配置
- 连接池 MaxIdle 最大连接数调整为无上限

### Bug Fixes

- 解决 server filter 去重问题
- 解决 ip 硬编码安全规范问题
- 解决代码规范问题

## [0.3.2](https://git.woa.com/trpc-go/trpc-go/tree/v0.3.2) (2020-06-18)

### Features

- 支持 server 端异步处理请求，解决非 trpc-go client 调用超时问题
- 框架内部默认 import uber automaxprocs，解决容器内调度延迟问题

### Bug Fixes

- 解决 client filter 覆盖清空问题
- 解决 http server CRLF 注入问题

## [0.3.1](https://git.woa.com/trpc-go/trpc-go/tree/v0.3.1) (2020-06-10)

### Features

- 支持用户自己设置 Listener
- 支持 http get 请求独立序列化方式

### Bug Fixes

- 解决 client filter 执行两次的问题
- 解决 server 回包无法指定序列化方式和压缩方式问题
- 解决 http client proxy 用户无法设置 protocol 的问题

## [0.3.0](https://git.woa.com/trpc-go/trpc-go/tree/v0.3.0) (2020-05-29)

### Features

- 支持传输层 tls 鉴权
- 支持 http2 protocol
- 支持 admin 动态设置不同 logger 不同 output 的日志等级
- 支持 http Put Delete 方法

## [0.2.8](https://git.woa.com/trpc-go/trpc-go/tree/v0.2.8) (2020-05-12)

### Features

- 代码 OWNER 制度更改，owners.txt 改成。code.yml，符合 epc 标准
- 支持 http client post form 请求
- 支持 client SendOnly 只发不收请求
- 支持自定义 http 路由 mux
- 支持 http.SetContentType 设置 http content-type 到 trpc serialization type 的映射关系，兼容不规范老 http 框架服务返回乱写的 content-type

### Bug Fixes

- 解决 http client rsp 没有反序列化问题
- 解决 tcp server 空闲时间不生效问题
- 解决多次调用 log.WithContextFields 新增字段不生效问题

## [0.2.7](https://git.woa.com/trpc-go/trpc-go/tree/v0.2.7) (2020-04-30)

### Bug Fixes

- 解决 flag 启动失败问题

## [0.2.6](https://git.woa.com/trpc-go/trpc-go/tree/v0.2.6) (2020-04-29)

### Features

- 复用 msg 结构体，echo 服务性能从 39w/s 提升至 41w/s
- 提升单元测试覆盖率至 84.6%
- 新增一致性哈希路由算法

### Bug Fixes

- tcp listener 没有 close
- 解决 NewServer flag 定义冲突问题

## [0.2.5](https://git.woa.com/trpc-go/trpc-go/tree/v0.2.5) (2020-04-20)

### Features

- 添加 trpc.NewServerWithConfig 允许用户自定义框架配置文件格式
- 支持 https client，支持 https 双向认证
- 支持 http mock
- 添加性能数据实时看板，readme benchmark icon 入口

### Bug Fixes

- 将所有 gogo protobuf 改成官方的 golang protobuf，解决兼容问题
- admin 启动失败直接 panic，解决 admin 启动失败无感知问题

## [0.2.4](https://git.woa.com/trpc-go/trpc-go/tree/v0.2.4) (2020-04-02)

### Features

- http server head 添加原始包体 ReqBody
- 配置文件支持 toml 序列化方式
- 添加 client CalleeMethod option，方便自定义监控方法名
- 添加 dns 寻址方式：dns://domain:port

### Bug Fixes

- 改造 log api，将 Warning 改成 Warn
- 更改 DefaultSelector 为接口方式

## [0.2.3](https://git.woa.com/trpc-go/trpc-go/tree/v0.2.3) (2020-03-24)

### Bug Fixes

- 禁用 client filter 时不加载 filter 配置

## [0.2.2](https://git.woa.com/trpc-go/trpc-go/tree/v0.2.2) (2020-03-23)

### Features

- 框架内部关键错误上报 metrics
- 多维监控使用数组形式

## [0.2.1](https://git.woa.com/trpc-go/trpc-go/tree/v0.2.1) (2020-03-19)

### Features

- 支持禁用 client 拦截器

## [0.2.0](https://git.woa.com/trpc-go/trpc-go/tree/v0.2.0) (2020-03-18)

### Bug Fixes

- 解决 golint 问题

### Features

- 支持 set 路由
- client config 支持配置下游的序列化方式和压缩方式
- 框架支持 metrics 标准多维监控接口
- 所有 wiki 文档全部转移到 iwiki

## [0.1.6](https://git.woa.com/trpc-go/trpc-go/tree/v0.1.6) (2020-03-11)

### Bug Fixes

- 新增插件初始化完成事件通知

## [0.1.5](https://git.woa.com/trpc-go/trpc-go/tree/v0.1.5) (2020-03-09)

### Bug Fixes

- 解决 golint 问题
- 解决 client transport 收包失败都返回 101 超时错误码问题

### Features

- client transport framer 复用
- http server decode 失败返回 400，encode 失败返回 500
- 新增更安全的多并发简易接口 trpc.GoAndWait
- 新增 http client 通用的 Post Get 方法
- server 拦截器未注册不让启动
- 日志 caller skip 支持配置
- 支持 https server
- 添加上游客户端主动断开连接，提前取消请求错误码 161

## [0.1.4](https://git.woa.com/trpc-go/trpc-go/tree/v0.1.4) (2020-02-18)

### Bug Fixes

- 客户端设置不自动解压缩失效问题

## [0.1.3](https://git.woa.com/trpc-go/trpc-go/tree/v0.1.3) (2020-02-13)

### Bug Fixes

- 插件初始化加载 bug

## [0.1.2](https://git.woa.com/trpc-go/trpc-go/tree/v0.1.2) (2020-02-12)

### Bug Fixes

- http client codec CalleeMethod 覆盖问题
- server/client mock api 失效问题

### Features

- 新增 go1.13 错误处理 error wrapper 模式
- 添加插件初始化依赖顺序逻辑
- 新增 trpc.BackgroundContext() 默认携带环境信息，避免用户使用错误

## [0.1.1](https://git.woa.com/trpc-go/trpc-go/tree/v0.1.1) (2020-01-21)

### Bug Fixes

- http client transport 无法设置 content-type 问题
- 天机阁 ClientFilter 取不到 CalleeMethod 问题
- http client transport 无法设置 host 问题

### Features

- 增加 disable_request_timeout 配置开关，允许用户自己决定是否继承上游超时时间，默认会继承
- 增加 callee framework error type，用以区分当前框架错误码，下游框架错误码，业务错误码
- 下游超时时，errmsg 自动添加耗时时间，方便定位问题
- http server 回包 header 增加 nosniff 安全 header
- http 被调 method 使用 url 上报

## [0.1.0](https://git.woa.com/trpc-go/trpc-go/tree/v0.1.0) (2020-01-10)

### Bug Fixes

- 滚动日志默认按大小，流水日志按日期
- 日志路径和文件名拼接 bug
- 指定环境名路由 bug

### Features

- 代码格式优化，符合 epc 标准
- 插件上报统计数据

## [0.1.0-rc.14](https://git.woa.com/trpc-go/trpc-go/tree/v0.1.0-rc.14) (2020-01-06)

### Bug Fixes

- 连接池默认最大空闲连接数过小导致频繁创建 fd，出现 timewait 爆满问题，改成默认 MaxIdle=2048
- server transport 没有 framer builder 导致请求 crash 问题

### Features

- 支持从名字服务获取被调方容器名

## [0.1.0-rc.13](https://git.woa.com/trpc-go/trpc-go/tree/v0.1.0-rc.13) (2019-12-30)

### Bug Fixes

- 连接池偶现 EOF 问题：server 端统一空闲时间 1min，client 端统一空闲时间 50s
- 高并发下超时设置 http header crash 问题：去除 service select 超时控制
- http 回包 json enum 变字符串 改成 enum 变数字，可配置
- http header 透传信息二进制设置失败问题，改成 transinfo base64 编码

### Features

- 支持无协议文件自定义 http 路由
- 支持请求 http 后端携带 header
- http 服务支持 reuseport 热重启

## [0.1.0-rc.12](https://git.woa.com/trpc-go/trpc-go/tree/v0.1.0-rc.12) (2019-12-24)

### Bug Fixes

- 包大小 uint16 限制
- metrics counter 锁 bug
- 单个插件初始化超时 3s，防止服务卡死
- 同名网卡 ip 覆盖
- 多 logger 失效

### Features

- 指定环境名路由
- http 新增自定义 ErrorHandler
- timer 改成插件模式
- 添加 godoc icon

## [0.1.0-rc.11](https://git.woa.com/trpc-go/trpc-go/tree/v0.1.0-rc.11) (2019-12-09)

### Bug Fixes

- udp client transport 对象池复用导致 buffer 错乱

## [0.1.0-rc.10](https://git.woa.com/trpc-go/trpc-go/tree/v0.1.0-rc.10) (2019-12-05)

### Bug Fixes

- udp client connected 模式 writeto 失败问题

## [0.1.0-rc.9](https://git.woa.com/trpc-go/trpc-go/tree/v0.1.0-rc.9) (2019-12-04)

### Bug Fixes

- 连接池超时控制无效
- 单测偶现失败
- 默认配置失效

### Features

- 新增多环境开关
- udp client transport 新增 connection mode，由用户自己控制请求模式
- udp 收包使用对象池，优化性能
- admin 新增性能分析接口

## [0.1.0-rc.8](https://git.woa.com/trpc-go/trpc-go/tree/v0.1.0-rc.8) (2019-11-26)

### Bug Fixes

- server WithProtocol option 漏了 transport
- 后端回包修改压缩方式不生效
- client namespace 配置不生效

### Features

- 支持 client 工具多环境路由
- 支持 admin 管理命令
- 支持 热重启
- 优化 日志打印

## [0.1.0-rc.7](https://git.woa.com/trpc-go/trpc-go/tree/v0.1.0-rc.7) (2019-11-21)

### Features

- 支持 client option 设置多环境

## [0.1.0-rc.6](https://git.woa.com/trpc-go/trpc-go/tree/v0.1.0-rc.6) (2019-11-20)

### Bug Fixes

- 支持一致性哈希路由

## [0.1.0-rc.5](https://git.woa.com/trpc-go/trpc-go/tree/v0.1.0-rc.5) (2019-11-08)

### Bug Fixes

- tconf api
- transport 空指针 bug

### Features

- 多环境治理
- 代码质量管理 owner 机制

## [0.1.0-rc.4](https://git.woa.com/trpc-go/trpc-go/tree/v0.1.0-rc.4) (2019-11-04)

### Bug Fixes

- frame builder 魔数校验，最大包限制默认 10M

### Features

- 提高单测覆盖率

## [0.1.0-rc.3](https://git.woa.com/trpc-go/trpc-go/tree/v0.1.0-rc.3) (2019-10-28)

### Bug Fixes

- http client codec

## [0.1.0-rc.2](https://git.woa.com/trpc-go/trpc-go/tree/v0.1.0-rc.2) (2019-10-25)

### Bug Fixes

- windows 连接池 bug

### Features

- 测试覆盖率提高到 83%

## [0.1.0-rc.1](https://git.woa.com/trpc-go/trpc-go/tree/v0.1.0-rc.1) (2019-10-25)

### Features

- 一发一收应答式服务模型
- 支持 tcp udp http 网络请求
- 支持 tcp 连接池，buffer 对象池
- 支持 server 业务处理函数前后链式拦截器，client 网络调用函数前后链式拦截器
- 提供 trpc 代码 [生成工具](https://git.woa.com/trpc-go/trpc-go-cmdline)，通过 protobuf idl 生成工程服务代码模板
- 提供 [rick 统一协议管理平台](http://trpc.rick.woa.com/rick/pb/list)，tRPC-Go 插件通过 proto 文件自动生成 pb.go 并自动 push 到 [统一 git](https://git.woa.com/trpcprotocol)
- 插件化支持 任意业务协议，目前已支持 trpc，[tars](https://git.woa.com/trpc-go/trpc-codec/tree/master/tars)，[oidb](https://git.woa.com/trpc-go/trpc-codec/tree/master/oidb)
- 插件化支持 任意序列化方式，目前已支持 protobuf，jce，json
- 插件化支持 任意压缩方式，目前已支持 gzip，snappy
- 插件化支持 任意链路跟踪系统，目前已使用拦截器方式支持 [天机阁](https://git.woa.com/trpc-go/trpc-opentracing-tjg) [jaeger](https://git.woa.com/trpc-go/trpc-opentracing-jaeger)
- 插件化支持 任意名字服务，目前已支持 [老 l5](https://git.woa.com/trpc-go/trpc-selector-cl5)，[cmlb](https://git.woa.com/trpc-go/trpc-selector-cmlb)，[北极星测试环境](https://git.woa.com/trpc-go/trpc-naming-polaris)
- 插件化支持 任意监控系统，目前已支持 [老 sng-monitor-attr 监控](https://git.woa.com/trpc-go/metrics-plugins/tree/master/attr)，[pcg 007 监控](https://git.woa.com/trpc-go/metrics-plugins/tree/master/m007)
- 插件化支持 多输出日志组件，包括 终端 console，本地文件 file，[远程日志 atta](https://git.woa.com/trpc-go/trpc-log-remote-atta)
- 插件化支持 任意负载均衡算法，目前已支持 roundrobin weightroundrobin
- 插件化支持 任意熔断器算法，目前已支持 北极星熔断器插件
- 插件化支持 任意配置中心系统，目前已支持 [tconf](https://git.woa.com/trpc-go/config-tconf)

### 压测报告

| 环境 | server | client | 数据 | tps | cpu |
| :--: | :--: |:--: |:--: |:--: |:--: |
| 1 | v8 虚拟机 9.87.179.247 | 星海平台 jmeter 9.21.148.88 | 10B 的 echo 请求 | 25w/s | null |
| 2 | b70 物理机 100.65.32.12 | 星海平台 jmeter 9.21.148.88 | 10B 的 echo 请求 | 42w/s | null |
| 3 | v8 虚拟机 9.87.179.247 | eab 工具，b70 物理机 100.65.32.13 | 10B 的 echo 请求 | 35w/s | 64% |
| 4 | b70 物理机 100.65.32.12 | eab 工具，b70 物理机 100.65.32.13 | 10B 的 echo 请求 | 60w/s | 45% |

### 测试报告

- 整体单元测试 [覆盖率 80%](http://devops.oa.com/console/pipeline/pcgtrpcproject/p-da0d17b2016f404fa725983ae020ed01/detail/b-5ee497f8d96348359b874ec062795ca5/output)
- 支持 [server mock 能力](server/mockserver)
- 支持 [client mock 能力](client/mockclient)

### 开发文档

- 每个 package 有 [README.md](server)
- [examples/features](examples/features) 有每个特性的代码示例
- [examples/helloworld](examples/helloworld) 具体工程服务示例
- [trpc wiki](https://iwiki.woa.com/pages/viewpage.action?pageId=89292279) 有详细的设计文档，开发指南，FAQ 等

### 下一版本功能规划

- 服务性能优化，提高 tps
- 完善开发文档，提高易用性
- 完善单元测试，提高测试覆盖率
- 支持 [更多协议](https://git.woa.com/trpc-go/trpc-codec)，打通全公司大部分存量平台框架
- admin 命令行系统
- auth 鉴权
- 多环境/set/idc/版本/哈希 路由能力
- 染色 key 能力
