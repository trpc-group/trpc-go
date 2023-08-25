# RPCZ

RPCZ is a tool for monitoring RPC, logging various events that occur in a single rpc, such as serialization/deserialization, compression/decompression and execution of interceptors.
It allows users to configure the events that need to be logged, and users can view the logged events through the admin tool, which can help them locate problems quickly and accurately.
In addition, since RPCZ records the duration of various events in RPC and the size of packets sent and received, it can help users analyze timeout events and optimize the performance of the service.

## Explanation of terms

### Event

Event (Event) [1, 2, 3] is used to describe that something (`Event.Name`) happened at a particular moment (`Event.Time`).

```go
type Event struct {
    Name string
    Time time.Time
}
```
In a normal RPC call, a series of events will occur, for example, the Client side of the request is sent in chronological order, and generally the following series of events will occur.

1. start running the pre-interceptor
2. finish running the pre-interceptor
3. start serialization
4. end serialization
5. start compression
6. end compression
7. start encoding protocol header fields
8. end encoding protocol header fields
9. start sending binaries to the network
10. end sending binary to network
11. start receiving binary files from the network
12. ends receiving binary files from the network
13. start decoding protocol header fields
14. end decoding protocol header fields
15. start decompression
16. end decompression
17. start deserialization
18. end deserialization
19. start running post-interceptor
20. finish running the post-interceptor

On the server side, where the request is processed, the following sequence of events typically occurs in chronological order.

1. start decoding the protocol header fields
2. finish decoding the protocol header fields
3. start decompression
4. end decompression
5. start deserialization
6. end deserialization
7. start running the pre-interceptor
8. finish running the pre-interceptor
9. start running user-defined handler
10. end running user-defined handler
11. start running post-interceptor
12. end running post-interceptor
13. start serialization
14. end serialization
15. start compression
16. end compression
17. start encoding protocol header fields
18. end decoding protocol header fields
19. start sending binary files to the network
20. end sending binary to network

### Span

Span[4, 5] is used to describe a single operation for a certain time interval (with a start time and an end time), such as a client sending a remote request and a server processing the request or a function call.
Depending on the size of the divided time interval, a large Span can contain multiple smaller Spans, just as multiple other functions may be called within a single function, creating a tree structured hierarchy.
Thus, a Span may contain many sub-Spans in addition to the name, the internal identifier span-id [6], the start time, the end time, and the set of events (Events) that occurred during this time.

There are two types of Span in rpcz: 

- client-Span: describes the actions of the client during the interval from the start of the request to the receipt of the reply (covering the series of events on the client side described in the previous section Event). 
- server-Span: describes the operation of the server from the time it starts receiving requests to the time it finishes sending replies (covers the series of events on the server side described in the previous section Event).
   When server-Span runs a user-defined processing function, it may create a client to call a downstream service, so server-Span will contain several sub-client-Span.

```
server-Span
    client-Span-1
    client-Span-2
    ......
    client-Span-n
```

Span is stored in context, rpcz will automatically call ContextWithSpan to store Span in context, you need to ensure that the Span in context will not be lost during the function call.

## Life cycle of Span

When examining the lifecycle of Span objects, most of the operations on Span in rpcz need to consider concurrency safety.
Pool and pre-allocated circular array are used to reduce the performance impact of memory allocation for Span.

### Span construction

rpcz initializes a global GlobalRPCZ at startup, which is used to generate and store Span.
There are only two possible locations where a Span can be constructed within the framework.
The first location is when the handle function of the transport layer on the server side first starts processing incoming requests.
The second location is when the Invoke function is called in the client-side stub code to start the rpc request.
Although the two locations create different types of Span, the code logic is similar, both will call `rpcz.NewSpanContext`, which actually performs following three operations successively: 

1. Call the SpanFromContext function to get the span from the context.
2. Call span.NewChild method to create a new child span. 
3. Call the ContextWithSpan function to set the newly created child span into context.

