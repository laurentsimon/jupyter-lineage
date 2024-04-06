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
	logger     logger.Logger
	allowHosts []string
	denyHosts  []string
}

// NOTE: We could use goproxy.ReqHostMatches(regexp.MustCompile("^.*$") with a list of regex as whown
// in https://github.com/elazarl/goproxy/blob/7cc037d33fb57d20c2fa7075adaf0e2d2862da78/README.md?plain=1#L139
func (h *handler) onRequest(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	req, resp, err := h.enforceHostAllowList(r, ctx)
	if err != nil {
		h.logger.Debugf("[http]: request error: %v", err)
		return req, resp
	}
	req, resp, err = h.enforceHostdenyHosts(r, ctx)
	if err != nil {
		h.logger.Debugf("[http]: request error: %v", err)
		return req, resp
	}
	return r, nil
}

func (h *handler) enforceHostAllowList(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response, error) {
	if len(h.allowHosts) == 0 {
		return r, nil, nil
	}
	// NOTE: We could also handle regexe like
	// shown in https://github.com/elazarl/goproxy/blob/master/examples/goproxy-eavesdropper/main.go#L24.
	for i := range h.allowHosts {
		val := &h.allowHosts[i]
		if r.Host == *val {
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

func (h *handler) enforceHostdenyHosts(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response, error) {
	for i := range h.denyHosts {
		// if goproxy.DstHostIs(*val)(r, ctx) includes port matching.
		val := &h.denyHosts[i]
		if r.Host == *val {
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
	// TODO: Dispatch here?
	/*
			Example:
			INFO/2024-04-06T02:20:08Z: [goproxy][060] INFO: Sending request GET https://huggingface.co:443/microsoft/trocr-small-handwritten/resolve/main/tokenizer_config.json

		INFO/2024-04-06T02:20:08Z: [goproxy][060] INFO: resp 200 OK

		DEBUG/2024-04-06T02:20:08Z: [http]: received ("huggingface.co"):
		Header:
		map["Accept-Ranges":["bytes"] "Access-Control-Allow-Origin":["https://huggingface.co"] "Access-Control-Expose-Headers":["X-Repo-Commit,X-Request-Id,X-Error-Code,X-Error-Message,ETag,Link,Accept-Ranges,Content-Range"] "Content-Disposition":["inline; filename*=UTF-8''tokenizer_config.json; filename=\"tokenizer_config.json\";"] "Content-Length":["327"] "Content-Security-Policy":["default-src none; sandbox"] "Content-Type":["text/plain; charset=utf-8"] "Cross-Origin-Opener-Policy":["same-origin"] "Date":["Sat, 06 Apr 2024 02:20:08 GMT"] "Etag":["\"9fa50ec37ab9f93c5f9d90e65827d3af0d5d4043\""] "Referrer-Policy":["strict-origin-when-cross-origin"] "Vary":["Origin"] "Via":["1.1 8f33c0d3c22e6034f8a41854a2ca274e.cloudfront.net (CloudFront)"] "X-Amz-Cf-Id":["8jh-ORgWfjrk2WRY5Hs5uBxUYlPB9CaMymq1VOpDCBYnRrWEVeJg7w=="] "X-Amz-Cf-Pop":["SEA900-P3"] "X-Cache":["Miss from cloudfront"] "X-Powered-By":["huggingface-moon"] "X-Repo-Commit":["55eb2010aeaaa246defc329d42939e0253d55c99"] "X-Request-Id":["Root=1-6610b158-6ddbb105010c82d262b0a1e0;206764df-7f26-462f-ae41-c762b90576c8"]]
		Body:
		"{\"bos_token\": \"<s>\", \"eos_token\": \"</s>\", \"unk_token\": \"<unk>\", \"sep_token\": \"</s>\", \"cls_token\": \"<s>\", \"pad_token\": \"<pad>\", \"mask_token\": {\"content\": \"<mask>\", \"single_word\": false, \"lstrip\": true, \"rstrip\": false, \"normalized\": true, \"__type\": \"AddedToken\"}, \"sp_model_kwargs\": {}, \"tokenizer_class\": \"XLMRobertaTokenizer\"}"

		In response we see:
		"Content-Disposition":["inline; filename*=UTF-8''tokenizer_config.json; filename=\"tokenizer_config.json\";"] "Content-Length":["327"]
		"X-Repo-Commit":["55eb2010aeaaa246defc329d42939e0253d55c99"]

		We can ignore HEAD requests. We get payload for others.
		We use the URL https://huggingface.co:443/microsoft/trocr-small-handwritten/resolve/main/tokenizer_config.json since it's a proxy
		resourceDescriptor{
			URI: https://huggingface.co:443/microsoft/trocr-small-handwritten/resolve/main/tokenizer_config.json
			DownloadLocation: https://huggingface.co:443/microsoft/trocr-small-handwritten/resolve/main/tokenizer_config.json,
			Digest.gitCommit: header[X-Repo-Commit],
			Digest.sha256: sha256(payload),
			MediaType: header[contenttype],
			Name: model://huggingface.co/microsoft/trocr-small-handwritten
		}
	*/
	h.logger.Debugf("[http]: received (%q):\nHeader:\n%q\nBody:\n%q", ctx.Req.Host, resp.Header, b)
	resp.Body.Close()
	resp.Body = ioutil.NopCloser(bytes.NewBufferString(string(b)))
	return resp
}
