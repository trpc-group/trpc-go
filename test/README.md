English | [中文](README.zh_CN.md)

# How to add integration test cases in tRPC-Go

## When to add integration test cases

Test cases can be divided into two dimensions: Test Size and Test Scope.
Test Size is divided into small, medium, and large tests; Test Scope is divided into unit tests, integration tests, and system tests.
The larger the dimension of the test case, the more CPU, IO, network, memory, and other resources are required during runtime, the slower the running speed, and the less reliable the obtained running results [1].
For a system, there is a rough guiding principle of the test pyramid [2] in software engineering practice, where the lower the dimension of the test case, the higher the proportion.

```text
        .                 System Test 5%
       ...                Integration Test 15%
................          Unit Test 80%
```

**Therefore, it is also recommended to write small-dimension small tests and unit tests in tRPC-Go.** 
The following scenarios may require adding integration test cases:

- Scenarios requiring multiple processes, such as testing the graceful restart of the server, can refer to the implementation in `graceful_restart_test.go`.
- Scenarios where a client needs to access services provided by a real server [^1] to verify functionality. 
Depending on the type of service provided by the server, you can refer to `trpc_test.go`, `http_test.go`, `restful_test.go`, and `streaming_test.go`.

[^1] The integration test network calls in tRPC-Go only use the localhost network, and the entire test can only run on a single machine.

## Test code organization

### Flat

Go language has a built-in `go test` command, which runs functions with the `TestXXX` naming rule in `_test.go` files, thus executing test code. 
Since go test does not impose any constraints on the organization of test code, many existing test codes use a very simple "flat" organization.
The advantage of this "flat" code organization is that each test function is unrelated, there is no additional abstraction in the code structure, and it is very easy to get started.
However, "flat" is not suitable for tRPC-Go's integration testing because tRPC-Go's integration testing has the following characteristics:

- There are many test cases for various user usage scenarios. "Flat" has no hierarchy, so the code will appear chaotic.
- Multiple test cases may be conceptually related, such as testing tRPC-Go data penetration. These cases should be organized together.
- Almost every test case needs to be executed in a specific environment with a server, and resources (mainly closing the started server) need to be cleaned up after the case is executed. This common logic should be extracted and shared among all cases.

### xUnit's family pattern

To address characteristics 1 and 2 of tRPC-Go's integration testing, the solution is to make the test code organization hierarchical and organize conceptually related cases together.
We adopted the typical three-layer test code organization of the xUnit family [3,4,5]:

```text
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

This code organization has three levels: a Test Project consists of several Test Suites, and each Test Suite contains multiple Test Cases.

Before Go 1.7, it was impossible to organize test code in the above form using Go's native tools and standard library.
The support for subtest[6] added in Go 1.7 allows this kind of organization in Go as well.
However, the widely popular testify in the community is a simple wrapper around Go's official testing package, which is convenient and easy to use.
Its suite[7] package provides similar capabilities, so tRPC-Go's integration testing mainly uses testify/suite to achieve the above code organization.

### Test fixtures

To address characteristic 3 of tRPC-Go's integration testing, the solution is to use the SetUp and TearDown functions in testify/suite to create/set and teardown/destroy test fixtures[8].

> A test fixture is an artificial, deterministic environment where a test case or a test suite (a group of test cases under it) is tested, and its test result is repeatable (the results of multiple test runs are the same).

Test fixtures are commonly found in the following scenarios:

- Create the necessary resources to start a server, start a server, close a server after the test is over, and clean up related resources
- Load a specific set of known data into the database, and clear the data after the test is over
- Copy a specific set of known files, and clear the files after the test is over

Of course, test fixtures can have different levels, corresponding to Test Project, Test Suite, and Test Case.
Therefore, the general execution flow of integration testing in tRPC-Go is as follows:

```text
Integration Test Package
   Test Start
       Test Project level fixture creation
           Test Suite level fixture creation (start server, etc.)
               Test Case level fixture creation
                   Run test (usually requires client to send request to interact with server, and then verify based on the returned result)
               Test Case level fixture teardown
           Test Suite level fixture teardown (close server, etc.)
       Test Project level fixture teardown
   Test End
```

### Code Example

Let's take the admin module in the tRPC-Go framework code as an example to analyze the code organization and execution flow adopted by tRPC-Go integration test code.

We analyze the code organization of the integration test by running the `TestAdmin` function in `admin_go.test`:

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

From the output of `go test`, we can see that it is more hierarchical than "flat", and we can easily see which functions/methods have been tested, the corresponding TestSuite, and each TestCase in the TestSuite.

```bash
TestRunSuite  <--- Corresponds to Test Project
    TestAdmin  <--- Corresponds to Test Suite1
       testCmds              ｜  <---- Corresponds to Test Case1
       testCmdsConfig        ｜  <---- Corresponds to Test Case2
       testCmdsLogLevel      ｜  <---- Corresponds to Test Case3
       testCustomHandleFunc  ｜  <---- Corresponds to Test Case4
       testIsHealthy         ｜  <---- Corresponds to Test Case5
