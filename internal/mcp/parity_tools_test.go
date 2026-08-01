package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/task"
)

// capturingServices records the commands the tools build, so the parity
// tests can assert the MCP boundary produces the same task.CreateCommand
// semantics REST's decodeTaskRequest does.
type capturingServices struct {
	fakeServices
	createCommand  *task.CreateCommand
	listScope      *task.ListScope
	listFilters    *task.ListFilters
	listPage       *core.Page
	collectibleTip *ledger.CollectibleTipSelection
	reviewer       *ledger.Reviewer
	sendCommand    *capturedSendCommand
}

// capturedSendCommand records what callSendCredits passed to the service.
type capturedSendCommand struct {
	source ledger.TransferSource
	target ledger.TransferTarget
	amount int64
	note   ledger.TransferNote
	key    string
}

func (services *capturingServices) SendCredits(ctx context.Context, actor core.UserID, source ledger.TransferSource, target ledger.TransferTarget, amount ledger.CreditAmount, note ledger.TransferNote, key ledger.IdempotencyKey) ledger.SendResult {
	*services.sendCommand = capturedSendCommand{source: source, target: target, amount: amount.Int64(), note: note, key: key.String()}
	return services.fakeServices.SendCredits(ctx, actor, source, target, amount, note, key)
}

func (services *capturingServices) CreateTask(ctx context.Context, command task.CreateCommand) task.CreateResult {
	*services.createCommand = command
	return services.fakeServices.CreateTask(ctx, command)
}

func (services *capturingServices) ListTasks(ctx context.Context, subject auth.Subject, scope task.ListScope, filters task.ListFilters, page core.Page) task.ListResult {
	*services.listScope = scope
	*services.listFilters = filters
	*services.listPage = page
	return services.fakeServices.ListTasks(ctx, subject, scope, filters, page)
}

func (services *capturingServices) ReviewAcceptSubmission(ctx context.Context, reviewer ledger.Reviewer, taskID core.TaskID, submissionID core.SubmissionID, key ledger.IdempotencyKey, creditSelection ledger.CreditReviewSelection, tipSelection ledger.TipSelection, collectibleTip ledger.CollectibleTipSelection) ledger.AcceptResult {
	*services.collectibleTip = collectibleTip
	*services.reviewer = reviewer
	return services.fakeServices.ReviewAcceptSubmission(ctx, reviewer, taskID, submissionID, key, creditSelection, tipSelection, collectibleTip)
}

func newCapturingServices() *capturingServices {
	return &capturingServices{
		createCommand:  &task.CreateCommand{},
		listScope:      new(task.ListScope),
		listFilters:    &task.ListFilters{},
		listPage:       &core.Page{},
		collectibleTip: new(ledger.CollectibleTipSelection),
		reviewer:       new(ledger.Reviewer),
		sendCommand:    &capturedSendCommand{},
	}
}

func TestToolFailureRendersStructuredErrorJSON(t *testing.T) {
	server := NewServer(fakeServices{rejectGet: true})
	response := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.get_task","arguments":{"task_id":"`+testTaskID(t)+`"}}`))
	var result toolCallResult
	if err := json.Unmarshal(mustResult(t, response), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected isError tool result")
	}
	if len(result.Content) != 1 {
		t.Fatalf("content count = %d, want 1", len(result.Content))
	}
	var body toolErrorBody
	if err := json.Unmarshal([]byte(result.Content[0].Text), &body); err != nil {
		t.Fatalf("failure text is not JSON: %q", result.Content[0].Text)
	}
	if body.Code != core.ErrorCodeInvalidState.String() {
		t.Fatalf("code = %q, want %q", body.Code, core.ErrorCodeInvalidState.String())
	}
	if body.Message != "task view access denied" {
		t.Fatalf("message = %q", body.Message)
	}
}

