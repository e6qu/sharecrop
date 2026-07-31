module Sharecrop.Labels exposing (..)

import Http
import Sharecrop.Generated.Agent as Agent
import Sharecrop.Generated.Collectible as Collectible
import Sharecrop.Generated.Events as Events
import Sharecrop.Generated.Ledger as Ledger
import Sharecrop.Generated.Notification as Notification
import Sharecrop.Generated.Submission as Submission
import Sharecrop.Generated.Task as Task


collectibleKindTag : Collectible.CollectibleKind -> String
collectibleKindTag kind =
    case kind of
        Collectible.CollectibleKindUnique ->
            "unique"

        Collectible.CollectibleKindEdition ->
            "edition"

        Collectible.CollectibleKindBadge ->
            "badge"


collectibleKindLabel : Collectible.CollectibleKind -> String
collectibleKindLabel kind =
    case kind of
        Collectible.CollectibleKindUnique ->
            "Unique"

        Collectible.CollectibleKindEdition ->
            "Edition"

        Collectible.CollectibleKindBadge ->
            "Badge"


collectiblePolicyTag : Collectible.CollectibleTransferPolicy -> String
collectiblePolicyTag policy =
    case policy of
        Collectible.CollectibleTransferPolicyNonTransferableExceptPayout ->
            "non_transferable_except_payout"

        Collectible.CollectibleTransferPolicyTransferableBetweenUsers ->
            "transferable_between_users"

        Collectible.CollectibleTransferPolicyTransferableWithinOrganization ->
            "transferable_within_organization"

        Collectible.CollectibleTransferPolicyIssuerControlled ->
            "issuer_controlled"


collectiblePolicyLabel : Collectible.CollectibleTransferPolicy -> String
collectiblePolicyLabel policy =
    case policy of
        Collectible.CollectibleTransferPolicyNonTransferableExceptPayout ->
            "Non-transferable except payout"

        Collectible.CollectibleTransferPolicyTransferableBetweenUsers ->
            "Transferable between users"

        Collectible.CollectibleTransferPolicyTransferableWithinOrganization ->
            "Transferable within organization"

        Collectible.CollectibleTransferPolicyIssuerControlled ->
            "Issuer controlled"


collectibleStateLabel : Collectible.CollectibleState -> String
collectibleStateLabel state =
    case state of
        Collectible.CollectibleStateMinted ->
            "minted"

        Collectible.CollectibleStateEscrowed ->
            "escrowed"

        Collectible.CollectibleStateAwarded ->
            "awarded"


participationPolicyTag : Task.TaskParticipationPolicy -> String
participationPolicyTag policy =
    case policy of
        Task.TaskParticipationPolicyOpen ->
            "open"

        Task.TaskParticipationPolicyReservationRequired ->
            "reservation_required"

        Task.TaskParticipationPolicyApprovalRequired ->
            "approval_required"


participationUsesReservation : String -> Bool
participationUsesReservation tag =
    tag == "reservation_required" || tag == "approval_required"


participationPolicyLabel : Task.TaskParticipationPolicy -> String
participationPolicyLabel policy =
    case policy of
        Task.TaskParticipationPolicyOpen ->
            "open submissions"

        Task.TaskParticipationPolicyReservationRequired ->
            "reservation required"

        Task.TaskParticipationPolicyApprovalRequired ->
            "approval required"


assigneeScopeTag : Task.TaskAssigneeScope -> String
assigneeScopeTag scope =
    case scope of
        Task.TaskAssigneeScopeUser ->
            "user"

        Task.TaskAssigneeScopeOrganizationTeam ->
            "organization_team"

        Task.TaskAssigneeScopeTeam ->
            "team"


assigneeScopeLabel : Task.TaskAssigneeScope -> String
assigneeScopeLabel scope =
    case scope of
        Task.TaskAssigneeScopeUser ->
            "user"

        Task.TaskAssigneeScopeOrganizationTeam ->
            "organization team"

        Task.TaskAssigneeScopeTeam ->
            "team"


availabilityKindLabel : Task.TaskAvailabilityKind -> String
availabilityKindLabel kind =
    case kind of
        Task.TaskAvailabilityKindAvailable ->
            "available to work"

        Task.TaskAvailabilityKindReserved ->
            "reserved"

        Task.TaskAvailabilityKindAwaitingApproval ->
            "awaiting approval"

        Task.TaskAvailabilityKindClosed ->
            -- Not a bare "closed": next to a draft's "open submissions"
            -- participation badge that read as a contradiction.
            "not open for work"


