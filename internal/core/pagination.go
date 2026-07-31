package core

const (
	pageLimitMinimum  = 1
	pageLimitMaximum  = 200
	pageDefaultLimit  = 100
	pageDefaultOffset = 0
	pageOffsetMinimum = 0
)

type Page struct {
	limit  int
	offset int
}

func (p Page) Limit() int {
	return p.limit
}

func (p Page) Offset() int {
	return p.offset
}

func DefaultPage() Page {
	return Page{limit: pageDefaultLimit, offset: pageDefaultOffset}
}

// Probe returns a page that fetches one row beyond this page's limit, so a
// list handler can detect whether a further page exists without a second
// query. The probe limit deliberately bypasses the public maximum: it is an
// internal fetch size, never a client-supplied limit. Callers truncate the
// fetched rows back to Limit() and report the extra row through next_offset.
func (p Page) Probe() Page {
	return Page{limit: p.limit + 1, offset: p.offset}
}

// ProbeListWindow sizes a list page fetched with Page.Probe() (limit+1 rows).
// It reports how many fetched rows belong on this page and the response's
// next_offset: 0 when this is the last page, offset+limit otherwise. A real
// further page never yields 0 because offset is non-negative and limit is
// positive.
func ProbeListWindow(fetched int, page Page) (visible int, nextOffset int) {
	if fetched > page.Limit() {
		return page.Limit(), page.Offset() + page.Limit()
	}
	return fetched, 0
}

type PageResult interface {
	pageResult()
}

type PageAccepted struct {
	Value Page
}

type PageRejected struct {
	Reason DomainError
}

func (PageAccepted) pageResult() {}

func (PageRejected) pageResult() {}

func NewPage(limit int, offset int) PageResult {
	if limit < pageLimitMinimum {
		return PageRejected{Reason: NewDomainError(ErrorCodeInvalidArgument, "page limit must be positive")}
	}
	if offset < pageOffsetMinimum {
		return PageRejected{Reason: NewDomainError(ErrorCodeInvalidArgument, "page offset must not be negative")}
	}
	clampedLimit := limit
	if clampedLimit > pageLimitMaximum {
		clampedLimit = pageLimitMaximum
	}
	return PageAccepted{Value: Page{limit: clampedLimit, offset: offset}}
}
