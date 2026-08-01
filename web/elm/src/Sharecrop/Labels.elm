module Sharecrop.Labels exposing (..)

import Http
import Json.Decode as Decode
import Sharecrop.Generated.Agent as Agent
import Time
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

        Collectible.CollectibleStateWithdrawn ->
            "withdrawn"


participationPolicyTag : Task.TaskParticipationPolicy -> String
participationPolicyTag policy =
    case policy of
        Task.TaskParticipationPolicyOpen ->
            "open"

        Task.TaskParticipationPolicyReservationRequired ->
            "reservation_required"


participationUsesReservation : String -> Bool
participationUsesReservation tag =
    tag == "reservation_required"


participationPolicyLabel : Task.TaskParticipationPolicy -> String
participationPolicyLabel policy =
    case policy of
        Task.TaskParticipationPolicyOpen ->
            -- "open participation", not "open submissions": next to a
            -- review queue the latter reads as "submissions are waiting".
            "open participation"

        Task.TaskParticipationPolicyReservationRequired ->
            "reservation required"


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
    , Agent.AgentScopeLedgerWrite
    , Agent.AgentScopeModerationRead
    , Agent.AgentScopeModerationManage
    , Agent.AgentScopePrivacyRead
    , Agent.AgentScopePrivacyManage
    , Agent.AgentScopePlatformAdmin
    , Agent.AgentScopeCredentialsManage
    , Agent.AgentScopeWebhooksRead
    , Agent.AgentScopeWebhooksManage
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

        Agent.AgentScopeLedgerWrite ->
            "ledger_write"

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

        Agent.AgentScopeLedgerWrite ->
            "Send credits"

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

        Submission.SubmissionStateSuperseded ->
            "superseded"


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

        Ledger.LedgerEntryKindPeerTransfer ->
            "Credit send"


{-| Human label for a notification kind, used as the compact heading of an
inbox row (the full sentence comes from notificationSentence).
-}
notificationKindLabel : Notification.NotificationKind -> String
notificationKindLabel kind =
    case kind of
        Notification.NotificationKindSubmissionCreated ->
            "Submission received"

        Notification.NotificationKindSubmissionAccepted ->
            "Submission accepted"

        Notification.NotificationKindSubmissionChangesRequested ->
            "Changes requested"

        Notification.NotificationKindSubmissionRejected ->
            "Submission rejected"

        Notification.NotificationKindSubmissionSuperseded ->
            "Submission superseded"

        Notification.NotificationKindSubmissionCommented ->
            "Submission comment"

        Notification.NotificationKindTaskFunded ->
            "Task funded"

        Notification.NotificationKindTaskCancelled ->
            "Task cancelled"

        Notification.NotificationKindTaskExpired ->
            "Task expired"

        Notification.NotificationKindTaskCommented ->
            "Task comment"

        Notification.NotificationKindSeriesCommented ->
            "Series comment"

        Notification.NotificationKindReservationRequested ->
            -- Reserving is immediate (no approval gate), so the wire kind's
            -- historical "requested" phrasing would misreport a done deal.
            "Task reserved"

        Notification.NotificationKindReservationApproved ->
            "Reservation approved"

        Notification.NotificationKindReservationDeclined ->
            "Reservation declined"

        Notification.NotificationKindReservationCancelled ->
            "Reservation cancelled"

        Notification.NotificationKindReservationExpired ->
            "Reservation expired"

        Notification.NotificationKindPayoutReceived ->
            "Payout received"

        Notification.NotificationKindCreditGranted ->
            "Credits granted"

        Notification.NotificationKindTipReceived ->
            "Tip received"

        Notification.NotificationKindCollectibleAwarded ->
            "Collectible awarded"

        Notification.NotificationKindCollectibleWithdrawn ->
            "Collectible withdrawn"

        Notification.NotificationKindCreditsReceived ->
            "Credits received"


