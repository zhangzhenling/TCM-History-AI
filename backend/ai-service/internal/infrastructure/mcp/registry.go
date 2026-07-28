package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"tcm-history-ai/backend/ai-service/internal/domain/entity"
	"tcm-history-ai/backend/ai-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/pagination"
)

// ToolMeta carries the full metadata for an MCP Tool, extending entity.Tool
// with schema and routing information.
type ToolMeta struct {
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	BackendService string          `json:"backend_service"`
	RPCMethod      string          `json:"rpc_method"`
	InputSchema    json.RawMessage `json:"input_schema"`
	OutputSchema   json.RawMessage `json:"output_schema"`
	RequiredScopes []string        `json:"required_scopes"`
	TimeoutMs      int             `json:"timeout_ms"`
	Degradation    string          `json:"degradation"` // none | cache | fallback
	Enabled        bool            `json:"enabled"`
	Version        string          `json:"version"`
}

// Registry holds the in-memory tool registry and supports hot-reloading.
type Registry struct {
	mu     sync.RWMutex
	tools  map[string]ToolMeta
	repo   repository.ToolRepository
	scopes map[string][]string // tool name -> required scopes (runtime override)
}

// NewRegistry constructs an empty registry backed by the given repository.
func NewRegistry(repo repository.ToolRepository) *Registry {
	return &Registry{
		tools:  make(map[string]ToolMeta),
		repo:   repo,
		scopes: make(map[string][]string),
	}
}

// Load fetches all enabled tools from the repository and rebuilds the registry.
func (r *Registry) Load(ctx context.Context) error {
	// Load all enabled tools without pagination for registry population.
	items, _, err := r.repo.ListEnabled(ctx, paginationZero())
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	clear(r.tools)
	for i := range items {
		t := items[i]
		meta := toolMetaFromEntity(&t)
		if meta.Enabled {
			r.tools[meta.Name] = meta
		}
	}
	return nil
}

// Register adds or replaces a tool in the registry without persisting to DB.
// Useful for registering the built-in TCM tools at startup.
func (r *Registry) Register(meta ToolMeta) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[meta.Name] = meta
}

// Unregister removes a tool from the registry.
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tools, name)
}

// Get looks up a single tool by name.
func (r *Registry) Get(name string) (ToolMeta, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.tools[name]
	return m, ok
}

// List returns all enabled tools in the registry, optionally filtered by scope.
func (r *Registry) List(haveScopes []string) []ToolMeta {
	r.mu.RLock()
	defer r.mu.RUnlock()

	set := make(map[string]struct{}, len(haveScopes))
	for _, s := range haveScopes {
		set[s] = struct{}{}
	}

	out := make([]ToolMeta, 0, len(r.tools))
	for _, m := range r.tools {
		if !m.Enabled {
			continue
		}
		if !scopesSatisfy(set, m.RequiredScopes) {
			continue
		}
		out = append(out, m)
	}
	return out
}

// scopesSatisfy reports whether the caller's scope set covers all required scopes.
func scopesSatisfy(set map[string]struct{}, required []string) bool {
	if len(required) == 0 {
		return true
	}
	for _, s := range required {
		if _, ok := set[s]; !ok {
			return false
		}
	}
	return true
}

// toolMetaFromEntity maps a GORM entity to ToolMeta.
// The extra MCP fields (backend_service, rpc_method, schemas, scopes, timeout,
// degradation) are stored in the entity's ParametersJSON as an envelope so that
// the existing ai_tools table does not need a schema migration.
func toolMetaFromEntity(t *entity.Tool) ToolMeta {
	m := ToolMeta{
		Name:        t.Name,
		Description: t.Description,
		Enabled:     t.IsEnabled,
		Version:     t.Version,
	}
	// Attempt to unmarshal the extended MCP envelope from parameters_json.
	var env struct {
		BackendService string          `json:"backend_service"`
		RPCMethod      string          `json:"rpc_method"`
		InputSchema    json.RawMessage `json:"input_schema"`
		OutputSchema   json.RawMessage `json:"output_schema"`
		RequiredScopes []string        `json:"required_scopes"`
		TimeoutMs      int             `json:"timeout_ms"`
		Degradation    string          `json:"degradation"`
	}
	_ = json.Unmarshal(t.ParametersJSON, &env)
	m.BackendService = env.BackendService
	m.RPCMethod = env.RPCMethod
	m.InputSchema = env.InputSchema
	m.OutputSchema = env.OutputSchema
	m.RequiredScopes = env.RequiredScopes
	m.TimeoutMs = env.TimeoutMs
	if m.TimeoutMs <= 0 {
		m.TimeoutMs = 3000
	}
	m.Degradation = env.Degradation
	if m.Degradation == "" {
		m.Degradation = "none"
	}
	return m
}