viewerActionLabel : Task.TaskViewerAction -> String
viewerActionLabel action =
    case action of
        Task.TaskViewerActionSubmit ->
            "submit"

        Task.TaskViewerActionReserve ->
            "reserve"

        Task.TaskViewerActionRequestApproval ->
            "request approval"

        Task.TaskViewerActionWait ->
            "wait"

        Task.TaskViewerActionNone ->
            "none"


reservationStateLabel : Task.TaskReservationState -> String
reservationStateLabel state =
    case state of
        Task.TaskReservationStateRequested ->
            "requested"

        Task.TaskReservationStateActive ->
            "active"

        Task.TaskReservationStateDeclined ->
            "declined"

        Task.TaskReservationStateCancelledByRequester ->
            "cancelled by requester"

        Task.TaskReservationStateCancelledByWorker ->
            "cancelled by worker"

        Task.TaskReservationStateExpired ->
            "expired"

        Task.TaskReservationStateSubmitted ->
            "submitted"


allScopes : List Agent.AgentScope
allScopes =
    [ Agent.AgentScopeTasksRead
    , Agent.AgentScopeTasksWrite
    , Agent.AgentScopeSubmissionsWrite
    , Agent.AgentScopeSubmissionsRead
    , Agent.AgentScopeSubmissionsReview
    , Agent.AgentScopeOrgRead
    , Agent.AgentScopeOrgManage
    , Agent.AgentScopeCollectiblesRead
    , Agent.AgentScopeCollectiblesManage
    , Agent.AgentScopeNotificationsRead
    , Agent.AgentScopeNotificationsManage
    , Agent.AgentScopeUsersRead
    , Agent.AgentScopeLedgerRead
    , Agent.AgentScopeModerationRead
    , Agent.AgentScopeModerationManage
    , Agent.AgentScopePrivacyRead
    , Agent.AgentScopePrivacyManage
    , Agent.AgentScopePlatformAdmin
    , Agent.AgentScopeCredentialsManage
    ]


scopeTag : Agent.AgentScope -> String
scopeTag scope =
    case scope of
        Agent.AgentScopeTasksRead ->
            "tasks_read"

        Agent.AgentScopeTasksWrite ->
            "tasks_write"

        Agent.AgentScopeSubmissionsWrite ->
            "submissions_write"

        Agent.AgentScopeSubmissionsRead ->
            "submissions_read"

        Agent.AgentScopeSubmissionsReview ->
            "submissions_review"

        Agent.AgentScopeOrgRead ->
            "org_read"

        Agent.AgentScopeOrgManage ->
            "org_manage"

        Agent.AgentScopeCollectiblesRead ->
            "collectibles_read"

        Agent.AgentScopeCollectiblesManage ->
            "collectibles_manage"

        Agent.AgentScopeNotificationsRead ->
            "notifications_read"

        Agent.AgentScopeNotificationsManage ->
            "notifications_manage"

        Agent.AgentScopeUsersRead ->
            "users_read"

        Agent.AgentScopeLedgerRead ->
            "ledger_read"

        Agent.AgentScopeModerationRead ->
            "moderation_read"

        Agent.AgentScopeModerationManage ->
            "moderation_manage"

        Agent.AgentScopePrivacyRead ->
            "privacy_read"

        Agent.AgentScopePrivacyManage ->
            "privacy_manage"

        Agent.AgentScopePlatformAdmin ->
            "platform_admin"

        Agent.AgentScopeCredentialsManage ->
            "credentials_manage"

        Agent.AgentScopeWebhooksRead ->
            "webhooks_read"

        Agent.AgentScopeWebhooksManage ->
            "webhooks_manage"


scopeLabel : Agent.AgentScope -> String
scopeLabel scope =
    case scope of
        Agent.AgentScopeTasksRead ->
            "Read tasks"

        Agent.AgentScopeTasksWrite ->
            "Create tasks"

        Agent.AgentScopeSubmissionsWrite ->
            "Submit work"

        Agent.AgentScopeSubmissionsRead ->
            "Read submissions"

        Agent.AgentScopeSubmissionsReview ->
            "Review submissions"

        Agent.AgentScopeOrgRead ->
            "Read organizations"

        Agent.AgentScopeOrgManage ->
            "Manage organizations"

        Agent.AgentScopeCollectiblesRead ->
            "Read collectibles"

        Agent.AgentScopeCollectiblesManage ->
            "Manage collectibles"

        Agent.AgentScopeNotificationsRead ->
            "Read notifications"

        Agent.AgentScopeNotificationsManage ->
            "Manage notifications"

        Agent.AgentScopeUsersRead ->
            "Read user directory"

        Agent.AgentScopeLedgerRead ->
            "Read ledger"

        Agent.AgentScopeModerationRead ->
            "Read moderation reports"

        Agent.AgentScopeModerationManage ->
            "Triage moderation reports"

        Agent.AgentScopePrivacyRead ->
            "Read privacy requests"

        Agent.AgentScopePrivacyManage ->
            "Manage privacy requests"

        Agent.AgentScopePlatformAdmin ->
            "Platform admin"

        Agent.AgentScopeCredentialsManage ->
            "Manage own credentials"

        Agent.AgentScopeWebhooksRead ->
            "Read webhook subscriptions"

        Agent.AgentScopeWebhooksManage ->
            "Manage webhook subscriptions"


