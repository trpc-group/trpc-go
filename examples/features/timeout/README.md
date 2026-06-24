# Timeout

The following are some brief introductions and usage examples of trpc-go timeout feature. You can understand how the timeout mechanism of trpc-go works from these examples.

## Usage

Steps to use the feature. Typically:

* start the server

```shell
cd server && go build -v && ./server
```

* open another terminal, and start the forward server


```shell
cd forwardserver && go build -v && ./forwardserver
```


* open another terminal, start the client


```shell
cd client &&  go build -v && ./client
```

The above example shows the "ForwardServer Full-Link Timeout".
For more timeout scenarios, see the table in the next section.
You can modify the timeout configuration to simulate other timeout scenarios.


## Timeout Scenarios Explanation

### Call Chain
```
Client -> ForwardServer -> Server
```


| Timeout Scenario | Client Timeout | ForwardServer Timeout | ForwardServer Sleep | ForwardServer→Server Timeout | Server Timeout | Server Sleep | ForwardServer Error | Client Error |
|:----------------|:--------------:|:--------------------:|:------------------:|:---------------------------:|:--------------:|:------------:|:-------------------|:-------------|
| No Timeout | 4 | 5 | 1 | 3 | 2 | 1 | nil | nil |
| Silent Server Timeout | 4 | 5 | 1 | 3 | 1 | 2 | nil | nil |
| ForwardServer Normal Timeout | 4 | 2 | 3 | 3 | 2 | 1 | RetClientFullLinkTimeout | RetServerTimeout |
| Client Full-Link Timeout | 3 | 5 | 4 | 3 | 2 | 1 | RetClientFullLinkTimeout | RetClientFullLinkTimeout |
| ForwardServer→Server Client Timeout | 4 | 5 | 1 | 1 | 3 | 2 | RetClientTimeout | RetClientTimeout |
| ForwardServer Full-Link Timeout | 4 | 5 | 6 | 3 | 2 | 1 | RetClientTimeout | RetClientFullLinkTimeout |

1. **Normal Case (No Timeout)**
  - All services complete within their timeout limits
  - All timeouts: Client(4s) -> ForwardServer(5s) -> Server(2s)
  - No errors returned

2. **Silent Server Timeout**
  - Server sleeps(2s) longer than its timeout(1s)
  - But since Server doesn't actively handle timeout, no error is propagated
  - Both Client and ForwardServer remain unaware of the timeout

3. **Client Full-Link Timeout**
  - Client timeout(3s) < ForwardServer processing time(4s)
  - Results in full-link timeout propagation
  - Both services receive RetClientFullLinkTimeout

4. **ForwardServer->Server Client Timeout**
  - ForwardServer->Server timeout(1s) < Server processing time(2s)
  - Results in simple client timeout
  - Both receive RetClientTimeout

5. **ForwardServer Normal Timeout**
  - ForwardServer timeout(2s) < processing time(3s)
  - Client receives RetServerTimeout
  - ForwardServer receives RetClientFullLinkTimeout

6. **ForwardServer Full-Link Timeout**
  - ForwardServer processing time(6s) exceeds all timeouts
  - ForwardServer detects timeout but doesn't send response as Client already abandoned request
  - ForwardServer reports RetServerFullLinkTimeout to monitoring system
  - Results in RetClientTimeout and RetClientFullLinkTimeout


### Note
All times are in seconds, and errors indicate where in the chain the timeout occurred and how it propagated through the system.

## timeout mechanism in trpc-go

The timeout mechanism of trpc-go is as follows:

```raw
                                                +------------------+-----------------------+
                                                | server B         |        single timeout |
                                                |                  |        +------------> |
                                                |                  |                       |
                                                |                  |        +--------------+                        +--------------+
                                                |                  |        |              | first remaining time   |              |
                                                |                  |        | call server C+----------------------> | server C     |
                                                |                  |        |              |                        |              |
                                                |                  |        +--------------+                        +--------------+
                                                |           The    |                       |
                                                |           overall|                       |
+--------------+                                |           timeout|                       |
|              | total timeout of the entire link           of     |        +--------------+                        +--------------+
|  client A    | ---------------------------->  |           the    |        |              | second remaining time  |              |
|              |                                |           current|        | call server D+----------------------> | server D     |
+--------------+                                |           request|        |              |                        |              |
                                                |                  |        +--------------+                        +--------------+
                                                |                  |                       |
                                                |                  |                       |
                                                |                  |                       |
                                                |                  |        +--------------+                        +--------------+
                                                |                  |        |              |  third remaining time  |              |
                                                |                  |        | call server E+----------------------> | server E     |
                                                |                  |        |              |                        |              |
                                                |                  |        +--------------+                        +--------------+
                                                |                  |                       |
                                                +------------------v-----------------------+

```

* Client configuration

  * The total timeout time of the downstream link

    When the client initiates a request, it needs to specify the timeout period reserved for the downstream in the business agreement. After the timeout period is exceeded, the request will be canceled to avoid invalid waiting.

    The total timeout time of the downstream link is configured as follows, timeout: 1000 means that the maximum processing time of all backend requests invoked by the client is 1000ms

    ```yaml
    client:                                            # Backend configuration for client calls.
      timeout: 1000                                    # The total timeout time of the downstream link, the longest request processing time for all backends.
      namespace: development                           # Environments for all backends.
      service:                                         # Configuration for a single backend.
        - name: trpc.test.helloworld.Greeter           # service name of the backend service.
          namespace: development                       # The environment of the backend service.
          network: tcp                                 # The network type of the backend service is tcp udp configuration priority.
          protocol: trpc                               # Application layer protocol: trpc http.
          timeout: 800                                 # Maximum request processing time.
    ```

  * Single service timeout

    The client may request multiple backend services at the same time. You can set the timeout period of the client call for each backend service separately. For example, the timeout: 800 configured under service above means that the timeout period for a single backend service is 800ms

* server configuration

    A server can provide one or more service services, and supports setting the timeout period for each service. As follows, timeout: 1000 means that the server processing time of trpc.test.helloworld.Greeter service is up to 1000ms, and if it exceeds 1000ms, it will return a timeout.

   ```yaml
    server:                                            # server configuration.
      app: test                                        # Business application name.
      server: Greeter                                  # process service name.
      service:                                         # The service provided by the business service can have multiple.
        - name: trpc.test.helloworld.Greeter           # service route name.
          ip: 127.0.0.1                                # The service listens to the ip address. You can use the placeholder ${ip}, choose one of ip and nic, and give priority to ip.
          port: 8000                                   # Service listening port can use placeholder ${port}.
          network: tcp                                 # Network monitoring type tcp udp.
          protocol: trpc                               # Application layer protocol trpc http.
          timeout: 1000                                # Request maximum processing time unit milliseconds.
          idletime: 300000                             # Connection idle time unit milliseconds.
    ```

* specified in the code

    It supports setting the timeout period in the code. In this example, the client timeout period is set to 1000ms through the `client.WithTimeout(time.Millisecond * 1000)` method.

    It is worth noting that the priority of code specification > the configuration file, set the timeout in the configuration file and the code at the same time, and finally adopt the configuration of the code specification, that is, the configuration takes precedence.