// ToEntity converts ToolMeta back to entity.Tool for persistence.
func (m ToolMeta) ToEntity(id int64) *entity.Tool {
	env := struct {
		BackendService string          `json:"backend_service"`
		RPCMethod      string          `json:"rpc_method"`
		InputSchema    json.RawMessage `json:"input_schema"`
		OutputSchema   json.RawMessage `json:"output_schema"`
		RequiredScopes []string        `json:"required_scopes"`
		TimeoutMs      int             `json:"timeout_ms"`
		Degradation    string          `json:"degradation"`
	}{
		BackendService: m.BackendService,
		RPCMethod:      m.RPCMethod,
		InputSchema:    m.InputSchema,
		OutputSchema:   m.OutputSchema,
		RequiredScopes: m.RequiredScopes,
		TimeoutMs:      m.TimeoutMs,
		Degradation:    m.Degradation,
	}
	paramsJSON, _ := json.Marshal(env)
	t := &entity.Tool{
		Name:           m.Name,
		Description:    m.Description,
		ParametersJSON: paramsJSON,
		Category:       m.BackendService,
		IsEnabled:      m.Enabled,
		Version:        m.Version,
	}
	// ID is defined on the embedded BaseModel; set it explicitly to avoid
	// "unknown field" errors in composite struct literals across packages.
	t.ID = id
	return t
}

// paginationZero returns a zero-value pagination for internal loading.
func paginationZero() pagination.Params {
	return pagination.Params{}
}

