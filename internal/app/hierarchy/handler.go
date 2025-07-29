package hierarchy

import (
	"context"
	"github.com/66gu1/easygodocs/internal/app/hierarchy/dto"
	"github.com/66gu1/easygodocs/internal/infrastructure/httputil"
	"net/http"
)

type Handler struct {
	svc Service
}

//go:generate minimock -i github.com/66gu1/easygodocs/internal/app/hierarchy.Service -o ./mock -s _mock.go
type Service interface {
	GetTree(ctx context.Context) (dto.Tree, error)
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GetTree(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tree, err := h.svc.GetTree(ctx)
	if err != nil {
		httputil.ReturnError(ctx, w, err)
		return
	}

	httputil.WriteJSON(ctx, w, http.StatusOK, tree)
}
