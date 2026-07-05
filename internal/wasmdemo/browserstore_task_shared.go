package wasmdemo

import (
	"encoding/json"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/task"
)

// storedTaskRecord is the shared, low-level task persistence format used by
// TaskBrowserStore, LedgerBrowserStore, and the collectible-reward-funding
// half of AssetBrowserStore - these three domains' real Postgres stores all
// read and write the same `tasks` table directly (task state checks inside
// fund/refund, reward_kind flips on funding, cancellation on refund), so
// their browser-store equivalents share one underlying record too, rather
// than each keeping an independent, driftable copy of task state.
type storedTaskRecord struct {
	ID                     string `json:"id"`
	OwnerKind              string `json:"owner_kind"`
	OwnerUserID            string `json:"owner_user_id,omitempty"`
	OwnerTeamID            string `json:"owner_team_id,omitempty"`
	OwnerOrganizationID    string `json:"owner_organization_id,omitempty"`
	Title                  string `json:"title"`
	Description            string `json:"description"`
	TaskType               string `json:"task_type"`
	ReferenceURL           string `json:"reference_url"`
	RewardKind             string `json:"reward_kind"`
	RewardCreditAmount     int64  `json:"reward_credit_amount"`
	RewardCollectibleCount int    `json:"reward_collectible_count"`
	Participation          string `json:"participation_policy"`
	AssigneeScope          string `json:"assignee_scope"`
	ReservationTTLHours    int    `json:"reservation_ttl_hours"`
	State                  string `json:"state"`
	VisibilityKind         string `json:"visibility_kind"`
	VisibilityUserID       string `json:"visibility_user_id,omitempty"`
	VisibilityTeamID       string `json:"visibility_team_id,omitempty"`
	VisibilityOrgID        string `json:"visibility_organization_id,omitempty"`
	ResponseSchemaJSON     string `json:"response_schema_json"`
	PayloadKind            string `json:"payload_kind"`
	PayloadJSON            string `json:"payload_json,omitempty"`
	CreatedBy              string `json:"created_by"`
}

func taskRecordKey(id string) string { return "task:record:" + id }
func taskUserIndexKey(userID string) string {
	return "task:user_index:" + userID
}
func taskPublicIndexKey() string { return "task:public_index" }

// putStorageString and getStorageString are the shared, type-free byte-level
// primitives every typed put*/get* helper below builds on - kept to plain
// strings (not a generic helper over an unconstrained type) so each stored
// type gets its own small, explicit marshal/unmarshal wrapper instead of one
// generic one.
func putStorageString(storage BrowserStorage, rawKey string, value string) bool {
	keyResult := NewStorageKey(rawKey)
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return false
	}
	_, matched := storage.Put(key.Value, value).(StorageWritten)
	return matched
}

func getStorageString(storage BrowserStorage, rawKey string) (string, bool, bool) {
	keyResult := NewStorageKey(rawKey)
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return "", false, false
	}
	readResult := storage.Get(key.Value)
	if _, missing := readResult.(StorageMissing); missing {
		return "", false, true
	}
	read, readMatched := readResult.(StorageRead)
	if !readMatched {
		return "", false, false
	}
	return read.Value, true, true
}

func putStoredTaskRecordJSON(storage BrowserStorage, rawKey string, record storedTaskRecord) bool {
	encoded, err := json.Marshal(record)
	if err != nil {
		return false
	}
	return putStorageString(storage, rawKey, string(encoded))
}

func getStoredTaskRecordJSON(storage BrowserStorage, rawKey string) (storedTaskRecord, bool, bool) {
	raw, found, ok := getStorageString(storage, rawKey)
	if !ok || !found {
		return storedTaskRecord{}, found, ok
	}
	var record storedTaskRecord
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return storedTaskRecord{}, false, false
	}
	return record, true, true
}

func loadStoredTaskRecord(storage BrowserStorage, taskID string) (storedTaskRecord, bool, *core.DomainError) {
	record, found, ok := getStoredTaskRecordJSON(storage, taskRecordKey(taskID))
	if !ok {
		reason := invalidState("task lookup failed")
		return storedTaskRecord{}, false, &reason
	}
	if !found {
		return storedTaskRecord{}, false, nil
	}
	return record, true, nil
}

func saveStoredTaskRecord(storage BrowserStorage, record storedTaskRecord) bool {
	return putStoredTaskRecordJSON(storage, taskRecordKey(record.ID), record)
}

