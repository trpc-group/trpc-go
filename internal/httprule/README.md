# About HttpRule

To support RESTful API, the difficulty is how to map fields in Proto Message to HTTP request/response. This mapping does not exist natively.

Therefore, we need to define a specification to specify the mapping details. This specification is called ***HttpRule***:

Inside pb file, we use **trpc.api.http** option to specify HttpRule. During the mapping, the leaf fields in the message is handled by the following **three** scenarios:

1. Field is referenced by url path in HttpRule: If url path in HttpRule uses one or more fields in rpc request message, these message will be passed through url path. But these fields must be native data types. Array type and message type are not supported.

2. Field is referenced by the body of HttpRule: If the body of HttpRule specifies the mapped field, then these fields in rpc request message are passed through HTTP request body.

3. Other fields: Other fields will be generated as URL request parameters automatically. If the field is repeatable, the same URL request parameter is permitted to query multiple times.

**Other things to notice**:

1. If "*" is used in HttpRule's body without specifying exact fields, then every request message field that has not been bind to URL path can be passed through the HTTP request body.

2. If the body of HttpRule is empty, then every request message field that has not been bind to URL path will become URL request parameter automatically.

## About httprule package

Apparently, the difficulty of HttpRule is the matching of URL path.

First of all, the URL path of RESTful request should be unified into a template:

```Go
 Template = "/" Segments [ Verb ] ;
 Segments = Segment { "/" Segment } ;
 Segment  = "*" | "**" | LITERAL | Variable ;
 Variable = "{" FieldPath [ "=" Segments ] "}" ;
 FieldPath = IDENT { "." IDENT } ;
 Verb     = ":" LITERAL ;
```

Every URL path in the HttpRule must follow this template.

Package ```httprule``` provides ```Parse``` method to parse URL path in HttpRule into the type ```PathTemplate```.

```PathTemplate``` provides ```Match``` method to match the variable value from the authentic HTTP request URL path.

***Example 1:***

   If the HttpRule URL path specified in pb Option **trpc.api.http** is ```/foobar/{foo}/bar/{baz}```, where ```foo``` and ```baz``` are variables, then it is parsed into the template:

   ```Go
      tpl, _ := httprule.Parse("/foobar/{foo}/bar/{baz}")
   ```

   the template is used to match the URL path from an HTTP request:

   ```Go
      captured, _ := tpl.Match("/foobar/x/bar/y")
   ```

   captured is:

   ```Go
      reflect.DeepEqual(captured, map[string]string{"foo":"x", "baz":"y"})
   ```

***Example 2:***

   If the HttpRule URL path specified in pb Option **trpc.api.http** is ```/foobar/{foo=x/*}```, where ```foo``` is a variable, then it is parsed into the template:

   ```Go
      tpl, _ := httprule.Parse("/foobar/{foo=x/*}")
   ```

   the template is used to match the URL path from an HTTP request:

   ```Go
      captured, _ := tpl.Match("/foobar/x/y")
   ```

   captured is:

   ```Go
      reflect.DeepEqual(captured, map[string]string{"foo":"x/y"})
   ```
