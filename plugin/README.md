# tRPC-Go Plugin Management

## How it works
- tRPC-Go framework only defines a general plugin interface and provides a map for plugin registration. A plugin of any component can be registered by implementing a "bridge" which is as belows.
![plugin management](TODO: 开源后，图片上传到 github，务必调整图片链接)

## Two kinds of tRPC-Go plugins
- One kind of plugins must be loaded from a config file i.e. selector, log, registry, discovery, loadbalance, circuitbreaker, config, filter, transport.
- The other kind of plugins don't rely on any config file i.e. codec, serializer, compressor.

## Different kinds of plugins, different ways of registration
- To register plugins that rely on the config file, register the plugin factory first. The plugin factory will then parse the config file, instantiate the plugin and register the plugin as certain type.
    ```golang
        // Factory is the interface for plugin factory abstraction.
        // Custom Plugins need to implement this interface to be registered as a plugin with certain type.
        type Factory interface {
            // Type returns type of the plugin, i.e. selector, log, config, tracing.
            Type() string
            // Setup loads plugin by configuration.
            // The data structure of the configuration of the plugin needs to be defined in advance。
            Setup(name string, dec Decoder) error
        }
    ```
    ```golang
        plugin.Register("default", &log.LogFactory{})
    ```

       Take log plugin initialization as example:
    ```golang
        // Setup loads configuration and register log plugin.
        func (f *LogFactory) Setup(name string, configDec plugin.Decoder) error {
        
            var conf Config  // configuration defined by log plugin itself
            err := configDec.Decode(&conf) // parse the configuration
            if err != nil {
                return err
            }
        
            logger := NewZapLog(conf) // instantiate the plugin
        
            Register(name, logger) // register the logger with name, use this logger can be like log.Get(name).Debug("xxx")
        
            if name == "default" {
                SetLogger(logger) // set this logger as default logger, calling log.Debug("xxx") will use this logger
            }
        
            return nil
        }
    ```

- Registering plugins that don't rely on any config file needs to call the plugins' own registration method.
    ```golang
        // Codec defines the interface of business communication protocol,
        // which contains head and body. It only parses the body in binary,
        // and then the business body struct will be handled by serializer.
        // In common, the body's protocol is pb, json, jce, etc. Specially,
        // we can register our own serializer to handle other body type.
        type Codec interface {
            // Encode pack the body into binary buffer.
            // client: Encode(msg, reqbody)(request-buffer, err)
            // server: Encode(msg, rspbody)(response-buffer, err)
            Encode(message Msg, body []byte) (buffer []byte, err error)

            // Decode unpack the body from binary buffer
            // server: Decode(msg, request-buffer)(reqbody, err)
            // client: Decode(msg, response-buffer)(rspbody, err)
            Decode(message Msg, buffer []byte) (body []byte, err error)
        }
    ```
    ```golang
        codec.Register("trpc", trpc.DefaultServerCodec, trpc.DefaultClientCodec)
    ```
