# tRPC-Go 插件管理体系

## 插件实现原理
- 框架只定义插件标准接口，并提供 map 注册方式，业务自己通过桥梁实现其他组件的接口并注册到框架中
  ![插件管理](TODO: 开源后，图片上传到 github，务必调整图片链接)

## tRPC-Go 插件主要有两种：
- 需要通过配置文件才能实例化的插件，如名字服务，远程日志等，包括 selector log registry discovery loadbalance circuitbreaker config filter transport
- 不需要配置文件的插件，如打解包协议，序列化方式等，包括 codec serializer compressor

## 针对不同的插件需要通过不同的注册方式来实现
- 需要配置文件才能实例化的插件，首先需要先注册插件工厂，插件工厂通过解析配置文件实例化插件并注册到具体的插件类型里面
    ```golang
        // Factory 插件工厂统一抽象 外部插件需要实现该接口，通过该工厂接口生成具体的插件并注册到具体的插件类型里面
        type Factory interface {
        	// Type 插件的类型 如 registry selector log config tracing，对应到 trpc_go.yaml 配置文件 plugins 下的第一层名字
        	Type() string
        	// Setup 根据配置项节点装载插件
        	// 用户自己先定义好具体插件的配置数据结构，通过 decoder 解析出来，开始实例化具体插件，对应到 trpc_go.yaml 配置文件 plugins 下的第二层名字
        	Setup(name string, configDec Decoder) error
        }
    ```
    ```golang
        plugin.Register("default", &log.LogFactory{})
    ```

       以初始化日志插件为例：
    ```golang
        // Setup 启动加载配置 并注册日志
        func (f *LogFactory) Setup(name string, configDec plugin.Decoder) error {
        
        	var conf Config // 日志插件自己定义的配置结构
        	err := configDec.Decode(&conf) // 解析配置
        	if err != nil {
        		return err
        	}
        
        	logger := NewZapLog(conf) // 通过配置实例化具体插件
        
        	Register(name, logger) // 注册到具体插件类型的 map 里面，同时支持不同的日志，用户通过 log.Get(name).Debug("xxx") 打日志
        
        	if name == "default" {
        		SetLogger(logger) // 设置默认日志，用户通过 log.Debug("xxx") 打日志
        	}
        
        	return nil
        }
    ```
- 不需要配置文件的插件，可直接注册到具体的插件类型里面
    ```golang
        type Codec interface {
            //server 解包 从完整的二进制网络数据包解析出二进制请求包体
            Decode(message Msg, request-buffer []byte) (reqbody []byte, err error)
            //server 回包 把二进制响应包体打包成一个完整的二进制网络数据
            Encode(message Msg, rspbody []byte) (response-buffer []byte, err error)
        }
    ```
    ```golang
        codec.Register("trpc", trpc.DefaultServerCodec, trpc.DefaultClientCodec)
    ```