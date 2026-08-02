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
		{Name: "limit", Type: QueryParameterInteger, Description: "Page size (minimum 1; values above 200 are clamped to 200). Defaults to 100."},
		{Name: "offset", Type: QueryParameterInteger, Description: "Number of rows to skip. Defaults to 0."},
	}
}

func withPaging(parameters ...QueryParameter) []QueryParameter {
	return append(parameters, pagingParameters()...)
}

func taskTypeEnum() []string {
	return []string{
		"general", "code_review", "security_review", "product_review", "ui_ux_review", "qa_testing",
		"document_review", "documentation_writing", "diagram_writing", "planning", "research", "data_extraction",
		"troubleshooting", "code_analysis", "architecture_review", "threat_analysis",
	}
}

// taskFilterParameters is the shared task filter set parsed by
// parseTaskListFilters, reused by every endpoint that lists tasks (the task
// listing itself plus the team-work view; scope parameters are declared only
// where the handler reads them).
func taskFilterParameters() []QueryParameter {
	return []QueryParameter{
		{Name: "state", Type: QueryParameterString, Description: "Task state filter; repeatable to match several states.", Enum: []string{"draft", "open", "closed", "cancelled", "expired"}},
		{Name: "participation_policy", Type: QueryParameterString, Description: "Participation policy filter.", Enum: []string{"open", "reservation_required"}},
		{Name: "task_type", Type: QueryParameterString, Description: "Task type filter.", Enum: taskTypeEnum()},
		{Name: "query", Type: QueryParameterString, Description: "Substring match on title and description."},
		{Name: "sort", Type: QueryParameterString, Description: "Sort order. Defaults to newest.", Enum: []string{"newest", "oldest", "title_asc", "title_desc", "reward_desc", "reward_asc"}, Default: "newest"},
		{Name: "created_after", Type: QueryParameterString, Description: "RFC3339 instant; only tasks created strictly after it are listed."},
		{Name: "funded", Type: QueryParameterString, Description: "Funded-state filter; absent lists tasks regardless of reward funding.", Enum: []string{"reward_funded", "reward_unfunded", "no_credit_reward"}},
	}
}

// namedQueryParameter declares the free-text "query" search parameter with an
// endpoint-specific description.
func namedQueryParameter(description string) QueryParameter {
	return QueryParameter{Name: "query", Type: QueryParameterString, Description: description}
}

// pagingOnly declares endpoints whose only query parameters are limit/offset.
func pagingOnly(operationIDs ...string) []EndpointQueryParameters {
	declared := make([]EndpointQueryParameters, 0, len(operationIDs))
	for _, operationID := range operationIDs {
		declared = append(declared, EndpointQueryParameters{OperationID: operationID, Parameters: pagingParameters()})
	}
	return declared
}

// QueryParameters returns the declared query parameters for every list or
// filter endpoint the OpenAPI document names them on.
func QueryParameters() []EndpointQueryParameters {
	declared := []EndpointQueryParameters{
		{
			OperationID: "listTasks",
			Parameters: withPaging(append([]QueryParameter{
				{Name: "scope", Type: QueryParameterString, Description: "Listing scope. Defaults to public. scope=organization requires organization_id; scope=team requires team_id.", Enum: []string{"public", "user", "organization", "team"}, Default: "public"},
				{Name: "organization_id", Type: QueryParameterString, Description: "Organization to list for; required with scope=organization."},
				{Name: "team_id", Type: QueryParameterString, Description: "Team to list for; required with scope=team."},
			}, append(taskFilterParameters(),
				QueryParameter{Name: "include_reserved", Type: QueryParameterBoolean, Description: "Include open tasks another worker has actively reserved. Defaults to false."},
			)...)...),
		},
		{
			// The team-work view reuses the shared task filter set; the team
			// comes from the path and reserved tasks are always included.
			OperationID: "getTeamWork",
			Parameters:  withPaging(taskFilterParameters()...),
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
				{Name: "wait", Type: QueryParameterInteger, Description: "Long-poll hold in whole seconds (0-25; larger values are capped). When the page would be empty, the server holds the request until an event arrives or the wait elapses."},
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
		{
			// The admin audit listing filters are free strings server-side
			// (unknown values match nothing rather than erroring), so none of
			// them declares an enum.
			OperationID: "listAuditEvents",
			Parameters: withPaging(
				QueryParameter{Name: "action", Type: QueryParameterString, Description: "Audit action filter (for example admin_credit_granted or submission_accepted); absent lists every action."},
				QueryParameter{Name: "subject_kind", Type: QueryParameterString, Description: "Subject kind filter (for example user, organization, task, submission, collectible, moderation_report, privacy_request)."},
				QueryParameter{Name: "subject_id", Type: QueryParameterString, Description: "Subject id filter; combine with subject_kind to trace one record."},
			),
		},
		{
			OperationID: "listUsers",
			Parameters:  withPaging(namedQueryParameter("Substring match over the user directory (display name and email); at most 160 characters.")),
		},
		{
			OperationID: "listSavedQueueViews",
			Parameters: withPaging(
				QueryParameter{Name: "scope", Type: QueryParameterString, Description: "Scope filter; absent lists every saved view.", Enum: []string{"team_work", "organization_tasks"}},
			),
		},
		{
			OperationID: "listOrganizations",
			Parameters:  withPaging(namedQueryParameter("Substring match on organization names.")),
		},
		{
			OperationID: "listOrganizationTeams",
			Parameters:  withPaging(namedQueryParameter("Substring match on team names.")),
		},
		{
			OperationID: "listStandaloneTeams",
			Parameters:  withPaging(namedQueryParameter("Substring match on team names.")),
		},
	}
	return append(declared, pagingOnly(
		"creditsLedger",
		"organizationCreditsLedger",
		"listTaskSubmissions",
		"getUserSubmissions",
		"getUserWork",
		"listOrganizationAuditEvents",
		"listOrganizationMembers",
		"listCollectibles",
		"listOrganizationCollectibles",
		"listTeamCollectibles",
		"listAgentCredentials",
		"listOrgCredentials",
		"listPrivacyRequests",
		"listAdminPrivacyRequests",
		"listTaskSeries",
		"listTaskComments",
		"listSubmissionComments",
		"listTaskSeriesComments",
		"listPlatformAdmins",
		"listTaskReservations",
	)...)
}
