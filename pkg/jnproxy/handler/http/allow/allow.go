package allow

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	handler "github.com/laurentsimon/jupyter-lineage/pkg/jnproxy/handler/http"
	"github.com/laurentsimon/jupyter-lineage/pkg/slsa"
)

type AllowCb interface {
	// Caller decides if they want to record the response.
	// For example, they may decide not to record 0-length payload
	// responses if they believe the headers are not manipulable by developers.
	WantRecord(*http.Response, handler.Context) (bool, *http.Header)
}

type Option func(*Allow) error

type Allow struct {
	handler.HandlerImpl
	cb AllowCb
}

func New(options ...Option) (*Allow, error) {
	self := &Allow{}
	self.SetName("Allow/v0.1")
	// Set optional parameters.
	for _, option := range options {
		err := option(self)
		if err != nil {
			return nil, err
		}
	}

	return self, nil
}

func (h *Allow) OnRequest(req *http.Request, ctx handler.Context) (*http.Request, *http.Response, bool, error) {
	return req, nil, true, nil
}

func (h *Allow) wantRecord(resp *http.Response, ctx handler.Context) (bool, *http.Header) {
	if h.cb == nil {
		// By default, we do not record headers.
		return true, nil
	}
	return h.cb.WantRecord(resp, ctx)
}

func (h *Allow) OnResponse(resp *http.Response, ctx handler.Context) (*http.Response, error) {
	rec, headerRecord := h.wantRecord(resp, ctx)
	if !rec {
		return nil, nil
	}

	// We record dependencies including the headers, because we do not know
	// if the contacted host is controlled by the developer or not.
	// If it is controlled by the dev, headers may contain code.
	b, _ := ioutil.ReadAll(resp.Body)
	//ctx.Logger.Debugf("[http]: received (%q %q):\nHeader:\n%q\nBody:\n%q", ctx.Req.Method, ctx.Req.Host+ctx.Req.URL.Path, resp.Header, b)
	ctx.Logger.Debugf("[http]: received (%q %q):\nHeader:\n%q", ctx.Req.Method, ctx.Req.Host+ctx.Req.URL.Path, resp.Header)
	resp.Body.Close()
	resp.Body = ioutil.NopCloser(bytes.NewBufferString(string(b)))

	// Parse headers.
	header := resp.Header
	var err error
	var hLen int
	contentLen := header.Get("Content-Length")
	if contentLen != "" {
		hLen, err = strconv.Atoi(header.Get("Content-Length"))
		if err != nil {
			msg := fmt.Sprintf("[http/%s] conversion to int: %v", h.Name(), err)
			ctx.Logger.Errorf(msg)
			return handler.NewResponse(ctx.Req, handler.ContentTypeText, http.StatusInternalServerError, msg), nil
		}
	}
	// https://www.rfc-editor.org/rfc/rfc9110.html#name-content-length
	// "a server MUST NOT send Content-Length in such a response unless
	// its field value equals the decimal number of octets that would have
	// been sent in the content of a response if the same request had used the GET method"
	// HEAD response may have a non-zero Content-Length.
	aLen := len(b)
	if hLen != aLen && aLen == 0 && ctx.Req.Method != "HEAD" {
		msg := fmt.Sprintf("length mismatch. Header (%v) != actual (%v)", hLen, aLen)
		ctx.Logger.Errorf(msg)
		return handler.NewResponse(ctx.Req, handler.ContentTypeText, http.StatusInternalServerError, msg), nil
	}
	contentType, ok := header["Content-Type"]
	if !ok {
		msg := "Content-Type is empty"
		ctx.Logger.Errorf(msg)
		return handler.NewResponse(ctx.Req, handler.ContentTypeText, http.StatusInternalServerError, msg), nil
	}
	hash := sha256.New()
	hash.Write(b)
	hh := fmt.Sprintf("%x", hash.Sum(nil))
	url := constructURL(ctx.Req.URL.Host, ctx.Req.URL.Path, ctx.Req.URL.Query())
	if err != nil {
		msg := err.Error()
		ctx.Logger.Errorf(msg)
		return handler.NewResponse(ctx.Req, handler.ContentTypeText, http.StatusInternalServerError, msg), nil
	}
	aLen64 := uint64(aLen)
	rd := slsa.ResourceDescriptor{
		DownloadLocation: url,
		URI:              url,
		ContentLength:    &aLen64,
		DigestSet: slsa.DigestSet{
			"sha256": hh,
		},
		// TODO(#12): Re-generate the string.
		Annotations: map[string]any{
			"Handler": h.Name(),
			"HTTP": map[string]any{
				"Method": ctx.Req.Method,
				"Header": map[string]any{
					"Content-Length": hLen,
					"Content-Type":   strings.Join(contentType, ";"),
				},
			},
		},
	}

	// Only record what the caller wants to record.
	// WARNING: There may be secrets in the resonse header!
	if headerRecord != nil {
		rd.Annotations["HTTPHeader"] = *headerRecord
	}

	h.Store(ctx.ID, rd)
	ctx.Logger.Debugf("[http]: RD %q", rd)
	// TODO(#12): Overwrite the header with our own to prevent side channels like encoding
	// data in spaces or other. Maybe this is done automatically by the framework...?
	// TODO: callback to decide if we need to store or not.
	return resp, nil
}

func constructURL(host, path string, q url.Values) string {
	url := host + path
	encoded := q.Encode()
	if encoded != "" {
		return url + "?" + encoded
	}
	return url
}

func WithCallback(cb AllowCb) Option {
	return func(h *Allow) error {
		h.cb = cb
		return nil
	}
}
