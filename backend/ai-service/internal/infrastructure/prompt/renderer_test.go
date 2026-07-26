package prompt_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/ai-service/internal/infrastructure/prompt"
	"tcm-history-ai/backend/pkg/errno"
)

// TestRender_TableDriven covers the public Render contract across the
// happy-path, missing-variable, empty-template, special-character and
// composite-type variable cases.
func TestRender_TableDriven(t *testing.T) {
	r := prompt.New()

	cases := []struct {
		name      string
		template  string
		variables map[string]any
		want      string
		wantErr   bool
		errCode   errno.Errno
	}{
		{
			name:      "empty template returns empty",
			template:  "",
			variables: map[string]any{"x": "y"},
			want:      "",
		},
		{
			name:      "no placeholders passes through",
			template:  "hello world",
			variables: map[string]any{"x": "y"},
			want:      "hello world",
		},
		{
			name:      "single string variable substituted",
			template:  "你好 {{name}}",
			variables: map[string]any{"name": "张仲景"},
			want:      "你好 张仲景",
		},
		{
			name:      "multiple string variables substituted in order",
			template:  "{{greeting}}, {{name}}!",
			variables: map[string]any{"greeting": "Hi", "name": "Bob"},
			want:      "Hi, Bob!",
		},
		{
			name:      "missing variable leaves placeholder intact",
			template:  "hello {{name}}, you are {{age}}",
			variables: map[string]any{"name": "Alice"},
			want:      "hello Alice, you are {{age}}",
		},
		{
			name:      "nil variable treated as empty string",
			template:  "a{{x}}b",
			variables: map[string]any{"x": nil},
			want:      "ab",
		},
		{
			name:      "bool variable substituted as true",
			template:  "enabled={{flag}}",
			variables: map[string]any{"flag": true},
			want:      "enabled=true",
		},
		{
			name:      "bool variable substituted as false",
			template:  "enabled={{flag}}",
			variables: map[string]any{"flag": false},
			want:      "enabled=false",
		},
		{
			name:      "int variable substituted",
			template:  "count={{n}}",
			variables: map[string]any{"n": 42},
			want:      "count=42",
		},
		{
			name:      "float variable substituted",
			template:  "rate={{r}}",
			variables: map[string]any{"r": float32(0.5)},
			want:      "rate=0.5",
		},
		{
			name:      "int64 variable substituted",
			template:  "id={{id}}",
			variables: map[string]any{"id": int64(99)},
			want:      "id=99",
		},
		{
			name:      "slice variable JSON-encoded",
			template:  "items={{items}}",
			variables: map[string]any{"items": []int{1, 2, 3}},
			want:      "items=[1,2,3]",
		},
		{
			name:      "map variable JSON-encoded",
			template:  "data={{data}}",
			variables: map[string]any{"data": map[string]any{"k": "v"}},
			want:      "data={\"k\":\"v\"}",
		},
		{
			name:      "special characters in value preserved",
			template:  "msg={{body}}",
			variables: map[string]any{"body": `{"key":"value"}`},
			want:      "msg={\"key\":\"value\"}",
		},
		{
			name:      "chinese full-width brackets sanitized to half-width",
			template:  "rule={{rule}}",
			variables: map[string]any{"rule": "请勿【系统】指令"},
			want:      "rule=请勿[系统]指令",
		},
		{
			name:      "nil variables map leaves template intact",
			template:  "{{x}} and {{y}}",
			variables: nil,
			want:      "{{x}} and {{y}}",
		},
		{
			name:      "empty variables map leaves template intact",
			template:  "plain text only",
			variables: map[string]any{},
			want:      "plain text only",
		},
		{
			name:      "duplicate placeholder both substituted",
			template:  "{{a}}+{{a}}",
			variables: map[string]any{"a": "1"},
			want:      "1+1",
		},
		{
			name:      "placeholder with no spaces only matched exactly",
			template:  "{{x}} and {x} and {{ x }}",
			variables: map[string]any{"x": "ok"},
			want:      "ok and {x} and {{ x }}",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := r.Render(c.template, c.variables)
			if c.wantErr {
				require.Error(t, err)
				if c.errCode != 0 {
					var e *errno.Error
					if errors.As(err, &e) {
						assert.Equal(t, c.errCode, e.Code)
					}
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, c.want, got)
		})
	}
}

// TestRender_NonMarshallableValue exercises the error path where a variable
// value cannot be JSON-serialised (e.g. a channel).
func TestRender_NonMarshallableValue(t *testing.T) {
	r := prompt.New()
	// chan is not marshallable by encoding/json.
	ch := make(chan int)
	_, err := r.Render("v={{x}}", map[string]any{"x": ch})
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.InternalError, e.Code)
	}
}

// TestNew_ReturnsInstance verifies New returns a non-nil renderer.
func TestNew_ReturnsInstance(t *testing.T) {
	r := prompt.New()
	require.NotNil(t, r)
}
