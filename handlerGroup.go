package lightwork

import (
	"context"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// HandlerGroup is the object that allows you to add middleware and routes/handlers.
// Middleware registered within a HandlerGroup is scoped to that group, and will only be used for handlers within that group, or within child groups.
type HandlerGroup struct {
	s              *Server
	parent         *HandlerGroup
	middlewareList []Middleware
	basePath       string
}

func (hg *HandlerGroup) handlerShim(h Handler) httprouter.Handle {
	h = hg.middlewareHandler(h)
	return func(rw http.ResponseWriter, req *http.Request, p httprouter.Params) {
		c := &Context{
			server:  hg.s,
			Context: SimpleCtx{Context: context.Background()},
		}
		c.Response = ContextResponse{c: c, rw: &loggingResponseWriter{rw: rw}}
		c.Request = ContextRequest{c: c, req: req, params: p}
		rlb := hg.s.NewRequestLogger(c)
		c.Log = &RequestLogger{
			b: rlb,
		}

		err := h(c)
		if err != nil {
			c.Log.Errorf("Error returned from request handler: %v", err)
		}
		if c.Response.rw.statusCode == 0 {
			c.Log.WTF("Handler didn't write a response")
		}
		c.Log.b.WriteLogs()
	}
}

func (hg *HandlerGroup) middlewareHandler(userHandler Handler) (fullHandler Handler) {
	fullHandler = userHandler
	ml := hg.middlewareList
	for i := len(ml) - 1; i >= 0; i-- {
		fullHandler = ml[i](fullHandler)
	}
	return
}

// GetHandlerGroup returns a child HandlerGroup, which can be used to add middleware and handlers in a separate scope.
func (ohg *HandlerGroup) GetHandlerGroup(basePath string) (hg *HandlerGroup) {
	hg = &HandlerGroup{
		s:              ohg.s,
		parent:         ohg,
		middlewareList: make([]Middleware, len(ohg.middlewareList)),
		basePath:       ohg.basePath + basePath,
	}
	copy(hg.middlewareList, ohg.middlewareList)
	return
}

// AddHandlerGroup is a more convenient method of retrieving and using a child HandlerGroup to add middleware and functions.
func (ohg *HandlerGroup) AddHandlerGroup(basePath string, registerFunc func(hg *HandlerGroup)) {
	registerFunc(ohg.GetHandlerGroup(basePath))
}

// AddMiddleware registers one or more middleware handlers.
// Middleware is called in the order that it gets registered, and will only be applied to handlers that are added after it.
func (hg *HandlerGroup) AddMiddleware(m ...Middleware) {
	hg.middlewareList = append(hg.middlewareList, m...)
}

// DELETE registers a handler using the DELETE HTTP Method
func (hg *HandlerGroup) DELETE(path string, h Handler) {
	path = hg.basePath + path
	hg.s.router.DELETE(path, hg.handlerShim(h))
}

// GET registers a handler using the GET HTTP Method
func (hg *HandlerGroup) GET(path string, h Handler) {
	path = hg.basePath + path
	hg.s.router.GET(path, hg.handlerShim(h))
}

// HEAD registers a handler using the HEAD HTTP Method
func (hg *HandlerGroup) HEAD(path string, h Handler) {
	path = hg.basePath + path
	hg.s.router.HEAD(path, hg.handlerShim(h))
}

// OPTIONS registers a handler using the OPTIONS HTTP Method
func (hg *HandlerGroup) OPTIONS(path string, h Handler) {
	path = hg.basePath + path
	hg.s.router.OPTIONS(path, hg.handlerShim(h))
}

// PATCH registers a handler using the PATCH HTTP Method
func (hg *HandlerGroup) PATCH(path string, h Handler) {
	path = hg.basePath + path
	hg.s.router.PATCH(path, hg.handlerShim(h))
}

// POST registers a handler using the POST HTTP Method
func (hg *HandlerGroup) POST(path string, h Handler) {
	path = hg.basePath + path
	hg.s.router.POST(path, hg.handlerShim(h))
}

// PUT registers a handler using the PUT HTTP Method
func (hg *HandlerGroup) PUT(path string, h Handler) {
	path = hg.basePath + path
	hg.s.router.PUT(path, hg.handlerShim(h))
}
