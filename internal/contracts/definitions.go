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
		notificationModule(),
		privacyModule(),
		moderationModule(),
		savedQueueViewsModule(),
	}
}

func moderationModule() Module {
	return Module{
		Name: NewModuleName("Sharecrop.Generated.Moderation"),
		Definitions: []Definition{
			Enum{
				Name: NewElmTypeName("ModerationSubjectKind"),
				Variants: []Variant{
					{Name: NewElmTypeName("ModerationSubjectKindTask"), Tag: "task"},
					{Name: NewElmTypeName("ModerationSubjectKindSubmission"), Tag: "submission"},
					{Name: NewElmTypeName("ModerationSubjectKindTaskComment"), Tag: "task_comment"},
					{Name: NewElmTypeName("ModerationSubjectKindSubmissionComment"), Tag: "submission_comment"},
					{Name: NewElmTypeName("ModerationSubjectKindTaskSeriesComment"), Tag: "task_series_comment"},
					{Name: NewElmTypeName("ModerationSubjectKindUser"), Tag: "user"},
					{Name: NewElmTypeName("ModerationSubjectKindOrganization"), Tag: "organization"},
					{Name: NewElmTypeName("ModerationSubjectKindTeam"), Tag: "team"},
					{Name: NewElmTypeName("ModerationSubjectKindCollectible"), Tag: "collectible"},
				},
			},
			Enum{
				Name: NewElmTypeName("ModerationReason"),
				Variants: []Variant{
					{Name: NewElmTypeName("ModerationReasonSpam"), Tag: "spam"},
					{Name: NewElmTypeName("ModerationReasonAbuse"), Tag: "abuse"},
					{Name: NewElmTypeName("ModerationReasonPII"), Tag: "pii"},
					{Name: NewElmTypeName("ModerationReasonPolicy"), Tag: "policy"},
					{Name: NewElmTypeName("ModerationReasonOther"), Tag: "other"},
				},
			},
			Product{
				Name: NewElmTypeName("ModerationReportResponse"),
				Fields: []Field{
					{Name: NewElmValueName("id"), JSONName: NewJSONFieldName("id"), Type: StringRef{}},
					{Name: NewElmValueName("subjectKind"), JSONName: NewJSONFieldName("subject_kind"), Type: StringRef{}},
					{Name: NewElmValueName("subjectID"), JSONName: NewJSONFieldName("subject_id"), Type: StringRef{}},
					{Name: NewElmValueName("subjectHref"), JSONName: NewJSONFieldName("subject_href"), Type: StringRef{}},
					{Name: NewElmValueName("reason"), JSONName: NewJSONFieldName("reason"), Type: StringRef{}},
					{Name: NewElmValueName("details"), JSONName: NewJSONFieldName("details"), Type: StringRef{}},
					{Name: NewElmValueName("reporterUserID"), JSONName: NewJSONFieldName("reporter_user_id"), Type: StringRef{}},
					{Name: NewElmValueName("createdAt"), JSONName: NewJSONFieldName("created_at"), Type: StringRef{}},
					{Name: NewElmValueName("state"), JSONName: NewJSONFieldName("state"), Type: StringRef{}},
					{Name: NewElmValueName("resolutionNote"), JSONName: NewJSONFieldName("resolution_note"), Type: StringRef{}},
					{Name: NewElmValueName("updatedBy"), JSONName: NewJSONFieldName("updated_by"), Type: StringRef{}},
					{Name: NewElmValueName("updatedAt"), JSONName: NewJSONFieldName("updated_at"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("ModerationReportsResponse"),
				Fields: []Field{
					{Name: NewElmValueName("reports"), JSONName: NewJSONFieldName("reports"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("ModerationReportResponse")}}},
				},
			},
		},
	}
}

func savedQueueViewsModule() Module {
	return Module{
		Name: NewModuleName("Sharecrop.Generated.SavedQueueViews"),
		Definitions: []Definition{
			Product{
				Name: NewElmTypeName("SavedQueueViewResponse"),
				Fields: []Field{
					{Name: NewElmValueName("id"), JSONName: NewJSONFieldName("id"), Type: StringRef{}},
					{Name: NewElmValueName("scope"), JSONName: NewJSONFieldName("scope"), Type: StringRef{}},
					{Name: NewElmValueName("name"), JSONName: NewJSONFieldName("name"), Type: StringRef{}},
					{Name: NewElmValueName("query"), JSONName: NewJSONFieldName("query"), Type: StringRef{}},
					{Name: NewElmValueName("stateFilter"), JSONName: NewJSONFieldName("state_filter"), Type: StringRef{}},
					{Name: NewElmValueName("typeFilter"), JSONName: NewJSONFieldName("type_filter"), Type: StringRef{}},
					{Name: NewElmValueName("sort"), JSONName: NewJSONFieldName("sort"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("SavedQueueViewsResponse"),
				Fields: []Field{
					{Name: NewElmValueName("views"), JSONName: NewJSONFieldName("views"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("SavedQueueViewResponse")}}},
				},
			},
		},
	}
}

func privacyModule() Module {
	return Module{
		Name: NewModuleName("Sharecrop.Generated.Privacy"),
		Definitions: []Definition{
			Enum{
				Name: NewElmTypeName("PrivacyRequestKind"),
				Variants: []Variant{
					{Name: NewElmTypeName("PrivacyRequestKindDataExport"), Tag: "data_export"},
					{Name: NewElmTypeName("PrivacyRequestKindSensitiveFieldDeletion"), Tag: "sensitive_field_deletion"},
				},
			},
			Product{
				Name: NewElmTypeName("PrivacyRequestResponse"),
				Fields: []Field{
					{Name: NewElmValueName("id"), JSONName: NewJSONFieldName("id"), Type: StringRef{}},
					{Name: NewElmValueName("kind"), JSONName: NewJSONFieldName("kind"), Type: StringRef{}},
					{Name: NewElmValueName("status"), JSONName: NewJSONFieldName("status"), Type: StringRef{}},
					{Name: NewElmValueName("requestedBy"), JSONName: NewJSONFieldName("requested_by"), Type: StringRef{}},
					{Name: NewElmValueName("exportJSON"), JSONName: NewJSONFieldName("export_json"), Type: StringRef{}},
					{Name: NewElmValueName("resolutionNote"), JSONName: NewJSONFieldName("resolution_note"), Type: StringRef{}},
					{Name: NewElmValueName("createdAt"), JSONName: NewJSONFieldName("created_at"), Type: StringRef{}},
					{Name: NewElmValueName("resolvedAt"), JSONName: NewJSONFieldName("resolved_at"), Type: StringRef{}},
					{Name: NewElmValueName("redactedFieldCount"), JSONName: NewJSONFieldName("redacted_field_count"), Type: IntRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("PrivacyRequestsResponse"),
				Fields: []Field{
					{Name: NewElmValueName("requests"), JSONName: NewJSONFieldName("requests"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("PrivacyRequestResponse")}}},
				},
			},
			Product{
				Name: NewElmTypeName("PrivacyRetentionRunResponse"),
				Fields: []Field{
					{Name: NewElmValueName("redactedFieldCount"), JSONName: NewJSONFieldName("redacted_field_count"), Type: IntRef{}},
				},
			},
		},
	}
}

func notificationModule() Module {
	return Module{
		Name: NewModuleName("Sharecrop.Generated.Notification"),
		Definitions: []Definition{
			Product{
				Name: NewElmTypeName("NotificationResponse"),
				Fields: []Field{
					{Name: NewElmValueName("id"), JSONName: NewJSONFieldName("id"), Type: StringRef{}},
					{Name: NewElmValueName("recipientUserID"), JSONName: NewJSONFieldName("recipient_user_id"), Type: StringRef{}},
					{Name: NewElmValueName("actorUserID"), JSONName: NewJSONFieldName("actor_user_id"), Type: StringRef{}},
					{Name: NewElmValueName("kind"), JSONName: NewJSONFieldName("kind"), Type: StringRef{}},
					{Name: NewElmValueName("subjectKind"), JSONName: NewJSONFieldName("subject_kind"), Type: StringRef{}},
					{Name: NewElmValueName("subjectID"), JSONName: NewJSONFieldName("subject_id"), Type: StringRef{}},
					{Name: NewElmValueName("state"), JSONName: NewJSONFieldName("state"), Type: StringRef{}},
					{Name: NewElmValueName("metadataJSON"), JSONName: NewJSONFieldName("metadata_json"), Type: StringRef{}},
					{Name: NewElmValueName("createdAt"), JSONName: NewJSONFieldName("created_at"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("NotificationsResponse"),
				Fields: []Field{
					{Name: NewElmValueName("notifications"), JSONName: NewJSONFieldName("notifications"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("NotificationResponse")}}},
				},
			},
		},
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
			Product{
				Name: NewElmTypeName("PlatformAdminResponse"),
				Fields: []Field{
					{Name: NewElmValueName("userID"), JSONName: NewJSONFieldName("user_id"), Type: StringRef{}},
					{Name: NewElmValueName("source"), JSONName: NewJSONFieldName("source"), Type: StringRef{}},
					{Name: NewElmValueName("createdAt"), JSONName: NewJSONFieldName("created_at"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("PlatformAdminsResponse"),
				Fields: []Field{
					{Name: NewElmValueName("admins"), JSONName: NewJSONFieldName("admins"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("PlatformAdminResponse")}}},
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
					{Name: NewElmValueName("organizationID"), JSONName: NewJSONFieldName("organization_id"), Type: StringRef{}},
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
					{Name: NewElmValueName("username"), JSONName: NewJSONFieldName("username"), Type: StringRef{}},
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
					{Name: NewElmTypeName("TaskAssigneeScopeTeam"), Tag: "team"},
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
				Name: NewElmTypeName("TaskAttachmentResponse"),
				Fields: []Field{
					{Name: NewElmValueName("name"), JSONName: NewJSONFieldName("name"), Type: StringRef{}},
					{Name: NewElmValueName("contentType"), JSONName: NewJSONFieldName("content_type"), Type: StringRef{}},
					{Name: NewElmValueName("sizeBytes"), JSONName: NewJSONFieldName("size_bytes"), Type: IntRef{}},
					{Name: NewElmValueName("dataURL"), JSONName: NewJSONFieldName("data_url"), Type: StringRef{}},
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
					{Name: NewElmValueName("allocatedCredits"), JSONName: NewJSONFieldName("allocated_credits"), Type: IntRef{}},
					{Name: NewElmValueName("allocatedCollectibleIDs"), JSONName: NewJSONFieldName("allocated_collectible_ids"), Type: ListRef{Element: StringRef{}}},
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
					{Name: NewElmValueName("attachments"), JSONName: NewJSONFieldName("attachments"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("TaskAttachmentResponse")}}},
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
					{Name: NewElmValueName("issuedWorkerCredential"), JSONName: NewJSONFieldName("issued_worker_credential"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("TaskReservationsResponse"),
				Fields: []Field{
					{Name: NewElmValueName("reservations"), JSONName: NewJSONFieldName("reservations"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("TaskReservationResponse")}}},
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
				Name: NewElmTypeName("SubmissionSensitiveFieldResponse"),
				Fields: []Field{
					{Name: NewElmValueName("path"), JSONName: NewJSONFieldName("path"), Type: StringRef{}},
					{Name: NewElmValueName("category"), JSONName: NewJSONFieldName("category"), Type: StringRef{}},
					{Name: NewElmValueName("retention"), JSONName: NewJSONFieldName("retention"), Type: StringRef{}},
					{Name: NewElmValueName("redaction"), JSONName: NewJSONFieldName("redaction"), Type: StringRef{}},
					{Name: NewElmValueName("state"), JSONName: NewJSONFieldName("state"), Type: StringRef{}},
					{Name: NewElmValueName("redactedAt"), JSONName: NewJSONFieldName("redacted_at"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("SubmissionAttachmentResponse"),
				Fields: []Field{
					{Name: NewElmValueName("name"), JSONName: NewJSONFieldName("name"), Type: StringRef{}},
					{Name: NewElmValueName("contentType"), JSONName: NewJSONFieldName("content_type"), Type: StringRef{}},
					{Name: NewElmValueName("sizeBytes"), JSONName: NewJSONFieldName("size_bytes"), Type: IntRef{}},
					{Name: NewElmValueName("dataURL"), JSONName: NewJSONFieldName("data_url"), Type: StringRef{}},
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
					{Name: NewElmValueName("attachments"), JSONName: NewJSONFieldName("attachments"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("SubmissionAttachmentResponse")}}},
					{Name: NewElmValueName("validationErrors"), JSONName: NewJSONFieldName("validation_errors"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("SubmissionValidationErrorResponse")}}},
					{Name: NewElmValueName("sensitiveFields"), JSONName: NewJSONFieldName("sensitive_fields"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("SubmissionSensitiveFieldResponse")}}},
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
					{Name: NewElmTypeName("LedgerEntryKindTaskFund"), Tag: "task_fund"},
					{Name: NewElmTypeName("LedgerEntryKindTaskRefund"), Tag: "task_refund"},
					{Name: NewElmTypeName("LedgerEntryKindTaskPayout"), Tag: "task_payout"},
					{Name: NewElmTypeName("LedgerEntryKindTaskTip"), Tag: "task_tip"},
					{Name: NewElmTypeName("LedgerEntryKindManualAdjustment"), Tag: "manual_adjustment"},
				},
			},
			Product{
				Name: NewElmTypeName("BalanceResponse"),
				Fields: []Field{
					{Name: NewElmValueName("spendableCredits"), JSONName: NewJSONFieldName("spendable_credits"), Type: IntRef{}},
					{Name: NewElmValueName("allocatedCredits"), JSONName: NewJSONFieldName("allocated_credits"), Type: IntRef{}},
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
				Name: NewElmTypeName("TaskFundResponse"),
				Fields: []Field{
					{Name: NewElmValueName("taskID"), JSONName: NewJSONFieldName("task_id"), Type: StringRef{}},
					{Name: NewElmValueName("creditAmount"), JSONName: NewJSONFieldName("credit_amount"), Type: IntRef{}},
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
					{Name: NewElmTypeName("AgentScopeOrgRead"), Tag: "org_read"},
					{Name: NewElmTypeName("AgentScopeOrgManage"), Tag: "org_manage"},
					{Name: NewElmTypeName("AgentScopeCollectiblesRead"), Tag: "collectibles_read"},
					{Name: NewElmTypeName("AgentScopeCollectiblesManage"), Tag: "collectibles_manage"},
					{Name: NewElmTypeName("AgentScopeNotificationsRead"), Tag: "notifications_read"},
					{Name: NewElmTypeName("AgentScopeNotificationsManage"), Tag: "notifications_manage"},
					{Name: NewElmTypeName("AgentScopeUsersRead"), Tag: "users_read"},
					{Name: NewElmTypeName("AgentScopeLedgerRead"), Tag: "ledger_read"},
					{Name: NewElmTypeName("AgentScopeModerationRead"), Tag: "moderation_read"},
					{Name: NewElmTypeName("AgentScopeModerationManage"), Tag: "moderation_manage"},
					{Name: NewElmTypeName("AgentScopePrivacyRead"), Tag: "privacy_read"},
					{Name: NewElmTypeName("AgentScopePrivacyManage"), Tag: "privacy_manage"},
					{Name: NewElmTypeName("AgentScopePlatformAdmin"), Tag: "platform_admin"},
					{Name: NewElmTypeName("AgentScopeCredentialsManage"), Tag: "credentials_manage"},
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
					{Name: NewElmValueName("expiresAt"), JSONName: NewJSONFieldName("expires_at"), Type: StringRef{}},
					{Name: NewElmValueName("taskID"), JSONName: NewJSONFieldName("task_id"), Type: StringRef{}},
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
			// Organization-wide credentials reuse agent.Scope/agent.State
			// directly on the Go side (internal/orgcred/models.go), so their
			// wire types live here too rather than in a new module: generated
			// Elm modules don't import each other, and duplicating the scope
			// enum would risk the two copies drifting apart.
			Product{
				Name: NewElmTypeName("OrgCredentialResponse"),
				Fields: []Field{
					{Name: NewElmValueName("id"), JSONName: NewJSONFieldName("id"), Type: StringRef{}},
					{Name: NewElmValueName("organizationID"), JSONName: NewJSONFieldName("organization_id"), Type: StringRef{}},
					{Name: NewElmValueName("label"), JSONName: NewJSONFieldName("label"), Type: StringRef{}},
					{Name: NewElmValueName("scopes"), JSONName: NewJSONFieldName("scopes"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("AgentScope")}}},
					{Name: NewElmValueName("state"), JSONName: NewJSONFieldName("state"), Type: NamedRef{Name: NewElmTypeName("AgentCredentialState")}},
					{Name: NewElmValueName("expiresAt"), JSONName: NewJSONFieldName("expires_at"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("OrgCredentialsResponse"),
				Fields: []Field{
					{Name: NewElmValueName("credentials"), JSONName: NewJSONFieldName("credentials"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("OrgCredentialResponse")}}},
				},
			},
			Product{
				Name: NewElmTypeName("OrgCredentialCreatedResponse"),
				Fields: []Field{
					{Name: NewElmValueName("credential"), JSONName: NewJSONFieldName("credential"), Type: NamedRef{Name: NewElmTypeName("OrgCredentialResponse")}},
					{Name: NewElmValueName("secret"), JSONName: NewJSONFieldName("secret"), Type: StringRef{}},
				},
			},
		},
	}
}