{-| An inbox row as one readable sentence: who did what to which subject,
e.g. "Ren Okafor submitted a response to 'Weekly QA sweep'." Falls back to
neutral nouns when the actor or subject title is blank (system events,
redacted or deleted subjects). The metadata JSON supplies the credit amount
for money kinds ("Ren Okafor sent you 25 credits."), degrading to a
plain "credits" when absent.
-}
notificationSentence : String -> String -> String -> Notification.NotificationKind -> String
notificationSentence actorName subjectTitle metadataJSON kind =
    let
        actor =
            personOrSomeone actorName

        task =
            titledOr "a task" subjectTitle

        series =
            titledOr "a series" subjectTitle

        credits =
            creditsPhrase metadataJSON
    in
    case kind of
        Notification.NotificationKindSubmissionCreated ->
            actor ++ " submitted a response to " ++ task ++ "."

        Notification.NotificationKindSubmissionAccepted ->
            actor ++ " accepted your submission to " ++ task ++ "."

        Notification.NotificationKindSubmissionChangesRequested ->
            actor ++ " requested changes on your submission to " ++ task ++ "."

        Notification.NotificationKindSubmissionRejected ->
            actor ++ " rejected your submission to " ++ task ++ "."

        Notification.NotificationKindSubmissionSuperseded ->
            "Your submission to " ++ task ++ " was superseded by an accepted submission."

        Notification.NotificationKindSubmissionCommented ->
            actor ++ " commented on a submission to " ++ task ++ "."

        Notification.NotificationKindTaskFunded ->
            actor ++ " funded " ++ task ++ "."

        Notification.NotificationKindTaskCancelled ->
            capitalized task ++ " was cancelled."

        Notification.NotificationKindTaskExpired ->
            capitalized task ++ " expired."

        Notification.NotificationKindTaskCommented ->
            actor ++ " commented on " ++ task ++ "."

        Notification.NotificationKindSeriesCommented ->
            actor ++ " commented on the series " ++ series ++ "."

        Notification.NotificationKindReservationRequested ->
            -- Reserving is immediate now; "requested" would read as pending.
            actor ++ " reserved " ++ task ++ "."

        Notification.NotificationKindReservationApproved ->
            actor ++ " approved a reservation on " ++ task ++ "."

        Notification.NotificationKindReservationDeclined ->
            actor ++ " declined a reservation on " ++ task ++ "."

        Notification.NotificationKindReservationCancelled ->
            actor ++ " cancelled a reservation on " ++ task ++ "."

        Notification.NotificationKindReservationExpired ->
            "A reservation on " ++ task ++ " expired."

        Notification.NotificationKindPayoutReceived ->
            "You received a payout for " ++ task ++ "."

        Notification.NotificationKindCreditGranted ->
            actor ++ " granted you " ++ credits ++ "."

        Notification.NotificationKindTipReceived ->
            actor ++ " sent you a tip for " ++ task ++ "."

        Notification.NotificationKindCollectibleAwarded ->
            "You were awarded " ++ titledOr "a collectible" subjectTitle ++ "."

        Notification.NotificationKindCollectibleWithdrawn ->
            withdrawingActor actorName ++ " withdrew " ++ titledOr "a collectible" subjectTitle ++ " from your inventory."

        Notification.NotificationKindCreditsReceived ->
            actor ++ " sent you " ++ credits ++ "."


{-| Withdrawing a collectible is an admin act; a blank actor still reads as
an authority ("An administrator withdrew…"), not an anonymous "Someone".
-}
withdrawingActor : String -> String
withdrawingActor actorName =
    if String.trim actorName == "" then
        "An administrator"

    else
        String.trim actorName


{-| "25 credits" when the event metadata carries the amount, else "credits".
-}
creditsPhrase : String -> String
creditsPhrase metadataJSON =
    case Decode.decodeString (Decode.field "amount" Decode.int) metadataJSON of
        Ok amount ->
            String.fromInt amount ++ " credits"

        Err _ ->
            "credits"


{-| An activity-feed row as one readable sentence, built from the event's
actor display name and task title.
-}
eventSentence : String -> String -> Events.DomainEventKind -> String
eventSentence actorName taskTitle kind =
    let
        actor =
            personOrSomeone actorName

        task =
            titledOr "a task" taskTitle
    in
    case kind of
        Events.DomainEventKindTaskOpened ->
            actor ++ " opened " ++ task ++ "."

        Events.DomainEventKindTaskFunded ->
            actor ++ " funded " ++ task ++ "."

        Events.DomainEventKindTaskCancelled ->
            actor ++ " cancelled " ++ task ++ "."

        Events.DomainEventKindTaskExpired ->
            capitalized task ++ " expired."

        Events.DomainEventKindTaskCommented ->
            actor ++ " commented on " ++ task ++ "."

        Events.DomainEventKindSeriesCommented ->
            actor ++ " commented on a series."

        Events.DomainEventKindReservationRequested ->
            -- Reserving is immediate now; "requested" would read as pending.
            actor ++ " reserved " ++ task ++ "."

        Events.DomainEventKindReservationApproved ->
            actor ++ " approved a reservation on " ++ task ++ "."

        Events.DomainEventKindReservationDeclined ->
            actor ++ " declined a reservation on " ++ task ++ "."

        Events.DomainEventKindReservationCancelled ->
            actor ++ " cancelled a reservation on " ++ task ++ "."

        Events.DomainEventKindReservationExpired ->
            "A reservation on " ++ task ++ " expired."

        Events.DomainEventKindSubmissionCreated ->
            actor ++ " submitted a response to " ++ task ++ "."

        Events.DomainEventKindSubmissionAccepted ->
            actor ++ " accepted a submission to " ++ task ++ "."

        Events.DomainEventKindSubmissionChangesRequested ->
            actor ++ " requested changes on a submission to " ++ task ++ "."

        Events.DomainEventKindSubmissionRejected ->
            actor ++ " rejected a submission to " ++ task ++ "."

        Events.DomainEventKindSubmissionSuperseded ->
            "A submission to " ++ task ++ " was superseded by an accepted submission."

        Events.DomainEventKindSubmissionCommented ->
            actor ++ " commented on a submission to " ++ task ++ "."

        Events.DomainEventKindPayoutReceived ->
            actor ++ " received a payout for " ++ task ++ "."

        Events.DomainEventKindCreditGranted ->
            actor ++ " received a credit grant."

        Events.DomainEventKindTipReceived ->
            actor ++ " received a tip for " ++ task ++ "."

        Events.DomainEventKindCollectibleAwarded ->
            actor ++ " was awarded a collectible."

        Events.DomainEventKindCollectibleWithdrawn ->
            "A collectible was withdrawn."

        Events.DomainEventKindCreditsSent ->
            actor ++ " sent credits."