func TestCreateTaskFullParityArguments(t *testing.T) {
	services := newCapturingServices()
	server := NewServer(services)
	subject := testSubject(t)

	organizationID := core.NewOrganizationID().(core.OrganizationIDCreated).Value
	teamID := core.NewTeamID().(core.TeamIDCreated).Value
	seriesID := core.NewTaskSeriesID().(core.TaskSeriesIDCreated).Value
	collectibleA := core.NewCollectibleID().(core.CollectibleIDCreated).Value
	collectibleB := core.NewCollectibleID().(core.CollectibleIDCreated).Value
	expiresAt := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Second)

	rawArguments := `{
		"title": "Review the release",
		"description": "Full parity create.",
		"response_schema_json": "{\"kind\":\"freeform\"}",
		"owner": {"kind": "organization_team", "organization_id": "` + organizationID.String() + `", "team_id": "` + teamID.String() + `"},
		"visibility_kind": "organization_team",
		"visibility_organization_id": "` + organizationID.String() + `",
		"visibility_team_id": "` + teamID.String() + `",
		"reward_kind": "bundle",
		"reward_credit_amount": 25,
		"reward_collectible_ids": ["` + collectibleA.String() + `", "` + collectibleB.String() + `"],
		"participation_policy": "reservation_required",
		"assignee_scope": "organization_team",
		"reservation_expiry_hours": 72,
		"task_type": "code_review",
		"reference_url": "https://example.com/pr/7",
		"series_id": "` + seriesID.String() + `",
		"series_position": 3,
		"payload_json": "{\"batch\":\"A\"}",
		"attachments": [{"name": "brief.txt", "content_type": "text/plain", "data_url": "data:text/plain;base64,aGVsbG8="}],
		"expires_at": "` + expiresAt.Format(time.RFC3339) + `"
	}`
	response := server.Handle(context.Background(), subject, CallerCredential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.create_task","arguments":`+rawArguments+`}`))
	if response.Error != nil {
		t.Fatalf("create_task error: %s", response.Error.Message)
	}

	command := *services.createCommand
	owner, ownerMatched := command.Owner.(task.OrganizationTeamOwner)
	if !ownerMatched || owner.OrganizationID != organizationID || owner.TeamID != teamID {
		t.Fatalf("owner = %#v", command.Owner)
	}
	visibility, visibilityMatched := command.Visibility.(task.OrganizationTeamVisibility)
	if !visibilityMatched || visibility.OrganizationID != organizationID || visibility.TeamID != teamID {
		t.Fatalf("visibility = %#v", command.Visibility)
	}
	reward, rewardMatched := command.Reward.(task.BundleRewardSpec)
	if !rewardMatched || reward.Credit.Int64() != 25 || reward.Collectible.Int() != 2 {
		t.Fatalf("reward = %#v", command.Reward)
	}
	if len(command.FundCollectibleIDs) != 2 || command.FundCollectibleIDs[0] != collectibleA || command.FundCollectibleIDs[1] != collectibleB {
		t.Fatalf("fund collectible ids = %#v", command.FundCollectibleIDs)
	}
	if command.Participation != task.ParticipationPolicyReservationRequired {
		t.Fatalf("participation = %s", command.Participation.String())
	}
	if command.AssigneeScope != task.AssigneeScopeOrganizationTeam {
		t.Fatalf("assignee scope = %s", command.AssigneeScope.String())
	}
	if command.ReservationTTL.Hours() != 72 {
		t.Fatalf("reservation ttl = %d", command.ReservationTTL.Hours())
	}
	placement, placementMatched := command.Placement.(task.ExistingSeriesPlacement)
	if !placementMatched || placement.SeriesID != seriesID || placement.Position.Int() != 3 {
		t.Fatalf("placement = %#v", command.Placement)
	}
	payload, payloadMatched := command.Payload.(task.JSONDataPayload)
	if !payloadMatched || payload.Source.String() != `{"batch":"A"}` {
		t.Fatalf("payload = %#v", command.Payload)
	}
	if len(command.Attachments) != 1 || command.Attachments[0].Name.String() != "brief.txt" {
		t.Fatalf("attachments = %#v", command.Attachments)
	}
	expiration, expirationMatched := command.Expiration.(task.ExpiresAt)
	if !expirationMatched || !expiration.Instant.Equal(expiresAt) {
		t.Fatalf("expiration = %#v", command.Expiration)
	}
	if command.Type.String() != "code_review" {
		t.Fatalf("task type = %s", command.Type.String())
	}
}