func ownerSQLColumnsBrowser(owner task.Owner) (kind string, userID string, teamID string, organizationID string) {
	switch typed := owner.(type) {
	case task.UserOwner:
		return task.OwnerKindUser.String(), typed.UserID.String(), "", ""
	case task.TeamOwner:
		return task.OwnerKindTeam.String(), "", typed.TeamID.String(), ""
	case task.OrganizationOwner:
		return task.OwnerKindOrganization.String(), "", "", typed.OrganizationID.String()
	case task.OrganizationTeamOwner:
		return task.OwnerKindOrganizationTeam.String(), "", typed.TeamID.String(), typed.OrganizationID.String()
	default:
		return "", "", "", ""
	}
}

func visibilitySQLColumnsBrowser(visibility task.Visibility) (kind string, userID string, teamID string, organizationID string) {
	switch typed := visibility.(type) {
	case task.PublicVisibility:
		return task.VisibilityKindPublic.String(), "", "", ""
	case task.UserVisibility:
		return task.VisibilityKindUser.String(), typed.UserID.String(), "", ""
	case task.TeamVisibility:
		return task.VisibilityKindTeam.String(), "", typed.TeamID.String(), ""
	case task.OrganizationVisibility:
		return task.VisibilityKindOrganization.String(), "", "", typed.OrganizationID.String()
	case task.OrganizationTeamVisibility:
		return task.VisibilityKindOrganizationTeam.String(), "", typed.TeamID.String(), typed.OrganizationID.String()
	default:
		return "", "", "", ""
	}
}

func rewardSQLColumnsBrowser(reward task.RewardSpec) (kind string, creditAmount int64, collectibleCount int) {
	switch typed := reward.(type) {
	case task.NoRewardSpec:
		return task.RewardKindNone.String(), 0, 0
	case task.CreditRewardSpec:
		return task.RewardKindCredit.String(), typed.Amount.Int64(), 0
	case task.CollectibleRewardSpec:
		return task.RewardKindCollectible.String(), 0, typed.Count.Int()
	case task.BundleRewardSpec:
		return task.RewardKindBundle.String(), typed.Credit.Int64(), typed.Collectible.Int()
	default:
		return task.RewardKindNone.String(), 0, 0
	}
}

func payloadSQLColumnsBrowser(payload task.DataPayload) (kind string, source string) {
	switch typed := payload.(type) {
	case task.NoDataPayload:
		return "none", ""
	case task.JSONDataPayload:
		return "json", typed.Source.String()
	default:
		return "none", ""
	}
}

