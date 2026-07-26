package embedding_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/knowledge-service/internal/domain/service"
	"tcm-history-ai/backend/knowledge-service/internal/infrastructure/embedding"
	"tcm-history-ai/backend/pkg/errno"
)

// newStub builds a *StubProvider with the given model/dim via the public
// New constructor + type assertion, so tests don't reach into the unexported
// model/dim fields directly.
func newStub(t *testing.T, model string, dim int) *embedding.StubProvider {
	t.Helper()
	p, err := embedding.New(embedding.Config{Provider: "stub", Model: model, Dim: dim})
	require.NoError(t, err)
	sp, ok := p.(*embedding.StubProvider)
	require.True(t, ok, "expected *StubProvider, got %T", p)
	return sp
}

// TestNew_StubProvider exercises the "", "stub", and "local" branches which all
// return a StubProvider with the configured model/dim. provider="openai" now
// returns a real *OpenAIProvider (见 TestNew_OpenAIProvider)。
func TestNew_StubProvider(t *testing.T) {
	cases := []struct {
		name     string
		provider string
	}{
		{"empty defaults to stub", ""},
		{"explicit stub", "stub"},
		{"local falls back to stub", "local"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := embedding.New(embedding.Config{
				Provider: tc.provider,
				Model:    "bge-large-zh-v1.5",
				Dim:      1024,
			})
			require.NoError(t, err)
			require.NotNil(t, p)
			assert.Equal(t, "bge-large-zh-v1.5", p.Model())
			assert.Equal(t, 1024, p.Dim())
		})
	}
}

// TestNew_OpenAIProvider verifies that provider="openai" returns a real
// *OpenAIProvider (而非 stub 回退)，且 model/dim 透传正确。
func TestNew_OpenAIProvider(t *testing.T) {
	p, err := embedding.New(embedding.Config{
		Provider: "openai",
		Endpoint: "https://api.openai.com/v1",
		APIKey:   "sk-test",
		Model:    "text-embedding-3-small",
		Dim:      1536,
		Timeout:  10,
	})
	require.NoError(t, err)
	require.NotNil(t, p)
	_, ok := p.(*embedding.OpenAIProvider)
	assert.True(t, ok, "expected *OpenAIProvider, got %T", p)
	assert.Equal(t, "text-embedding-3-small", p.Model())
	assert.Equal(t, 1536, p.Dim())
}

// TestNew_OpenAIProvider_Defaults confirms empty endpoint/model/dim fall back
// to the OpenAI defaults inside NewOpenAIProvider.
func TestNew_OpenAIProvider_Defaults(t *testing.T) {
	p, err := embedding.New(embedding.Config{Provider: "openai", APIKey: "sk-test"})
	require.NoError(t, err)
	op, ok := p.(*embedding.OpenAIProvider)
	require.True(t, ok)
	assert.Contains(t, op.String(), "https://api.openai.com/v1") // String() 含 base URL
	assert.Contains(t, op.String(), "text-embedding-3-small")    // 默认 model
	assert.Contains(t, op.String(), "dim=1536")                  // 默认 dim
	assert.Equal(t, "text-embedding-3-small", p.Model())
	assert.Equal(t, 1536, p.Dim())
}

// TestNew_UnknownProvider verifies that an unrecognised provider string
// produces an InvalidParams *errno.Error.
func TestNew_UnknownProvider(t *testing.T) {
	p, err := embedding.New(embedding.Config{Provider: "totally-bogus"})
	require.Error(t, err)
	assert.Nil(t, p)
	var e *errno.Error
	require.True(t, errors.As(err, &e), "expected *errno.Error")
	assert.Equal(t, errno.InvalidParams, e.Code)
	assert.Contains(t, e.Message, "totally-bogus")
}

// TestStubProvider_EmptyInputReturnsNil exercises the len(texts)==0 fast path.
func TestStubProvider_EmptyInputReturnsNil(t *testing.T) {
	s := newStub(t, "stub-m", 4)
	out, err := s.Embed(context.Background(), nil)
	require.NoError(t, err)
	assert.Nil(t, out)
}

// TestStubProvider_Embed_DefaultDim confirms that a zero Dim field falls back
// to the 1024 default inside Embed.
func TestStubProvider_Embed_DefaultDim(t *testing.T) {
	s := newStub(t, "stub-m", 0)
	out, err := s.Embed(context.Background(), []string{"abc"})
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Len(t, out[0], 1024)
}

// TestStubProvider_Embed_DeterministicByLength verifies the stub seeds
// vec[0] with len(text)%dim/100, so equal-length texts produce equal vectors
// and differing lengths produce differing seeds.
func TestStubProvider_Embed_DeterministicByLength(t *testing.T) {
	s := newStub(t, "stub-m", 1024)

	short := "abc"
	long := strings.Repeat("a", 999)

	vecShort, err := s.Embed(context.Background(), []string{short})
	require.NoError(t, err)
	vecLong, err := s.Embed(context.Background(), []string{long})
	require.NoError(t, err)

	require.Len(t, vecShort, 1)
	require.Len(t, vecLong, 1)
	require.Len(t, vecShort[0], 1024)
	require.Len(t, vecLong[0], 1024)

	// Same length text should yield the same seed.
	vecShort2, err := s.Embed(context.Background(), []string{"def"})
	require.NoError(t, err)
	assert.Equal(t, vecShort[0][0], vecShort2[0][0])

	// Different lengths yield different seeds.
	assert.NotEqual(t, vecShort[0][0], vecLong[0][0])
}

// TestStubProvider_Embed_MultipleTexts verifies the stub returns one vector
// per input text.
func TestStubProvider_Embed_MultipleTexts(t *testing.T) {
	s := newStub(t, "stub-m", 8)
	out, err := s.Embed(context.Background(), []string{"a", "bb", "ccc"})
	require.NoError(t, err)
	require.Len(t, out, 3)
	for _, v := range out {
		assert.Len(t, v, 8)
	}
}

// TestStubProvider_Model_Dim confirms the simple accessors.
func TestStubProvider_Model_Dim(t *testing.T) {
	s := newStub(t, "m", 7)
	assert.Equal(t, "m", s.Model())
	assert.Equal(t, 7, s.Dim())
}

// TestStubProvider_String confirms the debug representation contains the model
// and dimension so logs can disambiguate instances.
func TestStubProvider_String(t *testing.T) {
	s := newStub(t, "m", 7)
	str := s.String()
	assert.Contains(t, str, "model=m")
	assert.Contains(t, str, "dim=7")
}

// TestNew_AssignsPort guarantees the returned provider satisfies the
// service.EmbeddingProvider interface (compile-time check at test scope).
func TestNew_AssignsPort(t *testing.T) {
	p, err := embedding.New(embedding.Config{Provider: "stub", Model: "m", Dim: 4})
	require.NoError(t, err)
	var _ service.EmbeddingProvider = p
}