### Span passing in context

The created span is stored in context until it is committed, and is passed along the link of the rpc call.
Use `rpcz.AddEvent` to add a new event to the span in the current context on the call link.

### Span commits

After the request is processed by the handle function at the transport layer on the server side, `ender.End()` is called to commit the Span to GlobalRPCZ.
After that, the Span is still stored in the context, but semantically, the Span is not allowed to be manipulated again after the End function has been called, and its behavior is undefined.

### Accessing Span in admin

The admin module calls `rpcz.Query` and `rpcz.BatchQuery` to read the Span from GlobalRPCZ.
One thing to note is that the Span obtained by admin is a read-only Span (ReadOnlySpan), which is exported from a writable Span for the sake of concurrent access security.

### Delete redundant Span

When too many Spans are stored in the hash table, it is necessary to remove the redundant Spans according to some elimination rules.
The current implementation removes the oldest Span when the number of Spans in the GlobalRPCZ exceeds the maximum capacity limit.

## Origin of RPCZ name

Regarding the origin of the name "RPCZ", the suffix -z has two general meanings in English [7]: it is used in nouns to change the singular to plural, e.g. Boy**z** are always trouble; and it is used in verbs to change the verb form He love**z** me.
In summary, adding -z to a word has the same effect as adding -s.
So "RPCZ" refers to various types of RPCs, and this does hold true from a distributed global call-link perspective, where there is a tree-like parent-child relationship of various RPC calls that combine to form the "RPCZ".

The term "RPCZ" first came from Google's internal RPC framework Stubby, based on which Google implemented a similar function in the open source grpc channelz [8], which not only includes information about various channels, but also covers trace information.
After that, Baidu's open source brpc implemented a non-distributed trace tool based on the distributed trace system Dapper paper [9] published by google, imitating channelz named brpc-rpcz [10].
The next step is that users need a tool similar to brpc-rpcz for debugging and optimization in tRPC, so tRPC-Cpp first supports similar functionality [11, 12], still keeping the name RPCZ.

The last thing is to support similar functionality to "RPCZ" in tRPC-Go. During the implementation process, it was found that with the development of distributed tracing systems, open source systems of opentracing [13] and opentelemetry [14] emerged in the community, and the company also made tianji pavilion [15] internally.
tRPC-Go-RPCZ partially borrows the go language implementation of opentelemetry-trace for span and event design, and can be considered as a trace system inside the tRPC-Go framework.
Strictly speaking, tRPC-Go-RPCZ is non-distributed, because there is no communication between the different services at the protocol level.
Now it seems that brpc, tRPC-Cpp and the tRPC-Go implementation of rpcz, named spanz, might be more in line with the original meaning of the suffix "-z".


## How to configure rpcz

The configuration of rpcz includes basic configuration, advanced configuration and code configuration, see `config_test.go` for more configuration examples.

### Basic configuration

Configure admin on the server side, and configure rpcz inside admin:

```yaml
server:
  admin:
    ip: 127.0.0.1
    port: 9028
    rpcz:
      fraction: 1.0
      capacity: 10000
```

- `fraction`: the sampling rate, the range is `[0.0, 1.0]`, the default value is 0.0 which means no sampling, you need to configure it manually.
- `capacity`: the storage capacity of rpcz, the default value is 10000, which means the maximum number of spans can be stored.

### Advanced configuration

Advanced configuration allows you to filter the span of interest. Before using advanced configuration, you need to understand the sampling mechanism of rpcz.

#### Sampling mechanism

rpcz uses the sampling mechanism to control performance overhead and filter spans that are not of interest to you.
Sampling may occur at different stages of a Span's lifecycle, with the earliest sampling occurring before a Span is created and the latest sampling occurring before a Span is committed.

##### Sampling results table

Only Spans that are sampled before both creation and commit will eventually be collected in GlobalRPCZ for you to query through the admin interface.

