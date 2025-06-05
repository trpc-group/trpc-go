//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 THL A29 Limited, a Tencent company.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

package admin

import (
	"fmt"
	"net/http"
	"runtime"
	"sync"

	"trpc.group/trpc-go/trpc-go/internal/report"
	"trpc.group/trpc-go/trpc-go/log"
)

// PanicBufLen is len of buffer used for stack trace logging
// when the goroutine panics, 1024 by default.
const panicBufLen = 1024

// Router Routing table interface, register routing information through the structure that implements this interface.
type Router interface {
	// Config Set the handler function, cannot be overridden.
	Config(patten string, handler func(w http.ResponseWriter, r *http.Request)) *RouterHandler

	// ServeHTTP dispatches the request to the handler whose pattern most closely matches the request URL.
	ServeHTTP(w http.ResponseWriter, req *http.Request)

	// List current registration methods.
	List() []*RouterHandler
}

// NewRouter creates a new Router.
func NewRouter() Router {
	return &router{
		ServeMux: http.NewServeMux(),
	}
}

// NewRouterHandler creates a new restful route info handler.
func NewRouterHandler(patten string, handler func(w http.ResponseWriter, r *http.Request)) *RouterHandler {
	return &RouterHandler{
		patten:  patten,
		handler: handler,
	}
}

// router struct.
type router struct {
	*http.ServeMux

	sync.RWMutex
	handleFuncMap map[string]*RouterHandler
}

// Config configures a routing pattern and handler function.
func (r *router) Config(patten string, handler func(w http.ResponseWriter, r *http.Request)) *RouterHandler {
	r.Lock()
	defer r.Unlock()

	r.ServeMux.HandleFunc(patten, handler)
	if r.handleFuncMap == nil {
		r.handleFuncMap = make(map[string]*RouterHandler)
	}

	handle := NewRouterHandler(patten, handler)
	r.handleFuncMap[patten] = handle
	return handle
}

// ServeHTTP handles incoming http requests.
func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			var ret = newDefaultRes()
			ret[ReturnErrCodeParam] = http.StatusInternalServerError
			ret[ReturnMessageParam] = fmt.Sprintf("PANIC : %v", err)
			buf := make([]byte, panicBufLen)
			buf = buf[:runtime.Stack(buf, false)]
			log.Errorf("[PANIC]%v\n%s\n", err, buf)
			report.AdminPanicNum.Incr()
			_ = json.NewEncoder(w).Encode(ret)
		}
	}()
	r.ServeMux.ServeHTTP(w, req)
}

// List returns a list of configured routes.
func (r *router) List() []*RouterHandler {
	l := make([]*RouterHandler, 0, len(r.handleFuncMap))
	for _, handler := range r.handleFuncMap {
		l = append(l, handler)
	}
	return l
}

// RouterHandler routing information handler.
type RouterHandler struct {
	handler func(w http.ResponseWriter, r *http.Request)
	patten  string
	desc    string
}

// GetHandler returns a routing information handle function.
func (r *RouterHandler) GetHandler() func(w http.ResponseWriter, r *http.Request) {
	return r.handler
}

// GetDesc returns route description/remarks.
func (r *RouterHandler) GetDesc() string {
	return r.desc
}

// GetPatten returns template for routing configuration.
func (r *RouterHandler) GetPatten() string {
	return r.patten
}

// Desc sets description information.
func (r *RouterHandler) Desc(desc string) *RouterHandler {
	r.desc = desc
	return r
}
