# tRPC-GO RESTful

---

## 转码器原理

和 tRPC-Go 框架其他协议插件不同的是，RESTful 协议插件在 Transport 层就基于 tRPC HttpRule 实现了一个 tRPC 和 HTTP/JSON 的转码器，这样就不再需要走 Codec 编解码的流程，转码完成得到 PB 后直接到 trpc 工具为其专门生成的 REST Stub 中进行处理

## 转码器核心：HttpRule

关于 HttpRule 的详细说明，请查看 trpc-go/internal/httprule 包

下面的几个例子，能直观地展示 HttpRule 到底要怎么使用：

**一、将 URL Path 里面匹配 messages/* 的内容作为 name 字段值：**

```protobuf
service Messaging {
  rpc GetMessage(GetMessageRequest) returns (Message) {
    option (trpc.api.http) = {
        get: "/v1/{name=messages/*}"
    };
  }
}

message GetMessageRequest {
  string name = 1; // Mapped to URL path.
}

message Message {
  string text = 1; // The resource content.
}
```

上述 HttpRule 可得以下映射：

 | HTTP | tRPC |
 | ----- | ----- |
 | GET /v1/messages/123456 | GetMessage(name: "messages/123456") |
 
**二、较为复杂的嵌套 message 构造，URL Path 里的 123456 作为 message_id，sub.subfield 的值作为嵌套 message 里的 subfield：**
 
```protobuf
service Messaging {
  rpc GetMessage(GetMessageRequest) returns (Message) {
    option (trpc.api.http) = {
        get:"/v1/messages/{message_id}"
    };
  }
}

message GetMessageRequest {
  message SubMessage {
    string subfield = 1;
  }
  string message_id = 1; // Mapped to URL path.
  int64 revision = 2;    // Mapped to URL query parameter `revision`.
  SubMessage sub = 3;    // Mapped to URL query parameter `sub.subfield`.
}
```

上述 HttpRule 可得以下映射：

 | HTTP | tRPC |
 | ----- | ----- |
 | GET /v1/messages/123456?revision=2&sub.subfield=foo | GetMessage(message_id: "123456" revision: 2 sub: SubMessage(subfield: "foo")) |
 
**三、将 HTTP Body 的整体作为 Message 类型解析，即将 "Hi!" 作为 message.text 的值：**
 
```protobuf
service Messaging {
  rpc UpdateMessage(UpdateMessageRequest) returns (Message) {
    option (trpc.api.http) = {
      post: "/v1/messages/{message_id}"
      body: "message"
    };
  }
}

message UpdateMessageRequest {
  string message_id = 1; // mapped to the URL
  Message message = 2;   // mapped to the body
}
```

上述 HttpRule 可得以下映射：

 | HTTP | tRPC |
 | ----- | ----- |
 | POST /v1/messages/123456 { "text": "Hi!" } | UpdateMessage(message_id: "123456" message { text: "Hi!" }) |
 
**四、将 HTTP Body 里的字段解析为 Message 的 text 字段：**
 
```protobuf
service Messaging {
  rpc UpdateMessage(Message) returns (Message) {
    option (trpc.api.http) = {
      post: "/v1/messages/{message_id}"
      body: "*"
    };
  }
}

message Message {
  string message_id = 1;
  string text = 2;
}
```

上述 HttpRule 可得以下映射：

 | HTTP | tRPC |
 | ----- | ----- |
 | POST/v1/messages/123456 { "text": "Hi!" } | UpdateMessage(message_id: "123456" text: "Hi!") |
 
**五、使用 additional_bindings 表示追加绑定的 API：**
 
```protobuf
service Messaging {
  rpc GetMessage(GetMessageRequest) returns (Message) {
    option (trpc
            .api.http) = {
      get: "/v1/messages/{message_id}"
      additional_bindings {
        get: "/v1/users/{user_id}/messages/{message_id}"
      }
    };
  }
}

message GetMessageRequest {
  string message_id = 1;
  string user_id = 2;
}
```

上述 HttpRule 可得以下映射：

 | HTTP | tRPC |
 | ----- | ----- |
 | GET /v1/messages/123456 | GetMessage(message_id: "123456") |
 | GET /v1/users/me/messages/123456 | GetMessage(user_id: "me" message_id: "123456") |
