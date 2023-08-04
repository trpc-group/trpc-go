# Timeout
The following are some brief introductions and usage examples of trpc-go timeout feature. You can understand how the timeout mechanism of trpc-go works from these examples.
## Usage
Steps to use the feature. Typically:
* start the server
```
cd server && go build -v && ./server
```

*  start the client

  open another terminal.
```
cd client &&  go build -v && ./client
```
In the demo, there are two RPC calls, SayHello and SayHi.
You have set different client timeout values for the TestSayHello and TestSayHi interfaces. The client timeout value for TestSayHi is 1000ms.

```go
opts := []client.Option{
		client.WithTarget(addr),
		client.WithTimeout(time.Millisecond * 1000),
}
````

The TestSayHello interface will call the SayHi RPC. You have set the timeout value for this call to 2000ms.
```go
opts := []client.Option{
		client.WithTarget(addr),
		client.WithTimeout(time.Millisecond * 2000),
}
```

In the SayHi method of the server, you have set a sleep time of 1100ms for the thread.
```
time.Sleep(time.Millisecond * 1100ms)
```

When executing `./client`, you found that the TestSayHi interface timed out, while the TestSayHello interface returned normally.


## timeout mechanism in trpc-go 

The timeout mechanism of trpc-go is as follows:

```
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


- Client configuration

    - The total timeout time of the downstream link

    When the client initiates a request, it needs to specify the timeout period reserved for the downstream in the business agreement. After the timeout period is exceeded, the request will be canceled to avoid invalid waiting.
        
    The total timeout time of the downstream link is configured as follows, timeout: 1000 means that the maximum processing time of all backend requests invoked by the client is 1000ms
    
    ```
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

    - Single service timeout
    
    The client may request multiple backend services at the same time. You can set the timeout period of the client call for each backend service separately. For example, the timeout: 800 configured under service above means that the timeout period for a single backend service is 800ms
    
- server configuration

    A server can provide one or more service services, and supports setting the timeout period for each service. As follows, timeout: 1000 means that the server processing time of trpc.test.helloworld.Greeter service is up to 1000ms, and if it exceeds 1000ms, it will return a timeout.
   
   ```
    server:                                            # server configuration.
      app: test                                        # Business application name.
      server: Greeter                                  # process service name.
      bin_path: /usr/local/trpc/bin/                   # The path where the binary executable and framework configuration files are located.
      conf_path: /usr/local/trpc/conf/                 # The path where the business configuration file is located.
      data_path: /usr/local/trpc/data/                 # The path where the business data file is located.
      service:                                         # The service provided by the business service can have multiple.
        - name: trpc.test.helloworld.Greeter           # service route name.
          ip: 127.0.0.1                                # The service listens to the ip address. You can use the placeholder ${ip}, choose one of ip and nic, and give priority to ip.
          port: 8000                                   # Service listening port can use placeholder ${port}.
          network: tcp                                 # Network monitoring type tcp udp.
          protocol: trpc                               # Application layer protocol trpc http.
          timeout: 1000                                # Request maximum processing time unit milliseconds.
          idletime: 300000                             # Connection idle time unit milliseconds.
    ```
    
- specified in the code

    It supports setting the timeout period in the code. In this example, the client timeout period is set to 1000ms through the `client.WithTimeout(time.Millisecond * 1000)` method.

    It is worth noting that the priority of code specification > the configuration file, set the timeout in the configuration file and the code at the same time, and finally adopt the configuration of the code specification, that is, the configuration takes precedence.




