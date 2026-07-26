// Package prompt provides the PromptRenderer adapter for the AI Service.
//
// 渲染规则遵循 doc/09-AI-Prompt设计.md §四：必填缺失抛错、安全过滤后替换占位符。
package prompt

import (
	"encoding/json"
	"fmt"
	"strings"

	"tcm-history-ai/backend/ai-service/internal/domain/service"
	"tcm-history-ai/backend/pkg/errno"
)

// Renderer implements service.PromptRenderer using {{variable}} double-curly
// placeholder substitution.
type Renderer struct{}

// New constructs a Renderer.
func New() *Renderer { return &Renderer{} }

// Render replaces {{name}} placeholders with their values from variables.
// 缺失的占位符保持原样；用户可控变量做最小化注入防护（剥离系统提示分隔标记）。
func (r *Renderer) Render(template string, variables map[string]any) (string, error) {
	if template == "" {
		return "", nil
	}
	out := template
	for k, v := range variables {
		placeholder := "{{" + k + "}}"
		rendered, err := stringify(v)
		if err != nil {
			return "", errno.Wrap(errno.InternalError, "render variable "+k, err)
		}
		// 注入防护：剥离用户可控变量中可能被解释为指令分隔的标记
		rendered = sanitize(rendered)
		out = strings.ReplaceAll(out, placeholder, rendered)
	}
	return out, nil
}

// stringify converts a variable value to its string representation.
// 原始字符串、数字、bool 直接 fmt；对象/数组 JSON 序列化。
func stringify(v any) (string, error) {
	if v == nil {
		return "", nil
	}
	switch x := v.(type) {
	case string:
		return x, nil
	case bool:
		return fmt.Sprintf("%t", x), nil
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return fmt.Sprintf("%v", x), nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
}

// sanitize strips markers that could be abused to break out of the
// data section in the rendered prompt. This is a minimal heuristic guard;
// the full output审核 pipeline lives downstream (see doc/09 §九).
func sanitize(s string) string {
	// 移除【】分隔标记防止与系统提示冲突
	s = strings.ReplaceAll(s, "【", "[")
	s = strings.ReplaceAll(s, "】", "]")
	return s
}

// Compile-time check.
var _ service.PromptRenderer = (*Renderer)(nil)