personOrSomeone : String -> String
personOrSomeone name =
    if String.trim name == "" then
        "Someone"

    else
        String.trim name


titledOr : String -> String -> String
titledOr fallback title =
    if String.trim title == "" then
        fallback

    else
        "'" ++ String.trim title ++ "'"


capitalized : String -> String
capitalized phrase =
    case String.uncons phrase of
        Just ( first, rest ) ->
            String.cons (Char.toUpper first) rest

        Nothing ->
            phrase


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
            -- Reserving is immediate (no approval gate); see the matching
            -- notification-kind label.
            "Task reserved"

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

        Events.DomainEventKindSubmissionSuperseded ->
            "Submission superseded"

        Events.DomainEventKindSubmissionCommented ->
            "Submission commented"

        Events.DomainEventKindPayoutReceived ->
            "Payout received"

        Events.DomainEventKindCreditGranted ->
            "Credit granted"

        Events.DomainEventKindTipReceived ->
            "Tip received"

        Events.DomainEventKindCollectibleAwarded ->
            "Collectible awarded"

        Events.DomainEventKindCollectibleWithdrawn ->
            "Collectible withdrawn"

        Events.DomainEventKindCreditsSent ->
            "Credits sent"


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

        Events.DomainEventKindSubmissionSuperseded ->
            "submission_superseded"

        Events.DomainEventKindSubmissionCommented ->
            "submission_commented"

        Events.DomainEventKindPayoutReceived ->
            "payout_received"

        Events.DomainEventKindCreditGranted ->
            "credit_granted"

        Events.DomainEventKindTipReceived ->
            "tip_received"

        Events.DomainEventKindCollectibleAwarded ->
            "collectible_awarded"

        Events.DomainEventKindCollectibleWithdrawn ->
            "collectible_withdrawn"

        Events.DomainEventKindCreditsSent ->
            "credits_sent"


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


{-| "2 hours ago"-style phrasing for an RFC3339 timestamp, relative to the
model's clock. Falls back to the raw timestamp when the clock has not ticked
yet (now = epoch) or the timestamp does not parse - a wrong relative time
would be worse than a precise absolute one.
-}
relativeTimeLabel : Time.Posix -> String -> String
relativeTimeLabel now raw =
    case posixFromRFC3339 raw of
        Nothing ->
            raw

        Just instant ->
            if Time.posixToMillis now == 0 then
                raw

            else
                relativePhrase (Time.posixToMillis now - Time.posixToMillis instant) raw