| Sampled before Span creation? | Sampled before Span commit? | Will the Span eventually be collected? |
|:------------------------------|:---------------------------:|:--------------------------------------:|
| true                          |            true             |                  true                  | 
| true                          |            false            |                 false                  | 
| false                         |            true             |                 false                  | 
| false                         |            false            |                 false                  | 

##### Sampling before Span creation

Span is created only when it is sampled, otherwise it is not created, which avoids a series of subsequent operations on Span and thus reduces the performance overhead to a large extent.
The sampling policy with fixed sampling rate [16, 17] has only one configurable floating-point parameter `rpcz.fraction`, for example, `rpcz.fraction` is 0.0001, which means one request is sampled for every 10000 (1/0.0001) requests.
When `rpcz.fraction` is less than 0, it is fetched up by 0. When `rpcz.fraction` is greater than 1, it is fetched down by 1.

##### Sampling before Span commit

Spans that have been created will record all kinds of information in the rpc, but you may only care about spans that contain certain information, such as spans with rpc errors, spans that are highly time consuming, and spans that contain certain property information.
In this case, it is necessary to sample only the Span that you needs before the Span is finally committed.
rpcz provides a flexible external interface that allows you to set the `rpcz.record` field in the configuration file to customize the sampling logic before the service is started.

```yaml
server:
  admin:
    rpcz:
      record_when:
        error_codes: [0,]      
        min_duration: 1000ms # ms or s
        sampling_fraction: 1 # [0.0, 1.0]
```

- `error_codes`: Only sample spans containing any of these error codes, e.g. 0(RetOk), 21(RetServerTimeout).
- `min_duration`: Only sample spans that last longer than `min_duration`, which can be used for time-consuming analysis.
- `sampling_fraction`: The sampling rate, in the range of `[0, 1]`.

#### Example of configuration

##### commits span containing error code 1 (RetServerDecodeFail) or with duration greater than 1s

```yaml
server:
  admin:
    ip: 127.0.0.1
    port: 9028
    rpcz:
      fraction: 1.0
      capacity: 10000
      record_when:
        error_codes: 1       
        min_duration: 1000ms
        sampling_fraction: 1         
```

##### commits spans containing error codes of 1 (RetServerDecodeFail) or 21 (RetServerTimeout) or with a duration greater than 2s with a probability of 1/2

```yaml
server:
  admin:
    ip: 127.0.0.1
    port: 9028
    rpcz:
      fraction: 1.0
      capacity: 10000
      record_when:
        error_codes: [1, 21]     
        min_duration: 2s
        sampling_fraction: 0.5      
```

### Code configuration

After reading the configuration file and before the service starts, rpcz can be configured with `rpcz.GlobalRPCZ`, where the commit sampling logic needs to implement the `ShouldRecord` function.

```go
// ShouldRecord determines if the Span should be recorded.
type ShouldRecord = func(Span) bool
```

##### commits only for Span containing the "SpecialAttribute" attribute

```go
const attributeName = "SpecialAttribute"
rpcz.GlobalRPCZ = rpcz.NewRPCZ(&rpcz.Config{
    Fraction: 1.0,
    Capacity: 1000,
    ShouldRecord: func(s rpcz.Span) bool {
        _, ok = s.Attribute(attributeName)
        return ok
    },
})
```

### Query the summary information of the most recently submitted multiple span

To query the summary information of the last num span, you can access the following url:

```html
http://ip:port/cmds/rpcz/spans?num=xxx
```

For example, executing `curl http://ip:port/cmds/rpcz/spans?num=2` will return the summary information for 2 spans as follows.

```html
1:
  span: (client, 65744150616107367)
    time: (Dec 1 20:57:43.946627, Dec 1 20:57:43.946947)
    duration: (0, 319.792µs, 0)
    attributes: (RPCName, /trpc.testing.end2end.TestTRPC/EmptyCall), (Error, <nil>)
2:
  span: (server, 1844470940819923952)
    time: (Dec 1 20:57:43.946677, Dec 1 20:57:43.946912)
    duration: (0, 235.5µs, 0)
    attributes: (RequestSize, 125),(ResponseSize, 18),(RPCName, /trpc.testing.end2end.TestTRPC/EmptyCall),(Error, success)
```