// BuiltInTools returns the 8 pre-defined TCM tools per doc/08-MCP设计.md §8.3.
func BuiltInTools() []ToolMeta {
	return []ToolMeta{
		{
			Name:           "tcm.timeline.query",
			Description:    "按朝代或起止年份查询中医历史事件时间轴",
			BackendService: "history-service",
			RPCMethod:      "HistoryEventService.QueryTimeline",
			InputSchema:    json.RawMessage(`{"type":"object","properties":{"dynasty":{"type":"string"},"start_year":{"type":"integer"},"end_year":{"type":"integer"},"category":{"type":"string","enum":["medical","political","cultural"]},"limit":{"type":"integer","default":20,"maximum":100}},"oneOf":[{"required":["dynasty"]},{"required":["start_year","end_year"]}]}`),
			OutputSchema:   json.RawMessage(`{"type":"object","properties":{"events":{"type":"array","items":{"type":"object","properties":{"event_id":{"type":"integer"},"title":{"type":"string"},"year":{"type":"integer"},"dynasty":{"type":"string"},"summary":{"type":"string"},"related_persons":{"type":"array","items":{"type":"string"}},"related_books":{"type":"array","items":{"type":"string"}}}}},"degraded":{"type":"boolean"}}}`),
			RequiredScopes: []string{"history:read"},
			TimeoutMs:      3000,
			Degradation:    "cache",
			Enabled:        true,
			Version:        "v1",
		},
		{
			Name:           "tcm.person.query",
			Description:    "查询历史人物信息与师承关系",
			BackendService: "history-service",
			RPCMethod:      "PersonService.Query",
			InputSchema:    json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"},"person_id":{"type":"integer"},"include_lineage":{"type":"boolean","default":true},"lineage_depth":{"type":"integer","default":2,"maximum":5}},"oneOf":[{"required":["name"]},{"required":["person_id"]}]}`),
			OutputSchema:   json.RawMessage(`{"type":"object","properties":{"person":{"type":"object","properties":{"person_id":{"type":"integer"},"name":{"type":"string"},"courtesy_name":{"type":"string"},"alias":{"type":"string"},"dynasty":{"type":"string"},"birth_year":{"type":"integer"},"death_year":{"type":"integer"},"hometown":{"type":"string"},"school":{"type":"string"},"major_works":{"type":"array","items":{"type":"string"}},"biography":{"type":"string"}}},"lineage":{"type":"object","properties":{"masters":{"type":"array","items":{"type":"string"}},"disciples":{"type":"array","items":{"type":"string"}}}},"lineage_degraded":{"type":"boolean"}}}`),
			RequiredScopes: []string{"person:read"},
			TimeoutMs:      2500,
			Degradation:    "fallback",
			Enabled:        true,
			Version:        "v1",
		},
		{
			Name:           "tcm.school.query",
			Description:    "查询学派信息与代表人物",
			BackendService: "knowledge-service",
			RPCMethod:      "SchoolService.Query",
			InputSchema:    json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"},"dynasty":{"type":"string"},"include_representatives":{"type":"boolean","default":true}},"required":["name"]}`),
			OutputSchema:   json.RawMessage(`{"type":"object","properties":{"school":{"type":"object","properties":{"school_id":{"type":"integer"},"name":{"type":"string"},"founded_dynasty":{"type":"string"},"core_theory":{"type":"string"},"classics":{"type":"array","items":{"type":"string"}},"representatives":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"dynasty":{"type":"string"},"contribution":{"type":"string"}}}}}}}}`),
			RequiredScopes: []string{"school:read"},
			TimeoutMs:      2000,
			Degradation:    "cache",
			Enabled:        true,
			Version:        "v1",
		},
		{
			Name:           "tcm.classic.query",
			Description:    "查询经典著作内容与章节",
			BackendService: "knowledge-service",
			RPCMethod:      "ClassicService.Query",
			InputSchema:    json.RawMessage(`{"type":"object","properties":{"title":{"type":"string"},"book_id":{"type":"integer"},"chapter_id":{"type":"integer"},"keyword":{"type":"string"},"return_content":{"type":"boolean","default":true}},"oneOf":[{"required":["title"]},{"required":["book_id"]}]}`),
			OutputSchema:   json.RawMessage(`{"type":"object","properties":{"book":{"type":"object","properties":{"book_id":{"type":"integer"},"title":{"type":"string"},"author":{"type":"string"},"dynasty":{"type":"string"},"completed_year":{"type":"integer"},"volumes":{"type":"integer"},"summary":{"type":"string"}}},"chapters":{"type":"array","items":{"type":"object","properties":{"chapter_id":{"type":"integer"},"title":{"type":"string"},"content":{"type":"string"}}}},"content_degraded":{"type":"boolean"}}}`),
			RequiredScopes: []string{"book:read"},
			TimeoutMs:      3000,
			Degradation:    "fallback",
			Enabled:        true,
			Version:        "v1",
		},
		{
			Name:           "tcm.graph.path",
			Description:    "查询知识图谱关联路径",
			BackendService: "graph-service",
			RPCMethod:      "GraphService.FindPath",
			InputSchema:    json.RawMessage(`{"type":"object","properties":{"source":{"type":"object","properties":{"type":{"type":"string","enum":["Person","Book","School","Prescription","Medicine","Disease","Dynasty","Event"]},"name":{"type":"string"}},"required":["type","name"]},"target":{"type":"object","properties":{"type":{"type":"string","enum":["Person","Book","School","Prescription","Medicine","Disease","Dynasty","Event"]},"name":{"type":"string"}},"required":["type","name"]},"max_hops":{"type":"integer","default":3,"maximum":6},"rel_types":{"type":"array","items":{"type":"string"}}},"required":["source","target"]}`),
			OutputSchema:   json.RawMessage(`{"type":"object","properties":{"paths":{"type":"array","items":{"type":"object","properties":{"length":{"type":"integer"},"nodes":{"type":"array","items":{"type":"object","properties":{"type":{"type":"string"},"name":{"type":"string"}}}}},"edges":{"type":"array","items":{"type":"object","properties":{"type":{"type":"string"},"direction":{"type":"string","enum":["out","in"]}}}}}}},"graph_degraded":{"type":"boolean"}}}`),
			RequiredScopes: []string{"graph:read"},
			TimeoutMs:      4000,
			Degradation:    "fallback",
			Enabled:        true,
			Version:        "v1",
		},
		{
			Name:           "tcm.search",
			Description:    "全文检索中医内容",
			BackendService: "knowledge-service",
			RPCMethod:      "SearchService.Search",
			InputSchema:    json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"},"indexes":{"type":"array","items":{"type":"string","enum":["book","person","prescription","medicine"]},"default":["book","person"]},"filters":{"type":"object","properties":{"dynasty":{"type":"string"},"school":{"type":"string"}}},"limit":{"type":"integer","default":10,"maximum":50}},"required":["query"]}`),
			OutputSchema:   json.RawMessage(`{"type":"object","properties":{"hits":{"type":"array","items":{"type":"object","properties":{"index":{"type":"string"},"id":{"type":"integer"},"title":{"type":"string"},"snippet":{"type":"string"},"score":{"type":"number"}}}}},"estimated_total":{"type":"integer"},"search_degraded":{"type":"boolean"}}}`),
			RequiredScopes: []string{"search:read"},
			TimeoutMs:      2000,
			Degradation:    "fallback",
			Enabled:        true,
			Version:        "v1",
		},
		{
			Name:           "tcm.medicine.query",
			Description:    "查询中药信息",
			BackendService: "graph-service",
			RPCMethod:      "MedicineService.Query",
			InputSchema:    json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"},"medicine_id":{"type":"integer"},"include_prescriptions":{"type":"boolean","default":false}},"oneOf":[{"required":["name"]},{"required":["medicine_id"]}]}`),
			OutputSchema:   json.RawMessage(`{"type":"object","properties":{"medicine":{"type":"object","properties":{"medicine_id":{"type":"integer"},"name":{"type":"string"},"pinyin":{"type":"string"},"nature":{"type":"string"},"flavor":{"type":"string"},"meridians":{"type":"array","items":{"type":"string"}},"efficacy":{"type":"string"},"indications":{"type":"array","items":{"type":"string"}},"source":{"type":"string"},"contraindication":{"type":"string"}}},"prescriptions":{"type":"array","items":{"type":"string"}}}}`),
			RequiredScopes: []string{"medicine:read"},
			TimeoutMs:      2000,
			Degradation:    "cache",
			Enabled:        true,
			Version:        "v1",
		},
		{
			Name:           "tcm.prescription.query",
			Description:    "查询方剂组成与主治",
			BackendService: "graph-service",
			RPCMethod:      "PrescriptionService.Query",
			InputSchema:    json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"},"prescription_id":{"type":"integer"},"include_modifications":{"type":"boolean","default":false}},"oneOf":[{"required":["name"]},{"required":["prescription_id"]}]}`),
			OutputSchema:   json.RawMessage(`{"type":"object","properties":{"prescription":{"type":"object","properties":{"prescription_id":{"type":"integer"},"name":{"type":"string"},"source_book":{"type":"string"},"author":{"type":"string"},"dynasty":{"type":"string"},"composition":{"type":"array","items":{"type":"object","properties":{"medicine":{"type":"string"},"dosage":{"type":"string"},"role":{"type":"string","enum":["君","臣","佐","使"]}}}},"indications":{"type":"array","items":{"type":"string"}},"preparation":{"type":"string"},"usage":{"type":"string"},"modifications":{"type":"array","items":{"type":"object","properties":{"condition":{"type":"string"},"add":{"type":"array","items":{"type":"string"}},"remove":{"type":"array","items":{"type":"string"}}}}}}}}}},"composition_degraded":{"type":"boolean"}}}`),
			RequiredScopes: []string{"prescription:read"},
			TimeoutMs:      2500,
			Degradation:    "fallback",
			Enabled:        true,
			Version:        "v1",
		},
	}
}

// fmt.Stringer for ToolMeta.
func (m ToolMeta) String() string {
	return fmt.Sprintf("ToolMeta{%s %s/%s enabled=%v}", m.Name, m.BackendService, m.RPCMethod, m.Enabled)
}