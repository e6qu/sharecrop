package agent

import "github.com/e6qu/sharecrop/internal/core"

// Credential is an opaque, scoped agent access token owned by a user.
type Credential struct {
	ID     core.AgentCredentialID
	UserID core.UserID
	Label  Label
	Scopes ScopeSet
	State  State
}

// ScopeSet is an unordered, de-duplicated collection of granted scopes.
type ScopeSet struct {
	values []Scope
}

func NewScopeSet(scopes []Scope) ScopeSet {
	unique := make([]Scope, 0, len(scopes))
	for _, scope := range scopes {
		if containsScope(unique, scope) {
			continue
		}
		unique = append(unique, scope)
	}
	return ScopeSet{values: unique}
}

func (set ScopeSet) Values() []Scope {
	copied := make([]Scope, len(set.values))
	copy(copied, set.values)
	return copied
}

func (set ScopeSet) Allows(scope Scope) ScopeCheck {
	if containsScope(set.values, scope) {
		return ScopeGranted{}
	}
	return ScopeDenied{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "agent credential is missing the "+scope.String()+" scope")}
}

func (set ScopeSet) IsEmpty() bool {
	return len(set.values) == 0
}

func containsScope(scopes []Scope, scope Scope) bool {
	for _, existing := range scopes {
		if existing == scope {
			return true
		}
	}
	return false
}

type ScopeCheck interface {
	scopeCheck()
}

type ScopeGranted struct{}

type ScopeDenied struct {
	Reason core.DomainError
}

func (ScopeGranted) scopeCheck() {}

func (ScopeDenied) scopeCheck() {}