credentialStateLabel : Agent.AgentCredentialState -> String
credentialStateLabel state =
    case state of
        Agent.AgentCredentialStateActive ->
            "active"

        Agent.AgentCredentialStateRevoked ->
            "revoked"


taskStateGuidance : Task.TaskState -> String
taskStateGuidance state =
    case state of
        Task.TaskStateDraft ->
            "Next step: fund this task (if it offers a reward) and then open it so workers can submit."

        Task.TaskStateOpen ->
            "Workers can submit now. Review submissions below to accept, request changes, or reject."

        Task.TaskStateClosed ->
            "This task is closed. An accepted submission has been settled."

        Task.TaskStateCancelled ->
            "This task was cancelled. Any allocated reward was returned to the funder's spendable balance."

        Task.TaskStateExpired ->
            "This task expired without an accepted submission."


taskStateLabel : Task.TaskState -> String
taskStateLabel state =
    case state of
        Task.TaskStateDraft ->
            "draft"

        Task.TaskStateOpen ->
            "open"

        Task.TaskStateClosed ->
            "closed"

        Task.TaskStateCancelled ->
            "cancelled"

        Task.TaskStateExpired ->
            "expired"


submissionStateLabel : Submission.SubmissionState -> String
submissionStateLabel state =
    case state of
        Submission.SubmissionStateSubmitted ->
            "submitted"

        Submission.SubmissionStateInvalid ->
            "invalid"

        Submission.SubmissionStateAccepted ->
            "accepted"

        Submission.SubmissionStateRejected ->
            "rejected"

        Submission.SubmissionStateChangesRequested ->
            "changes requested"


kindLabel : Ledger.LedgerEntryKind -> String
kindLabel kind =
    case kind of
        Ledger.LedgerEntryKindSignupGrant ->
            "Signup grant"

        Ledger.LedgerEntryKindTaskFund ->
            "Task funding"

        Ledger.LedgerEntryKindTaskRefund ->
            "Task refund"

        Ledger.LedgerEntryKindTaskPayout ->
            "Task payout"

        Ledger.LedgerEntryKindTaskTip ->
            "Task tip"

        Ledger.LedgerEntryKindManualAdjustment ->
            "Manual adjustment"


{-| The wire tag for a notification kind, unchanged from when the field was a
raw string, so inbox rows read exactly as before the kind became an enum.
-}
notificationKindLabel : Notification.NotificationKind -> String
notificationKindLabel kind =
    case kind of
        Notification.NotificationKindSubmissionCreated ->
            "submission_created"

        Notification.NotificationKindSubmissionAccepted ->
            "submission_accepted"

        Notification.NotificationKindSubmissionChangesRequested ->
            "submission_changes_requested"

        Notification.NotificationKindSubmissionRejected ->
            "submission_rejected"

        Notification.NotificationKindSubmissionCommented ->
            "submission_commented"

        Notification.NotificationKindTaskFunded ->
            "task_funded"

        Notification.NotificationKindTaskCancelled ->
            "task_cancelled"

        Notification.NotificationKindTaskExpired ->
            "task_expired"

        Notification.NotificationKindTaskCommented ->
            "task_commented"

        Notification.NotificationKindSeriesCommented ->
            "series_commented"

        Notification.NotificationKindReservationRequested ->
            "reservation_requested"

        Notification.NotificationKindReservationApproved ->
            "reservation_approved"

        Notification.NotificationKindReservationDeclined ->
            "reservation_declined"

        Notification.NotificationKindReservationCancelled ->
            "reservation_cancelled"

        Notification.NotificationKindReservationExpired ->
            "reservation_expired"

        Notification.NotificationKindPayoutReceived ->
            "payout_received"

        Notification.NotificationKindTipReceived ->
            "tip_received"

        Notification.NotificationKindCollectibleAwarded ->
            "collectible_awarded"


