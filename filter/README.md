# Filter

To intercept requests, the framework provides four positions: before/after server handler and before/after client `call`
function.

## How to Create a Filter

### Define Process Logics Function

```golang
func ServerFilter() filter.Filter {
    return func(ctx context.Context, req interface{}, next filter.HandleFunc) (interface{}, error) {
		
		// preposition logics
		
        err = next(ctx, req, rsp)
        
		// postposition logics
        
        return rsp, err // return err to upper layer
    }
}
```
```golang
func ClientFilter() filter.Filter {
    return func(ctx context.Context, req, rsp interface{}, next filter.HandleFunc) error {
        
        // preposition logics
        
        err = next(ctx, req, rsp)
        
        // postposition logics
        
        return err // return err to upper layer
    }
}
```

### Register to framework
```golang
serverFilter := ServerFilter()
clientFilter := ClientFilter()

filter.Register("name", serverFilter, clientFilter)
```

### enable in configuration
```yaml
server:
 ...
 filter:
  ...
  - name 

client:
 ...
 filter:
  ...
  - name 
```
