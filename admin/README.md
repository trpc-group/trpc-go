# tRPC-Go Management command module function and implementation [中文主页](README_CN.md)
- The Server interface defines the basic functions of starting, closing, routing settings and querying.
- The processing method of adminConfig can be passed in to configure the admin service, such as setting the version number, enabling/disabling TLS, setting the listening port, etc.
- admin default listen port 9028, The TLS connection is not enabled, and the read and write timeout is 3s. Can be customized as above.

# admin partial interface analysis
- `Run`: start up adminServer，Listen to the specified port, use the incoming parameters to adjust the configuration, listen to the port, and receive external requests.
- `Close`: shut down adminServer
- `HandleFunc` Register the route handler function, cannot be overridden

### 
* TLS support
*