```

We analyze the test execution flow by looking at the code logic of `TestRunSuite`, `TestAdmin`, and `testXXX(testCmds, testCmdsConfig, ...)` in the order of function execution.

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

In the first step, `TestRunSuite` creates a `TestSuite` instance and runs the member methods of `TestSuite` that conform to the `TestXXX` naming rule, such as `TestAdmin` here. You can think of the `TestSuite` class as managing all Test Suites in tRPC-Go integration testing, with `TestAdmin` being one of them [^2].

In the second step, before executing `TestAdmin`, the `SetupSuite()` and `SetupTest()` of `TestSuite` will be executed in sequence to create Test Project-level and Test Suite-level fixtures.

In the third step, the `testCmds`, `testCmdsConfig`, `testCmdsLogLevel`, `testCustomHandleFunc`, and `testIsHealthy` Test Cases in `TestAdmin` will be executed using the `s.Run` method. Here, Test Case-level fixtures can be created at the beginning of each function and destroyed using the defer function method.

In the fourth step, actual testing will be performed in each Test Case, such as sending an HTTP Get request to the admin server in `testCmds` and verifying based on the returned result.

In the fifth step, after each `TestAdmin` is finished, the `TearDownTest` of `TestSuite` will be run to destroy Test Suite-level fixtures, and the `TearDownSuite` will be run after all Test Suites are finished to destroy Test Project-level fixtures.

[^2]: In tRPC-Go, the semantics of the `TestSuite` structure and the suite in testify/suite are slightly different, with the latter's suite referring to a Test Suite.


### Adding a new integration test case demonstration

Generally, in tRPC-Go integration testing, adding a test case usually requires starting a server first, then creating a client to access the service provided by the server, and finally verifying based on the returned result.
The following example demonstrates testing the tRPC protocol's one-to-one timeout:

```go
1 func (s *TestSuite) TestClientTimeoutAtUnaryCall() {
2 	s.startServer(&TRPCService{unaryCallSleepTime: time.Second})
3 
4 	c := s.newTRPCClient()
5 	_, err := c.UnaryCall(trpc.BackgroundContext(), s.defaultSimpleRequest, client.WithTimeout(100*time.Millisecond))
6 	require.Equal(s.T(), errs.RetClientTimeout, errs.Code(err))
7 }
```

#### Step 1: Organize test cases according to the tRPC-Go test code organization

If a Test Suite contains only one Test Case, the position of the Test Case can be raised.
Based on the functionality to be tested, choose a function name that conforms to the TestXXX syntax and is easy to understand. 
In line 1 of the code, a member method of `*TestSuite` named `TestClientTimeoutAtUnaryCall` is defined.

#### Step 2: Start a server providing trpc service

In line 2 of the code, the `startServer` member method of `*TestSuite` is called to start a trpc service, and the sleep time in the one-to-one service is set to 1s. 
tRPC-Go integration testing can provide 3 levels of flexibility for starting services:

- Different types of services can be started. 
The `service_imp.go` file implements various types of services defined in the protocols directory's `test.proto`, including `TRPCService`, `StreamingService`, `testHTTPService`, and `testRESTfulService`, covering trpc, streaming, http, and restful protocols.
- The processing logic of methods in the service can be customized. 
For example, when constructing the `TRPCService`, you can fill in the`EmptyCallF` field, and when the client initiates the `EmptyCall` call, the custom processing logic will be executed.

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
- The behavior of methods in the service can be changed.
For example, in this example, the unaryCallSleepTime is set to 1s.

#### Step 3: Create a client and initiate a regular RPC call

In line 4 of the code, a trpc client is created by calling the `newTRPCClient` member method of `*TestSuite`.
This method sets the client's Target, timeout, and disables multiplexing.

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

In line 5 of the code, a regular RPC `UnaryCall` is initiated.
The request parameter `s.defaultSimpleRequest` is of type `*testpb.SimpleRequest`, which can be constructed from the stub code package `testpb "trpc.group/trpc-go/trpc-go/test/protocols"`.

#### Step 4: Verify based on the returned result

In line 6 of the code, the error code is verified using the `require` package to check if the error code type matches the expected `errs.RetClientTimeout`.

## References

1. Winters, Titus, Tom Manshreck, and Hyrum Wright. Software engineering at Google: Lessons learned from programming over time. O'Reilly Media, 2020.
2. Mike Cohn, Succeeding with Agile: Software Development Using Scrum (New York: Addison-Wesley Professional, 2009)
3. xUnit's family testing frameworks are widely popular in Java and Python languages, initially established by extreme programming advocates Kent Beck and Erich Gamma, see Meszaros, Gerard. 
xUnit test patterns: Refactoring test code. Pearson Education, 2007.
4. JUnit: https://junit.org/junit5/docs/current/user-guide/
5. PyUnit: https://pyunit.sourceforge.net/pyunit.html
6. https://pkg.go.dev/testing#hdr-Subtests_and_Sub_benchmarks
7. https://pkg.go.dev/github.com/stretchr/testify/suite
8. Test fixture:Software https://en.wikipedia.org/wiki/Test_fixture