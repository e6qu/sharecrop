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
					{Name: NewElmValueName("createdBy"), JSONName: NewJSONFieldName("created_by"), Type: StringRef{}},
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
					{Name: NewElmValueName("organizationID"), JSONName: NewJSONFieldName("organization_id"), Type: StringRef{}},
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
			Product{
				Name: NewElmTypeName("TaskListItemResponse"),
				Fields: []Field{
					{Name: NewElmValueName("id"), JSONName: NewJSONFieldName("id"), Type: StringRef{}},
					{Name: NewElmValueName("ownerKind"), JSONName: NewJSONFieldName("owner_kind"), Type: NamedRef{Name: NewElmTypeName("TaskOwnerKind")}},
					{Name: NewElmValueName("title"), JSONName: NewJSONFieldName("title"), Type: StringRef{}},
					{Name: NewElmValueName("state"), JSONName: NewJSONFieldName("state"), Type: NamedRef{Name: NewElmTypeName("TaskState")}},
					{Name: NewElmValueName("visibilityKind"), JSONName: NewJSONFieldName("visibility_kind"), Type: NamedRef{Name: NewElmTypeName("TaskVisibilityKind")}},
					{Name: NewElmValueName("createdBy"), JSONName: NewJSONFieldName("created_by"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("TasksResponse"),
				Fields: []Field{
					{Name: NewElmValueName("tasks"), JSONName: NewJSONFieldName("tasks"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("TaskListItemResponse")}}},
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
				},
			},
			Enum{
				Name: NewElmTypeName("SubmitterKind"),
				Variants: []Variant{
					{Name: NewElmTypeName("SubmitterKindAuthenticated"), Tag: "authenticated"},
					{Name: NewElmTypeName("SubmitterKindAnonymous"), Tag: "anonymous"},
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
					{Name: NewElmValueName("submitterKind"), JSONName: NewJSONFieldName("submitter_kind"), Type: NamedRef{Name: NewElmTypeName("SubmitterKind")}},
					{Name: NewElmValueName("state"), JSONName: NewJSONFieldName("state"), Type: NamedRef{Name: NewElmTypeName("SubmissionState")}},
					{Name: NewElmValueName("responseJSON"), JSONName: NewJSONFieldName("response_json"), Type: StringRef{}},
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