// parseStoredTaskRecord converts the shared storage record back into the
// real task.Task domain type, the same shape internal/db's TaskStore
// returns from a Postgres row.
func parseStoredTaskRecord(record storedTaskRecord) (task.Task, *core.DomainError) {
	idResult := core.ParseTaskID(record.ID)
	id, idMatched := idResult.(core.TaskIDCreated)
	if !idMatched {
		reason := idResult.(core.TaskIDRejected).Reason
		return task.Task{}, &reason
	}

	owner, ownerErr := parseStoredOwner(record)
	if ownerErr != nil {
		return task.Task{}, ownerErr
	}

	titleResult := task.NewTitle(record.Title)
	title, titleMatched := titleResult.(task.TitleAccepted)
	if !titleMatched {
		reason := titleResult.(task.TitleRejected).Reason
		return task.Task{}, &reason
	}
	descriptionResult := task.NewDescription(record.Description)
	description, descriptionMatched := descriptionResult.(task.DescriptionAccepted)
	if !descriptionMatched {
		reason := descriptionResult.(task.DescriptionRejected).Reason
		return task.Task{}, &reason
	}
	typeResult := task.ParseTaskType(record.TaskType)
	taskType, typeMatched := typeResult.(task.TaskTypeAccepted)
	if !typeMatched {
		reason := typeResult.(task.TaskTypeRejected).Reason
		return task.Task{}, &reason
	}
	referenceResult := task.NewReferenceURL(record.ReferenceURL)
	reference, referenceMatched := referenceResult.(task.ReferenceURLAccepted)
	if !referenceMatched {
		reason := referenceResult.(task.ReferenceURLRejected).Reason
		return task.Task{}, &reason
	}

	reward, rewardErr := parseStoredReward(record)
	if rewardErr != nil {
		return task.Task{}, rewardErr
	}

	participationResult := task.ParseParticipationPolicy(record.Participation)
	participation, participationMatched := participationResult.(task.ParticipationPolicyAccepted)
	if !participationMatched {
		reason := participationResult.(task.ParticipationPolicyRejected).Reason
		return task.Task{}, &reason
	}
	assigneeScopeResult := task.ParseAssigneeScope(record.AssigneeScope)
	assigneeScope, assigneeScopeMatched := assigneeScopeResult.(task.AssigneeScopeAccepted)
	if !assigneeScopeMatched {
		reason := assigneeScopeResult.(task.AssigneeScopeRejected).Reason
		return task.Task{}, &reason
	}
	ttlResult := task.NewReservationTTL(record.ReservationTTLHours)
	ttl, ttlMatched := ttlResult.(task.ReservationTTLAccepted)
	if !ttlMatched {
		reason := ttlResult.(task.ReservationTTLRejected).Reason
		return task.Task{}, &reason
	}
	stateResult := task.ParseState(record.State)
	state, stateMatched := stateResult.(task.StateAccepted)
	if !stateMatched {
		reason := stateResult.(task.StateRejected).Reason
		return task.Task{}, &reason
	}

	visibility, visibilityErr := parseStoredVisibility(record)
	if visibilityErr != nil {
		return task.Task{}, visibilityErr
	}

	responseSchemaResult := task.NewResponseSchemaSource(record.ResponseSchemaJSON)
	responseSchema, responseSchemaMatched := responseSchemaResult.(task.ResponseSchemaSourceAccepted)
	if !responseSchemaMatched {
		reason := responseSchemaResult.(task.ResponseSchemaSourceRejected).Reason
		return task.Task{}, &reason
	}

	var payload task.DataPayload
	if record.PayloadKind == "json" {
		sourceResult := task.NewPayloadSource(record.PayloadJSON)
		source, sourceMatched := sourceResult.(task.PayloadSourceAccepted)
		if !sourceMatched {
			reason := sourceResult.(task.PayloadSourceRejected).Reason
			return task.Task{}, &reason
		}
		payload = task.JSONDataPayload{Source: source.Value}
	} else {
		payload = task.NoDataPayload{}
	}

	createdByResult := core.ParseUserID(record.CreatedBy)
	createdBy, createdByMatched := createdByResult.(core.UserIDCreated)
	if !createdByMatched {
		reason := createdByResult.(core.UserIDRejected).Reason
		return task.Task{}, &reason
	}

	return task.Task{
		ID:             id.Value,
		Owner:          owner,
		Title:          title.Value,
		Description:    description.Value,
		Type:           taskType.Value,
		Reference:      reference.Value,
		Reward:         reward,
		Participation:  participation.Value,
		AssigneeScope:  assigneeScope.Value,
		ReservationTTL: ttl.Value,
		State:          state.Value,
		Visibility:     visibility,
		Placement:      task.StandalonePlacement{},
		ResponseSchema: responseSchema.Value,
		Payload:        payload,
		Attachments:    nil,
		CreatedBy:      createdBy.Value,
	}, nil
}

func parseStoredOwner(record storedTaskRecord) (task.Owner, *core.DomainError) {
	switch record.OwnerKind {
	case task.OwnerKindUser.String():
		result := core.ParseUserID(record.OwnerUserID)
		created, matched := result.(core.UserIDCreated)
		if !matched {
			reason := result.(core.UserIDRejected).Reason
			return nil, &reason
		}
		return task.UserOwner{UserID: created.Value}, nil
	case task.OwnerKindTeam.String():
		result := core.ParseTeamID(record.OwnerTeamID)
		created, matched := result.(core.TeamIDCreated)
		if !matched {
			reason := result.(core.TeamIDRejected).Reason
			return nil, &reason
		}
		return task.TeamOwner{TeamID: created.Value}, nil
	case task.OwnerKindOrganization.String():
		result := core.ParseOrganizationID(record.OwnerOrganizationID)
		created, matched := result.(core.OrganizationIDCreated)
		if !matched {
			reason := result.(core.OrganizationIDRejected).Reason
			return nil, &reason
		}
		return task.OrganizationOwner{OrganizationID: created.Value}, nil
	case task.OwnerKindOrganizationTeam.String():
		orgResult := core.ParseOrganizationID(record.OwnerOrganizationID)
		orgCreated, orgMatched := orgResult.(core.OrganizationIDCreated)
		if !orgMatched {
			reason := orgResult.(core.OrganizationIDRejected).Reason
			return nil, &reason
		}
		teamResult := core.ParseTeamID(record.OwnerTeamID)
		teamCreated, teamMatched := teamResult.(core.TeamIDCreated)
		if !teamMatched {
			reason := teamResult.(core.TeamIDRejected).Reason
			return nil, &reason
		}
		return task.OrganizationTeamOwner{OrganizationID: orgCreated.Value, TeamID: teamCreated.Value}, nil
	default:
		reason := core.NewDomainError(core.ErrorCodeInvalidState, "task owner kind is invalid")
		return nil, &reason
	}
}