// TestCreateTaskRequiresExplicitVisibilityKind pins the invisible-task trap
// closed: creating without visibility_kind fails with a message naming every
// valid value, instead of silently defaulting to a private task.
func TestCreateTaskRequiresExplicitVisibilityKind(t *testing.T) {
	services := newCapturingServices()
	server := NewServer(services)
	subject := testSubject(t)
	response := server.Handle(context.Background(), subject, CallerCredential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.create_task","arguments":{"title":"T","description":"D","response_schema_json":"{\"kind\":\"freeform\"}","reward_kind":"none"}}`))
	if response.Error == nil || response.Error.Code != codeInvalidParams {
		t.Fatalf("expected invalid-params without visibility_kind, got %+v", response.Error)
	}
	for _, expected := range []string{"visibility_kind is required", "public", "user", "team", "organization", "organization_team"} {
		if !strings.Contains(response.Error.Message, expected) {
			t.Fatalf("error message %q missing %q", response.Error.Message, expected)
		}
	}

	// The old implicit-default spelling is also rejected with the same
	// guidance rather than silently producing a private task.
	legacy := server.Handle(context.Background(), subject, CallerCredential{Scopes: allScopes()}, request(`2`, "tools/call", `{"name":"sharecrop.create_task","arguments":{"title":"T","description":"D","response_schema_json":"{\"kind\":\"freeform\"}","reward_kind":"none","visibility_kind":"default"}}`))
	if legacy.Error == nil || legacy.Error.Code != codeInvalidParams {
		t.Fatalf("expected invalid-params for visibility_kind \"default\", got %+v", legacy.Error)
	}
}

func TestCreateTaskDefaultsMirrorRESTOwnerAndUserVisibility(t *testing.T) {
	services := newCapturingServices()
	server := NewServer(services)
	subject := testSubject(t)
	response := server.Handle(context.Background(), subject, CallerCredential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.create_task","arguments":{"title":"T","description":"D","response_schema_json":"{\"kind\":\"freeform\"}","reward_kind":"none","visibility_kind":"user"}}`))
	if response.Error != nil {
		t.Fatalf("create_task error: %s", response.Error.Message)
	}
	command := *services.createCommand
	owner, ownerMatched := command.Owner.(task.UserOwner)
	if !ownerMatched || owner.UserID != subject.ID {
		t.Fatalf("owner = %#v", command.Owner)
	}
	// visibility_kind "user" with no explicit id means the caller's own user.
	visibility, visibilityMatched := command.Visibility.(task.UserVisibility)
	if !visibilityMatched || visibility.UserID != subject.ID {
		t.Fatalf("visibility = %#v", command.Visibility)
	}
	if _, matched := command.Expiration.(task.NoExpiration); !matched {
		t.Fatalf("expiration = %#v", command.Expiration)
	}
	if command.ReservationTTL.Hours() != task.DefaultReservationTTL().Hours() {
		t.Fatalf("reservation ttl = %d", command.ReservationTTL.Hours())
	}
}

func TestCreateTaskCollectibleRewardCountIsHonest(t *testing.T) {
	services := newCapturingServices()
	server := NewServer(services)
	collectibleA := core.NewCollectibleID().(core.CollectibleIDCreated).Value
	collectibleB := core.NewCollectibleID().(core.CollectibleIDCreated).Value
	response := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.create_task","arguments":{"title":"T","description":"D","response_schema_json":"{\"kind\":\"freeform\"}","visibility_kind":"user","reward_kind":"collectible","reward_collectible_ids":["`+collectibleA.String()+`","`+collectibleB.String()+`"]}}`))
	if response.Error != nil {
		t.Fatalf("create_task error: %s", response.Error.Message)
	}
	reward, matched := services.createCommand.Reward.(task.CollectibleRewardSpec)
	if !matched || reward.Count.Int() != 2 {
		t.Fatalf("reward = %#v", services.createCommand.Reward)
	}
	if len(services.createCommand.FundCollectibleIDs) != 2 {
		t.Fatalf("fund collectible ids = %#v", services.createCommand.FundCollectibleIDs)
	}

	// A collectible reward with no ids is rejected, exactly like REST.
	missing := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`2`, "tools/call", `{"name":"sharecrop.create_task","arguments":{"title":"T","description":"D","response_schema_json":"{\"kind\":\"freeform\"}","visibility_kind":"user","reward_kind":"collectible"}}`))
	if missing.Error == nil || missing.Error.Code != codeInvalidParams {
		t.Fatalf("expected invalid-params for a collectible reward without ids, got %+v", missing.Error)
	}
}

func TestCreateTaskRejectsPastExpiration(t *testing.T) {
	services := newCapturingServices()
	server := NewServer(services)
	past := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	response := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.create_task","arguments":{"title":"T","description":"D","response_schema_json":"{\"kind\":\"freeform\"}","visibility_kind":"user","reward_kind":"none","expires_at":"`+past+`"}}`))
	if response.Error == nil || response.Error.Code != codeInvalidParams {
		t.Fatalf("expected invalid-params for a past expiration, got %+v", response.Error)
	}
}

func TestAcceptSubmissionCollectibleTip(t *testing.T) {
	services := newCapturingServices()
	server := NewServer(services)
	collectibleID := core.NewCollectibleID().(core.CollectibleIDCreated).Value
	submissionID := core.NewSubmissionID().(core.SubmissionIDCreated).Value
	response := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.accept_submission","arguments":{"task_id":"`+testTaskID(t)+`","submission_id":"`+submissionID.String()+`","idempotency_key":"accept-1","tip_collectible_id":"`+collectibleID.String()+`"}}`))
	if response.Error != nil {
		t.Fatalf("accept_submission error: %s", response.Error.Message)
	}
	selected, matched := (*services.collectibleTip).(ledger.CollectibleTipSelected)
	if !matched || selected.ID != collectibleID {
		t.Fatalf("collectible tip = %#v", *services.collectibleTip)
	}
}

