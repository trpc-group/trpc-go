# 如何在 tRPC-Go 中添加集成测试用例

## 什么时候该添加集成测试用例

测试用例可以从 Test Size 和 Test Scope 两个维度进行划分，按 Test Size 从小到大划分，包含小型测试，中型测试和大型测试；按 Test Scope 从小到大划分，包含单元测试，集成测试和系统测试。测试用例的维度越大，运行时需要的 CPU，IO，网络，内存等资源越多，运行速度越慢，得到的运行结果也越不可靠 [1]。对于一个系统来说，这几种测试的占比在软件工程实践中，存在如下一个测试金字塔 [2] 的粗略指导性原则，维度越低的测试用例占比应该越高。

```
        .                 系统测试 5%
       ...                集成测试 15%
................          单元测试 80%
```

**因此在 tRPC-Go 中也建议尽量编写小维度的小型测试和单元测试**。以下场景可能需要添加集成测试用例：

1. 需要使用到多进程的情况，例如测试 server 的优雅重启，可以仿照 `graceful_restart_test.go` 中的实现。
2. 需要创建 client 来访问一个真实 server [^1] 提供的服务来验证功能的场景，根据 server 提供的服务类型的不同，可仿照 `trpc_test.go`、 `http_test.go`、`restful_test.go` 和`streaming_test.go`。

[^1]: tRPC-Go 的集成测试网络调用只使用了 localhost 网络，同时整个测试只能运行在单机上。

## 测试代码的组织形式

### 平铺

Go 语言内置 `go test` 命令，该命令会运行 `_test.go` 文件中符合 `TestXXX` 命名规则的函数，进而实现测试代码的执行。因为 `go test` 并没有对测试代码的组织形式提出任何约束条件，所以现有很多测试代码都采用了十分简单的“平铺”组织。这种“平铺”的代码组织形式优点是每个测试函数都是互不相关的，在代码结构上没有额外的抽象，上手非常容易。然而， “平铺”并不适用于 tRPC-Go 的集成测试，因为  tRPC-Go 的集成测试存在以下特点：

1. 会测试各种用户使用场景，存在大量的测试用例，“平铺”由于没有层次感，代码就会显得比较混乱；
2. 多个测试用例在概念上可能是相关的，如都是针对 tRPC-Go 数据透传的测试，应该把这些用例组织在一起；
3. 几乎每个测试用例前都需要在存在某个 server 的特定环境中执行，且在用例执行完后需要清理资源（主要是关闭启动的 server），应该把这些共有逻辑提炼出来，在所有用例之间实现共享。

### xUnit 家族模式

针对 tRPC-Go 的集成测试的特点 1 和特点 2，解决办法是让测试代码组成形式具有层次，将概念上相关的用例在组织一起，我们采用了 xUnit 家族 [3，4，5] 典型的三层测试代码组织形式：

```bash
Test Project
    Test Suite1
        Test Case1
        ...
        Test CaseN
    Test Suite2 
        Test Case1
        ...
        Test CaseN
    ... 
    Test SuiteN
  	   Test Case1
       ...
       Test CaseN
```

这种代码组织形式包含三个层次，一个 Test Project 由若干个 Test Suite 组成，而每个 Test Suite 又包含多个 Test Case。

在 Go 1.7 版本之前，使用 Go 原生工具和标准库是无法按照上述形式组织测试代码的，在 Go 1.7 中加入的对 subtest[6] 的支持允许在 Go 中也可以使用上面这种方式组织测试代码。然而社区中广泛流行 testify 在 Go 官方 testing 包的基础上做了简单封装，方便好用，其中的 suite[7] 包提供了类似的能力，所以 tRPC-Go 的集成测试主要利用了 testify/suite 来实现以上的代码组织形式。

### 测试固件

针对 tRPC-Go 的集成测试的特点 3，解决办法是利用 testify/suite 中的 SetUp 和 TearDown 函数分别来创建/设置和拆除/销毁测试固件[8]。

> 测试固件是指一个人造的、确定性的环境，一个测试用例或一个测试套件（下的一组测试用例）在这个环境中进行测试，其测试结果是可重复的（多次测试运行的结果是相同的）。

