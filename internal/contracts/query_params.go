package contracts

// This file declares the documented query parameters of the list/filter REST
// endpoints, keyed by the handler function name that the OpenAPI extractor
// already uses as the operationId. The OpenAPI generator emits these
// declarations into docs/openapi.json, so parameter documentation lives with
// the other wire contracts instead of being hand-maintained inside the
// generator. Enum tags are string literals by the same convention the Elm
// contract enums use: the duplicated tags are pinned by tests and generation
// diffs rather than by importing the domain packages.

// QueryParameterType is the JSON schema type of one query parameter value.
type QueryParameterType struct {
	value string
}

var (
	QueryParameterString  = QueryParameterType{value: "string"}
	QueryParameterInteger = QueryParameterType{value: "integer"}
	QueryParameterBoolean = QueryParameterType{value: "boolean"}
)

func (parameterType QueryParameterType) String() string {
	return parameterType.value
}

// QueryParameter declares one optional query parameter: its name, value
// type, prose description, and (when the value is a closed set) its enum
// tags and served default.
type QueryParameter struct {
	Name        string
	Type        QueryParameterType
	Description string
	Enum        []string
	Default     string
}

// EndpointQueryParameters binds declared query parameters to one REST
// handler by its operation id.
type EndpointQueryParameters struct {
	OperationID string
	Parameters  []QueryParameter
}

func pagingParameters() []QueryParameter {
	return []QueryParameter{
		{Name: "limit", Type: QueryParameterInteger, Description: "Page size (1-100). Defaults to the server page size."},
		{Name: "offset", Type: QueryParameterInteger, Description: "Number of rows to skip. Defaults to 0."},
	}
}

func withPaging(parameters ...QueryParameter) []QueryParameter {
	return append(parameters, pagingParameters()...)
}

func taskTypeEnum() []string {
	return []string{"general", "code_review", "security_review", "product_review", "ui_ux_review", "qa_testing"}
}

// QueryParameters returns the declared query parameters for every list or
// filter endpoint the OpenAPI document names them on.
func QueryParameters() []EndpointQueryParameters {
	return []EndpointQueryParameters{
		{
			OperationID: "listTasks",
			Parameters: withPaging(
				QueryParameter{Name: "scope", Type: QueryParameterString, Description: "Listing scope. Defaults to public. scope=organization requires organization_id; scope=team requires team_id.", Enum: []string{"public", "user", "organization", "team"}, Default: "public"},
				QueryParameter{Name: "organization_id", Type: QueryParameterString, Description: "Organization to list for; required with scope=organization."},
				QueryParameter{Name: "team_id", Type: QueryParameterString, Description: "Team to list for; required with scope=team."},
				QueryParameter{Name: "state", Type: QueryParameterString, Description: "Task state filter; repeatable to match several states.", Enum: []string{"draft", "open", "closed", "cancelled", "expired"}},
				QueryParameter{Name: "participation_policy", Type: QueryParameterString, Description: "Participation policy filter.", Enum: []string{"open", "reservation_required", "approval_required"}},
				QueryParameter{Name: "task_type", Type: QueryParameterString, Description: "Task type filter.", Enum: taskTypeEnum()},
				QueryParameter{Name: "query", Type: QueryParameterString, Description: "Substring match on title and description."},
				QueryParameter{Name: "sort", Type: QueryParameterString, Description: "Sort order. Defaults to newest.", Enum: []string{"newest", "oldest", "title_asc", "title_desc", "reward_desc", "reward_asc"}, Default: "newest"},
				QueryParameter{Name: "created_after", Type: QueryParameterString, Description: "RFC3339 instant; only tasks created strictly after it are listed."},
				QueryParameter{Name: "include_reserved", Type: QueryParameterBoolean, Description: "Include open tasks another worker has actively reserved. Defaults to false."},
			),
		},
		{
			OperationID: "listNotifications",
			Parameters: withPaging(
				QueryParameter{Name: "state", Type: QueryParameterString, Description: "State filter; absent lists the whole inbox.", Enum: []string{"unread"}},
			),
		},
		{
			OperationID: "listEvents",
			Parameters: []QueryParameter{
				{Name: "after", Type: QueryParameterString, Description: "Resume cursor; absent starts from the beginning of the caller's visible stream."},
				{Name: "limit", Type: QueryParameterInteger, Description: "Page size (1-100). Defaults to the server page size."},
			},
		},
		{
			OperationID: "streamEvents",
			Parameters: []QueryParameter{
				{Name: "after", Type: QueryParameterString, Description: "Resume cursor; the Last-Event-ID header takes precedence on reconnect."},
				{Name: "limit", Type: QueryParameterInteger, Description: "Batch size per poll (1-100). Defaults to the server page size."},
			},
		},
		{
			OperationID: "creditsLedger",
			Parameters:  pagingParameters(),
		},
		{
			OperationID: "organizationCreditsLedger",
			Parameters:  pagingParameters(),
		},
		{
			OperationID: "listTaskSubmissions",
			Parameters:  pagingParameters(),
		},
		{
			OperationID: "getUserSubmissions",
			Parameters:  pagingParameters(),
		},
		{
			OperationID: "listWebhookSubscriptions",
			Parameters: withPaging(
				QueryParameter{Name: "organization_id", Type: QueryParameterString, Description: "List an organization's subscriptions instead of the caller's own."},
			),
		},
		{
			OperationID: "listWebhookDeliveries",
			Parameters: withPaging(
				QueryParameter{Name: "organization_id", Type: QueryParameterString, Description: "Act for an organization-owned subscription instead of a personal one."},
			),
		},
		{
			OperationID: "listAdminModerationReports",
			Parameters: withPaging(
				QueryParameter{Name: "state", Type: QueryParameterString, Description: "Triage state filter; absent lists every report.", Enum: []string{"open", "resolved", "dismissed"}},
			),
		},
	}
}
