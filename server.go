package lightwork

import (
	"io"

	"github.com/julienschmidt/httprouter"

	"net/http"
	"net/http/httptest"
)

type Server struct {
	router *httprouter.Router

	// EncodeStruct will be used to serialise objects for HTTP responses.
	// Typically, this would be a JSON or XML encoder.
	EncodeStruct func(c *Context, input interface{}, output io.Writer) (err error)
	// DecodeStruct will be used to deserialise HTTP requests into objects.
	// Typically, this would be a JSON or XML decoder.
	// "result" must be a pointer to the object that this will deserialise into.
	DecodeStruct func(c *Context, input io.Reader, result interface{}) (err error)

	// ValidateStruct will be used to validate objects.
	ValidateStruct func(c *Context, input interface{}) (err error)

	// NewRequestLogger will be called at the beginning of every request to get a logger to be used for that request.
	NewRequestLogger func(c *Context) (rlb RequestLoggerBase)
}

func NewServer() (server *Server) {
	return &Server{
		router: httprouter.New(),
	}
}

// Router returns the underlying julienschmidt/httprouter Router instance.
func (s *Server) Router() (router *httprouter.Router) {
	return s.router
}

// GetHandlerGroup returns a HandlerGroup, which can be used to add middleware and handlers.
func (s *Server) GetHandlerGroup(basePath string) (hg *HandlerGroup) {
	hg = &HandlerGroup{
		s:              s,
		parent:         nil,
		middlewareList: make([]Middleware, 0),
		basePath:       basePath,
	}
	return
}

// AddHandlerGroup is a more convenient method of retrieving and using a HandlerGroup to add middleware and functions.
func (s *Server) AddHandlerGroup(basePath string, registerFunc func(hg *HandlerGroup)) {
	registerFunc(s.GetHandlerGroup(basePath))
}

// Start listens on the provided address, and starts serving requests.
func (s *Server) Start(address string) (err error) {
	return http.ListenAndServe(address, s.router)
}

// StartTest starts and returns an *httptest.Server, which can be used for automated testing
func (s *Server) StartTest() (testServer *httptest.Server) {
	return httptest.NewServer(s.router)
}
