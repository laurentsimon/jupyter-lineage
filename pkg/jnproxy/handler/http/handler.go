package http

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/laurentsimon/jupyter-lineage/pkg/logger"
	"github.com/laurentsimon/jupyter-lineage/pkg/slsa"
)

// Context contains metadata about callbacks.
type Context struct {
	// TODO: need to ensure no collision here.
	ID     int64         // Unique ID identifying the request <-> response
	Req    *http.Request // Request that led to the callback.
	Logger logger.Logger
}

type Handler interface {
	// Unique name of the handler
	Name() string
	// OnRequest is called when a client makes a new request.
	// ct.Req is the same pointer as req.
	// - Request: The modified request. nill if unchanged.
	// - Response: The response to send the client. nil to let the request continue.
	// - bool indicates whether the handler want to receive
	// the response via OnResponse().
	OnRequest(req *http.Request, ctx Context) (*http.Request, *http.Response, bool, error)
	// OnResponse is called when a server responds to a client request.
	// ctx.Req point to the origial request
	OnResponse(resp *http.Response, ctx Context) (*http.Response, error)
	// Dependencies returns the results identified by the handler.
	// On return, the function must erase the dependencies from its internal state.
	// TODO(#13): structure to host other info, like the type of dependencies
	// (sw package, model, dataset, etc)
	Dependencies(ctx Context) ([]slsa.ResourceDescriptor, error)
}

func NewResponse(r *http.Request, contentType string, status int, body string) *http.Response {
	resp := &http.Response{}
	resp.Request = r
	resp.TransferEncoding = r.TransferEncoding
	resp.Header = make(http.Header)
	resp.Header.Add("Content-Type", contentType)
	resp.StatusCode = status
	resp.Status = http.StatusText(status)
	buf := bytes.NewBufferString(body)
	resp.ContentLength = int64(buf.Len())
	resp.Body = ioutil.NopCloser(buf)
	return resp
}

const (
	ContentTypeText = "text/plain"
	ContentTypeHtml = "text/html"
)

// Alias for NewResponse(r,ContentTypeText,http.StatusAccepted,text)
func TextResponse(r *http.Request, text string) *http.Response {
	return NewResponse(r, ContentTypeText, http.StatusAccepted, text)
}

type HandlerImpl struct {
	mu   sync.Mutex
	m    sync.Map
	name string
}

func (h *HandlerImpl) Dependencies(ctx Context) ([]slsa.ResourceDescriptor, error) {
	var deps []slsa.ResourceDescriptor
	var e error
	defer h.mu.Unlock()
	h.mu.Lock()
	h.m.Range(func(key, value any) bool {
		v, ok := value.(slsa.ResourceDescriptor)
		if !ok {
			e = fmt.Errorf("[%s]: invalid type (%T) for key (%q)", h.name, value, key)
			return false
		}
		deps = append(deps, v)
		return true
	})
	h.m.Range(func(key, value any) bool {
		h.m.Delete(key)
		return true
	})
	return deps, e
}

func (h *HandlerImpl) Name() string {
	return h.name
}
func (h *HandlerImpl) SetName(name string) {
	h.name = name
}

func (h *HandlerImpl) Store(id int64, rd slsa.ResourceDescriptor) {
	h.m.Store(id, rd)
}