The summary information for each span matches the following template.

```html
span: (name, id)
time: (startTime, endTime)
duration: (preDur, middleDur, postDur)
attributes: (name1, value1) (name2, value2)
```

The meaning of each of these fields is explained as follows.

- name: the name of the span
- id: the unique identifier of the span, which can be used to query the details of a specific span
- startTime: the creation time of the span
- endTime: the commit time of the span, when the span is not successfully committed, the value of this field is "unknown"
- duration: contains a time period to describe the duration of currentSpan and parentSpan
  - preDur: currentSpan.startTime - parentSpan.startTime
  - middleDur: currentSpan.endTime - currentSpan.startTime, when currentSpan.endTime is "unknown", the value of middleDur is also "unknown".
  - postDur: parentSpan.endTime - currentSpan.endTime, when parentSpan.endTime or currentSpan.endTime is "unknown", the value of postDur is also "unknown"
- attributes: attributes of the span, each attribute consists of (attribute name, attribute value), usually the following three attributes are displayed
  - RequestSize: request packet size (byte)
  - ResponseSize: response packet size (byte)
  - RPCName: the service name of the counterpart + interface name (/trpc.app.server.service/method)
  - Error: error message, according to the framework return code to determine whether the request is successful, success or nil means success

If you do not specify the number of queries, the following query will default to return a summary of the [^1] 10 most recently submitted successful spans.

```html
http://ip:port/cmds/rpcz/spans
```

[^1]: **The most recently committed span is not sorted strictly by time, there may be multiple goroutines submitting spans at the same time, and they are sorted by the most recently committed span.**

### Query the details of a span

To query the details of a span containing an id, you can access the following url.

```html
http://ip:port/cmds/rpcz/spans/{id}
```

For example, execute `curl http://ip:port/cmds/rpcz/spans/6673650005084645130` to query the details of a span with the span id 6673650005084645130.

