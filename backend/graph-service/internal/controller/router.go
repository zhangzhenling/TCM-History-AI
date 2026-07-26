package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"tcm-history-ai/backend/pkg/response"
)

// Deps bundles every controller the router needs. It is populated by wire.
type Deps struct {
	Node  *NodeController
	Edge  *EdgeController
	Query *QueryController
	Sync  *SyncController
}

// RegisterRoutes wires every Graph Service route onto the Hertz server.
// Routes follow RESTful conventions under /api/v1/graph (doc/05 §5.7).
func RegisterRoutes(h *server.Hertz, deps *Deps) {
	h.GET("/health", func(ctx context.Context, c *app.RequestContext) {
		response.OKWith(ctx, c, "graph-service up", map[string]any{
			"service": "graph-service",
			"status":  "ok",
		})
	})

	v1 := h.Group("/api/v1/graph")

	// Nodes
	v1.GET("/nodes", deps.Node.List)
	v1.POST("/nodes", deps.Node.Create)
	v1.GET("/nodes/:uid", deps.Node.Get)
	v1.PUT("/nodes/:uid", deps.Node.Update)
	v1.DELETE("/nodes/:uid", deps.Node.Delete)

	// Edges
	v1.GET("/edges", deps.Edge.List)
	v1.POST("/edges", deps.Edge.Create)
	v1.GET("/edges/:uid", deps.Edge.Get)
	v1.PUT("/edges/:uid", deps.Edge.Update)
	v1.DELETE("/edges/:uid", deps.Edge.Delete)

	// Complex queries (doc/05 §5.5)
	v1.GET("/persons/:uid/works", deps.Query.GetPersonWorks)
	v1.GET("/schools/:name/lineage", deps.Query.GetSchoolLineage)
	v1.GET("/paths/shortest", deps.Query.FindShortestPath)
	v1.GET("/dynasties/:name/figures", deps.Query.GetDynastyFigures)
	v1.GET("/prescriptions/:uid/detail", deps.Query.GetPrescriptionDetail)
	v1.GET("/subgraph", deps.Query.GetSubgraph)
	v1.GET("/search", deps.Query.Search)

	// Sync
	v1.POST("/sync", deps.Sync.TriggerSync)

	// Suppress unused-import warning for consts.
	_ = consts.StatusOK
}
