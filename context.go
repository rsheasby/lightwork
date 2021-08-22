package lightwork

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

type loggingResponseWriter struct {
	rw            http.ResponseWriter
	statusCode    int
	contentLength int
}

func (lrw *loggingResponseWriter) Header() (h http.Header) {
	return lrw.rw.Header()
}

func (lrw *loggingResponseWriter) Write(b []byte) (n int, err error) {
	n, err = lrw.rw.Write(b)
	lrw.contentLength += n
	return
}

func (lrw *loggingResponseWriter) WriteHeader(statusCode int) {
	lrw.statusCode = statusCode
	lrw.rw.WriteHeader(statusCode)
}

type Handler func(c *Context) (err error)

// SimpleCtx is a simple wrapper around a context that includes the SetValue convenience function
type SimpleCtx struct {
	context.Context
}

// SetValue is a helper that replaces the context with a new context including the provided value.
func (ctx *SimpleCtx) SetValue(key, value interface{}) {
	ctx.Context = context.WithValue(ctx.Context, key, value)
}

// Context is the object provided to an HTTP handler.
// It contains sub-objects for reading the request, returning a response, and logging.
// It also includes an extended context.Context, which can be used for deadlines, cancellation, or storage of arbitrary values.
type Context struct {
	Context         SimpleCtx
	Log             *RequestLogger
	Response        ContextResponse
	Request         ContextRequest
	server          *Server
	escapeHatchUsed bool
}

// EscapeHatch returns the *Request and ResponseWriter for the request.
// The use of this function assumes you need lower-level control, and thus disables some built-in functionality.
func (c *Context) EscapeHatch() (rw http.ResponseWriter, req *http.Request) {
	c.escapeHatchUsed = true
	return c.Response.rw, c.Request.req
}

// ContextResponse contains the methods used to return an HTTP response
type ContextResponse struct {
	rw *loggingResponseWriter
	c  *Context
}

// Header returns the header object for the response.
func (cr ContextResponse) Header() (h http.Header) {
	return cr.rw.Header()
}

// setHeaderIfNotAlreadySet checks if a header with the provided key already exists, and if not, sets it to the provided value.
func (cr ContextResponse) setHeaderIfNotAlreadySet(key, value string) {
	header := cr.Header()
	v := header.Get(key)
	if v != "" {
		return
	}
	header.Set(key, value)
}

// Status returns the provided status code, with no response body.
func (cr ContextResponse) Status(statusCode int) (err error) {
	cr.Header().Set("Content-Length", "0")
	cr.rw.WriteHeader(statusCode)
	return nil
}

// Bytes returns the provided status code and body.
// If the Content-Type header is not already set, it will be set to application/octet-stream.
func (cr ContextResponse) Bytes(statusCode int, body []byte) (err error) {
	cr.setHeaderIfNotAlreadySet("Content-Type", "application/octet-stream")
	cr.Header().Set("Content-Length", strconv.Itoa(len(body)))
	cr.rw.WriteHeader(statusCode)
	_, err = cr.rw.Write(body)
	if err != nil {
		return fmt.Errorf("failed to write body: %w", err)
	}
	return
}

// String returns the provided status code and body.
// If the Content-Type header is not already set, it will be set to text/plain.
func (cr ContextResponse) String(statusCode int, body string) (err error) {
	cr.setHeaderIfNotAlreadySet("Content-Type", "text/plain")
	return cr.Bytes(statusCode, []byte(body))
}

// Struct returns the provided status code, and a serialised struct
// The struct will be serialised using the server's configured StructEncoder.
func (cr ContextResponse) Struct(statusCode int, s interface{}) (err error) {
	cr.rw.WriteHeader(statusCode)
	return cr.c.server.EncodeStruct(cr.c, s, cr.rw)
}

// Stream returns the provided status code, then streams the provided Reader as the body.
// Go will automatically set the Content-Type based on the first 512 bytes of the stream, if the header is not already set.
// If you don't want Go to infer the Content-Type, you should explicitly set the header BEFORE using this function.
// For relatively short streams, Go will automatically buffer the output and set the Content-Length after reading the full stream.
// For longer streams, Go will use chunked encoding.
func (cr ContextResponse) Stream(statusCode int, stream io.Reader) (err error) {
	cr.rw.WriteHeader(statusCode)
	_, err = io.Copy(cr.rw, stream)
	return
}