```
span: (server, 6673650005084645130)
  time: (Dec  2 10:43:55.295935, Dec  2 10:43:55.399262)
  duration: (0, 103.326ms, 0)
  attributes: (RequestSize, 125),(ResponseSize, 18),(RPCName, /trpc.testing.end2end.TestTRPC/EmptyCall),(Error, success)
  span: (DecodeProtocolHead, 6673650005084645130)
    time: (Dec  2 10:43:55.295940, Dec  2 10:43:55.295952)
    duration: (4.375µs, 12.375µs, 103.30925ms)
  span: (Decompress, 6673650005084645130)
    time: (Dec  2 10:43:55.295981, Dec  2 10:43:55.295982)
    duration: (45.875µs, 791ns, 103.279334ms)
  span: (Unmarshal, 6673650005084645130)
    time: (Dec  2 10:43:55.295982, Dec  2 10:43:55.295983)
    duration: (47.041µs, 334ns, 103.278625ms)
  span: (filter1, 6673650005084645130)
    time: (Dec  2 10:43:55.296161, Dec  2 10:43:55.399249)
    duration: (225.708µs, 103.088ms, 12.292µs)
    event: (your annotation at pre-filter, Dec  2 10:43:55.296163)
    span: (filter2, 6673650005084645130)
      time: (Dec  2 10:43:55.296164, Dec  2 10:43:55.399249)
      duration: (2.75µs, 103.085ms, 250ns)
      event: (your annotation at pre-filter, Dec  2 10:43:55.296165)
      span: (server.WithFilter, 6673650005084645130)
        time: (Dec  2 10:43:55.296165, Dec  2 10:43:55.399249)
        duration: (1.208µs, 103.083625ms, 167ns)
        event: (your annotation at pre-filter, Dec  2 10:43:55.296165)
        span: (, 6673650005084645130)
          time: (Dec  2 10:43:55.296166, Dec  2 10:43:55.399249)
          duration: (792ns, 103.082583ms, 250ns)
          span: (HandleFunc, 6673650005084645130)
            time: (Dec  2 10:43:55.296177, Dec  2 10:43:55.399249)
            duration: (11.583µs, 103.070917ms, 83ns)
            event: (handling EmptyCallF, Dec  2 10:43:55.296179)
            span: (client, 6673650005084645130)
              time: (Dec  2 10:43:55.296187, Dec  2 10:43:55.297871)
              duration: (9.125µs, 1.684625ms, 101.377167ms)
              attributes: (RPCName, /trpc.testing.end2end.TestTRPC/UnaryCall),(Error, <nil>)
              span: (filter1, 6673650005084645130)
                time: (Dec  2 10:43:55.296192, Dec  2 10:43:55.297870)
                duration: (5.292µs, 1.678542ms, 791ns)
                span: (client.WithFilter, 6673650005084645130)
                  time: (Dec  2 10:43:55.296192, Dec  2 10:43:55.297870)
                  duration: (542ns, 1.677875ms, 125ns)
                  span: (selector, 6673650005084645130)
                    time: (Dec  2 10:43:55.296193, Dec  2 10:43:55.297870)
                    duration: (541ns, 1.677209ms, 125ns)
                    span: (CallFunc, 6673650005084645130)
                      time: (Dec  2 10:43:55.296200, Dec  2 10:43:55.297869)
                      duration: (7.459µs, 1.668541ms, 1.209µs)
                      attributes: (RequestSize, 405),(ResponseSize, 338)
                      span: (Marshal, 6673650005084645130)
                        time: (Dec  2 10:43:55.296202, Dec  2 10:43:55.296341)
                        duration: (1.375µs, 138.875µs, 1.528291ms)
                      span: (Compress, 6673650005084645130)
                        time: (Dec  2 10:43:55.296341, Dec  2 10:43:55.296341)
                        duration: (140.708µs, 333ns, 1.5275ms)
                      span: (EncodeProtocolHead, 6673650005084645130)
                        time: (Dec  2 10:43:55.296342, Dec  2 10:43:55.296345)
                        duration: (141.458µs, 3.333µs, 1.52375ms)
                      span: (SendMessage, 6673650005084645130)
                        time: (Dec  2 10:43:55.297540, Dec  2 10:43:55.297555)
                        duration: (1.339375ms, 15.708µs, 313.458µs)
                      span: (ReceiveMessage, 6673650005084645130)
                        time: (Dec  2 10:43:55.297556, Dec  2 10:43:55.297860)
                        duration: (1.355666ms, 303.75µs, 9.125µs)
                      span: (DecodeProtocolHead, 6673650005084645130)
                        time: (Dec  2 10:43:55.297862, Dec  2 10:43:55.297865)
                        duration: (1.661916ms, 2.5µs, 4.125µs)
                      span: (Decompress, 6673650005084645130)
                        time: (Dec  2 10:43:55.297866, Dec  2 10:43:55.297866)
                        duration: (1.665583ms, 167ns, 2.791µs)
                      span: (Unmarshal, 6673650005084645130)
                        time: (Dec  2 10:43:55.297866, Dec  2 10:43:55.297868)
                        duration: (1.666041ms, 1.709µs, 791ns)
            span: (sleep, 6673650005084645130)
              time: (Dec  2 10:43:55.297876, unknown)
              duration: (1.698709ms, unknown, unknown)
              event: (awake, Dec  2 10:43:55.398954)
            span: (client, 6673650005084645130)
              time: (Dec  2 10:43:55.398979, Dec  2 10:43:55.399244)
              duration: (102.80125ms, 265.417µs, 4.25µs)
              attributes: (RPCName, /trpc.testing.end2end.TestTRPC/UnaryCall),(Error, <nil>)
              span: (filter2, 6673650005084645130)
                time: (Dec  2 10:43:55.398986, Dec  2 10:43:55.399244)
                duration: (6.834µs, 258.25µs, 333ns)
                span: (client.WithFilter, 6673650005084645130)
                  time: (Dec  2 10:43:55.398987, Dec  2 10:43:55.399244)
                  duration: (1.708µs, 256.458µs, 84ns)
                  span: (selector, 6673650005084645130)
                    time: (Dec  2 10:43:55.398988, Dec  2 10:43:55.399244)
                    duration: (417ns, 255.916µs, 125ns)
                    span: (CallFunc, 6673650005084645130)
                      time: (Dec  2 10:43:55.399005, Dec  2 10:43:55.399243)
                      duration: (16.833µs, 238.375µs, 708ns)
                      attributes: (RequestSize, 405),(ResponseSize, 338)
                      span: (Marshal, 6673650005084645130)
                        time: (Dec  2 10:43:55.399006, Dec  2 10:43:55.399017)
                        duration: (1.792µs, 10.458µs, 226.125µs)
                      span: (Compress, 6673650005084645130)
                        time: (Dec  2 10:43:55.399017, Dec  2 10:43:55.399017)
                        duration: (12.583µs, 167ns, 225.625µs)
                      span: (EncodeProtocolHead, 6673650005084645130)
                        time: (Dec  2 10:43:55.399018, Dec  2 10:43:55.399023)
                        duration: (12.958µs, 4.917µs, 220.5µs)
                      span: (SendMessage, 6673650005084645130)
                        time: (Dec  2 10:43:55.399041, Dec  2 10:43:55.399070)
                        duration: (36.375µs, 29.083µs, 172.917µs)
                      span: (ReceiveMessage, 6673650005084645130)
                        time: (Dec  2 10:43:55.399070, Dec  2 10:43:55.399239)
                        duration: (65.75µs, 168.25µs, 4.375µs)
                      span: (DecodeProtocolHead, 6673650005084645130)
                        time: (Dec  2 10:43:55.399240, Dec  2 10:43:55.399241)
                        duration: (235.417µs, 1.375µs, 1.583µs)
                      span: (Decompress, 6673650005084645130)
                        time: (Dec  2 10:43:55.399242, Dec  2 10:43:55.399242)
                        duration: (237µs, 125ns, 1.25µs)
                      span: (Unmarshal, 6673650005084645130)
                        time: (Dec  2 10:43:55.399242, Dec  2 10:43:55.399243)
                        duration: (237.292µs, 750ns, 333ns)
        event: (your annotation at post-filter, Dec  2 10:43:55.399249)
      event: (your annotation at post-filter, Dec  2 10:43:55.399249)
    event: (your annotation at post-filter, Dec  2 10:43:55.399249)
  span: (Marshal, 6673650005084645130)
    time: (Dec  2 10:43:55.399250, Dec  2 10:43:55.399251)
    duration: (103.314625ms, 1.208µs, 10.167µs)
  span: (Compress, 6673650005084645130)
    time: (Dec  2 10:43:55.399252, Dec  2 10:43:55.399252)
    duration: (103.315958ms, 125ns, 9.917µs)
  span: (EncodeProtocolHead, 6673650005084645130)
    time: (Dec  2 10:43:55.399252, Dec  2 10:43:55.399253)
    duration: (103.316208ms, 750ns, 9.042µs)
  span: (SendMessage, 6673650005084645130)
    time: (Dec  2 10:43:55.399253, Dec  2 10:43:55.399261)
    duration: (103.317333ms, 8.333µs, 334ns)
```

