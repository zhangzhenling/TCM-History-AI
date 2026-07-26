package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/graph-service/internal/application/usecase"
)

// QueryController exposes HTTP handlers for complex graph queries.
type QueryController struct {
	uc *usecase.QueryUseCase
}

// NewQueryController constructs a QueryController.
func NewQueryController(uc *usecase.QueryUseCase) *QueryController {
	return &QueryController{uc: uc}
}

// GetPersonWorks GET /api/v1/graph/persons/:uid/works
func (h *QueryController) GetPersonWorks(ctx context.Context, c *app.RequestContext) {
	uid, ok := pathUID(c)
	if !ok {
		return
	}
	resp, err := h.uc.GetPersonWorks(ctx, uid)
	okOrFail(ctx, c, resp, err)
}

// GetSchoolLineage GET /api/v1/graph/schools/:name/lineage?max_depth=
func (h *QueryController) GetSchoolLineage(ctx context.Context, c *app.RequestContext) {
	name, ok := pathName(c)
	if !ok {
		return
	}
	maxDepth := queryInt(c, "max_depth", 6)
	resp, err := h.uc.GetSchoolLineage(ctx, name, maxDepth)
	okOrFail(ctx, c, resp, err)
}

// FindShortestPath GET /api/v1/graph/paths/shortest?start_uid=&end_uid=&max_hops=
func (h *QueryController) FindShortestPath(ctx context.Context, c *app.RequestContext) {
	startUID := queryString(c, "start_uid")
	endUID := queryString(c, "end_uid")
	maxHops := queryInt(c, "max_hops", 8)
	resp, err := h.uc.FindShortestPath(ctx, startUID, endUID, maxHops)
	okOrFail(ctx, c, resp, err)
}

// GetDynastyFigures GET /api/v1/graph/dynasties/:name/figures
func (h *QueryController) GetDynastyFigures(ctx context.Context, c *app.RequestContext) {
	name, ok := pathName(c)
	if !ok {
		return
	}
	resp, err := h.uc.GetDynastyFigures(ctx, name)
	okOrFail(ctx, c, resp, err)
}

// GetPrescriptionDetail GET /api/v1/graph/prescriptions/:uid/detail
func (h *QueryController) GetPrescriptionDetail(ctx context.Context, c *app.RequestContext) {
	uid, ok := pathUID(c)
	if !ok {
		return
	}
	resp, err := h.uc.GetPrescriptionDetail(ctx, uid)
	okOrFail(ctx, c, resp, err)
}

// GetSubgraph GET /api/v1/graph/subgraph?center_uid=&depth=&limit=
func (h *QueryController) GetSubgraph(ctx context.Context, c *app.RequestContext) {
	centerUID := queryString(c, "center_uid")
	depth := queryInt(c, "depth", 2)
	limit := queryInt(c, "limit", 100)
	resp, err := h.uc.GetSubgraph(ctx, centerUID, depth, limit)
	okOrFail(ctx, c, resp, err)
}
