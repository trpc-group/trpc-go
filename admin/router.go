package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"

	"trpc.group/trpc-go/trpc-go/internal/report"
	"trpc.group/trpc-go/trpc-go/log"
)

// PanicBufLen is the length of the buffer used for stack trace logging
// when goroutine panics, default is 1024.
const panicBufLen = 1024

// newRouter creates a new Router.
func newRouter() *router {
	return &router{
		ServeMux: http.NewServeMux(),
	}
}

// newRouterHandler creates a new restful route info handler.
func newRouterHandler(patten string, handler func(w http.ResponseWriter, r *http.Request)) *routerHandler {
	return &routerHandler{
		pattern: patten,
		handler: handler,
	}
}

type router struct {
	*http.ServeMux

	sync.RWMutex
	handlers map[string]*routerHandler
}

// add adds a routing pattern and handler function.
func (r *router) add(patten string, handler func(w http.ResponseWriter, r *http.Request)) *routerHandler {
	r.Lock()
	defer r.Unlock()

	r.ServeMux.HandleFunc(patten, handler)
	if r.handlers == nil {
		r.handlers = make(map[string]*routerHandler)
	}

	h := newRouterHandler(patten, handler)
	r.handlers[patten] = h
	return h
}

// ServeHTTP handles incoming HTTP requests.
func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			buf := make([]byte, panicBufLen)
			buf = buf[:runtime.Stack(buf, false)]
			log.Errorf("[PANIC]%v\n%s\n", err, buf)
			report.AdminPanicNum.Incr()
			ret := newDefaultRes()
			ret[retErrCode] = http.StatusInternalServerError
			ret[retMessage] = fmt.Sprintf("PANIC : %v", err)
			_ = json.NewEncoder(w).Encode(ret)
		}
	}()
	r.ServeMux.ServeHTTP(w, req)
}

// list returns a list of configured routes.
func (r *router) list() []*routerHandler {
	l := make([]*routerHandler, 0, len(r.handlers))
	for _, handler := range r.handlers {
		l = append(l, handler)
	}
	return l
}

// routerHandler routing information handler.
type routerHandler struct {
	handler func(w http.ResponseWriter, r *http.Request)
	pattern string
}