A new `event` field has been added to the span details, along with an embedded subspan.

- event: describes what happened at a given moment, similar to a log.
  Events that can be inserted by you, such as `Nov 4 14:39:23.594147: your annotation at pre-filter` in the example above.
- span: While the server is processing your custom function, a new client may be created to call the downstream service, and a sub-span will be created
  As you can see, all subspans occur within `HandleFunc`.

Note that the values of middleDur and postDur in endTime, duration may be ``unknown'', for example, the above span contains the following subspan.

```
span: (sleep, 6673650005084645130)
time: (Dec 2 10:43:55.297876, unknown)
duration: (1.698709ms, unknown, unknown)
event: (awake, Dec 2 10:43:55.398954)
```

## Span Interface

You can call `rpcz.SpanFromContext`[^2] to get the current `Span` in the `context` and then use the following interface to manipulate Span.

```go
type Span interface {
	// AddEvent adds an event.
	AddEvent(name string)

	// SetAttribute sets Attribute with (name, value).
	SetAttribute(name string, value interface{})
	
	// ID returns SpanID.
	ID() SpanID

	// NewChild creates a child span from current span.
	// Ender ends this span if related operation is completed. 
	NewChild(name string) (Span, Ender)
}
```

[^2]: Return a `noopSpan` when the `context` does not contain any `span`, any operation on the `noopSpan` is null and will not take effect.

