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
		eventsModule(),
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
					{Name: NewElmTypeName("ModerationReasonDispute"), Tag: "dispute"},
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
					{Name: NewElmValueName("nextOffset"), JSONName: NewJSONFieldName("next_offset"), Type: IntRef{}},
					// total counts every row matching the filter, ignoring
					// limit/offset.
					{Name: NewElmValueName("total"), JSONName: NewJSONFieldName("total"), Type: IntRef{}},
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
					{Name: NewElmValueName("nextOffset"), JSONName: NewJSONFieldName("next_offset"), Type: IntRef{}},
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
					{Name: NewElmValueName("nextOffset"), JSONName: NewJSONFieldName("next_offset"), Type: IntRef{}},
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
			Enum{
				Name: NewElmTypeName("NotificationKind"),
				Variants: []Variant{
					{Name: NewElmTypeName("NotificationKindSubmissionCreated"), Tag: "submission_created"},
					{Name: NewElmTypeName("NotificationKindSubmissionAccepted"), Tag: "submission_accepted"},
					{Name: NewElmTypeName("NotificationKindSubmissionChangesRequested"), Tag: "submission_changes_requested"},
					{Name: NewElmTypeName("NotificationKindSubmissionRejected"), Tag: "submission_rejected"},
					{Name: NewElmTypeName("NotificationKindSubmissionSuperseded"), Tag: "submission_superseded"},
					{Name: NewElmTypeName("NotificationKindSubmissionCommented"), Tag: "submission_commented"},
					{Name: NewElmTypeName("NotificationKindTaskFunded"), Tag: "task_funded"},
					{Name: NewElmTypeName("NotificationKindTaskCancelled"), Tag: "task_cancelled"},
					{Name: NewElmTypeName("NotificationKindTaskExpired"), Tag: "task_expired"},
					{Name: NewElmTypeName("NotificationKindTaskCommented"), Tag: "task_commented"},
					{Name: NewElmTypeName("NotificationKindSeriesCommented"), Tag: "series_commented"},
					{Name: NewElmTypeName("NotificationKindReservationRequested"), Tag: "reservation_requested"},
					{Name: NewElmTypeName("NotificationKindReservationApproved"), Tag: "reservation_approved"},
					{Name: NewElmTypeName("NotificationKindReservationDeclined"), Tag: "reservation_declined"},
					{Name: NewElmTypeName("NotificationKindReservationCancelled"), Tag: "reservation_cancelled"},
					{Name: NewElmTypeName("NotificationKindReservationExpired"), Tag: "reservation_expired"},
					{Name: NewElmTypeName("NotificationKindPayoutReceived"), Tag: "payout_received"},
					{Name: NewElmTypeName("NotificationKindCreditGranted"), Tag: "credit_granted"},
					{Name: NewElmTypeName("NotificationKindTipReceived"), Tag: "tip_received"},
					{Name: NewElmTypeName("NotificationKindCollectibleAwarded"), Tag: "collectible_awarded"},
					{Name: NewElmTypeName("NotificationKindCollectibleWithdrawn"), Tag: "collectible_withdrawn"},
					{Name: NewElmTypeName("NotificationKindCollectibleReleased"), Tag: "collectible_released"},
					{Name: NewElmTypeName("NotificationKindCreditsReceived"), Tag: "credits_received"},
				},
			},
			Product{
				Name: NewElmTypeName("NotificationResponse"),
				Fields: []Field{
					{Name: NewElmValueName("id"), JSONName: NewJSONFieldName("id"), Type: StringRef{}},
					{Name: NewElmValueName("recipientUserID"), JSONName: NewJSONFieldName("recipient_user_id"), Type: StringRef{}},
					{Name: NewElmValueName("actorUserID"), JSONName: NewJSONFieldName("actor_user_id"), Type: StringRef{}},
					{Name: NewElmValueName("actorDisplayName"), JSONName: NewJSONFieldName("actor_display_name"), Type: StringRef{}},
					{Name: NewElmValueName("kind"), JSONName: NewJSONFieldName("kind"), Type: NamedRef{Name: NewElmTypeName("NotificationKind")}},
					{Name: NewElmValueName("subjectKind"), JSONName: NewJSONFieldName("subject_kind"), Type: StringRef{}},
					{Name: NewElmValueName("subjectID"), JSONName: NewJSONFieldName("subject_id"), Type: StringRef{}},
					{Name: NewElmValueName("subjectTitle"), JSONName: NewJSONFieldName("subject_title"), Type: StringRef{}},
					{Name: NewElmValueName("state"), JSONName: NewJSONFieldName("state"), Type: StringRef{}},
					{Name: NewElmValueName("metadataJSON"), JSONName: NewJSONFieldName("metadata_json"), Type: StringRef{}},
					{Name: NewElmValueName("createdAt"), JSONName: NewJSONFieldName("created_at"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("NotificationsResponse"),
				Fields: []Field{
					{Name: NewElmValueName("notifications"), JSONName: NewJSONFieldName("notifications"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("NotificationResponse")}}},
					{Name: NewElmValueName("nextOffset"), JSONName: NewJSONFieldName("next_offset"), Type: IntRef{}},
					// total counts every row matching the filter, ignoring
					// limit/offset.
					{Name: NewElmValueName("total"), JSONName: NewJSONFieldName("total"), Type: IntRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("NotificationUnreadCountResponse"),
				Fields: []Field{
					{Name: NewElmValueName("unreadCount"), JSONName: NewJSONFieldName("unread_count"), Type: IntRef{}},
				},
			},
		},
	}
}

// eventsModule carries the live event feed AND the webhook management wire
// types in one module: webhook responses reference DomainEventKind, and
// generated Elm modules don't import each other, so splitting them would
// force a duplicated kind enum that could drift (the same reason org
// credentials live inside the agent module).
func eventsModule() Module {
	return Module{
		Name: NewModuleName("Sharecrop.Generated.Events"),
		Definitions: []Definition{
			Enum{
				Name: NewElmTypeName("DomainEventKind"),
				Variants: []Variant{
					{Name: NewElmTypeName("DomainEventKindTaskOpened"), Tag: "task_opened"},
					{Name: NewElmTypeName("DomainEventKindTaskFunded"), Tag: "task_funded"},
					{Name: NewElmTypeName("DomainEventKindTaskCancelled"), Tag: "task_cancelled"},
					{Name: NewElmTypeName("DomainEventKindTaskExpired"), Tag: "task_expired"},
					{Name: NewElmTypeName("DomainEventKindTaskCommented"), Tag: "task_commented"},
					{Name: NewElmTypeName("DomainEventKindSeriesCommented"), Tag: "series_commented"},
					{Name: NewElmTypeName("DomainEventKindReservationRequested"), Tag: "reservation_requested"},
					{Name: NewElmTypeName("DomainEventKindReservationApproved"), Tag: "reservation_approved"},
					{Name: NewElmTypeName("DomainEventKindReservationDeclined"), Tag: "reservation_declined"},
					{Name: NewElmTypeName("DomainEventKindReservationCancelled"), Tag: "reservation_cancelled"},
					{Name: NewElmTypeName("DomainEventKindReservationExpired"), Tag: "reservation_expired"},
					{Name: NewElmTypeName("DomainEventKindSubmissionCreated"), Tag: "submission_created"},
					{Name: NewElmTypeName("DomainEventKindSubmissionAccepted"), Tag: "submission_accepted"},
					{Name: NewElmTypeName("DomainEventKindSubmissionChangesRequested"), Tag: "submission_changes_requested"},
					{Name: NewElmTypeName("DomainEventKindSubmissionRejected"), Tag: "submission_rejected"},
					{Name: NewElmTypeName("DomainEventKindSubmissionSuperseded"), Tag: "submission_superseded"},
					{Name: NewElmTypeName("DomainEventKindSubmissionCommented"), Tag: "submission_commented"},
					{Name: NewElmTypeName("DomainEventKindPayoutReceived"), Tag: "payout_received"},
					{Name: NewElmTypeName("DomainEventKindCreditGranted"), Tag: "credit_granted"},
					{Name: NewElmTypeName("DomainEventKindTipReceived"), Tag: "tip_received"},
					{Name: NewElmTypeName("DomainEventKindCollectibleAwarded"), Tag: "collectible_awarded"},
					{Name: NewElmTypeName("DomainEventKindCollectibleWithdrawn"), Tag: "collectible_withdrawn"},
					{Name: NewElmTypeName("DomainEventKindCollectibleReleased"), Tag: "collectible_released"},
					{Name: NewElmTypeName("DomainEventKindCreditsSent"), Tag: "credits_sent"},
				},
			},
			Enum{
				Name: NewElmTypeName("EventActorKind"),
				Variants: []Variant{
					{Name: NewElmTypeName("EventActorKindUser"), Tag: "user"},
					{Name: NewElmTypeName("EventActorKindSystem"), Tag: "system"},
				},
			},
			Product{
				Name: NewElmTypeName("EventResponse"),
				Fields: []Field{
					{Name: NewElmValueName("id"), JSONName: NewJSONFieldName("id"), Type: StringRef{}},
					{Name: NewElmValueName("kind"), JSONName: NewJSONFieldName("kind"), Type: NamedRef{Name: NewElmTypeName("DomainEventKind")}},
					{Name: NewElmValueName("actorKind"), JSONName: NewJSONFieldName("actor_kind"), Type: NamedRef{Name: NewElmTypeName("EventActorKind")}},
					{Name: NewElmValueName("actorUserID"), JSONName: NewJSONFieldName("actor_user_id"), Type: StringRef{}},
					{Name: NewElmValueName("actorDisplayName"), JSONName: NewJSONFieldName("actor_display_name"), Type: StringRef{}},
					{Name: NewElmValueName("occurredAt"), JSONName: NewJSONFieldName("occurred_at"), Type: StringRef{}},
					{Name: NewElmValueName("cursor"), JSONName: NewJSONFieldName("cursor"), Type: StringRef{}},
					{Name: NewElmValueName("taskID"), JSONName: NewJSONFieldName("task_id"), Type: StringRef{}},
					{Name: NewElmValueName("taskTitle"), JSONName: NewJSONFieldName("task_title"), Type: StringRef{}},
					{Name: NewElmValueName("submissionID"), JSONName: NewJSONFieldName("submission_id"), Type: StringRef{}},
					{Name: NewElmValueName("reservationID"), JSONName: NewJSONFieldName("reservation_id"), Type: StringRef{}},
					{Name: NewElmValueName("seriesID"), JSONName: NewJSONFieldName("series_id"), Type: StringRef{}},
					{Name: NewElmValueName("organizationID"), JSONName: NewJSONFieldName("organization_id"), Type: StringRef{}},
					{Name: NewElmValueName("collectibleID"), JSONName: NewJSONFieldName("collectible_id"), Type: StringRef{}},
					{Name: NewElmValueName("metadataJSON"), JSONName: NewJSONFieldName("metadata_json"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("EventListResponse"),
				Fields: []Field{
					{Name: NewElmValueName("events"), JSONName: NewJSONFieldName("events"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("EventResponse")}}},
					{Name: NewElmValueName("nextCursor"), JSONName: NewJSONFieldName("next_cursor"), Type: StringRef{}},
				},
			},
			Enum{
				Name: NewElmTypeName("WebhookSubscriptionState"),
				Variants: []Variant{
					{Name: NewElmTypeName("WebhookSubscriptionStateActive"), Tag: "active"},
					{Name: NewElmTypeName("WebhookSubscriptionStateRevoked"), Tag: "revoked"},
				},
			},
			Enum{
				Name: NewElmTypeName("WebhookDeliveryState"),
				Variants: []Variant{
					{Name: NewElmTypeName("WebhookDeliveryStatePending"), Tag: "pending"},
					{Name: NewElmTypeName("WebhookDeliveryStateDelivered"), Tag: "delivered"},
					{Name: NewElmTypeName("WebhookDeliveryStateDead"), Tag: "dead"},
				},
			},
			Enum{
				Name: NewElmTypeName("WebhookOwnerKind"),
				Variants: []Variant{
					{Name: NewElmTypeName("WebhookOwnerKindUser"), Tag: "user"},
					{Name: NewElmTypeName("WebhookOwnerKindOrganization"), Tag: "organization"},
				},
			},
			// WebhookAudience: recipient subscriptions deliver events addressed
			// to the owner; marketplace subscriptions deliver every public open
			// task_opened event, optionally narrowed by task type and minimum
			// credit reward.
			Enum{
				Name: NewElmTypeName("WebhookAudience"),
				Variants: []Variant{
					{Name: NewElmTypeName("WebhookAudienceRecipient"), Tag: "recipient"},
					{Name: NewElmTypeName("WebhookAudienceMarketplace"), Tag: "marketplace"},
				},
			},
			Product{
				Name: NewElmTypeName("WebhookSubscriptionResponse"),
				Fields: []Field{
					{Name: NewElmValueName("id"), JSONName: NewJSONFieldName("id"), Type: StringRef{}},
					{Name: NewElmValueName("ownerKind"), JSONName: NewJSONFieldName("owner_kind"), Type: NamedRef{Name: NewElmTypeName("WebhookOwnerKind")}},
					{Name: NewElmValueName("ownerUserID"), JSONName: NewJSONFieldName("owner_user_id"), Type: StringRef{}},
					{Name: NewElmValueName("ownerOrganizationID"), JSONName: NewJSONFieldName("owner_organization_id"), Type: StringRef{}},
					{Name: NewElmValueName("url"), JSONName: NewJSONFieldName("url"), Type: StringRef{}},
					{Name: NewElmValueName("kinds"), JSONName: NewJSONFieldName("kinds"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("DomainEventKind")}}},
					{Name: NewElmValueName("state"), JSONName: NewJSONFieldName("state"), Type: NamedRef{Name: NewElmTypeName("WebhookSubscriptionState")}},
					{Name: NewElmValueName("createdAt"), JSONName: NewJSONFieldName("created_at"), Type: StringRef{}},
					{Name: NewElmValueName("audience"), JSONName: NewJSONFieldName("audience"), Type: NamedRef{Name: NewElmTypeName("WebhookAudience")}},
					// The marketplace narrowing filters: empty / 0 mean no
					// filter, and recipient subscriptions always carry them
					// empty / 0.
					{Name: NewElmValueName("filterTaskType"), JSONName: NewJSONFieldName("filter_task_type"), Type: StringRef{}},
					{Name: NewElmValueName("filterMinCreditReward"), JSONName: NewJSONFieldName("filter_min_credit_reward"), Type: IntRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("WebhookSubscriptionCreatedResponse"),
				Fields: []Field{
					{Name: NewElmValueName("subscription"), JSONName: NewJSONFieldName("subscription"), Type: NamedRef{Name: NewElmTypeName("WebhookSubscriptionResponse")}},
					{Name: NewElmValueName("secret"), JSONName: NewJSONFieldName("secret"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("WebhookSubscriptionsResponse"),
				Fields: []Field{
					{Name: NewElmValueName("subscriptions"), JSONName: NewJSONFieldName("subscriptions"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("WebhookSubscriptionResponse")}}},
					{Name: NewElmValueName("nextOffset"), JSONName: NewJSONFieldName("next_offset"), Type: IntRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("WebhookDeliveryResponse"),
				Fields: []Field{
					{Name: NewElmValueName("id"), JSONName: NewJSONFieldName("id"), Type: StringRef{}},
					{Name: NewElmValueName("eventCursor"), JSONName: NewJSONFieldName("event_cursor"), Type: StringRef{}},
					{Name: NewElmValueName("state"), JSONName: NewJSONFieldName("state"), Type: NamedRef{Name: NewElmTypeName("WebhookDeliveryState")}},
					{Name: NewElmValueName("attemptCount"), JSONName: NewJSONFieldName("attempt_count"), Type: IntRef{}},
					{Name: NewElmValueName("nextAttemptAt"), JSONName: NewJSONFieldName("next_attempt_at"), Type: StringRef{}},
					{Name: NewElmValueName("lastStatus"), JSONName: NewJSONFieldName("last_status"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("WebhookDeliveriesResponse"),
				Fields: []Field{
					{Name: NewElmValueName("deliveries"), JSONName: NewJSONFieldName("deliveries"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("WebhookDeliveryResponse")}}},
					{Name: NewElmValueName("nextOffset"), JSONName: NewJSONFieldName("next_offset"), Type: IntRef{}},
					// total counts every row matching the filter, ignoring
					// limit/offset.
					{Name: NewElmValueName("total"), JSONName: NewJSONFieldName("total"), Type: IntRef{}},
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
			// OperationsCountersResponse mirrors GET
			// /api/admin/operations/counters: outbox and webhook-delivery
			// health plus the current UTC day's economy totals.
			// oldestPendingWebhookAgeSeconds is 0 when no delivery is pending.
			Product{
				Name: NewElmTypeName("OperationsCountersResponse"),
				Fields: []Field{
					{Name: NewElmValueName("outboxRecordedBacklog"), JSONName: NewJSONFieldName("outbox_recorded_backlog"), Type: IntRef{}},
					{Name: NewElmValueName("outboxDispatchFailed"), JSONName: NewJSONFieldName("outbox_dispatch_failed"), Type: IntRef{}},
					{Name: NewElmValueName("webhookDeliveriesPending"), JSONName: NewJSONFieldName("webhook_deliveries_pending"), Type: IntRef{}},
					{Name: NewElmValueName("webhookDeliveriesDead"), JSONName: NewJSONFieldName("webhook_deliveries_dead"), Type: IntRef{}},
					{Name: NewElmValueName("oldestPendingWebhookAgeSeconds"), JSONName: NewJSONFieldName("oldest_pending_webhook_age_seconds"), Type: IntRef{}},
					{Name: NewElmValueName("signupGrantsToday"), JSONName: NewJSONFieldName("signup_grants_today"), Type: IntRef{}},
					{Name: NewElmValueName("peerTransfersToday"), JSONName: NewJSONFieldName("peer_transfers_today"), Type: IntRef{}},
					{Name: NewElmValueName("peerTransferCreditsToday"), JSONName: NewJSONFieldName("peer_transfer_credits_today"), Type: IntRef{}},
					{Name: NewElmValueName("budgetRefusalsToday"), JSONName: NewJSONFieldName("budget_refusals_today"), Type: IntRef{}},
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
					{Name: NewElmValueName("nextOffset"), JSONName: NewJSONFieldName("next_offset"), Type: IntRef{}},
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
					{Name: NewElmValueName("nextOffset"), JSONName: NewJSONFieldName("next_offset"), Type: IntRef{}},
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
					{Name: NewElmTypeName("CollectibleStateWithdrawn"), Tag: "withdrawn"},
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
					// catalogSlug names the catalog entry a catalog-awarded
					// instance came from; empty for custom mints.
					{Name: NewElmValueName("catalogSlug"), JSONName: NewJSONFieldName("catalog_slug"), Type: StringRef{}},
					// editionNumber is the mint sequence number of an edition
					// instance; 0 for unnumbered instances.
					{Name: NewElmValueName("editionNumber"), JSONName: NewJSONFieldName("edition_number"), Type: IntRef{}},
					// issuerDisplayName names the minting/awarding user on
					// list reads; empty on mutation responses.
					{Name: NewElmValueName("issuerDisplayName"), JSONName: NewJSONFieldName("issuer_display_name"), Type: StringRef{}},
					// ownerDisplayName labels the current owner (user display
					// name, organization name, or team name, per owner_kind)
					// on list reads; empty on mutation responses.
					{Name: NewElmValueName("ownerDisplayName"), JSONName: NewJSONFieldName("owner_display_name"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("CollectiblesResponse"),
				Fields: []Field{
					{Name: NewElmValueName("collectibles"), JSONName: NewJSONFieldName("collectibles"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("CollectibleResponse")}}},
					{Name: NewElmValueName("nextOffset"), JSONName: NewJSONFieldName("next_offset"), Type: IntRef{}},
				},
			},
			// CollectibleCatalogEntryState is the catalog entry lifecycle:
			// available entries are awardable; withdrawn entries stay listed
			// (their instances remain in circulation) but can no longer be
			// awarded.
			Enum{
				Name: NewElmTypeName("CollectibleCatalogEntryState"),
				Variants: []Variant{
					{Name: NewElmTypeName("CollectibleCatalogEntryStateAvailable"), Tag: "available"},
					{Name: NewElmTypeName("CollectibleCatalogEntryStateWithdrawn"), Tag: "withdrawn"},
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
					{Name: NewElmValueName("state"), JSONName: NewJSONFieldName("state"), Type: NamedRef{Name: NewElmTypeName("CollectibleCatalogEntryState")}},
					// maxEditions bounds how many instances may ever be
					// minted: 1 for uniques, the run size for editions, 0 for
					// uncapped badges.
					{Name: NewElmValueName("maxEditions"), JSONName: NewJSONFieldName("max_editions"), Type: IntRef{}},
					// mintedCount counts the entry's live (non-withdrawn)
					// instances.
					{Name: NewElmValueName("mintedCount"), JSONName: NewJSONFieldName("minted_count"), Type: IntRef{}},
					// liveOwnerCount counts the distinct owners holding live
					// instances.
					{Name: NewElmValueName("liveOwnerCount"), JSONName: NewJSONFieldName("live_owner_count"), Type: IntRef{}},
					// ownerDisplayName labels the holder of a unique entry's
					// live instance; empty for non-unique entries and
					// unminted slots.
					{Name: NewElmValueName("ownerDisplayName"), JSONName: NewJSONFieldName("owner_display_name"), Type: StringRef{}},
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
					{Name: NewElmValueName("authorDisplayName"), JSONName: NewJSONFieldName("author_display_name"), Type: StringRef{}},
					{Name: NewElmValueName("body"), JSONName: NewJSONFieldName("body"), Type: StringRef{}},
					{Name: NewElmValueName("createdAt"), JSONName: NewJSONFieldName("created_at"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("TaskSeriesListResponse"),
				Fields: []Field{
					{Name: NewElmValueName("series"), JSONName: NewJSONFieldName("series"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("TaskSeriesResponse")}}},
					{Name: NewElmValueName("nextOffset"), JSONName: NewJSONFieldName("next_offset"), Type: IntRef{}},
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
			// ErrorCode mirrors internal/core/domain_error.go: the ten domain
			// codes plus "unavailable", which handlers write for 5xx
			// infrastructure failures.
			Enum{
				Name: NewElmTypeName("ErrorCode"),
				Variants: []Variant{
					{Name: NewElmTypeName("ErrorCodeInvalidID"), Tag: "invalid_id"},
					{Name: NewElmTypeName("ErrorCodeInvalidEnum"), Tag: "invalid_enum"},
					{Name: NewElmTypeName("ErrorCodeInvalidState"), Tag: "invalid_state"},
					{Name: NewElmTypeName("ErrorCodeInvalidArgument"), Tag: "invalid_argument"},
					{Name: NewElmTypeName("ErrorCodeNotFound"), Tag: "not_found"},
					{Name: NewElmTypeName("ErrorCodePermissionDenied"), Tag: "permission_denied"},
					{Name: NewElmTypeName("ErrorCodeConflict"), Tag: "conflict"},
					{Name: NewElmTypeName("ErrorCodeUnauthenticated"), Tag: "unauthenticated"},
					{Name: NewElmTypeName("ErrorCodeRateLimited"), Tag: "rate_limited"},
					{Name: NewElmTypeName("ErrorCodeBudgetExceeded"), Tag: "budget_exceeded"},
					{Name: NewElmTypeName("ErrorCodeUnavailable"), Tag: "unavailable"},
				},
			},
			Product{
				Name: NewElmTypeName("ErrorResponse"),
				Fields: []Field{
					{Name: NewElmValueName("error"), JSONName: NewJSONFieldName("error"), Type: StringRef{}},
					{Name: NewElmValueName("code"), JSONName: NewJSONFieldName("code"), Type: NamedRef{Name: NewElmTypeName("ErrorCode")}},
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
					{Name: NewElmValueName("displayName"), JSONName: NewJSONFieldName("display_name"), Type: StringRef{}},
					// "unverified" or "verified" for user sessions (the
					// signup grant lands at first verification); empty for
					// guest sessions, which have no email.
					{Name: NewElmValueName("emailVerificationState"), JSONName: NewJSONFieldName("email_verification_state"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("AccountProfileResponse"),
				Fields: []Field{
					{Name: NewElmValueName("id"), JSONName: NewJSONFieldName("id"), Type: StringRef{}},
					{Name: NewElmValueName("email"), JSONName: NewJSONFieldName("email"), Type: StringRef{}},
					{Name: NewElmValueName("displayName"), JSONName: NewJSONFieldName("display_name"), Type: StringRef{}},
					// "unverified" or "verified"; the signup credit grant
					// lands when the account first becomes verified.
					{Name: NewElmValueName("emailVerificationState"), JSONName: NewJSONFieldName("email_verification_state"), Type: StringRef{}},
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
					{Name: NewElmValueName("nextOffset"), JSONName: NewJSONFieldName("next_offset"), Type: IntRef{}},
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
					{Name: NewElmValueName("nextOffset"), JSONName: NewJSONFieldName("next_offset"), Type: IntRef{}},
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
					{Name: NewElmValueName("nextOffset"), JSONName: NewJSONFieldName("next_offset"), Type: IntRef{}},
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
					{Name: NewElmTypeName("TaskAvailabilityKindClosed"), Tag: "closed"},
				},
			},
			Enum{
				Name: NewElmTypeName("TaskViewerAction"),
				Variants: []Variant{
					{Name: NewElmTypeName("TaskViewerActionSubmit"), Tag: "submit"},
					{Name: NewElmTypeName("TaskViewerActionReserve"), Tag: "reserve"},
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
			Enum{
				Name: NewElmTypeName("TaskFundedState"),
				Variants: []Variant{
					{Name: NewElmTypeName("TaskFundedStateRewardFunded"), Tag: "reward_funded"},
					{Name: NewElmTypeName("TaskFundedStateRewardUnfunded"), Tag: "reward_unfunded"},
					{Name: NewElmTypeName("TaskFundedStateNoCreditReward"), Tag: "no_credit_reward"},
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
					{Name: NewElmValueName("creatorDisplayName"), JSONName: NewJSONFieldName("creator_display_name"), Type: StringRef{}},
					// holderDisplayName names the user holding the active
					// reservation when the reservation is user-assigned;
					// empty otherwise.
					{Name: NewElmValueName("holderDisplayName"), JSONName: NewJSONFieldName("holder_display_name"), Type: StringRef{}},
					{Name: NewElmValueName("funded"), JSONName: NewJSONFieldName("funded"), Type: NamedRef{Name: NewElmTypeName("TaskFundedState")}},
					// pendingReviewCount is populated only on tasks the caller
					// created; every other row reports 0.
					{Name: NewElmValueName("pendingReviewCount"), JSONName: NewJSONFieldName("pending_review_count"), Type: IntRef{}},
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
					{Name: NewElmValueName("expiresAt"), JSONName: NewJSONFieldName("expires_at"), Type: StringRef{}},
					// creatorDisplayName is resolved on the task detail read
					// path; create and state-change responses leave it empty.
					{Name: NewElmValueName("creatorDisplayName"), JSONName: NewJSONFieldName("creator_display_name"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("TaskCommentResponse"),
				Fields: []Field{
					{Name: NewElmValueName("id"), JSONName: NewJSONFieldName("id"), Type: StringRef{}},
					{Name: NewElmValueName("taskID"), JSONName: NewJSONFieldName("task_id"), Type: StringRef{}},
					{Name: NewElmValueName("authorUserID"), JSONName: NewJSONFieldName("author_user_id"), Type: StringRef{}},
					{Name: NewElmValueName("authorDisplayName"), JSONName: NewJSONFieldName("author_display_name"), Type: StringRef{}},
					{Name: NewElmValueName("body"), JSONName: NewJSONFieldName("body"), Type: StringRef{}},
					{Name: NewElmValueName("createdAt"), JSONName: NewJSONFieldName("created_at"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("TasksResponse"),
				Fields: []Field{
					{Name: NewElmValueName("tasks"), JSONName: NewJSONFieldName("tasks"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("TaskListItemResponse")}}},
					{Name: NewElmValueName("nextOffset"), JSONName: NewJSONFieldName("next_offset"), Type: IntRef{}},
					// total counts every row matching the filter, ignoring
					// limit/offset.
					{Name: NewElmValueName("total"), JSONName: NewJSONFieldName("total"), Type: IntRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("UserProfileResponse"),
				Fields: []Field{
					{Name: NewElmValueName("id"), JSONName: NewJSONFieldName("id"), Type: StringRef{}},
					{Name: NewElmValueName("displayName"), JSONName: NewJSONFieldName("display_name"), Type: StringRef{}},
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
					{Name: NewElmValueName("holderDisplayName"), JSONName: NewJSONFieldName("holder_display_name"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("TaskReservationsResponse"),
				Fields: []Field{
					{Name: NewElmValueName("reservations"), JSONName: NewJSONFieldName("reservations"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("TaskReservationResponse")}}},
					{Name: NewElmValueName("nextOffset"), JSONName: NewJSONFieldName("next_offset"), Type: IntRef{}},
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
					{Name: NewElmTypeName("SubmissionStateSuperseded"), Tag: "superseded"},
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
					{Name: NewElmValueName("submitterDisplayName"), JSONName: NewJSONFieldName("submitter_display_name"), Type: StringRef{}},
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
					{Name: NewElmValueName("nextOffset"), JSONName: NewJSONFieldName("next_offset"), Type: IntRef{}},
					// total counts every row matching the filter, ignoring
					// limit/offset.
					{Name: NewElmValueName("total"), JSONName: NewJSONFieldName("total"), Type: IntRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("SubmissionCommentResponse"),
				Fields: []Field{
					{Name: NewElmValueName("id"), JSONName: NewJSONFieldName("id"), Type: StringRef{}},
					{Name: NewElmValueName("submissionID"), JSONName: NewJSONFieldName("submission_id"), Type: StringRef{}},
					{Name: NewElmValueName("authorUserID"), JSONName: NewJSONFieldName("author_user_id"), Type: StringRef{}},
					{Name: NewElmValueName("authorDisplayName"), JSONName: NewJSONFieldName("author_display_name"), Type: StringRef{}},
					{Name: NewElmValueName("body"), JSONName: NewJSONFieldName("body"), Type: StringRef{}},
					{Name: NewElmValueName("createdAt"), JSONName: NewJSONFieldName("created_at"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("SubmissionCommentsResponse"),
				Fields: []Field{
					{Name: NewElmValueName("comments"), JSONName: NewJSONFieldName("comments"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("SubmissionCommentResponse")}}},
					{Name: NewElmValueName("nextOffset"), JSONName: NewJSONFieldName("next_offset"), Type: IntRef{}},
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
			// BanSelection is the reject-review choice about the submitting
			// worker: "none" leaves the worker free to submit again;
			// "ban_implementor" blocks the worker from this task.
			Enum{
				Name: NewElmTypeName("BanSelection"),
				Variants: []Variant{
					{Name: NewElmTypeName("BanSelectionNone"), Tag: "none"},
					{Name: NewElmTypeName("BanSelectionBanImplementor"), Tag: "ban_implementor"},
				},
			},
			Enum{
				Name: NewElmTypeName("LedgerEntryKind"),
				Variants: []Variant{
					{Name: NewElmTypeName("LedgerEntryKindSignupGrant"), Tag: "signup_grant"},
					{Name: NewElmTypeName("LedgerEntryKindTaskFund"), Tag: "task_fund"},
					{Name: NewElmTypeName("LedgerEntryKindTaskRefund"), Tag: "task_refund"},
					{Name: NewElmTypeName("LedgerEntryKindTaskPayout"), Tag: "task_payout"},
					{Name: NewElmTypeName("LedgerEntryKindTaskTip"), Tag: "task_tip"},
					{Name: NewElmTypeName("LedgerEntryKindManualAdjustment"), Tag: "manual_adjustment"},
					{Name: NewElmTypeName("LedgerEntryKindPeerTransfer"), Tag: "peer_transfer"},
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
					// note is the entry's stored note (for example the required
					// explanation on a platform-admin credit grant); empty for
					// entry kinds that record no note.
					{Name: NewElmValueName("note"), JSONName: NewJSONFieldName("note"), Type: StringRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("CreditGrantResponse"),
				Fields: []Field{
					{Name: NewElmValueName("entryID"), JSONName: NewJSONFieldName("entry_id"), Type: StringRef{}},
					{Name: NewElmValueName("amount"), JSONName: NewJSONFieldName("amount"), Type: IntRef{}},
				},
			},
			// CreditTransferSourceKind says whose balance a peer credit send
			// debits: the sender's own, or an organization's (the sender needs
			// its billing permission).
			Enum{
				Name: NewElmTypeName("CreditTransferSourceKind"),
				Variants: []Variant{
					{Name: NewElmTypeName("CreditTransferSourceSelf"), Tag: "self"},
					{Name: NewElmTypeName("CreditTransferSourceOrganization"), Tag: "organization"},
				},
			},
			// CreditTransferTargetKind says whose account a peer credit send
			// credits.
			Enum{
				Name: NewElmTypeName("CreditTransferTargetKind"),
				Variants: []Variant{
					{Name: NewElmTypeName("CreditTransferTargetUser"), Tag: "user"},
					{Name: NewElmTypeName("CreditTransferTargetOrganization"), Tag: "organization"},
				},
			},
			// CreditTransferResponse reports the sender-side ledger entry of a
			// completed peer credit send (POST /api/credits/transfers).
			Product{
				Name: NewElmTypeName("CreditTransferResponse"),
				Fields: []Field{
					{Name: NewElmValueName("entryID"), JSONName: NewJSONFieldName("entry_id"), Type: StringRef{}},
					{Name: NewElmValueName("amount"), JSONName: NewJSONFieldName("amount"), Type: IntRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("LedgerResponse"),
				Fields: []Field{
					{Name: NewElmValueName("entries"), JSONName: NewJSONFieldName("entries"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("LedgerEntryResponse")}}},
					{Name: NewElmValueName("nextOffset"), JSONName: NewJSONFieldName("next_offset"), Type: IntRef{}},
					// total counts every row matching the filter, ignoring
					// limit/offset.
					{Name: NewElmValueName("total"), JSONName: NewJSONFieldName("total"), Type: IntRef{}},
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
					{Name: NewElmTypeName("AgentScopeLedgerWrite"), Tag: "ledger_write"},
					{Name: NewElmTypeName("AgentScopeModerationRead"), Tag: "moderation_read"},
					{Name: NewElmTypeName("AgentScopeModerationManage"), Tag: "moderation_manage"},
					{Name: NewElmTypeName("AgentScopePrivacyRead"), Tag: "privacy_read"},
					{Name: NewElmTypeName("AgentScopePrivacyManage"), Tag: "privacy_manage"},
					{Name: NewElmTypeName("AgentScopePlatformAdmin"), Tag: "platform_admin"},
					{Name: NewElmTypeName("AgentScopeCredentialsManage"), Tag: "credentials_manage"},
					{Name: NewElmTypeName("AgentScopeWebhooksRead"), Tag: "webhooks_read"},
					{Name: NewElmTypeName("AgentScopeWebhooksManage"), Tag: "webhooks_manage"},
				},
			},
			Enum{
				Name: NewElmTypeName("AgentCredentialState"),
				Variants: []Variant{
					{Name: NewElmTypeName("AgentCredentialStateActive"), Tag: "active"},
					{Name: NewElmTypeName("AgentCredentialStateRevoked"), Tag: "revoked"},
				},
			},
			// WorkSeekingState is the default-deny work-seeking switch: every
			// credential is minted work_seeking_disabled and the owner must
			// enable it with a daily task budget before the credential can
			// reserve tasks or submit on its own.
			Enum{
				Name: NewElmTypeName("WorkSeekingState"),
				Variants: []Variant{
					{Name: NewElmTypeName("WorkSeekingStateDisabled"), Tag: "work_seeking_disabled"},
					{Name: NewElmTypeName("WorkSeekingStateEnabled"), Tag: "work_seeking_enabled"},
				},
			},
			// AgentWorkPolicyResponse is the stored work budget. Allowance
			// fields use 0 / empty for "not configured" (unlimited
			// concurrency and spend, every task type, no reward floor, no
			// advisory token budget); a disabled policy carries every
			// allowance as 0 / empty. The token budget is advisory only: the
			// server stores and returns it but never enforces it.
			Product{
				Name: NewElmTypeName("AgentWorkPolicyResponse"),
				Fields: []Field{
					{Name: NewElmValueName("workSeeking"), JSONName: NewJSONFieldName("work_seeking"), Type: NamedRef{Name: NewElmTypeName("WorkSeekingState")}},
					{Name: NewElmValueName("maxTasksPerDay"), JSONName: NewJSONFieldName("max_tasks_per_day"), Type: IntRef{}},
					{Name: NewElmValueName("maxConcurrentReservations"), JSONName: NewJSONFieldName("max_concurrent_reservations"), Type: IntRef{}},
					{Name: NewElmValueName("maxCreditsPerDay"), JSONName: NewJSONFieldName("max_credits_per_day"), Type: IntRef{}},
					{Name: NewElmValueName("taskTypes"), JSONName: NewJSONFieldName("task_types"), Type: ListRef{Element: StringRef{}}},
					{Name: NewElmValueName("minRewardCredits"), JSONName: NewJSONFieldName("min_reward_credits"), Type: IntRef{}},
					{Name: NewElmValueName("tokenBudgetTokens"), JSONName: NewJSONFieldName("token_budget_tokens"), Type: IntRef{}},
					{Name: NewElmValueName("tokenBudgetNote"), JSONName: NewJSONFieldName("token_budget_note"), Type: StringRef{}},
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
					{Name: NewElmValueName("workPolicy"), JSONName: NewJSONFieldName("work_policy"), Type: NamedRef{Name: NewElmTypeName("AgentWorkPolicyResponse")}},
					// Today's consumption (current UTC day): consumed
					// daily-task units, credits spent via the credential, and
					// the credential's still-active reservations.
					{Name: NewElmValueName("tasksUsedToday"), JSONName: NewJSONFieldName("tasks_used_today"), Type: IntRef{}},
					{Name: NewElmValueName("creditsSpentToday"), JSONName: NewJSONFieldName("credits_spent_today"), Type: IntRef{}},
					{Name: NewElmValueName("activeReservations"), JSONName: NewJSONFieldName("active_reservations"), Type: IntRef{}},
				},
			},
			Product{
				Name: NewElmTypeName("AgentCredentialsResponse"),
				Fields: []Field{
					{Name: NewElmValueName("credentials"), JSONName: NewJSONFieldName("credentials"), Type: ListRef{Element: NamedRef{Name: NewElmTypeName("AgentCredentialResponse")}}},
					{Name: NewElmValueName("nextOffset"), JSONName: NewJSONFieldName("next_offset"), Type: IntRef{}},
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
					{Name: NewElmValueName("nextOffset"), JSONName: NewJSONFieldName("next_offset"), Type: IntRef{}},
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
