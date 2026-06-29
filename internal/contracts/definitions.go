package contracts

func Modules() []Module {
	return []Module{
		idsModule(),
		errorModule(),
		authModule(),
		organizationModule(),
		teamModule(),
		taskModule(),
		taskSeriesModule(),
		submissionModule(),
		ledgerModule(),
		agentModule(),
		collectibleModule(),
		adminModule(),
	}
}

func adminModule() Module {
	return Module{
		Name: NewModuleName("Sharecrop.Generated.Admin"),
		Definitions: []Definition{
			Product{
				Name: NewElmTypeName("OperationsResponse"),
				Fields: []Field{
					{Name: NewElmValueName("status"), JSONName: NewJSONFieldName("status"), Type: StringRef{}},
					{Name: NewElmValueName("accountTokenDelivery"), JSONName: NewJSONFieldName("account_token_delivery"), Type: StringRef{}},
					{Name: NewElmValueName("mcpStorage"), JSONName: NewJSONFieldName("mcp_storage"), Type: StringRef{}},
					{Name: NewElmValueName("rateLimitStorage"), JSONName: NewJSONFieldName("rate_limit_storage"), Type: StringRef{}},
					{Name: NewElmValueName("activeMCPSessions"), JSONName: NewJSONFieldName("active_mcp_sessions"), Type: IntRef{}},
					{Name: NewElmValueName("activeIPRateBuckets"), JSONName: NewJSONFieldName("active_ip_rate_buckets"), Type: IntRef{}},
					{Name: NewElmValueName("activeSubjectRateBuckets"), JSONName: NewJSONFieldName("active_subject_rate_buckets"), Type: IntRef{}},
					{Name: NewElmValueName("secureCookies"), JSONName: NewJSONFieldName("secure_cookies"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("AuditEventResponse"),
				Fields: []Field{
					{Name: NewElmValueName("id"), JSONName: NewJSONFieldName("id"), Type: StringRef{}},
					{Name: NewElmValueName("actorUserID"), JSONName: NewJSONFieldName("actor_user_id"), Type: StringRef{}},
					{Name: NewElmValueName("action"), JSONName: NewJSONFieldName("action"), Type: StringRef{}},
					{Name: NewElmValueName("subjectKind"), JSONName: NewJSONFieldName("subject_kind"), Type: StringRef{}},
					{Name: NewElmValueName("subjectID"), JSONName: NewJSONFieldName("subject_id"), Type: StringRef{}},
					{Name: NewElmValueName("metadataJSON"), JSONName: NewJSONFieldName("metadata_json"), Type: StringRef{}},
					{Name: NewElmValueName("createdAt"), JSONName: NewJSONFieldName("created_at"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("AuditEventsResponse"),
				Fields: []Field{
					{Name: NewElmValueName("events"), JSONName: NewJSONFieldName("events"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("AuditEventResponse")}}},
				},
			},
		},
	}
}

func collectibleModule() Module {
	return Module{
		Name: NewModuleName("Sharecrop.Generated.Collectible"),
		Definitions: []Definition{
			Enum{
				Name: NewElmTypeName("CollectibleKind"),
				Variants: []Variant{
					{Name: NewElmTypeName("CollectibleKindUnique"), Tag: "unique"},
					{Name: NewElmTypeName("CollectibleKindEdition"), Tag: "edition"},
					{Name: NewElmTypeName("CollectibleKindBadge"), Tag: "badge"},
				},
			},
			Enum{
				Name: NewElmTypeName("CollectibleState"),
				Variants: []Variant{
					{Name: NewElmTypeName("CollectibleStateMinted"), Tag: "minted"},
					{Name: NewElmTypeName("CollectibleStateEscrowed"), Tag: "escrowed"},
					{Name: NewElmTypeName("CollectibleStateAwarded"), Tag: "awarded"},
				},
			},
			Enum{
				Name: NewElmTypeName("CollectibleTransferPolicy"),
				Variants: []Variant{
					{Name: NewElmTypeName("CollectibleTransferPolicyNonTransferableExceptPayout"), Tag: "non_transferable_except_payout"},
					{Name: NewElmTypeName("CollectibleTransferPolicyTransferableBetweenUsers"), Tag: "transferable_between_users"},
					{Name: NewElmTypeName("CollectibleTransferPolicyTransferableWithinOrganization"), Tag: "transferable_within_organization"},
					{Name: NewElmTypeName("CollectibleTransferPolicyIssuerControlled"), Tag: "issuer_controlled"},
				},
			},
			Product{
				Name: NewElmTypeName("CollectibleResponse"),
				Fields: []Field{
					{Name: NewElmValueName("id"), JSONName: NewJSONFieldName("id"), Type: StringRef{}},
					{Name: NewElmValueName("name"), JSONName: NewJSONFieldName("name"), Type: StringRef{}},
					{Name: NewElmValueName("kind"), JSONName: NewJSONFieldName("kind"), Type: NamedRef{Name: NewElmTypeName("CollectibleKind")}},
					{Name: NewElmValueName("state"), JSONName: NewJSONFieldName("state"), Type: NamedRef{Name: NewElmTypeName("CollectibleState")}},
					{Name: NewElmValueName("transferPolicy"), JSONName: NewJSONFieldName("transfer_policy"), Type: NamedRef{Name: NewElmTypeName("CollectibleTransferPolicy")}},
					{Name: NewElmValueName("ownerID"), JSONName: NewJSONFieldName("owner_id"), Type: StringRef{}},
					{Name: NewElmValueName("ownerKind"), JSONName: NewJSONFieldName("owner_kind"), Type: StringRef{}},
					{Name: NewElmValueName("art"), JSONName: NewJSONFieldName("art"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("CollectiblesResponse"),
				Fields: []Field{
					{Name: NewElmValueName("collectibles"), JSONName: NewJSONFieldName("collectibles"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("CollectibleResponse")}}},
				},
			},
			Product{
				Name: NewElmTypeName("CollectibleCatalogEntry"),
				Fields: []Field{
					{Name: NewElmValueName("slug"), JSONName: NewJSONFieldName("slug"), Type: StringRef{}},
					{Name: NewElmValueName("name"), JSONName: NewJSONFieldName("name"), Type: StringRef{}},
					{Name: NewElmValueName("kind"), JSONName: NewJSONFieldName("kind"), Type: NamedRef{Name: NewElmTypeName("CollectibleKind")}},
					{Name: NewElmValueName("transferPolicy"), JSONName: NewJSONFieldName("transfer_policy"), Type: NamedRef{Name: NewElmTypeName("CollectibleTransferPolicy")}},
					{Name: NewElmValueName("art"), JSONName: NewJSONFieldName("art"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("CollectibleCatalogResponse"),
				Fields: []Field{
					{Name: NewElmValueName("entries"), JSONName: NewJSONFieldName("entries"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("CollectibleCatalogEntry")}}},
				},
			},
		},
	}
}

func taskSeriesModule() Module {
	return Module{
		Name: NewModuleName("Sharecrop.Generated.TaskSeries"),
		Definitions: []Definition{
			Product{
				Name: NewElmTypeName("TaskSeriesResponse"),
				Fields: []Field{
					{Name: NewElmValueName("id"), JSONName: NewJSONFieldName("id"), Type: StringRef{}},
					{Name: NewElmValueName("ownerKind"), JSONName: NewJSONFieldName("owner_kind"), Type: StringRef{}},
					{Name: NewElmValueName("title"), JSONName: NewJSONFieldName("title"), Type: StringRef{}},
					{Name: NewElmValueName("description"), JSONName: NewJSONFieldName("description"), Type: StringRef{}},
					{Name: NewElmValueName("state"), JSONName: NewJSONFieldName("state"), Type: StringRef{}},
					{Name: NewElmValueName("createdBy"), JSONName: NewJSONFieldName("created_by"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("SeriesCommentResponse"),
				Fields: []Field{
					{Name: NewElmValueName("id"), JSONName: NewJSONFieldName("id"), Type: StringRef{}},
					{Name: NewElmValueName("seriesID"), JSONName: NewJSONFieldName("series_id"), Type: StringRef{}},
					{Name: NewElmValueName("authorUserID"), JSONName: NewJSONFieldName("author_user_id"), Type: StringRef{}},
					{Name: NewElmValueName("body"), JSONName: NewJSONFieldName("body"), Type: StringRef{}},
					{Name: NewElmValueName("createdAt"), JSONName: NewJSONFieldName("created_at"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("TaskSeriesListResponse"),
				Fields: []Field{
					{Name: NewElmValueName("series"), JSONName: NewJSONFieldName("series"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("TaskSeriesResponse")}}},
				},
			},
		},
	}
}

func idsModule() Module {
	return Module{
		Name: NewModuleName("Sharecrop.Generated.Ids"),
		Definitions: []Definition{
			Alias{Name: NewElmTypeName("UserID"), Type: StringRef{}},
			Alias{Name: NewElmTypeName("GuestID"), Type: StringRef{}},
			Alias{Name: NewElmTypeName("TaskID"), Type: StringRef{}},
			Alias{Name: NewElmTypeName("TaskSeriesID"), Type: StringRef{}},
			Alias{Name: NewElmTypeName("TaskCapabilityTokenID"), Type: StringRef{}},
			Alias{Name: NewElmTypeName("SubmissionID"), Type: StringRef{}},
			Alias{Name: NewElmTypeName("SubmissionReceiptTokenID"), Type: StringRef{}},
			Alias{Name: NewElmTypeName("CreditAccountID"), Type: StringRef{}},
			Alias{Name: NewElmTypeName("LedgerEntryID"), Type: StringRef{}},
			Alias{Name: NewElmTypeName("AgentCredentialID"), Type: StringRef{}},
			Alias{Name: NewElmTypeName("CollectibleID"), Type: StringRef{}},
			Alias{Name: NewElmTypeName("OrganizationID"), Type: StringRef{}},
			Alias{Name: NewElmTypeName("OrganizationMembershipID"), Type: StringRef{}},
			Alias{Name: NewElmTypeName("TeamID"), Type: StringRef{}},
			Alias{Name: NewElmTypeName("AccessToken"), Type: StringRef{}},
		},
	}
}

func errorModule() Module {
	return Module{
		Name: NewModuleName("Sharecrop.Generated.Error"),
		Definitions: []Definition{
			Product{
				Name: NewElmTypeName("ErrorResponse"),
				Fields: []Field{
					{Name: NewElmValueName("error"), JSONName: NewJSONFieldName("error"), Type: StringRef{}},
				},
			},
		},
	}
}

func authModule() Module {
	return Module{
		Name: NewModuleName("Sharecrop.Generated.Auth"),
		Definitions: []Definition{
			Enum{
				Name: NewElmTypeName("SubjectKind"),
				Variants: []Variant{
					{Name: NewElmTypeName("SubjectKindUser"), Tag: "user"},
					{Name: NewElmTypeName("SubjectKindGuest"), Tag: "guest"},
				},
			},
			Product{
				Name: NewElmTypeName("AuthResponse"),
				Fields: []Field{
					{Name: NewElmValueName("subjectKind"), JSONName: NewJSONFieldName("subject_kind"), Type: NamedRef{Name: NewElmTypeName("SubjectKind")}},
					{Name: NewElmValueName("subjectID"), JSONName: NewJSONFieldName("subject_id"), Type: StringRef{}},
					{Name: NewElmValueName("accessToken"), JSONName: NewJSONFieldName("access_token"), Type: StringRef{}},
					{Name: NewElmValueName("role"), JSONName: NewJSONFieldName("role"), Type: StringRef{}},
				},
			},
		},
	}
}

func organizationModule() Module {
	return Module{
		Name: NewModuleName("Sharecrop.Generated.Organization"),
		Definitions: []Definition{
			Enum{
				Name: NewElmTypeName("MembershipStatus"),
				Variants: []Variant{
					{Name: NewElmTypeName("MembershipStatusActive"), Tag: "active"},
					{Name: NewElmTypeName("MembershipStatusDeactivated"), Tag: "deactivated"},
					{Name: NewElmTypeName("MembershipStatusRemoved"), Tag: "removed"},
				},
			},
			Enum{
				Name: NewElmTypeName("OrganizationRole"),
				Variants: []Variant{
					{Name: NewElmTypeName("OrganizationRoleOwner"), Tag: "owner"},
					{Name: NewElmTypeName("OrganizationRoleAdmin"), Tag: "admin"},
					{Name: NewElmTypeName("OrganizationRoleMember"), Tag: "member"},
					{Name: NewElmTypeName("OrganizationRoleBilling"), Tag: "billing"},
					{Name: NewElmTypeName("OrganizationRoleReviewer"), Tag: "reviewer"},
					{Name: NewElmTypeName("OrganizationRolePublicPublisher"), Tag: "public_publisher"},
				},
			},
			Product{
				Name: NewElmTypeName("OrganizationResponse"),
				Fields: []Field{
					{Name: NewElmValueName("id"), JSONName: NewJSONFieldName("id"), Type: StringRef{}},
					{Name: NewElmValueName("name"), JSONName: NewJSONFieldName("name"), Type: StringRef{}},
					{Name: NewElmValueName("createdBy"), JSONName: NewJSONFieldName("created_by"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("OrganizationsResponse"),
				Fields: []Field{
					{Name: NewElmValueName("organizations"), JSONName: NewJSONFieldName("organizations"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("OrganizationResponse")}}},
				},
			},
			Product{
				Name: NewElmTypeName("OrganizationMemberResponse"),
				Fields: []Field{
					{Name: NewElmValueName("id"), JSONName: NewJSONFieldName("id"), Type: StringRef{}},
					{Name: NewElmValueName("organizationID"), JSONName: NewJSONFieldName("organization_id"), Type: StringRef{}},
					{Name: NewElmValueName("userID"), JSONName: NewJSONFieldName("user_id"), Type: StringRef{}},
					{Name: NewElmValueName("status"), JSONName: NewJSONFieldName("status"), Type: NamedRef{Name: NewElmTypeName("MembershipStatus")}},
					{Name: NewElmValueName("roles"), JSONName: NewJSONFieldName("roles"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("OrganizationRole")}}},
				},
			},
			Product{
				Name: NewElmTypeName("OrganizationMembersResponse"),
				Fields: []Field{
					{Name: NewElmValueName("members"), JSONName: NewJSONFieldName("members"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("OrganizationMemberResponse")}}},
				},
			},
		},
	}
}

func teamModule() Module {
	return Module{
		Name: NewModuleName("Sharecrop.Generated.Team"),
		Definitions: []Definition{
			Product{
				Name: NewElmTypeName("TeamResponse"),
				Fields: []Field{
					{Name: NewElmValueName("id"), JSONName: NewJSONFieldName("id"), Type: StringRef{}},
					{Name: NewElmValueName("ownerKind"), JSONName: NewJSONFieldName("owner_kind"), Type: StringRef{}},
					{Name: NewElmValueName("organizationID"), JSONName: NewJSONFieldName("organization_id"), Type: StringRef{}},
					{Name: NewElmValueName("ownerUserID"), JSONName: NewJSONFieldName("owner_user_id"), Type: StringRef{}},
					{Name: NewElmValueName("name"), JSONName: NewJSONFieldName("name"), Type: StringRef{}},
					{Name: NewElmValueName("createdBy"), JSONName: NewJSONFieldName("created_by"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("TeamsResponse"),
				Fields: []Field{
					{Name: NewElmValueName("teams"), JSONName: NewJSONFieldName("teams"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("TeamResponse")}}},
				},
			},
			Product{
				Name: NewElmTypeName("TeamDetailResponse"),
				Fields: []Field{
					{Name: NewElmValueName("team"), JSONName: NewJSONFieldName("team"), Type: NamedRef{Name: NewElmTypeName("TeamResponse")}},
					{Name: NewElmValueName("members"), JSONName: NewJSONFieldName("members"), Type: ListRef{Element: StringRef{}}},
				},
			},
		},
	}
}

func taskModule() Module {
	return Module{
		Name: NewModuleName("Sharecrop.Generated.Task"),
		Definitions: []Definition{
			Enum{
				Name: NewElmTypeName("TaskState"),
				Variants: []Variant{
					{Name: NewElmTypeName("TaskStateDraft"), Tag: "draft"},
					{Name: NewElmTypeName("TaskStateOpen"), Tag: "open"},
					{Name: NewElmTypeName("TaskStateClosed"), Tag: "closed"},
					{Name: NewElmTypeName("TaskStateCancelled"), Tag: "cancelled"},
					{Name: NewElmTypeName("TaskStateExpired"), Tag: "expired"},
				},
			},
			Enum{
				Name: NewElmTypeName("TaskOwnerKind"),
				Variants: []Variant{
					{Name: NewElmTypeName("TaskOwnerKindUser"), Tag: "user"},
					{Name: NewElmTypeName("TaskOwnerKindTeam"), Tag: "team"},
					{Name: NewElmTypeName("TaskOwnerKindOrganization"), Tag: "organization"},
					{Name: NewElmTypeName("TaskOwnerKindOrganizationTeam"), Tag: "organization_team"},
				},
			},
			Enum{
				Name: NewElmTypeName("TaskVisibilityKind"),
				Variants: []Variant{
					{Name: NewElmTypeName("TaskVisibilityKindPublic"), Tag: "public"},
					{Name: NewElmTypeName("TaskVisibilityKindUser"), Tag: "user"},
					{Name: NewElmTypeName("TaskVisibilityKindTeam"), Tag: "team"},
					{Name: NewElmTypeName("TaskVisibilityKindOrganization"), Tag: "organization"},
					{Name: NewElmTypeName("TaskVisibilityKindOrganizationTeam"), Tag: "organization_team"},
				},
			},
			Enum{
				Name: NewElmTypeName("TaskCapabilityTokenState"),
				Variants: []Variant{
					{Name: NewElmTypeName("TaskCapabilityTokenStateActive"), Tag: "active"},
					{Name: NewElmTypeName("TaskCapabilityTokenStateRevoked"), Tag: "revoked"},
				},
			},
			Enum{
				Name: NewElmTypeName("TaskParticipationPolicy"),
				Variants: []Variant{
					{Name: NewElmTypeName("TaskParticipationPolicyOpen"), Tag: "open"},
					{Name: NewElmTypeName("TaskParticipationPolicyReservationRequired"), Tag: "reservation_required"},
					{Name: NewElmTypeName("TaskParticipationPolicyApprovalRequired"), Tag: "approval_required"},
				},
			},
			Enum{
				Name: NewElmTypeName("TaskAssigneeScope"),
				Variants: []Variant{
					{Name: NewElmTypeName("TaskAssigneeScopeUser"), Tag: "user"},
					{Name: NewElmTypeName("TaskAssigneeScopeOrganizationTeam"), Tag: "organization_team"},
				},
			},
			Enum{
				Name: NewElmTypeName("TaskAvailabilityKind"),
				Variants: []Variant{
					{Name: NewElmTypeName("TaskAvailabilityKindAvailable"), Tag: "available"},
					{Name: NewElmTypeName("TaskAvailabilityKindReserved"), Tag: "reserved"},
					{Name: NewElmTypeName("TaskAvailabilityKindAwaitingApproval"), Tag: "awaiting_approval"},
					{Name: NewElmTypeName("TaskAvailabilityKindClosed"), Tag: "closed"},
				},
			},
			Enum{
				Name: NewElmTypeName("TaskViewerAction"),
				Variants: []Variant{
					{Name: NewElmTypeName("TaskViewerActionSubmit"), Tag: "submit"},
					{Name: NewElmTypeName("TaskViewerActionReserve"), Tag: "reserve"},
					{Name: NewElmTypeName("TaskViewerActionRequestApproval"), Tag: "request_approval"},
					{Name: NewElmTypeName("TaskViewerActionWait"), Tag: "wait"},
					{Name: NewElmTypeName("TaskViewerActionNone"), Tag: "none"},
				},
			},
			Enum{
				Name: NewElmTypeName("TaskReservationState"),
				Variants: []Variant{
					{Name: NewElmTypeName("TaskReservationStateRequested"), Tag: "requested"},
					{Name: NewElmTypeName("TaskReservationStateActive"), Tag: "active"},
					{Name: NewElmTypeName("TaskReservationStateDeclined"), Tag: "declined"},
					{Name: NewElmTypeName("TaskReservationStateCancelledByRequester"), Tag: "cancelled_by_requester"},
					{Name: NewElmTypeName("TaskReservationStateCancelledByWorker"), Tag: "cancelled_by_worker"},
					{Name: NewElmTypeName("TaskReservationStateExpired"), Tag: "expired"},
					{Name: NewElmTypeName("TaskReservationStateSubmitted"), Tag: "submitted"},
				},
			},
			Product{
				Name: NewElmTypeName("TaskListItemResponse"),
				Fields: []Field{
					{Name: NewElmValueName("id"), JSONName: NewJSONFieldName("id"), Type: StringRef{}},
					{Name: NewElmValueName("ownerKind"), JSONName: NewJSONFieldName("owner_kind"), Type: NamedRef{Name: NewElmTypeName("TaskOwnerKind")}},
					{Name: NewElmValueName("title"), JSONName: NewJSONFieldName("title"), Type: StringRef{}},
					{Name: NewElmValueName("rewardKind"), JSONName: NewJSONFieldName("reward_kind"), Type: StringRef{}},
					{Name: NewElmValueName("rewardCreditAmount"), JSONName: NewJSONFieldName("reward_credit_amount"), Type: IntRef{}},
					{Name: NewElmValueName("rewardCollectibleCount"), JSONName: NewJSONFieldName("reward_collectible_count"), Type: IntRef{}},
					{Name: NewElmValueName("participationPolicy"), JSONName: NewJSONFieldName("participation_policy"), Type: NamedRef{Name: NewElmTypeName("TaskParticipationPolicy")}},
					{Name: NewElmValueName("assigneeScope"), JSONName: NewJSONFieldName("assignee_scope"), Type: NamedRef{Name: NewElmTypeName("TaskAssigneeScope")}},
					{Name: NewElmValueName("reservationExpiryHours"), JSONName: NewJSONFieldName("reservation_expiry_hours"), Type: IntRef{}},
					{Name: NewElmValueName("state"), JSONName: NewJSONFieldName("state"), Type: NamedRef{Name: NewElmTypeName("TaskState")}},
					{Name: NewElmValueName("visibilityKind"), JSONName: NewJSONFieldName("visibility_kind"), Type: NamedRef{Name: NewElmTypeName("TaskVisibilityKind")}},
					{Name: NewElmValueName("availabilityKind"), JSONName: NewJSONFieldName("availability_kind"), Type: NamedRef{Name: NewElmTypeName("TaskAvailabilityKind")}},
					{Name: NewElmValueName("viewerAction"), JSONName: NewJSONFieldName("viewer_action"), Type: NamedRef{Name: NewElmTypeName("TaskViewerAction")}},
					{Name: NewElmValueName("reviewerAction"), JSONName: NewJSONFieldName("reviewer_action"), Type: StringRef{}},
					{Name: NewElmValueName("createdBy"), JSONName: NewJSONFieldName("created_by"), Type: StringRef{}},
					{Name: NewElmValueName("activeAssigneeKind"), JSONName: NewJSONFieldName("active_assignee_kind"), Type: StringRef{}},
					{Name: NewElmValueName("activeAssigneeID"), JSONName: NewJSONFieldName("active_assignee_id"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("TaskResponse"),
				Fields: []Field{
					{Name: NewElmValueName("id"), JSONName: NewJSONFieldName("id"), Type: StringRef{}},
					{Name: NewElmValueName("ownerKind"), JSONName: NewJSONFieldName("owner_kind"), Type: NamedRef{Name: NewElmTypeName("TaskOwnerKind")}},
					{Name: NewElmValueName("ownerID"), JSONName: NewJSONFieldName("owner_id"), Type: StringRef{}},
					{Name: NewElmValueName("title"), JSONName: NewJSONFieldName("title"), Type: StringRef{}},
					{Name: NewElmValueName("description"), JSONName: NewJSONFieldName("description"), Type: StringRef{}},
					{Name: NewElmValueName("taskType"), JSONName: NewJSONFieldName("task_type"), Type: StringRef{}},
					{Name: NewElmValueName("referenceURL"), JSONName: NewJSONFieldName("reference_url"), Type: StringRef{}},
					{Name: NewElmValueName("rewardKind"), JSONName: NewJSONFieldName("reward_kind"), Type: StringRef{}},
					{Name: NewElmValueName("rewardCreditAmount"), JSONName: NewJSONFieldName("reward_credit_amount"), Type: IntRef{}},
					{Name: NewElmValueName("rewardCollectibleCount"), JSONName: NewJSONFieldName("reward_collectible_count"), Type: IntRef{}},
					{Name: NewElmValueName("participationPolicy"), JSONName: NewJSONFieldName("participation_policy"), Type: NamedRef{Name: NewElmTypeName("TaskParticipationPolicy")}},
					{Name: NewElmValueName("assigneeScope"), JSONName: NewJSONFieldName("assignee_scope"), Type: NamedRef{Name: NewElmTypeName("TaskAssigneeScope")}},
					{Name: NewElmValueName("reservationExpiryHours"), JSONName: NewJSONFieldName("reservation_expiry_hours"), Type: IntRef{}},
					{Name: NewElmValueName("state"), JSONName: NewJSONFieldName("state"), Type: NamedRef{Name: NewElmTypeName("TaskState")}},
					{Name: NewElmValueName("visibilityKind"), JSONName: NewJSONFieldName("visibility_kind"), Type: NamedRef{Name: NewElmTypeName("TaskVisibilityKind")}},
					{Name: NewElmValueName("visibilityID"), JSONName: NewJSONFieldName("visibility_id"), Type: StringRef{}},
					{Name: NewElmValueName("availabilityKind"), JSONName: NewJSONFieldName("availability_kind"), Type: NamedRef{Name: NewElmTypeName("TaskAvailabilityKind")}},
					{Name: NewElmValueName("viewerAction"), JSONName: NewJSONFieldName("viewer_action"), Type: NamedRef{Name: NewElmTypeName("TaskViewerAction")}},
					{Name: NewElmValueName("reviewerAction"), JSONName: NewJSONFieldName("reviewer_action"), Type: StringRef{}},
					{Name: NewElmValueName("seriesKind"), JSONName: NewJSONFieldName("series_kind"), Type: StringRef{}},
					{Name: NewElmValueName("seriesID"), JSONName: NewJSONFieldName("series_id"), Type: StringRef{}},
					{Name: NewElmValueName("seriesPosition"), JSONName: NewJSONFieldName("series_position"), Type: IntRef{}},
					{Name: NewElmValueName("responseSchemaJSON"), JSONName: NewJSONFieldName("response_schema_json"), Type: StringRef{}},
					{Name: NewElmValueName("payloadKind"), JSONName: NewJSONFieldName("payload_kind"), Type: StringRef{}},
					{Name: NewElmValueName("payloadJSON"), JSONName: NewJSONFieldName("payload_json"), Type: StringRef{}},
					{Name: NewElmValueName("createdBy"), JSONName: NewJSONFieldName("created_by"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("TaskCommentResponse"),
				Fields: []Field{
					{Name: NewElmValueName("id"), JSONName: NewJSONFieldName("id"), Type: StringRef{}},
					{Name: NewElmValueName("taskID"), JSONName: NewJSONFieldName("task_id"), Type: StringRef{}},
					{Name: NewElmValueName("authorUserID"), JSONName: NewJSONFieldName("author_user_id"), Type: StringRef{}},
					{Name: NewElmValueName("body"), JSONName: NewJSONFieldName("body"), Type: StringRef{}},
					{Name: NewElmValueName("createdAt"), JSONName: NewJSONFieldName("created_at"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("TasksResponse"),
				Fields: []Field{
					{Name: NewElmValueName("tasks"), JSONName: NewJSONFieldName("tasks"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("TaskListItemResponse")}}},
				},
			},
			Product{
				Name: NewElmTypeName("UserProfileResponse"),
				Fields: []Field{
					{Name: NewElmValueName("id"), JSONName: NewJSONFieldName("id"), Type: StringRef{}},
					{Name: NewElmValueName("tasks"), JSONName: NewJSONFieldName("tasks"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("TaskListItemResponse")}}},
				},
			},
			Product{
				Name: NewElmTypeName("TaskReservationResponse"),
				Fields: []Field{
					{Name: NewElmValueName("id"), JSONName: NewJSONFieldName("id"), Type: StringRef{}},
					{Name: NewElmValueName("taskID"), JSONName: NewJSONFieldName("task_id"), Type: StringRef{}},
					{Name: NewElmValueName("assigneeKind"), JSONName: NewJSONFieldName("assignee_kind"), Type: NamedRef{Name: NewElmTypeName("TaskAssigneeScope")}},
					{Name: NewElmValueName("assigneeID"), JSONName: NewJSONFieldName("assignee_id"), Type: StringRef{}},
					{Name: NewElmValueName("state"), JSONName: NewJSONFieldName("state"), Type: NamedRef{Name: NewElmTypeName("TaskReservationState")}},
					{Name: NewElmValueName("requestedBy"), JSONName: NewJSONFieldName("requested_by"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("TaskReservationsResponse"),
				Fields: []Field{
					{Name: NewElmValueName("reservations"), JSONName: NewJSONFieldName("reservations"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("TaskReservationResponse")}}},
				},
			},
			Product{
				Name: NewElmTypeName("TaskCapabilityTokenResponse"),
				Fields: []Field{
					{Name: NewElmValueName("id"), JSONName: NewJSONFieldName("id"), Type: StringRef{}},
					{Name: NewElmValueName("taskID"), JSONName: NewJSONFieldName("task_id"), Type: StringRef{}},
					{Name: NewElmValueName("state"), JSONName: NewJSONFieldName("state"), Type: NamedRef{Name: NewElmTypeName("TaskCapabilityTokenState")}},
					{Name: NewElmValueName("token"), JSONName: NewJSONFieldName("token"), Type: StringRef{}},
				},
			},
		},
	}
}

func submissionModule() Module {
	return Module{
		Name: NewModuleName("Sharecrop.Generated.Submission"),
		Definitions: []Definition{
			Enum{
				Name: NewElmTypeName("SubmissionState"),
				Variants: []Variant{
					{Name: NewElmTypeName("SubmissionStateSubmitted"), Tag: "submitted"},
					{Name: NewElmTypeName("SubmissionStateInvalid"), Tag: "invalid"},
					{Name: NewElmTypeName("SubmissionStateAccepted"), Tag: "accepted"},
					{Name: NewElmTypeName("SubmissionStateRejected"), Tag: "rejected"},
					{Name: NewElmTypeName("SubmissionStateChangesRequested"), Tag: "changes_requested"},
				},
			},
			Product{
				Name: NewElmTypeName("SubmissionValidationErrorResponse"),
				Fields: []Field{
					{Name: NewElmValueName("path"), JSONName: NewJSONFieldName("path"), Type: StringRef{}},
					{Name: NewElmValueName("message"), JSONName: NewJSONFieldName("message"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("SubmissionResponse"),
				Fields: []Field{
					{Name: NewElmValueName("id"), JSONName: NewJSONFieldName("id"), Type: StringRef{}},
					{Name: NewElmValueName("taskID"), JSONName: NewJSONFieldName("task_id"), Type: StringRef{}},
					{Name: NewElmValueName("submitterID"), JSONName: NewJSONFieldName("submitter_id"), Type: StringRef{}},
					{Name: NewElmValueName("state"), JSONName: NewJSONFieldName("state"), Type: NamedRef{Name: NewElmTypeName("SubmissionState")}},
					{Name: NewElmValueName("responseJSON"), JSONName: NewJSONFieldName("response_json"), Type: StringRef{}},
					{Name: NewElmValueName("reviewNote"), JSONName: NewJSONFieldName("review_note"), Type: StringRef{}},
					{Name: NewElmValueName("validationErrors"), JSONName: NewJSONFieldName("validation_errors"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("SubmissionValidationErrorResponse")}}},
				},
			},
			Product{
				Name: NewElmTypeName("SubmissionsResponse"),
				Fields: []Field{
					{Name: NewElmValueName("submissions"), JSONName: NewJSONFieldName("submissions"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("SubmissionResponse")}}},
				},
			},
			Product{
				Name: NewElmTypeName("SubmissionCommentResponse"),
				Fields: []Field{
					{Name: NewElmValueName("id"), JSONName: NewJSONFieldName("id"), Type: StringRef{}},
					{Name: NewElmValueName("submissionID"), JSONName: NewJSONFieldName("submission_id"), Type: StringRef{}},
					{Name: NewElmValueName("authorUserID"), JSONName: NewJSONFieldName("author_user_id"), Type: StringRef{}},
					{Name: NewElmValueName("body"), JSONName: NewJSONFieldName("body"), Type: StringRef{}},
					{Name: NewElmValueName("createdAt"), JSONName: NewJSONFieldName("created_at"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("SubmissionCommentsResponse"),
				Fields: []Field{
					{Name: NewElmValueName("comments"), JSONName: NewJSONFieldName("comments"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("SubmissionCommentResponse")}}},
				},
			},
			Product{
				Name: NewElmTypeName("SubmissionCreatedResponse"),
				Fields: []Field{
					{Name: NewElmValueName("submission"), JSONName: NewJSONFieldName("submission"), Type: NamedRef{Name: NewElmTypeName("SubmissionResponse")}},
					{Name: NewElmValueName("receiptToken"), JSONName: NewJSONFieldName("receipt_token"), Type: StringRef{}},
				},
			},
		},
	}
}

func ledgerModule() Module {
	return Module{
		Name: NewModuleName("Sharecrop.Generated.Ledger"),
		Definitions: []Definition{
			Enum{
				Name: NewElmTypeName("LedgerEntryKind"),
				Variants: []Variant{
					{Name: NewElmTypeName("LedgerEntryKindSignupGrant"), Tag: "signup_grant"},
					{Name: NewElmTypeName("LedgerEntryKindTaskEscrow"), Tag: "task_escrow"},
					{Name: NewElmTypeName("LedgerEntryKindTaskRefund"), Tag: "task_refund"},
					{Name: NewElmTypeName("LedgerEntryKindTaskPayout"), Tag: "task_payout"},
					{Name: NewElmTypeName("LedgerEntryKindTaskTip"), Tag: "task_tip"},
					{Name: NewElmTypeName("LedgerEntryKindManualAdjustment"), Tag: "manual_adjustment"},
				},
			},
			Enum{
				Name: NewElmTypeName("EscrowState"),
				Variants: []Variant{
					{Name: NewElmTypeName("EscrowStateHeld"), Tag: "held"},
					{Name: NewElmTypeName("EscrowStateReleased"), Tag: "released"},
					{Name: NewElmTypeName("EscrowStateRefunded"), Tag: "refunded"},
				},
			},
			Product{
				Name: NewElmTypeName("BalanceResponse"),
				Fields: []Field{
					{Name: NewElmValueName("amount"), JSONName: NewJSONFieldName("amount"), Type: IntRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("LedgerEntryResponse"),
				Fields: []Field{
					{Name: NewElmValueName("id"), JSONName: NewJSONFieldName("id"), Type: StringRef{}},
					{Name: NewElmValueName("kind"), JSONName: NewJSONFieldName("kind"), Type: NamedRef{Name: NewElmTypeName("LedgerEntryKind")}},
					{Name: NewElmValueName("amount"), JSONName: NewJSONFieldName("amount"), Type: IntRef{}},
					{Name: NewElmValueName("taskID"), JSONName: NewJSONFieldName("task_id"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("LedgerResponse"),
				Fields: []Field{
					{Name: NewElmValueName("entries"), JSONName: NewJSONFieldName("entries"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("LedgerEntryResponse")}}},
				},
			},
			Product{
				Name: NewElmTypeName("TaskEscrowResponse"),
				Fields: []Field{
					{Name: NewElmValueName("taskID"), JSONName: NewJSONFieldName("task_id"), Type: StringRef{}},
					{Name: NewElmValueName("amount"), JSONName: NewJSONFieldName("amount"), Type: IntRef{}},
					{Name: NewElmValueName("state"), JSONName: NewJSONFieldName("state"), Type: NamedRef{Name: NewElmTypeName("EscrowState")}},
				},
			},
			Product{
				Name: NewElmTypeName("AcceptSubmissionResponse"),
				Fields: []Field{
					{Name: NewElmValueName("taskID"), JSONName: NewJSONFieldName("task_id"), Type: StringRef{}},
					{Name: NewElmValueName("submissionID"), JSONName: NewJSONFieldName("submission_id"), Type: StringRef{}},
					{Name: NewElmValueName("payoutKind"), JSONName: NewJSONFieldName("payout_kind"), Type: StringRef{}},
					{Name: NewElmValueName("payoutAmount"), JSONName: NewJSONFieldName("payout_amount"), Type: IntRef{}},
					{Name: NewElmValueName("workerUserID"), JSONName: NewJSONFieldName("worker_user_id"), Type: StringRef{}},
					{Name: NewElmValueName("collectibleIDs"), JSONName: NewJSONFieldName("collectible_ids"), Type: ListRef{Element: StringRef{}}},
					{Name: NewElmValueName("tipAmount"), JSONName: NewJSONFieldName("tip_amount"), Type: IntRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("ReviewSubmissionResponse"),
				Fields: []Field{
					{Name: NewElmValueName("taskID"), JSONName: NewJSONFieldName("task_id"), Type: StringRef{}},
					{Name: NewElmValueName("submissionID"), JSONName: NewJSONFieldName("submission_id"), Type: StringRef{}},
					{Name: NewElmValueName("state"), JSONName: NewJSONFieldName("state"), Type: StringRef{}},
					{Name: NewElmValueName("reviewNote"), JSONName: NewJSONFieldName("review_note"), Type: StringRef{}},
					{Name: NewElmValueName("payoutKind"), JSONName: NewJSONFieldName("payout_kind"), Type: StringRef{}},
					{Name: NewElmValueName("payoutAmount"), JSONName: NewJSONFieldName("payout_amount"), Type: IntRef{}},
					{Name: NewElmValueName("workerUserID"), JSONName: NewJSONFieldName("worker_user_id"), Type: StringRef{}},
					{Name: NewElmValueName("tipAmount"), JSONName: NewJSONFieldName("tip_amount"), Type: IntRef{}},
				},
			},
		},
	}
}

func agentModule() Module {
	return Module{
		Name: NewModuleName("Sharecrop.Generated.Agent"),
		Definitions: []Definition{
			Enum{
				Name: NewElmTypeName("AgentScope"),
				Variants: []Variant{
					{Name: NewElmTypeName("AgentScopeTasksRead"), Tag: "tasks_read"},
					{Name: NewElmTypeName("AgentScopeTasksWrite"), Tag: "tasks_write"},
					{Name: NewElmTypeName("AgentScopeSubmissionsWrite"), Tag: "submissions_write"},
					{Name: NewElmTypeName("AgentScopeSubmissionsRead"), Tag: "submissions_read"},
					{Name: NewElmTypeName("AgentScopeSubmissionsReview"), Tag: "submissions_review"},
				},
			},
			Enum{
				Name: NewElmTypeName("AgentCredentialState"),
				Variants: []Variant{
					{Name: NewElmTypeName("AgentCredentialStateActive"), Tag: "active"},
					{Name: NewElmTypeName("AgentCredentialStateRevoked"), Tag: "revoked"},
				},
			},
			Product{
				Name: NewElmTypeName("AgentCredentialResponse"),
				Fields: []Field{
					{Name: NewElmValueName("id"), JSONName: NewJSONFieldName("id"), Type: StringRef{}},
					{Name: NewElmValueName("label"), JSONName: NewJSONFieldName("label"), Type: StringRef{}},
					{Name: NewElmValueName("scopes"), JSONName: NewJSONFieldName("scopes"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("AgentScope")}}},
					{Name: NewElmValueName("state"), JSONName: NewJSONFieldName("state"), Type: NamedRef{Name: NewElmTypeName("AgentCredentialState")}},
				},
			},
			Product{
				Name: NewElmTypeName("AgentCredentialsResponse"),
				Fields: []Field{
					{Name: NewElmValueName("credentials"), JSONName: NewJSONFieldName("credentials"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("AgentCredentialResponse")}}},
				},
			},
			Product{
				Name: NewElmTypeName("AgentCredentialCreatedResponse"),
				Fields: []Field{
					{Name: NewElmValueName("credential"), JSONName: NewJSONFieldName("credential"), Type: NamedRef{Name: NewElmTypeName("AgentCredentialResponse")}},
					{Name: NewElmValueName("secret"), JSONName: NewJSONFieldName("secret"), Type: StringRef{}},
				},
			},
		},
	}
}
