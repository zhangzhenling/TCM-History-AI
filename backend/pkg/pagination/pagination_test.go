package pagination_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"tcm-history-ai/backend/pkg/pagination"
)

// TestParams_Normalise verifies clamping behaviour for page<=0, pageSize<=0,
// pageSize>MaxPageSize, and offset computation.
func TestParams_Normalise(t *testing.T) {
	cases := []struct {
		name           string
		page, pageSize int
		wantPage       int
		wantPageSize   int
		wantOffset     int
	}{
		{
			name:         "valid values",
			page:         3,
			pageSize:     50,
			wantPage:     3,
			wantPageSize: 50,
			wantOffset:   100,
		},
		{
			name:         "page<=0 -> default page 1",
			page:         0,
			pageSize:     10,
			wantPage:     1,
			wantPageSize: 10,
			wantOffset:   0,
		},
		{
			name:         "negative page -> default page 1",
			page:         -5,
			pageSize:     10,
			wantPage:     1,
			wantPageSize: 10,
			wantOffset:   0,
		},
		{
			name:         "pageSize<=0 -> default page size",
			page:         2,
			pageSize:     0,
			wantPage:     2,
			wantPageSize: pagination.DefaultPageSize,
			wantOffset:   pagination.DefaultPageSize,
		},
		{
			name:         "negative pageSize -> default page size",
			page:         2,
			pageSize:     -3,
			wantPage:     2,
			wantPageSize: pagination.DefaultPageSize,
			wantOffset:   pagination.DefaultPageSize,
		},
		{
			name:         "pageSize over max -> capped",
			page:         1,
			pageSize:     pagination.MaxPageSize + 500,
			wantPage:     1,
			wantPageSize: pagination.MaxPageSize,
			wantOffset:   0,
		},
		{
			name:         "pageSize exactly max -> unchanged",
			page:         1,
			pageSize:     pagination.MaxPageSize,
			wantPage:     1,
			wantPageSize: pagination.MaxPageSize,
			wantOffset:   0,
		},
		{
			name:         "both zero -> defaults",
			page:         0,
			pageSize:     0,
			wantPage:     pagination.DefaultPage,
			wantPageSize: pagination.DefaultPageSize,
			wantOffset:   0,
		},
		{
			name:         "page 5 page_size 20 -> offset 80",
			page:         5,
			pageSize:     20,
			wantPage:     5,
			wantPageSize: 20,
			wantOffset:   80,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := pagination.Params{Page: c.page, PageSize: c.pageSize}
			gotPage, gotPageSize, gotOffset := p.Normalise()
			assert.Equal(t, c.wantPage, gotPage, "page")
			assert.Equal(t, c.wantPageSize, gotPageSize, "page size")
			assert.Equal(t, c.wantOffset, gotOffset, "offset")
		})
	}
}

// TestParams_Normalise_DoesNotMutateReceiver verifies Normalise is a pure
// function and does not mutate the receiver.
func TestParams_Normalise_DoesNotMutateReceiver(t *testing.T) {
	p := pagination.Params{Page: -1, PageSize: 50000}
	_, _, _ = p.Normalise()
	assert.Equal(t, -1, p.Page, "receiver Page should be unchanged")
	assert.Equal(t, 50000, p.PageSize, "receiver PageSize should be unchanged")
}

// TestFrom verifies the simple constructor.
func TestFrom(t *testing.T) {
	p := pagination.From(7, 25)
	assert.Equal(t, 7, p.Page)
	assert.Equal(t, 25, p.PageSize)
}

// TestNewPage_TotalPageCalc covers total page calculation including ceiling
// behaviour and the zero-pageSize branch.
func TestNewPage_TotalPageCalc(t *testing.T) {
	t.Run("exact division", func(t *testing.T) {
		page := pagination.NewPage(1, 20, 60, []int{})
		assert.Equal(t, 1, page.Page)
		assert.Equal(t, 20, page.PageSize)
		assert.Equal(t, 60, page.Total)
		assert.Equal(t, 3, page.TotalPage)
	})

	t.Run("non-exact division rounds up", func(t *testing.T) {
		page := pagination.NewPage(1, 20, 61, []int{})
		assert.Equal(t, 4, page.TotalPage, "61 items / 20 per page = 4 pages (ceil)")
	})

	t.Run("single item on second page", func(t *testing.T) {
		page := pagination.NewPage(2, 20, 21, []int{})
		assert.Equal(t, 2, page.TotalPage)
	})

	t.Run("zero total", func(t *testing.T) {
		page := pagination.NewPage(1, 20, 0, []int{})
		assert.Equal(t, 0, page.TotalPage)
	})

	t.Run("page<=0 defaults to 1", func(t *testing.T) {
		page := pagination.NewPage(0, 20, 0, []int{})
		assert.Equal(t, pagination.DefaultPage, page.Page)
	})

	t.Run("pageSize<=0 defaults to default page size", func(t *testing.T) {
		page := pagination.NewPage(1, 0, 0, []int{})
		assert.Equal(t, pagination.DefaultPageSize, page.PageSize)
	})

	t.Run("items propagated", func(t *testing.T) {
		items := []string{"a", "b", "c"}
		page := pagination.NewPage(1, 10, 3, items)
		assert.Equal(t, items, page.Items)
	})
}

// TestPage_String verifies the log-line helper.
func TestPage_String(t *testing.T) {
	page := pagination.NewPage(3, 25, 70, []string{"x"})
	got := page.String()
	assert.Contains(t, got, "page=3")
	assert.Contains(t, got, "size=25")
	assert.Contains(t, got, "total=70")
	assert.True(t, strings.HasPrefix(got, "page="))
}

// TestPage_GenericType verifies Page works with both value and pointer slices.
func TestPage_GenericType(t *testing.T) {
	t.Run("with int slice", func(t *testing.T) {
		page := pagination.NewPage(1, 10, 5, []int{1, 2, 3, 4, 5})
		assert.Equal(t, 5, len(page.Items))
	})

	t.Run("with struct slice", func(t *testing.T) {
		type item struct{ Name string }
		items := []item{{Name: "alice"}, {Name: "bob"}}
		page := pagination.NewPage(1, 10, 2, items)
		assert.Equal(t, 2, len(page.Items))
		assert.Equal(t, "alice", page.Items[0].Name)
	})
}
