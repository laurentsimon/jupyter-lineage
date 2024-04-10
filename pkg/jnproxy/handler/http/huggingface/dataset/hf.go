package dataset

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/laurentsimon/jupyter-lineage/pkg/slsa"

	handler "github.com/laurentsimon/jupyter-lineage/pkg/jnproxy/handler/http"
)

type Dataset struct {
	handler.HandlerImpl
}

func New() (*Dataset, error) {
	self := &Dataset{}
	self.SetName("HuggingfaceDataset/v0.1")
	return self, nil
}

func (h *Dataset) OnRequest(req *http.Request, ctx handler.Context) (*http.Request, *http.Response, bool, error) {
	absPath, err := handler.AbsURLPath(req.URL.Path)
	if err != nil {
		msg := fmt.Sprintf("[http/%s] %v", h.Name(), err)
		ctx.Logger.Errorf(msg)
		return req, handler.NewResponse(ctx.Req, handler.ContentTypeText, http.StatusInternalServerError, msg), false, nil
	}
	// WARNING: absPath prefix must start and and with '/'.
	interested := (req.Host == "huggingface.co" && strings.Contains(absPath, "/datasets/")) ||
		(req.Host == "cdn-lfs.huggingface.co" && strings.Contains(absPath, "/datasets/")) ||
		// TODO: This is the API to list pqrquest files
		// https://huggingface.co/docs/datasets-server/en/parquet#using-the-dataset-viewer-api
		(req.Host == "datasets-server.huggingface.co") ||
		// WARNING: amazon bucket is not under huggingface.co so we need to match the path prefix.
		(req.Host == "s3.amazonaws.com" && strings.HasPrefix(absPath, "/datasets.huggingface.co/")) ||
		// WARNING: google bucket is not under huggingface.co so we need to match the path prefix.
		(req.Host == "storage.googleapis.com" &&
			(strings.HasPrefix(absPath, "/huggingface-nlp/") || strings.HasPrefix(absPath, "/cvdf-datasets/")))

	return req, nil, interested, nil
}

func (h *Dataset) OnResponse(resp *http.Response, ctx handler.Context) (*http.Response, error) {
	b, _ := ioutil.ReadAll(resp.Body)
	//ctx.Logger.Debugf("[http]: received (%q):\nHeader:\n%q\nBody:\n%q", ctx.Req.Host, resp.Header, b)
	ctx.Logger.Debugf("[http]: received (%q %q):\nHeader:\n%q", ctx.Req.Method, ctx.Req.Host+ctx.Req.URL.Path, resp.Header)
	if ctx.Req.Method == "HEAD" {
		return resp, nil
	}
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

	var rd slsa.ResourceDescriptor
	switch contentType[0] {
	case "todo/zip":
		// Extract zipped files.
		// https://pkg.go.dev/archive/zip#NewReader
		reader := bytes.NewReader(b)
		zipReader, err := zip.NewReader(reader, int64(len(b)))
		if err != nil {
			msg := fmt.Sprintf("zip reader: %v", err)
			ctx.Logger.Errorf(msg)
			return handler.NewResponse(ctx.Req, handler.ContentTypeText, http.StatusInternalServerError, msg), nil
		}
		for _, f := range zipReader.File {
			// f.FileInfo().IsDir()
			// fileInArchive, err := f.Open()
			// io.Copy(dstFile, fileInArchive)
			ctx.Logger.Debugf("unzipping file ", f.Name, f.CompressedSize, f.UncompressedSize64)
		}
	case "binary/octet-stream":
		// https://huggingface.co/docs/datasets-server/en/parquet
		// https://huggingface.co/docs/datasets/en/index
	case "application/x-gzip":
		// https://pkg.go.dev/compress/gzip#Reader.Read
		reader := bytes.NewReader(b)
		outputBytes := make([]byte, len(b))
		gzipReader, err := gzip.NewReader(reader)
		if err != nil {
			msg := fmt.Sprintf("gzip reader: %v", err)
			ctx.Logger.Errorf(msg)
			return handler.NewResponse(ctx.Req, handler.ContentTypeText, http.StatusInternalServerError, msg), nil
		}
		_, err = gzipReader.Read(outputBytes)
		if err != nil {
			msg := fmt.Sprintf("gzip read: %v", err)
			ctx.Logger.Errorf(msg)
			return handler.NewResponse(ctx.Req, handler.ContentTypeText, http.StatusInternalServerError, msg), nil
		}
	default:
		hash := sha256.New()
		hash.Write(b)
		hh := fmt.Sprintf("%x", hash.Sum(nil))
		aLen64 := uint64(len(b))
		rd = slsa.ResourceDescriptor{
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
	}

	xRepoCommit := header.Get("X-Repo-Commit")
	if xRepoCommit != "" {
		rd.DigestSet["hint:gitCommit"] = xRepoCommit
	}
	h.Store(ctx.ID, rd)
	ctx.Logger.Debugf("[http]: RD %q", rd)
	return resp, nil
}
