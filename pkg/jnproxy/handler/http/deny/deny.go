package deny

import (
	"fmt"
	"net/http"

	"github.com/laurentsimon/jupyter-lineage/pkg/errs"
	handler "github.com/laurentsimon/jupyter-lineage/pkg/jnproxy/handler/http"
	"github.com/laurentsimon/jupyter-lineage/pkg/slsa"
)

const name = "Handler/v0.1"

type Handler struct {
}

func New() (*Handler, error) {
	return &Handler{}, nil
}

func (h *Handler) Name() string {
	return name
}

func (h *Handler) OnRequest(req *http.Request, ctx handler.Context) (*http.Request, *http.Response, bool, error) {
	return req,
		handler.NewResponse(req, handler.ContentTypeText, http.StatusForbidden, "Forbidden"),
		false, nil
}

func (h *Handler) OnResponse(resp *http.Response, ctx handler.Context) (*http.Response, error) {
	if resp.StatusCode != http.StatusForbidden {
		return handler.NewResponse(ctx.Req, handler.ContentTypeText, http.StatusInternalServerError, "InternServerError"),
			fmt.Errorf("%w: received response (%q) not forbidden (%q)", errs.ErrorInvalid, ctx.Req.Host, resp.StatusCode)
	}
	return resp, nil
}

func (h *Handler) Dependencies(ctx handler.Context) ([]slsa.ResourceDescriptor, error) {
	return nil, nil
}