测试固件存在于以下常见场景：

- 创建启动 server 需要的相关资源并启动一个 server，测试结束后关闭一个 server，并清理相关资源

- 将一组已知的特定数据加载到数据库中，测试结束后清除这些数据
- 复制一组特定的已知文件，测试结束后清除这些文件

当然测试固件可以有不同的级别，分别对应 Test Project、Test Suite 和 Test Case，因此 tRPC-Go 中集成测试执行流一般如下：

```bash
 集成测试包
    测试开始
        Test Project 级别的固件创建
            Test Suite 级别的固件创建（启动 server 等）
                Test Case 级别的固件创建
                    运行测试（一般需要 client 发送请求同 server 进行交互，然后根据返回的结果进行验证）
                Test Case 级别的固件销毁
            Test Suite 级别的固件销毁（关闭 server 等）
        Test Project 级别的固件销毁
    测试结束
```

### 代码示例

下面以测试 tRPC-Go 框架代码中的 admin 模块为例子来分析下 tRPC-Go 集成测试代码所采用的代码组织形式和执行流。

我们通过运行 admin_go.test 中 的 TestAdmin 函数来分析集成测试的代码组织形式：

```bash
=== RUN   TestRunSuite
--- PASS: TestRunSuite (0.51s)
=== RUN   TestRunSuite/TestAdmin
2022-10-18 14:19:26.999	DEBUG	maxprocs/maxprocs.go:47	maxprocs: Leaving GOMAXPROCS=10: CPU quota undefined
2022-10-18 14:19:26.999	INFO	server/service.go:158	process:6983, trpc service:trpc.testing.end2end.TestTRPC launch success, tcp::0, serving ...
    --- PASS: TestRunSuite/TestAdmin (0.51s)
=== RUN   TestRunSuite/TestAdmin/cmds
        --- PASS: TestRunSuite/TestAdmin/cmds (0.00s)
=== RUN   TestRunSuite/TestAdmin/cmds-config
        --- PASS: TestRunSuite/TestAdmin/cmds-config (0.00s)
=== RUN   TestRunSuite/TestAdmin/cmds-loglevel
        --- PASS: TestRunSuite/TestAdmin/cmds-loglevel (0.00s)
=== RUN   TestRunSuite/TestAdmin/CustomHandleFunc
        --- PASS: TestRunSuite/TestAdmin/CustomHandleFunc (0.00s)
=== RUN   TestRunSuite/TestAdmin/is-healthy
2022-10-18 14:19:27.207	INFO	server/service.go:488	process:6983, trpc service:trpc.testing.end2end.TestTRPC, closing ...
2022-10-18 14:19:27.207	INFO	admin/admin.go:154	process:6983, admin server, closed
2022-10-18 14:19:27.508	INFO	server/service.go:508	process:6983, trpc service:trpc.testing.end2end.TestTRPC, closed
        --- PASS: TestRunSuite/TestAdmin/is-healthy (0.00s)
PASS
```

从中可以看到 `go test`的输出相对于“平铺”更有层次感，我们可以一眼看出对哪些函数/方法进行了测试、这些被测对象对应的 TestSuite 以及 TestSuite 中的每个 TestCase。

```bash
TestRunSuite  <--- 对应 Test Project
    TestAdmin  <--- 对应 Test Suite1
       testCmds              ｜  <---- 对应 Test Case1
       testCmdsConfig        ｜  <---- 对应 Test Case2
       testCmdsLogLevel      ｜  <---- 对应 Test Case3
       testCustomHandleFunc  ｜  <---- 对应 Test Case4
       testIsHealthy         ｜  <---- 对应 Test Case5
```

我们根据函数的执行顺序，依次查看 `TestRunSuite` 、`TestAdmin` 和 `testXXX(testCmds, testCmdsConfig, ...)` 的代码逻辑来分析测试执行流。