func parseStoredVisibility(record storedTaskRecord) (task.Visibility, *core.DomainError) {
	switch record.VisibilityKind {
	case task.VisibilityKindPublic.String():
		return task.PublicVisibility{}, nil
	case task.VisibilityKindUser.String():
		result := core.ParseUserID(record.VisibilityUserID)
		created, matched := result.(core.UserIDCreated)
		if !matched {
			reason := result.(core.UserIDRejected).Reason
			return nil, &reason
		}
		return task.UserVisibility{UserID: created.Value}, nil
	case task.VisibilityKindTeam.String():
		result := core.ParseTeamID(record.VisibilityTeamID)
		created, matched := result.(core.TeamIDCreated)
		if !matched {
			reason := result.(core.TeamIDRejected).Reason
			return nil, &reason
		}
		return task.TeamVisibility{TeamID: created.Value}, nil
	case task.VisibilityKindOrganization.String():
		result := core.ParseOrganizationID(record.VisibilityOrgID)
		created, matched := result.(core.OrganizationIDCreated)
		if !matched {
			reason := result.(core.OrganizationIDRejected).Reason
			return nil, &reason
		}
		return task.OrganizationVisibility{OrganizationID: created.Value}, nil
	case task.VisibilityKindOrganizationTeam.String():
		orgResult := core.ParseOrganizationID(record.VisibilityOrgID)
		orgCreated, orgMatched := orgResult.(core.OrganizationIDCreated)
		if !orgMatched {
			reason := orgResult.(core.OrganizationIDRejected).Reason
			return nil, &reason
		}
		teamResult := core.ParseTeamID(record.VisibilityTeamID)
		teamCreated, teamMatched := teamResult.(core.TeamIDCreated)
		if !teamMatched {
			reason := teamResult.(core.TeamIDRejected).Reason
			return nil, &reason
		}
		return task.OrganizationTeamVisibility{OrganizationID: orgCreated.Value, TeamID: teamCreated.Value}, nil
	default:
		reason := core.NewDomainError(core.ErrorCodeInvalidState, "task visibility kind is invalid")
		return nil, &reason
	}
}

func parseStoredReward(record storedTaskRecord) (task.RewardSpec, *core.DomainError) {
	switch record.RewardKind {
	case task.RewardKindNone.String():
		return task.NoRewardSpec{}, nil
	case task.RewardKindCredit.String():
		amountResult := task.NewCreditRewardAmount(record.RewardCreditAmount)
		amount, matched := amountResult.(task.CreditRewardAmountAccepted)
		if !matched {
			reason := amountResult.(task.CreditRewardAmountRejected).Reason
			return nil, &reason
		}
		return task.CreditRewardSpec{Amount: amount.Value}, nil
	case task.RewardKindCollectible.String():
		countResult := task.NewCollectibleRewardCount(record.RewardCollectibleCount)
		count, matched := countResult.(task.CollectibleRewardCountAccepted)
		if !matched {
			reason := countResult.(task.CollectibleRewardCountRejected).Reason
			return nil, &reason
		}
		return task.CollectibleRewardSpec{Count: count.Value}, nil
	case task.RewardKindBundle.String():
		amountResult := task.NewCreditRewardAmount(record.RewardCreditAmount)
		amount, amountMatched := amountResult.(task.CreditRewardAmountAccepted)
		if !amountMatched {
			reason := amountResult.(task.CreditRewardAmountRejected).Reason
			return nil, &reason
		}
		countResult := task.NewCollectibleRewardCount(record.RewardCollectibleCount)
		count, countMatched := countResult.(task.CollectibleRewardCountAccepted)
		if !countMatched {
			reason := countResult.(task.CollectibleRewardCountRejected).Reason
			return nil, &reason
		}
		return task.BundleRewardSpec{Credit: amount.Value, Collectible: count.Value}, nil
	default:
		reason := core.NewDomainError(core.ErrorCodeInvalidState, "task reward kind is invalid")
		return nil, &reason
	}
}