### Using AddEvent to add events

```go
// If no Span is currently set in ctx an implementation of a Span that performs no operations is returned.
span := SpanFromContext(ctx context.Context)

span.AddEvent("Acquiring lock")
mutex.Lock()

span.AddEvent("Got lock, doing work...")
// do some stuff ...

span.AddEvent("Unlocking")
mutex.Unlock()
```

### Use SetAttribute to set attributes

```go
ctx, msg := codec.EnsureMessage(ctx)
span := SpanFromContext(ctx context.Context)
span.SetAttribute("RPCName", msg.ClientRPCName())
span.SetAttribute("Error", msg.ClientRspErr())
```

### Create a new child span

**End() function should be called only once by the caller to end the life cycle of the child span; uncalled End and multiple calls to End are undefined**

```go
span := SpanFromContext(ctx context.Context)
cs, end := span.NewChild("Decompress")
reqBodyBuf, err := codec.Decompress(compressType, reqBodyBuf)
end.End()
```

## Reference

- [1] https://en.wikipedia.org/wiki/Event_(UML)
- [2] https://en.wikipedia.org/wiki/Event_(computing)
- [3] https://opentelemetry.io/docs/instrumentation/go/manual/#events
- [4] https://opentelemetry.io/docs/instrumentation/go/api/tracing/#starting-and-ending-a-span
- [5] https://opentelemetry.io/docs/concepts/observability-primer/#spans
- [6] span-id represented as an 8-byte array, satisfying the w3c trace-context specification. https://www.w3.org/TR/trace-context/#parent-id
- [7] https://en.wiktionary.org/wiki/-z#English
- [8] https://github.com/grpc/proposal/blob/master/A14-channelz.md
- [9] Dapper, a Large-Scale Distributed Systems Tracing Infrastructure: http://static.googleusercontent.com/media/research.google.com/en// pubs/archive/36356.pdf
- [10] brpc-rpcz: https://github.com/apache/incubator-brpc/blob/master/docs/cn/rpcz.md
- [11] tRPC-Cpp rpcz wiki. todo
- [12] tRPC-Cpp rpcz proposal. https://git.woa.com/trpc/trpc-proposal/blob/master/L17-cpp-rpcz.md
- [13] opentracing: https://opentracing.io/
- [14] opentelemetry: https://opentelemetry.io/
- [15] https://tpstelemetry.pages.woa.com/
- [16] open-telemetry 2.0-sdk-go: https://git.woa.com/opentelemetry/opentelemetry-go-ecosystem/blob/master/sdk/trace/dyeing_sampler.go
- [17] open-telemetry-sdk-go- traceIDRatioSampler: https://github.com/open-telemetry/opentelemetry-go/blob/main/sdk/trace/sampling.go