package http

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/laurentsimon/jupyter-lineage/pkg/errs"
	"github.com/laurentsimon/jupyter-lineage/pkg/logger"

	"github.com/elazarl/goproxy"
)

// TODO: public to let callers customize and provide their own handlers
// probably need to move this to root of folder. I think we only need to expose
// an Init, onResponse, onRequest APIs
type handler struct {
	logger       logger.Logger
	allowedHosts []string
	denyList     []string
}

// NOTE: We could use goproxy.ReqHostMatches(regexp.MustCompile("^.*$") with a list of regex as whown
// in https://github.com/elazarl/goproxy/blob/7cc037d33fb57d20c2fa7075adaf0e2d2862da78/README.md?plain=1#L139
func (h *handler) onRequest(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	req, resp, err := h.enforceHostAllowList(r, ctx)
	if err != nil {
		h.logger.Debugf("[http]: request error: %v", err)
		return req, resp
	}
	req, resp, err = h.enforceHostDenyList(r, ctx)
	if err != nil {
		h.logger.Debugf("[http]: request error: %v", err)
		return req, resp
	}
	return r, nil
}

func (h *handler) enforceHostAllowList(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response, error) {
	if len(h.allowedHosts) == 0 {
		return r, nil, nil
	}
	for i := range h.allowedHosts {
		val := &h.allowedHosts[i]
		if goproxy.ReqHostIs(*val)(r, ctx) {
			return r, nil, nil
		}
	}
	// WARNING: can be bypassed by connecting to random IP and setting this header to an allowed
	// value? No, because we connect outselve using this value.
	return r, goproxy.NewResponse(r,
			goproxy.ContentTypeText, http.StatusForbidden,
			"Forbidden"),
		fmt.Errorf("%w: destination (%q) not on allow list", errs.ErrorInvalid, r.Host)
}

func (h *handler) enforceHostDenyList(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response, error) {
	for i := range h.denyList {
		val := &h.denyList[i]
		if goproxy.ReqHostIs(*val)(r, ctx) {
			return r, goproxy.NewResponse(r,
					goproxy.ContentTypeText, http.StatusForbidden,
					"Forbidden"),
				fmt.Errorf("%w: destination (%q) on deny list", errs.ErrorInvalid, r.Host)

		}
	}
	return r, nil, nil
}

func (h *handler) onResponse(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
	b, _ := ioutil.ReadAll(resp.Body)
	// TODO: handle error
	h.logger.Debugf("[http]: received (%q):\nHeader:\n%q\nBody:\n%q", ctx.Req.Host, resp.Header, b)
	resp.Body.Close()
	resp.Body = ioutil.NopCloser(bytes.NewBufferString(string(b)))
	return resp
}