{-| Human phrasing for a domain event kind, for the Overview activity feed
and webhook subscription rows.
-}
domainEventKindLabel : Events.DomainEventKind -> String
domainEventKindLabel kind =
    case kind of
        Events.DomainEventKindTaskOpened ->
            "Task opened"

        Events.DomainEventKindTaskFunded ->
            "Task funded"

        Events.DomainEventKindTaskCancelled ->
            "Task cancelled"

        Events.DomainEventKindTaskExpired ->
            "Task expired"

        Events.DomainEventKindTaskCommented ->
            "Task commented"

        Events.DomainEventKindSeriesCommented ->
            "Series commented"

        Events.DomainEventKindReservationRequested ->
            "Reservation requested"

        Events.DomainEventKindReservationApproved ->
            "Reservation approved"

        Events.DomainEventKindReservationDeclined ->
            "Reservation declined"

        Events.DomainEventKindReservationCancelled ->
            "Reservation cancelled"

        Events.DomainEventKindReservationExpired ->
            "Reservation expired"

        Events.DomainEventKindSubmissionCreated ->
            "Submission received"

        Events.DomainEventKindSubmissionAccepted ->
            "Submission accepted"

        Events.DomainEventKindSubmissionChangesRequested ->
            "Changes requested"

        Events.DomainEventKindSubmissionRejected ->
            "Submission rejected"

        Events.DomainEventKindSubmissionCommented ->
            "Submission commented"

        Events.DomainEventKindPayoutReceived ->
            "Payout received"

        Events.DomainEventKindTipReceived ->
            "Tip received"

        Events.DomainEventKindCollectibleAwarded ->
            "Collectible awarded"


{-| The wire tag for a domain event kind - used for webhook checkbox testids
and the create-subscription request body.
-}
domainEventKindTag : Events.DomainEventKind -> String
domainEventKindTag kind =
    case kind of
        Events.DomainEventKindTaskOpened ->
            "task_opened"

        Events.DomainEventKindTaskFunded ->
            "task_funded"

        Events.DomainEventKindTaskCancelled ->
            "task_cancelled"

        Events.DomainEventKindTaskExpired ->
            "task_expired"

        Events.DomainEventKindTaskCommented ->
            "task_commented"

        Events.DomainEventKindSeriesCommented ->
            "series_commented"

        Events.DomainEventKindReservationRequested ->
            "reservation_requested"

        Events.DomainEventKindReservationApproved ->
            "reservation_approved"

        Events.DomainEventKindReservationDeclined ->
            "reservation_declined"

        Events.DomainEventKindReservationCancelled ->
            "reservation_cancelled"

        Events.DomainEventKindReservationExpired ->
            "reservation_expired"

        Events.DomainEventKindSubmissionCreated ->
            "submission_created"

        Events.DomainEventKindSubmissionAccepted ->
            "submission_accepted"

        Events.DomainEventKindSubmissionChangesRequested ->
            "submission_changes_requested"

        Events.DomainEventKindSubmissionRejected ->
            "submission_rejected"

        Events.DomainEventKindSubmissionCommented ->
            "submission_commented"

        Events.DomainEventKindPayoutReceived ->
            "payout_received"

        Events.DomainEventKindTipReceived ->
            "tip_received"

        Events.DomainEventKindCollectibleAwarded ->
            "collectible_awarded"


rewardLabel : String -> Int -> Int -> String
rewardLabel kind amount collectibleCount =
    case kind of
        "credit" ->
            if collectibleCount > 0 then
                String.fromInt amount ++ " credits + " ++ collectibleCountLabel collectibleCount

            else
                String.fromInt amount ++ " credits"

        "collectible" ->
            collectibleCountLabel collectibleCount

        "bundle" ->
            String.fromInt amount ++ " credits + " ++ collectibleCountLabel collectibleCount

        _ ->
            if collectibleCount > 0 then
                collectibleCountLabel collectibleCount

            else
                "no reward"


collectibleCountLabel : Int -> String
collectibleCountLabel count =
    if count == 1 then
        "1 collectible"

    else
        String.fromInt count ++ " collectibles"


httpErrorLabel : Http.Error -> String
httpErrorLabel error =
    case error of
        Http.BadUrl url ->
            "Bad URL: " ++ url

        Http.Timeout ->
            "The request timed out."

        Http.NetworkError ->
            "A network error occurred."

        Http.BadStatus status ->
            "The request failed with status " ++ String.fromInt status ++ "."

        Http.BadBody message ->
            -- Carries either the backend's own error message (the common
            -- case - see Api.expectJsonWithServerError) or, more rarely, a
            -- genuine response-decoding failure. Shown as-is either way:
            -- the former is already a complete, readable reason, and the
            -- latter is a real bug worth seeing rather than hiding.
            message