// StreamReadSeeker returns the provided status code, then streams the provided ReadSeeker as the body.
// Go will automatically set the Content-Type based on the first 512 bytes of the stream, if the header is not already set.
// If you don't want Go to infer the Content-Type, you should explicitly set the header BEFORE using this function.
func (cr ContextResponse) StreamReadSeeker(statusCode int, stream io.ReadSeeker) (err error) {
	currentPos, err := stream.Seek(0, io.SeekCurrent)
	if err != nil {
		cr.c.Log.Warningf("Unable to determine current stream position: %v", err)
		cr.c.Log.Info("Falling back to chunked streaming")
		return cr.Stream(statusCode, stream)
	}
	totalStreamLen, err := stream.Seek(0, io.SeekEnd)
	if err != nil {
		cr.c.Log.Warningf("Unable to determine total stream length: %v", err)
		cr.c.Log.Info("Falling back to chunked streaming")
		return cr.Stream(statusCode, stream)
	}
	_, err = stream.Seek(currentPos, io.SeekStart)
	if err != nil {
		cr.c.Log.Errorf("Unable to restore stream position after reading length: %v", err)
		return fmt.Errorf("failed to safely determine stream length - aborting")
	}

	cr.Header().Set("Content-Length", strconv.FormatInt(totalStreamLen-currentPos, 10))
	return cr.Stream(statusCode, stream)
}

// File returns the provided status code, then streams the provided file as the body.
// Go will automatically set the Content-Type based on the first 512 bytes of the file, if the header is not already set.
// If you don't want Go to infer the Content-Type, you should explicitly set the header BEFORE using this function.
func (cr ContextResponse) File(statusCode int, filename string) (err error) {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return cr.StreamReadSeeker(statusCode, file)
}

// GetStatusCode returns the status code that was sent in the response.
// If a response code has not been sent yet, this will return 0.
func (cr ContextResponse) GetStatusCode() (statusCode int) {
	return cr.rw.statusCode
}

// ContextRequest contains the methods used to retrieve the request details
type ContextRequest struct {
	c      *Context
	req    *http.Request
	params httprouter.Params
}

// ClientHost returns the hostname or IP address of the client making the request.
func (cr ContextRequest) ClientHost() (host string) {
	return cr.c.server.ClientHost(cr.c)
}

// Method returns the HTTP method of the request.
func (cr ContextRequest) Method() (m string) {
	return cr.req.Method
}

// URL returns the HTTP URL of the request.
func (cr ContextRequest) URL() (url *url.URL) {
	return cr.req.URL
}

func (cr ContextRequest) Header() (h *http.Header) {
	return &cr.req.Header
}

// Params returns the httprouter.Params object for the request.
func (cr ContextRequest) Params() (p httprouter.Params) {
	return cr.params
}

// GetParam is shorthand for Params().ByName.
func (cr ContextRequest) GetParam(name string) (value string) {
	return cr.params.ByName(name)
}

// BodyStream returns the body of the request as a io.ReadCloser.
func (cr ContextRequest) BodyStream() (stream io.ReadCloser) {
	return cr.req.Body
}

// BodyBytes returns the body of the request as a byte slice.
func (cr ContextRequest) BodyBytes() (body []byte) {
	buf := bytes.Buffer{}
	bodyStream := cr.BodyStream()
	defer bodyStream.Close()
	buf.ReadFrom(bodyStream)
	return buf.Bytes()
}

// BodyString returns the body of the request as a string.
func (cr ContextRequest) BodyString() (body string) {
	return string(cr.BodyBytes())
}

// BodyStruct reads and deserialises the body of the request into the provided struct.
// The result parameter must be a pointer to a struct.
func (cr ContextRequest) BodyStruct(result interface{}) (err error) {
	bodyStream := cr.BodyStream()
	return cr.c.server.DecodeStruct(cr.c, bodyStream, result)
}