relativePhrase : Int -> String -> String
relativePhrase deltaMillis raw =
    let
        seconds =
            deltaMillis // 1000
    in
    if seconds < 0 then
        -- A timestamp slightly ahead of the local clock (skew) is "just
        -- now"; anything genuinely in the future keeps its absolute form.
        if seconds > -120 then
            "just now"

        else
            raw

    else if seconds < 60 then
        "just now"

    else if seconds < 3600 then
        countPhrase (seconds // 60) "minute"

    else if seconds < 86400 then
        countPhrase (seconds // 3600) "hour"

    else if seconds < 30 * 86400 then
        countPhrase (seconds // 86400) "day"

    else
        -- Beyond a month, the calendar date reads better than "47 days ago".
        "on " ++ String.left 10 raw


countPhrase : Int -> String -> String
countPhrase count unit =
    if count == 1 then
        "1 " ++ unit ++ " ago"

    else
        String.fromInt count ++ " " ++ unit ++ "s ago"


{-| A humanized absolute instant, e.g. "Sep 15, 2026, 17:00 UTC" — for
expiry displays, where the exact calendar moment matters more than a
relative phrase. Falls back to the raw timestamp when it does not parse.
-}
absoluteTimeLabel : String -> String
absoluteTimeLabel raw =
    case posixFromRFC3339 raw of
        Nothing ->
            raw

        Just instant ->
            monthShortName (Time.toMonth Time.utc instant)
                ++ " "
                ++ String.fromInt (Time.toDay Time.utc instant)
                ++ ", "
                ++ String.fromInt (Time.toYear Time.utc instant)
                ++ ", "
                ++ String.padLeft 2 '0' (String.fromInt (Time.toHour Time.utc instant))
                ++ ":"
                ++ String.padLeft 2 '0' (String.fromInt (Time.toMinute Time.utc instant))
                ++ " UTC"


monthShortName : Time.Month -> String
monthShortName month =
    case month of
        Time.Jan ->
            "Jan"

        Time.Feb ->
            "Feb"

        Time.Mar ->
            "Mar"

        Time.Apr ->
            "Apr"

        Time.May ->
            "May"

        Time.Jun ->
            "Jun"

        Time.Jul ->
            "Jul"

        Time.Aug ->
            "Aug"

        Time.Sep ->
            "Sep"

        Time.Oct ->
            "Oct"

        Time.Nov ->
            "Nov"

        Time.Dec ->
            "Dec"


{-| Parses the RFC3339 instants the API emits ("2026-07-31T12:34:56Z",
optionally with fractional seconds or a +HH:MM offset) into a Posix instant.
Local code instead of a date library: the input space is one machine-emitted
format, not arbitrary user text.
-}
posixFromRFC3339 : String -> Maybe Time.Posix
posixFromRFC3339 raw =
    case String.split "T" raw of
        [ datePart, timePart ] ->
            Maybe.map2
                (\dayMillis timeOfDay -> Time.millisToPosix (dayMillis + timeOfDay))
                (dateMillis datePart)
                (timeOfDayMillis timePart)

        _ ->
            Nothing


dateMillis : String -> Maybe Int
dateMillis datePart =
    case List.map String.toInt (String.split "-" datePart) of
        [ Just year, Just month, Just day ] ->
            Just (daysFromCivil year month day * 86400000)

        _ ->
            Nothing


timeOfDayMillis : String -> Maybe Int
timeOfDayMillis timePart =
    let
        ( clockText, offsetMinutes ) =
            splitUTCOffset timePart
    in
    case List.map String.toInt (String.split ":" clockText) of
        [ Just hour, Just minute, Just second ] ->
            Just ((((hour * 60 + minute) - offsetMinutes) * 60 + second) * 1000)

        _ ->
            Nothing


{-| Splits "12:34:56.789+02:00" into the clock digits (fraction dropped) and
the offset in minutes. "Z" and a missing suffix both mean UTC.
-}
splitUTCOffset : String -> ( String, Int )
splitUTCOffset timePart =
    let
        withoutZ =
            if String.endsWith "Z" timePart then
                String.dropRight 1 timePart

            else
                timePart

        ( clock, offset ) =
            case String.indexes "+" withoutZ of
                position :: _ ->
                    ( String.left position withoutZ, offsetToMinutes (String.dropLeft (position + 1) withoutZ) )

                [] ->
                    case String.indexes "-" withoutZ of
                        position :: _ ->
                            ( String.left position withoutZ, -(offsetToMinutes (String.dropLeft (position + 1) withoutZ)) )

                        [] ->
                            ( withoutZ, 0 )
    in
    ( String.left 8 clock, offset )


offsetToMinutes : String -> Int
offsetToMinutes offsetText =
    case List.map String.toInt (String.split ":" offsetText) of
        [ Just hours, Just minutes ] ->
            hours * 60 + minutes

        _ ->
            0


{-| Days since 1970-01-01 for a proleptic-Gregorian civil date (Howard
Hinnant's algorithm). Valid for every date the API can emit.
-}
daysFromCivil : Int -> Int -> Int -> Int
daysFromCivil year month day =
    let
        shiftedYear =
            if month <= 2 then
                year - 1

            else
                year

        era =
            shiftedYear // 400

        yearOfEra =
            shiftedYear - era * 400

        monthIndex =
            modBy 12 (month + 9)

        dayOfYear =
            (153 * monthIndex + 2) // 5 + day - 1

        dayOfEra =
            yearOfEra * 365 + yearOfEra // 4 - yearOfEra // 100 + dayOfYear
    in
    era * 146097 + dayOfEra - 719468