func TestListTasksFiltersAndPaging(t *testing.T) {
	services := newCapturingServices()
	server := NewServer(services)
	response := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.list_tasks","arguments":{"scope":"public","states":["open","closed"],"participation_policy":"open","query":"receipt","task_type":"general","sort":"oldest","limit":5,"offset":10}}`))
	if response.Error != nil {
		t.Fatalf("list_tasks error: %s", response.Error.Message)
	}
	filters := *services.listFilters
	stateFilter, stateMatched := filters.State.(task.StateIn)
	if !stateMatched || len(stateFilter.Values) != 2 {
		t.Fatalf("state filter = %#v", filters.State)
	}
	if _, matched := filters.Participation.(task.ParticipationPolicyEquals); !matched {
		t.Fatalf("participation filter = %#v", filters.Participation)
	}
	search, searchMatched := filters.Search.(task.SearchContains)
	if !searchMatched || search.Value.String() != "receipt" {
		t.Fatalf("search filter = %#v", filters.Search)
	}
	if _, matched := filters.Type.(task.TypeEquals); !matched {
		t.Fatalf("type filter = %#v", filters.Type)
	}
	if filters.Sort != task.SortOldest {
		t.Fatalf("sort = %s", filters.Sort.String())
	}
	// The service receives the probe page (limit+1) so the tool can report
	// next_offset without a second query; the caller-visible window is 5.
	if services.listPage.Limit() != 6 || services.listPage.Offset() != 10 {
		t.Fatalf("page = %d/%d", services.listPage.Limit(), services.listPage.Offset())
	}

	// The deprecated single-state alias still works.
	alias := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`2`, "tools/call", `{"name":"sharecrop.list_tasks","arguments":{"scope":"public","state":"open"}}`))
	if alias.Error != nil {
		t.Fatalf("list_tasks state alias error: %s", alias.Error.Message)
	}
	if equals, matched := services.listFilters.State.(task.StateEquals); !matched || equals.Value != task.StateOpen {
		t.Fatalf("alias state filter = %#v", services.listFilters.State)
	}
}

func TestCreditsToolsRequireLedgerReadScope(t *testing.T) {
	server := NewServer(fakeServices{})
	granted := CallerCredential{Scopes: agent.NewScopeSet([]agent.Scope{agent.ScopeLedgerRead})}

	balanceText := decodeToolText(t, server.Handle(context.Background(), testSubject(t), granted, request(`1`, "tools/call", `{"name":"sharecrop.get_credit_balance","arguments":{}}`)))
	var balance creditBalancePayload
	if err := json.Unmarshal([]byte(balanceText), &balance); err != nil {
		t.Fatalf("decode balance: %v", err)
	}
	if balance.SpendableCredits != 25 || balance.AllocatedCredits != 5 {
		t.Fatalf("balance = %+v", balance)
	}

	ledgerText := decodeToolText(t, server.Handle(context.Background(), testSubject(t), granted, request(`2`, "tools/call", `{"name":"sharecrop.list_ledger","arguments":{"limit":10}}`)))
	var entries ledgerEntriesPayload
	if err := json.Unmarshal([]byte(ledgerText), &entries); err != nil {
		t.Fatalf("decode ledger: %v", err)
	}
	if entries.Entries == nil {
		t.Fatalf("entries should decode to an empty list")
	}

	denied := CallerCredential{Scopes: agent.NewScopeSet([]agent.Scope{agent.ScopeTasksRead})}
	response := server.Handle(context.Background(), testSubject(t), denied, request(`3`, "tools/call", `{"name":"sharecrop.get_credit_balance","arguments":{}}`))
	if response.Error == nil || response.Error.Code != codeScopeDenied {
		t.Fatalf("expected scope-denied for missing ledger_read, got %+v", response.Error)
	}
	listDenied := server.Handle(context.Background(), testSubject(t), denied, request(`4`, "tools/call", `{"name":"sharecrop.list_ledger","arguments":{}}`))
	if listDenied.Error == nil || listDenied.Error.Code != codeScopeDenied {
		t.Fatalf("expected scope-denied for missing ledger_read, got %+v", listDenied.Error)
	}
}

func TestListToolsAcceptPagination(t *testing.T) {
	server := NewServer(fakeServices{})
	response := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.list_task_submissions","arguments":{"task_id":"`+testTaskID(t)+`","limit":3,"offset":6}}`))
	if response.Error != nil {
		t.Fatalf("list_task_submissions with paging error: %s", response.Error.Message)
	}
	negative := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`2`, "tools/call", `{"name":"sharecrop.list_task_submissions","arguments":{"task_id":"`+testTaskID(t)+`","offset":-1}}`))
	if negative.Error == nil || negative.Error.Code != codeInvalidParams {
		t.Fatalf("expected invalid-params for a negative offset, got %+v", negative.Error)
	}
}

var _ Services = &capturingServices{}
