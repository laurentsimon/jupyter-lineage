package model

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/laurentsimon/jupyter-lineage/pkg/slsa"

	handler "github.com/laurentsimon/jupyter-lineage/pkg/jnproxy/handler/http"
)

type Model struct {
	handler.HandlerImpl
}

func New() (*Model, error) {
	self := &Model{}
	self.SetName("HuggingfaceModel/v0.1")
	return self, nil
}

func (h *Model) OnRequest(req *http.Request, ctx handler.Context) (*http.Request, *http.Response, bool, error) {
	absPath, err := handler.AbsURLPath(req.URL.Path)
	if err != nil {
		msg := fmt.Sprintf("[http/%s] %v", h.Name(), err)
		ctx.Logger.Errorf(msg)
		return req, handler.NewResponse(ctx.Req, handler.ContentTypeText, http.StatusInternalServerError, msg), false, nil
	}
	// WARNING: absPath prefix must start and and with '/'.
	interested := (req.Host == "huggingface.co" && !strings.Contains(absPath, "/datasets/")) ||
		(req.Host == "cdn-lfs.huggingface.co" && !strings.Contains(absPath, "/datasets/"))
	return req, nil, interested, nil
}

func (h *Model) OnResponse(resp *http.Response, ctx handler.Context) (*http.Response, error) {
	b, _ := ioutil.ReadAll(resp.Body)
	//ctx.Logger.Debugf("[http]: received (%q):\nHeader:\n%q\nBody:\n%q", ctx.Req.Host, resp.Header, b)
	ctx.Logger.Debugf("[http]: received (%q %q):\nHeader:\n%q", ctx.Req.Method, ctx.Req.Host+ctx.Req.URL.Path, resp.Header)
	if ctx.Req.Method == "HEAD" {
		return resp, nil
	}
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

		"Content-Disposition":["inline; filename*=UTF-8''special_tokens_map.json; filename=\"special_tokens_map.json\";"] "Content-Length":["238"]
		"Content-Type":["text/plain; charset=utf-8"]
	*/
	resp.Body.Close()
	resp.Body = ioutil.NopCloser(bytes.NewBufferString(string(b)))

	// Parse headers.
	header := resp.Header
	hLen, err := strconv.Atoi(header.Get("Content-Length"))
	if err != nil {
		msg := fmt.Sprintf("[http/%s] conversion to int: %v", h.Name(), err)
		ctx.Logger.Errorf(msg)
		return handler.NewResponse(ctx.Req, handler.ContentTypeText, http.StatusInternalServerError, msg), nil
	}
	if hLen != len(b) {
		msg := fmt.Sprintf("length mismatch. Header (%v) != actual (%v)", hLen, len(b))
		ctx.Logger.Errorf(msg)
		return handler.NewResponse(ctx.Req, handler.ContentTypeText, http.StatusInternalServerError, msg), nil
	}
	contentType, ok := header["Content-Type"]
	if !ok {
		msg := "Content-Type is empty"
		ctx.Logger.Errorf(msg)
		return handler.NewResponse(ctx.Req, handler.ContentTypeText, http.StatusInternalServerError, msg), nil
	}
	xRepoCommit := header.Get("X-Repo-Commit")
	hash := sha256.New()
	hash.Write(b)
	hh := fmt.Sprintf("%x", hash.Sum(nil))
	aLen64 := uint64(len(b))
	rd := slsa.ResourceDescriptor{
		// WARNING: We're not recording GET parameters.
		DownloadLocation: ctx.Req.URL.Host + ctx.Req.URL.Path,
		URI:              ctx.Req.URL.Host + ctx.Req.URL.Path,
		ContentLength:    &aLen64,
		DigestSet: slsa.DigestSet{
			"sha256": hh,
		},
		Annotations: map[string]any{
			// NOTE: No header recorded.
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
	if xRepoCommit != "" {
		rd.DigestSet["hint:gitCommit"] = xRepoCommit
	}
	h.Store(ctx.ID, rd)
	ctx.Logger.Debugf("[http]: RD %q", rd)
	return resp, nil
}