```go
func TestRunSuite(t *testing.T) {
   suite.Run(t, new(TestSuite))
}

func (s *TestSuite) SetupSuite() {}

func (s *TestSuite) SetupTest() {}

func (s *TestSuite) TestAdmin() {
  s.copyTRPCConfigFile(defaultTRPCWithAdminConfigPath)
	s.startTRPCServerWithListener(&TRPCService{})
	// wait a while until admin server has started.
	time.Sleep(200 * time.Millisecond)
	s.Run("cmds", s.testCmds)
	s.Run("cmds-config", s.testCmdsConfig)
	s.Run("cmds-loglevel", s.testCmdsLogLevel)
	s.Run("CustomHandleFunc", s.testCustomHandleFunc)
	s.Run("is-healthy", s.testIsHealthy)
}

func (s *TestSuite) testCmds(){
  resp, err := httpRequest(http.MethodGet, fmt.Sprintf("http://%s/cmds", defaultAdminListenAddr), "")
	require.Nil(s.T(), err)
	r := struct {
		Errcode int      `json:"errorcode"`
		Message string   `json:"message"`
		Cmds    []string `json:"cmds"`
	}{}
	require.Nil(s.T(), json.Unmarshal(resp, &r), "Unmarshal failed")
	require.ElementsMatch(
		s.T(),
		[]string{
			"/cmds",
			"/version",
			"/debug/pprof/",
			"/debug/pprof/symbol",
			"/debug/pprof/trace",
			"/cmds/loglevel",
			"/cmds/config",
			"/is_healthy/",
			"/debug/pprof/cmdline",
			"/debug/pprof/profile",
		},
		r.Cmds,
	)
}

func (s *TestSuite) TearDownTest() {}

func (s *TestSuite) TearDownSuite() {}
```

第一步， `TestRunSuite` 会创建一个 `TestSuite` 实例，并依次运行 `TestSuite` 中符合 `TestXXX` 命名规则的成员方法，例如这里的 `TestAdmin`。可以认为这里的 `TestSuite` 类管理着 tRPC-Go 集成测试的所有 Test Suite，`TestAdmin` 为其中一个 Test Suite [^2]。

第二步，在执行 `TestAdmin` 之前会依次执行 `TestSuite` 的 `SetupSuite()` 和 `SetupTest()` 来创建 Test Project 级别和 Test Suite 级别的固件。

第三步，会通过 `s.Run` 方法执行 `TestAdmin` 中的 `testCmds`、`testCmdsConfig`、`testCmdsLogLevel`、 `testCustomHandleFunc` 和 `testIsHealthy` 中的 Test Case，这里可以在每个函数的开始出创建 Test Case 级别的固件，用 defer 函数的方法来销毁 Test Case 级别的固件。

第四步，会在每个 Test Case 中进行实际的测试，例如 `testCmds` 会向 admin server 发送一个 HTTP Get 请求，根据返回的结果做验证。

第五步，会在每个 `TestAdmin` 结束后运行 `TestSuite` 的 `TearDownTest` 来销毁 Test Suite 级别的固件，会在所有的 Test Suite 结束后运行 `TearDownSuite` 来销毁 Test Project 级别的固件。

[^2]: tRPC-Go 中`TestSuite` 结构体的语义和 testify/suite 中 suite 的语义有些许区别，后者的 suite 就是指一个 Test Suite。

### 新增一个集成测试用例示范

一般来说，在 tRPC-Go 集成测试中，新增一个测试用例一般需要先启动一个 server，然后创建 client 来访问 server 提供了 service，最后根据返回结果验证。下面以测试 tRPC 协议 一应一答超时为例：

```go
1 func (s *TestSuite) TestClientTimeoutAtUnaryCall() {
2 	s.startServer(&TRPCService{unaryCallSleepTime: time.Second})
3 
4 	c := s.newTRPCClient()
5 	_, err := c.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest, client.WithTimeout(100*time.Millisecond))
6 	require.Equal(s.T(), errs.RetClientTimeout, errs.Code(err))
7 }
```

#### 第一步：根据 tRPC-Go 测试代码的组织形式来组织测试用例

