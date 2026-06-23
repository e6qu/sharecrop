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
