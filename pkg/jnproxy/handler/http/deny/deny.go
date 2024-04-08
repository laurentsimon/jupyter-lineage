package deny

import (
	"fmt"
	"net/http"

	"github.com/laurentsimon/jupyter-lineage/pkg/errs"
	handler "github.com/laurentsimon/jupyter-lineage/pkg/jnproxy/handler/http"
)

const name = "Deny"

type Deny struct {
}

func New() (*Deny, error) {
	return &Deny{}, nil
}

func (h *Deny) Name() string {
	return name
}

func (h *Deny) OnRequest(req *http.Request, ctx handler.Context) (*http.Request, *http.Response, bool, error) {
	return req,
		handler.NewResponse(req, handler.ContentTypeText, http.StatusForbidden, "Forbidden"),
		false, nil
}

func (h *Deny) OnResponse(resp *http.Response, ctx handler.Context) (*http.Response, error) {
	if resp.StatusCode != http.StatusForbidden {
		return handler.NewResponse(ctx.Req, handler.ContentTypeText, http.StatusInternalServerError, "InternServerError"),
			fmt.Errorf("%w: received response (%q) not forbidden (%q)", errs.ErrorInvalid, ctx.Req.Host, resp.StatusCode)
	}
	return resp, nil
}

func (h *Deny) Results() (handler.Results, error) {
	return handler.Results{}, nil
}