如果一个 Test Suite 中只包含 一个 Test Case，可以把该 Test Case 的位置往上提升。根据需要测试的功能，取一个符合 TestXXX 语法规则且简单易懂的函数名，代码的第 1 行定义了一个名为 `TestClientTimeoutAtUnaryCall`的`*TestSuite` 的成员方法。

#### 第二步：启动一个提供 trpc service 的 server

代码的第 2 行调用 `*TestSuite` 的成员方法 `startServer` 启动了一个 trpc service，并设置一应一答服务中的 sleep 时间为 1s。tRPC-Go 集成测试可以为启动的服务提供 3 种级别的灵活性：

- 可以启动不同类型的 service。service_imp.go 实现了 protocols 目录下 test.proto 定义的各种类型的 service，包括`TRPCService`、`StreamingService` 、`testHTTPService` 和 `testRESTfulService`，涵盖了 trpc、streaming、http 和 restful 协议。

- 可以自定义 service 中的 method 处理逻辑。如可以在构造 `TRPCService` 的填写`EmptyCallF`字段，则 client 发起 `EmptyCall` 调用时会执行自定义的处理逻辑。

  ```go
  // TRPCService to test tRPC service.
  type TRPCService struct {
  	// Customizable implementations of server handlers.
  	EmptyCallF func(ctx context.Context, in *testpb.Empty) (*testpb.Empty, error)
  
  	unaryCallSleepTime time.Duration
  }
  
  // EmptyCall to test empty call.
  func (s *TRPCService) EmptyCall(ctx context.Context, in *testpb.Empty) (*testpb.Empty, error) {
  	if s.EmptyCallF != nil {
  		return s.EmptyCallF(ctx, in)
  	}
  	return &testpb.Empty{}, nil
  }
  ```

- 可以改变 service 中的 method 的行为。例如这个示例里设置了`unaryCallSleepTime` 时间为 1s。

#### 第三步：创建 client 且发起一个普通 RPC 调用

代码的第 4 行通过调用`*TestSuite` 的成员方法 `newTRPCClient` 创建了一个 trpc client, 该方法设置 client 的 Target，超时时间和不采用多路复用。

```go
// newTRPCClient creates a tRPC client connected to this service that the test may use.
// The newly created client will be available in the client field of TestSuite.
func (s *TestSuite) newTRPCClient(opts ...client.Option) testpb.TestTRPCClientProxy {
	log.Debugf("client dial to %s.", s.serverAddress())
	const defaultTimeout = 1 * time.Second
	return testpb.NewTestTRPCClientProxy(
		append(
			opts,
			client.WithTarget(s.serverAddress()),
			client.WithTimeout(defaultTimeout),
			client.WithMultiplexed(s.tRPCEnv.clientMultiplexed),
		)...,
	)
}
```

代码的第 5 行会发起一个普通 RPC `UnaryCall`, 请求参数 `s.defaultSimpleRequest` 为 `*testpb.SimpleRequest` 类型，可以从 pb 生成的桩代码包 `testpb "trpc.group/trpc-go/trpc-go/test/protocols"` 中自行构建请求参数。

### 第四步：根据返回结果验证

代码的第 6 行根据返回的错误码，使用 require 包验证错误码的的类型是否符合预期的 `errs.RetClientTimeout`。

## 参考

1. Winters, Titus, Tom Manshreck, and Hyrum Wright. *Software engineering at Google: Lessons learned from programming over time*. O'Reilly Media, 2020.
2. Mike Cohn, *Succeeding with Agile: Software Development Using Scrum* (New York: Addison-Wesley Professional, 2009)
3. xUnit 家族测试框架在 Java 和 Python 语言中广为流行，最初由极限编程倡导者 Kent Beck 和 Erich Gamma 建立的，见 Meszaros, Gerard. *xUnit test patterns: Refactoring test code*. Pearson Education, 2007.
4. JUnit: https://junit.org/junit5/docs/current/user-guide/
5. PyUnit: https://pyunit.sourceforge.net/pyunit.html
6. https://pkg.go.dev/testing#hdr-Subtests_and_Sub_benchmarks
7. https://pkg.go.dev/github.com/stretchr/testify/suite
8. Test fixture:Software https://en.wikipedia.org/wiki/Test_fixture
