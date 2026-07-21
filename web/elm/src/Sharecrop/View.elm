module Sharecrop.View exposing (..)

import Browser
import Html exposing (Html, a, button, div, form, h3, label, main_, nav, option, p, select, span, table, tbody, td, text, th, thead, tr)
import Html.Keyed
import Html.Attributes exposing (attribute, checked, disabled, href, placeholder, selected, type_, value)
import Html.Events exposing (onCheck, onClick, onInput, onSubmit)
import Dict
import Json.Decode as Decode
import Json.Encode as Encode
import Sharecrop.Generated.Admin as Admin
import Sharecrop.Generated.Agent as Agent
import Sharecrop.Generated.Collectible as Collectible
import Sharecrop.Generated.Ledger as Ledger
import Sharecrop.Generated.Moderation as Moderation
import Sharecrop.Generated.Notification as Notification
import Sharecrop.Generated.Organization as Organization
import Sharecrop.Generated.Privacy as Privacy
import Sharecrop.Generated.Submission as Submission
import Sharecrop.Generated.Task as Task
import Sharecrop.Generated.TaskSeries as TaskSeries
import Sharecrop.Generated.Team as Team
import Sharecrop.ResponseSchema as ResponseSchema
import Sharecrop.Sprites as Sprites
import Sharecrop.Labels exposing (allScopes, assigneeScopeLabel, assigneeScopeTag, availabilityKindLabel, collectibleKindLabel, collectibleKindTag, collectiblePolicyLabel, collectiblePolicyTag, collectibleStateLabel, credentialStateLabel, kindLabel, participationPolicyLabel, participationPolicyTag, participationUsesReservation, reservationStateLabel, rewardLabel, scopeLabel, scopeTag, submissionStateLabel, taskStateGuidance, taskStateLabel, viewerActionLabel)
import Sharecrop.Types exposing (..)
import Sharecrop.Ui as Ui exposing (testId)


view : Model -> Browser.Document Msg
view model =
    { title = "Sharecrop"
    , body =
        [ main_ [ Html.Attributes.class "min-h-screen bg-slate-50 p-4 text-slate-950 sm:p-8" ]
            [ div [ Html.Attributes.class "mx-auto max-w-3xl space-y-6" ]
                [ sessionView model ]
            ]
        ]
    }


{-| The logged-out screen isn't part of the `Page` union (it's the one
unrouted entry point), so "Sharecrop" is its own meaningful `<h1>`. Once
logged in, every route renders its own page-specific `<h1>` instead (see
`pageView`) rather than repeating this static wordmark on every page.
-}
sessionView : Model -> Html Msg
sessionView model =
    case model.session of
        LoggedOut ->
            div [ Html.Attributes.class "space-y-6" ]
                [ Ui.pageTitle "Sharecrop"
                , authView model
                ]

        LoggedIn state ->
            loggedInView model state


authView : Model -> Html Msg
authView model =
    div
        [ Html.Attributes.class "space-y-4 rounded-lg border border-slate-200 bg-white p-6 shadow-sm" ]
        -- The login and password-reset controls are separate <form>s so that
        -- pressing Enter in a reset field submits the reset request rather than
        -- attempting a login. Each reset field is bound to the reset action
        -- that makes sense for it.
        (if model.shauth then
            [ p [ Html.Attributes.class "text-slate-600" ] [ text "Continue through Shauth to use your organization identity." ]
            , Ui.secondaryLink [ testId "shauth-login" ] "/api/auth/shauth" "Continue with Shauth"
            , maybeError model.authError "auth-error"
            ]

         else
            [ form [ Html.Attributes.class "space-y-4", onSubmit LoginClicked ]
            [ p [ Html.Attributes.class "text-slate-600" ] [ text "Sign in or create an account to view your credit ledger and set up agents." ]
            , Ui.textInput [ type_ "email", placeholder "Email", value model.email, onInput EmailChanged, testId "email" ]
            , Ui.textInput [ type_ "password", placeholder "Password", value model.password, onInput PasswordChanged, testId "password" ]
            , div [ Html.Attributes.class "flex gap-3" ]
                ([ Ui.primaryButton [ type_ "submit", testId "login" ] "Log in"
                 , Ui.secondaryButton [ type_ "button", onClick RegisterClicked, testId "register" ] "Register"
                 , Ui.secondaryLink [ testId "shauth-login" ] "/api/auth/shauth" "Continue with Shauth"
                 ]
                    -- Guest sessions only work against the demo backend; the
                    -- real API rejects the guest subject on every data route, so
                    -- offering the button there is a dead end (empty lists and
                    -- failing actions with no explanation).
                    ++ (if model.demo then
                            [ Ui.secondaryButton [ type_ "button", onClick GuestClicked, testId "guest-login" ] "Continue as guest" ]

                        else
                            []
                       )
                )
            ]
        , div [ Html.Attributes.class "space-y-2 border-t border-slate-100 pt-4" ]
            [ Ui.label_ "Password reset"
            , form [ Html.Attributes.class "space-y-2", onSubmit RequestPasswordResetClicked ]
                [ Ui.textInput [ type_ "email", placeholder "Account email", value model.resetEmail, onInput PasswordResetEmailChanged, testId "reset-email" ]
                , div [ Html.Attributes.class "flex flex-wrap gap-2" ]
                    [ Ui.primaryButton [ type_ "submit", testId "request-password-reset" ] "Create reset token" ]
                ]
            , form [ Html.Attributes.class "space-y-2", onSubmit ConfirmPasswordResetClicked ]
                [ Ui.textInput [ type_ "text", placeholder "Reset token", value model.resetToken, onInput PasswordResetTokenChanged, testId "reset-token" ]
                , Ui.textInput [ type_ "password", placeholder "New password", value model.resetPassword, onInput PasswordResetPasswordChanged, testId "reset-password" ]
                , Ui.primaryButton [ type_ "submit", testId "confirm-password-reset" ] "Reset password"
                ]
            ]
        , maybeError model.authError "auth-error"
        , case model.authNotice of
            Just notice ->
                Ui.successText "auth-notice" notice

            Nothing ->
                text ""
            ]
        )


loggedInView : Model -> LoggedInModel -> Html Msg
loggedInView model state =
    div [ Html.Attributes.class "space-y-6" ]
        [ navBar model.demo state.page state.subjectId state.username state.isAdmin state.openNavMenu
        , maybeError model.authError "logout-error"

        -- Keyed by route so navigating away and back always rebuilds the page
        -- fresh: without a key, Elm's virtual DOM can match two structurally
        -- similar pages (e.g. both starting with a `<details>` disclosure at
        -- the same position) as "the same node" and carry over native,
        -- Elm-invisible DOM state like an expanded/collapsed <details>.
        , Html.Keyed.node "div" [] [ ( pageToPath state.page, pageView model.origin state ) ]
        ]


{-| The primary nav: just Overview and Tasks stay flat now — Tasks is the
one hub for everything task-related (My tasks, Discover public tasks, My
submissions, Series, and a "+ New task" shortcut all live there, see
`tasksView`), so it no longer needs New task/Discovery/Work as siblings.
Everything else collapses into `Ui.navMenu` dropdowns (Manage, Account) so
the whole bar reads as one row instead of a wall of buttons. Every existing
`nav-*` link keeps its exact `data-testid`, so moving a link doesn't change
how a test finds it — only whether a surrounding menu needs opening first.
-}
navBar : Bool -> Page -> String -> String -> Bool -> Maybe String -> Html Msg
navBar demo current subjectId username isAdmin openNavMenu =
    let
        isCurrent target =
            pageToPath current == pageToPath target

        manageMenuActive =
            isCurrent FundingPage
                || isCurrent CollectiblesPage
                || isCurrent AgentsPage
                || isCurrent OrganizationsPage

        accountMenuActive =
            isCurrent (UserDetailPage subjectId)
                || isCurrent AdminPage
                || isCurrent InboxPage
                || isCurrent (UserSubmissionsPage subjectId)

        isMenuOpen identifier =
            openNavMenu == Just identifier
    in
    nav
        [ Html.Attributes.attribute "aria-label" "Primary", Html.Attributes.class "flex flex-wrap items-center gap-2" ]
        [ navLink current OverviewPage "overview" "Overview"
        , navLink current TasksPage "tasks" "Tasks"
        , Ui.navMenu "nav-manage-menu"
            True
            manageMenuActive
            (isMenuOpen "nav-manage-menu")
            (ToggleNavMenu "nav-manage-menu")
            "Manage"
            Nothing
            [ navLink current FundingPage "funding" "Funding"
            , navLink current CollectiblesPage "collectibles" "Collectibles"
            , navLink current AgentsPage "agents" "Agents"
            , navLink current OrganizationsPage "organizations" "Organizations"
            ]
        , Ui.navMenu "nav-account-menu"
            True
            accountMenuActive
            (isMenuOpen "nav-account-menu")
            (ToggleNavMenu "nav-account-menu")
            (if String.isEmpty username then
                "Account"

             else
                username
            )
            (if String.isEmpty username then
                Nothing

             else
                Just username
            )
            (navLink current (UserDetailPage subjectId) "profile" "Profile"
                :: navLink current InboxPage "inbox" "Inbox"
                :: (if isAdmin then
                        [ navLink current AdminPage "admin" "Admin" ]

                    else
                        []
                   )
                ++ [ Ui.secondaryButton [ type_ "button", onClick LogoutClicked, testId "logout", attribute "data-shauth-sign-out" "" ] "Log out" ]
                ++ (if demo then
                        [ Ui.secondaryButton [ type_ "button", onClick ResetDemoClicked, testId "reset-demo" ] "Reset demo" ]

                    else
                        []
                   )
            )
        ]


navLink : Page -> Page -> String -> String -> Html Msg
navLink current target identifier labelText =
    let
        styleClass =
            if pageToPath current == pageToPath target then
                Ui.primaryButtonClass

            else
                Ui.secondaryButtonClass
    in
    a
        [ href ("#" ++ pageToPath target)
        , Html.Attributes.class styleClass
        , testId ("nav-" ++ identifier)
        ]
        [ text labelText ]


adminView : LoggedInModel -> Html Msg
adminView state =
    Ui.card
        [ Ui.disclosure "admin-section-operations"
            True
            "Operations"
            [ case state.operations of
                Just operations ->
                    Html.dl [ Html.Attributes.class "grid gap-2 text-sm sm:grid-cols-2", testId "admin-operations" ]
                        [ operationFact "Status" operations.status
                        , operationFact "Account token delivery" operations.accountTokenDelivery
                        , operationFact "MCP storage" operations.mcpStorage
                        , operationFact "Rate limit storage" operations.rateLimitStorage
                        , operationFact "Secure cookies" operations.secureCookies
                        , operationFact "Active MCP sessions" (String.fromInt operations.activeMCPSessions)
                        , operationFact "IP rate buckets" (String.fromInt operations.activeIPRateBuckets)
                        , operationFact "Subject rate buckets" (String.fromInt operations.activeSubjectRateBuckets)
                        ]

                Nothing ->
                    p [ Html.Attributes.class "text-sm text-slate-500", testId "admin-operations-empty" ] [ text "Operations status is not loaded." ]
            ]
        , Ui.disclosure "admin-section-audit"
            False
            "Audit events"
            [ div [ Html.Attributes.class "grid gap-3 sm:grid-cols-3" ]
                [ Ui.fieldLabel "Action"
                    [ Ui.textInput [ placeholder "submission_accepted", value state.auditActionFilter, onInput AuditActionFilterChanged, testId "admin-audit-action" ] ]
                , Ui.fieldLabel "Subject kind"
                    [ Ui.textInput [ placeholder "submission", value state.auditSubjectKindFilter, onInput AuditSubjectKindFilterChanged, testId "admin-audit-subject-kind" ] ]
                , Ui.fieldLabel "Subject ID"
                    [ Ui.textInput [ placeholder "ID", value state.auditSubjectIDFilter, onInput AuditSubjectIDFilterChanged, testId "admin-audit-subject-id" ] ]
                ]
            , div [ Html.Attributes.class "flex flex-wrap gap-2" ]
                [ Ui.secondaryButton [ type_ "button", onClick SearchAuditEventsClicked, testId "admin-audit-search" ] "Search"
                ]
            , paginationControls "admin-audit-page" PreviousAuditEventsPageClicked NextAuditEventsPageClicked state.auditEventsOffset (List.length state.auditEvents)
            , if List.isEmpty state.auditEvents then
                p [ Html.Attributes.class "text-sm text-slate-500", testId "admin-audit-empty" ] [ text "No audit events." ]

              else
                div [ Html.Attributes.class "divide-y divide-slate-100", testId "admin-audit-events" ]
                    (List.map auditEventRow state.auditEvents)
            ]
        , Ui.disclosure "admin-section-platform-admins"
            False
            "Platform admins"
            [ Ui.fieldLabel "Grant user"
                [ userPicker "admin-platform-user" state.adminSelectedUserId state.userDirectoryQuery AdminSelectedUserChanged "Choose user" state.userDirectory state.userDirectoryOffset ]
            , div [ Html.Attributes.class "flex flex-wrap gap-2" ]
                [ Ui.secondaryButton [ type_ "button", onClick GrantPlatformAdminClicked, disabled (String.trim state.adminSelectedUserId == ""), testId "admin-grant-platform-admin" ] "Grant"
                ]
            , paginationControls "admin-platform-admins-page" PreviousPlatformAdminsPageClicked NextPlatformAdminsPageClicked state.platformAdminsOffset (List.length state.platformAdmins)
            , if List.isEmpty state.platformAdmins then
                p [ Html.Attributes.class "text-sm text-slate-500", testId "admin-platform-admins-empty" ] [ text "No platform admins." ]

              else
                div [ Html.Attributes.class "divide-y divide-slate-100", testId "admin-platform-admins" ]
                    (List.map platformAdminRow state.platformAdmins)
            ]
        , Ui.disclosure "admin-section-privacy"
            False
            "Privacy requests"
            [ Ui.fieldLabel "Resolution note"
                [ Ui.textInput [ placeholder "Export generated or fields redacted", value state.adminPrivacyResolutionNote, onInput AdminPrivacyResolutionNoteChanged, testId "admin-privacy-note" ] ]
            , div [ Html.Attributes.class "flex flex-wrap items-center gap-2" ]
                [ Ui.secondaryButton [ type_ "button", onClick RunPrivacyRetentionClicked, testId "admin-run-privacy-retention" ] "Run retention"
                , case state.adminRetentionRedactedFieldCount of
                    Just count ->
                        span [ Html.Attributes.class "text-xs text-slate-600", testId "admin-retention-count" ] [ text ("Redacted fields: " ++ String.fromInt count) ]

                    Nothing ->
                        text ""
                ]
            , paginationControls "admin-privacy-page" PreviousAdminPrivacyPageClicked NextAdminPrivacyPageClicked state.adminPrivacyOffset (List.length state.adminPrivacyRequests)
            , if List.isEmpty state.adminPrivacyRequests then
                p [ Html.Attributes.class "text-sm text-slate-500", testId "admin-privacy-empty" ] [ text "No privacy requests." ]

              else
                div [ Html.Attributes.class "divide-y divide-slate-100", testId "admin-privacy-requests" ]
                    (List.map (adminPrivacyRequestRow state.adminPrivacyResolutionNote) state.adminPrivacyRequests)
            ]
        , Ui.disclosure "admin-section-moderation"
            False
            "Moderation reports"
            [ div [ Html.Attributes.class "grid gap-3 sm:grid-cols-2" ]
                [ Ui.fieldLabel "State"
                    [ select [ Html.Attributes.class Ui.fieldClass, value state.adminModerationStateFilter, onInput AdminModerationStateFilterChanged, testId "admin-moderation-state" ]
                        [ blankOption "All states"
                        , stringOption state.adminModerationStateFilter ( "open", "Open" )
                        , stringOption state.adminModerationStateFilter ( "resolved", "Resolved" )
                        , stringOption state.adminModerationStateFilter ( "dismissed", "Dismissed" )
                        ]
                    ]
                , Ui.fieldLabel "Triage note"
                    [ Ui.textInput [ placeholder "Decision note", value state.adminModerationResolutionNote, onInput AdminModerationResolutionNoteChanged, testId "admin-moderation-note" ] ]
                ]
            , paginationControls "admin-moderation-page" PreviousAdminModerationPageClicked NextAdminModerationPageClicked state.adminModerationOffset (List.length state.adminModerationReports)
            , if List.isEmpty state.adminModerationReports then
                p [ Html.Attributes.class "text-sm text-slate-500", testId "admin-moderation-empty" ] [ text "No moderation reports." ]

              else
                div [ Html.Attributes.class "divide-y divide-slate-100", testId "admin-moderation-reports" ]
                    (List.map (adminModerationReportRow state.adminModerationResolutionNote) state.adminModerationReports)
            ]
        , maybeNote state.adminMessage "admin-message"
        ]


operationFact : String -> String -> Html Msg
operationFact labelText valueText =
    div [ Html.Attributes.class "rounded border border-slate-200 p-2" ]
        [ Html.dt [ Html.Attributes.class "text-xs font-semibold text-slate-500" ] [ text labelText ]
        , Html.dd [ Html.Attributes.class "break-words text-slate-900" ] [ text valueText ]
        ]


auditEventRow : { event | action : String, subjectKind : String, subjectID : String, actorUserID : String, createdAt : String, metadataJSON : String } -> Html Msg
auditEventRow event =
    div [ Html.Attributes.class "space-y-1 py-2 text-sm", testId "admin-audit-event" ]
        [ p [ Html.Attributes.class "font-medium" ] [ text (event.action ++ " on " ++ event.subjectKind) ]
        , p [ Html.Attributes.class "text-xs text-slate-500 break-words" ] [ text ("Subject " ++ event.subjectID ++ " · actor " ++ event.actorUserID ++ " · " ++ event.createdAt) ]
        , if event.metadataJSON == "{}" then
            text ""

          else
            Ui.codeBlock [ testId "admin-audit-metadata" ] event.metadataJSON
        ]


platformAdminRow : Admin.PlatformAdminResponse -> Html Msg
platformAdminRow admin =
    div [ Html.Attributes.class "flex flex-wrap items-center justify-between gap-3 py-3 text-sm", testId "admin-platform-admin" ]
        [ div [ Html.Attributes.class "space-y-1" ]
            [ a [ href ("#/users/" ++ admin.userID), Html.Attributes.class "block font-medium text-slate-900 break-all underline" ] [ text admin.userID ]
            , p [ Html.Attributes.class "text-xs text-slate-500" ] [ text (admin.source ++ " · " ++ admin.createdAt) ]
            ]
        , if admin.source == "bootstrap" then
            Ui.badge "bootstrap"

          else
            Ui.secondaryButton [ type_ "button", onClick (RevokePlatformAdminClicked admin.userID), testId "admin-revoke-platform-admin" ] "Revoke"
        ]


adminPrivacyRequestRow : String -> Privacy.PrivacyRequestResponse -> Html Msg
adminPrivacyRequestRow resolutionNote request =
    div [ Html.Attributes.class "space-y-2 py-3 text-sm", testId "admin-privacy-request" ]
        [ div [ Html.Attributes.class "flex flex-wrap items-center gap-2" ]
            [ privacyRequestStatusBadge request.status
            , span [ Html.Attributes.class "font-medium text-slate-900" ] [ text request.kind ]
            , span [ Html.Attributes.class "break-all text-xs text-slate-500" ] [ text request.id ]
            ]
        , Html.dl [ Html.Attributes.class "grid gap-2 sm:grid-cols-2" ]
            [ operationFact "Requested by" request.requestedBy
            , operationFact "Created" request.createdAt
            , operationFact "Resolved" (emptyLabel request.resolvedAt)
            , operationFact "Redacted fields" (String.fromInt request.redactedFieldCount)
            ]
        , if request.resolutionNote == "" then
            text ""

          else
            p [ Html.Attributes.class "text-xs text-slate-600", testId "admin-privacy-resolution-note" ] [ text request.resolutionNote ]
        , if request.exportJSON == "" then
            text ""

          else
            Ui.codeBlock [ testId "admin-privacy-export" ] request.exportJSON
        , if request.status == "queued" then
            Ui.secondaryButton [ type_ "button", onClick (ResolveAdminPrivacyRequestClicked request.id), disabled (String.trim resolutionNote == ""), testId "admin-resolve-privacy" ] "Resolve"

          else
            text ""
        ]


adminModerationReportRow : String -> Moderation.ModerationReportResponse -> Html Msg
adminModerationReportRow resolutionNote report =
    div [ Html.Attributes.class "space-y-2 py-3 text-sm", testId "admin-moderation-report" ]
        [ div [ Html.Attributes.class "flex flex-wrap items-center gap-2" ]
            [ moderationReportStateBadge report.state
            , Ui.badge report.reason
            , span [ Html.Attributes.class "font-medium text-slate-900" ] [ text report.subjectKind ]
            , if report.subjectHref == "" then
                span [ Html.Attributes.class "break-all text-xs text-slate-500" ] [ text report.subjectID ]

              else
                a [ href report.subjectHref, Html.Attributes.class "break-all text-xs font-medium text-emerald-700", testId "admin-moderation-subject-link" ] [ text report.subjectID ]
            ]
        , Html.dl [ Html.Attributes.class "grid gap-2 sm:grid-cols-2" ]
            [ operationFact "Reporter" report.reporterUserID
            , operationFact "Created" report.createdAt
            , operationFact "Updated by" (emptyLabel report.updatedBy)
            , operationFact "Updated" (emptyLabel report.updatedAt)
            ]
        , if report.resolutionNote == "" then
            text ""

          else
            p [ Html.Attributes.class "text-xs text-slate-600", testId "admin-moderation-resolution-note" ] [ text report.resolutionNote ]
        , if report.details == "" then
            text ""

          else
            p [ Html.Attributes.class "text-sm text-slate-700 break-words", testId "admin-moderation-details" ] [ text report.details ]
        , div [ Html.Attributes.class "flex flex-wrap gap-2" ]
            [ Ui.secondaryButton [ type_ "button", onClick (TriageModerationReportClicked report.id "open"), testId "admin-moderation-open" ] "Reopen"
            , Ui.secondaryButton [ type_ "button", onClick (TriageModerationReportClicked report.id "resolved"), disabled (String.trim resolutionNote == ""), testId "admin-moderation-resolve" ] "Resolve"
            , Ui.secondaryButton [ type_ "button", onClick (TriageModerationReportClicked report.id "dismissed"), disabled (String.trim resolutionNote == ""), testId "admin-moderation-dismiss" ] "Dismiss"
            ]
        ]


emptyLabel : String -> String
emptyLabel value =
    if String.trim value == "" then
        "none"

    else
        value


inboxView : LoggedInModel -> Html Msg
inboxView state =
    Ui.card
        [ if List.isEmpty state.notifications then
            p [ Html.Attributes.class "text-sm text-slate-500", testId "inbox-empty" ] [ text "No notifications." ]

          else
            div [ Html.Attributes.class "divide-y divide-slate-100", testId "inbox-list" ]
                (List.map notificationRow state.notifications)
        , paginationControls "inbox-page" PreviousNotificationsPageClicked NextNotificationsPageClicked state.notificationsOffset (List.length state.notifications)
        , maybeNote state.inboxMessage "inbox-message"
        ]


notificationRow : Notification.NotificationResponse -> Html Msg
notificationRow notification =
    div [ Html.Attributes.class "space-y-2 py-3 text-sm", testId "notification-row" ]
        [ div [ Html.Attributes.class "flex flex-wrap items-center justify-between gap-2" ]
            [ p [ Html.Attributes.class "font-medium text-slate-900" ] [ text (notification.kind ++ " on " ++ notification.subjectKind) ]
            , span [ Html.Attributes.class (notificationStateClass notification.state), testId "notification-state" ] [ text notification.state ]
            ]
        , p [ Html.Attributes.class "break-words text-xs text-slate-500" ]
            [ text ("Subject " ++ notification.subjectID ++ " · actor ")
            , a [ href ("#/users/" ++ notification.actorUserID), Html.Attributes.class "underline" ] [ text notification.actorUserID ]
            , text (" · " ++ notification.createdAt)
            ]
        , notificationTaskLink notification
        , if notification.metadataJSON == "{}" then
            text ""

          else
            Ui.codeBlock [ testId "notification-metadata" ] notification.metadataJSON
        , if notification.state == "unread" then
            Ui.secondaryButton [ type_ "button", onClick (MarkNotificationReadClicked notification.id), testId "notification-mark-read" ] "Mark read"

          else
            text ""
        ]


notificationTaskLink : Notification.NotificationResponse -> Html Msg
notificationTaskLink notification =
    case Decode.decodeString (Decode.field "task_id" Decode.string) notification.metadataJSON of
        Ok taskId ->
            a [ href ("#/tasks/" ++ taskId), Html.Attributes.class Ui.secondaryButtonClass, testId "notification-task-link" ] [ text "Open task" ]

        Err _ ->
            text ""


notificationStateClass : String -> String
notificationStateClass state =
    if state == "unread" then
        "rounded border border-amber-300 bg-amber-50 px-2 py-1 text-xs font-semibold text-amber-900"

    else
        "rounded border border-slate-200 bg-slate-50 px-2 py-1 text-xs font-semibold text-slate-600"


{-| Each route gets its own `<h1>` (WCAG 1.3.1/2.4.6) so the page's identity
doesn't depend solely on the persistent "Sharecrop" wordmark, which no longer
renders once logged in (see `sessionView`). Titles are chosen to read
distinctly from whatever `Ui.sectionTitle`/`<h2>` a page renders internally
(e.g. "New task" here vs. "Create a task" inside `createTaskView`) so the
heading hierarchy doesn't repeat the same text at two levels.
-}
pageView : String -> LoggedInModel -> Html Msg
pageView origin state =
    let
        ( title, content ) =
            case state.page of
                OverviewPage ->
                    ( "Overview", overviewView state )

                TasksPage ->
                    ( "Tasks", tasksView origin state )

                CreateTaskPage ->
                    ( "New task", createTaskView state )

                TaskDetailPage _ ->
                    ( "Task", taskDetailPageView origin state )

                FundingPage ->
                    ( "Funding", fundingView state )

                AgentsPage ->
                    ( "Agents", agentsView origin state )

                CollectiblesPage ->
                    ( "Collectibles", collectiblesView state )

                OrganizationsPage ->
                    ( "Organizations", organizationsView state )

                OrganizationDetailPage _ ->
                    ( "Organization", organizationDetailView state )

                UserDetailPage userId ->
                    ( "Profile", userDetailView origin userId state )

                UserWorkPage userId ->
                    ( "Work", userTaskListView "Currently working on" "user-work" userId state.userWork )

                UserSubmissionsPage userId ->
                    ( "Submissions", userSubmissionsView userId state )

                CollectibleDetailPage collectibleId ->
                    ( "Collectible", collectibleDetailView collectibleId state )

                SeriesDetailPage seriesId ->
                    ( "Series", seriesDetailView seriesId state )

                TeamDetailPage teamId ->
                    ( "Team", teamDetailView teamId state )

                AdminPage ->
                    ( "Admin", adminView state )

                InboxPage ->
                    ( "Inbox", inboxView state )

                NotFoundPage ->
                    ( "Page not found"
                    , Ui.card
                        [ p [ Html.Attributes.class "text-sm text-slate-600" ] [ text "That page does not exist." ]
                        , a [ href "#/", Html.Attributes.class Ui.secondaryButtonClass, testId "not-found-home" ] [ text "Go to overview" ]
                        ]
                    )
    in
    div [ Html.Attributes.class "space-y-4" ]
        [ Ui.pageTitle title
        , content
        ]


teamDetailView : String -> LoggedInModel -> Html Msg
teamDetailView teamId state =
    Ui.card
        [ case state.teamDetail of
            Just detail ->
                div [ Html.Attributes.class "space-y-2", testId "team-detail" ]
                    [ p [ Html.Attributes.class "text-2xl font-semibold", testId "team-detail-name" ] [ text detail.team.name ]
                    , Ui.label_ ("Team " ++ detail.team.id)
                    , p [ Html.Attributes.class "text-sm" ] [ text ("Owner kind: " ++ detail.team.ownerKind) ]
                    , Ui.sectionTitle "Members"
                    , if List.isEmpty detail.members then
                        p [ Html.Attributes.class "text-sm text-slate-500", testId "team-members-empty" ] [ text "No members yet." ]

                      else
                        div [ Html.Attributes.class "divide-y divide-slate-100", testId "team-members" ]
                            (List.map (\memberId -> a [ href ("#/users/" ++ memberId), Html.Attributes.class "block py-2 text-sm underline", testId "team-member-row" ] [ text memberId ]) detail.members)
                    -- Shown for the user-owner AND for org-owned teams: the
                    -- backend allows anyone with manage-teams permission on
                    -- the owning org to add members, and the client does not
                    -- load the viewer's org roles here, so it lets the server
                    -- decide (an unauthorized attempt gets a clear rejection
                    -- in team-member-message). Hiding it entirely made
                    -- org-owned teams impossible to populate in the browser.
                    , if detail.team.ownerKind == "organization" || (detail.team.ownerKind == "user" && detail.team.ownerUserID == state.subjectId) then
                        form [ Html.Attributes.class "flex flex-wrap items-end gap-2", onSubmit (AddTeamMemberClicked detail.team.id) ]
                            [ Ui.fieldLabel "Add member by email"
                                [ Ui.textInput [ type_ "email", placeholder "person@example.com", value state.teamMemberEmail, onInput TeamMemberEmailChanged, testId "team-member-email" ] ]
                            , Ui.primaryButton [ type_ "submit", testId "add-team-member" ] "Add member"
                            , maybeNote state.teamMemberMessage "team-member-message"
                            ]

                      else
                        text ""
                    , teamWorkDashboard detail.team.id state
                    , Ui.sectionTitle "Collectibles"
                    , collectiblesHoldingsList "team-collectibles" state.teamCollectibles
                    , maybeNote state.teamCollectiblesMessage "team-collectibles-message"
                    ]

            Nothing ->
                case state.teamDetailError of
                    Just message ->
                        p [ Html.Attributes.class "text-sm text-slate-700", testId "team-detail-missing" ] [ text ("Could not load this team: " ++ message) ]

                    Nothing ->
                        p [ Html.Attributes.class "text-sm text-slate-500", testId "team-detail-missing" ] [ text ("Loading team " ++ teamId ++ "…") ]
        ]


teamWorkDashboard : String -> LoggedInModel -> Html Msg
teamWorkDashboard teamId state =
    let
        filteredTasks =
            state.teamWork
                |> filterTeamWork teamId state.teamWorkFilter

        reviewTasks =
            List.filter (\item -> item.reviewerAction /= "none") filteredTasks

        readyForTeam =
            List.filter teamCanActOnTask filteredTasks

        assignedToTeam =
            List.filter (\item -> item.activeAssigneeID == teamId) filteredTasks
    in
    div [ Html.Attributes.class "space-y-4", testId "team-work-dashboard" ]
        [ Ui.disclosure "team-work-filters" False "Filters" <|
            [ Ui.fieldLabel "Search team work"
                [ Ui.textInput [ type_ "search", placeholder "Task title or ID", value state.teamWorkQuery, onInput TeamWorkQueryChanged, testId "team-work-query" ] ]
            , taskTypeFilterSelect "team-work-type" state.teamWorkTypeFilter TeamWorkTypeFilterChanged
            , taskSortSelect "team-work-sort" state.teamWorkSort TeamWorkSortChanged
            , div [ Html.Attributes.class "flex flex-wrap gap-2" ]
                [ Ui.secondaryButton [ type_ "button", onClick SearchTeamWorkClicked, testId "team-work-search" ] "Search"
                ]
            , div [ Html.Attributes.class "flex flex-wrap gap-2", testId "team-work-filter" ]
                (List.map (teamWorkFilterButton state.teamWorkFilter) teamWorkFilterOptions)
            , queueSavedViews
                { nameValue = state.teamWorkSavedViewName
                , nameChanged = TeamWorkSavedViewNameChanged
                , saveClicked = SaveTeamWorkViewClicked
                , applyClicked = ApplyTeamWorkViewClicked
                , views = state.teamWorkSavedViews
                , prefix = "team-work"
                }
            ]
        , paginationControls "team-work-page" PreviousTeamWorkPageClicked NextTeamWorkPageClicked state.teamWorkOffset (List.length state.teamWork)
        , teamWorkSection state.subjectId "Review queue" "team-review-queue" "No submissions waiting for team review." reviewTasks
        , teamWorkSection state.subjectId "Ready for team" "team-ready-work" "No team-visible tasks are ready for action." readyForTeam
        , teamWorkSection state.subjectId "Assigned to team" "team-assigned-work" "No tasks are currently assigned to this team." assignedToTeam
        , maybeNote state.teamWorkMessage "team-work-message"
        ]


teamWorkFilterOptions : List ( String, String )
teamWorkFilterOptions =
    [ ( "", "All" )
    , ( "review", "Review" )
    , ( "ready", "Ready" )
    , ( "assigned", "Assigned" )
    ]


teamWorkFilterButton : String -> ( String, String ) -> Html Msg
teamWorkFilterButton selected ( tag, labelText ) =
    Ui.chooserButton (selected == tag)
        (TeamWorkFilterChanged tag)
        ("team-work-filter-"
            ++ (if tag == "" then
                    "all"

                else
                    tag
               )
        )
        labelText


queueSavedViews config =
    div [ Html.Attributes.class "space-y-2 rounded-md border border-slate-200 bg-white p-3", testId (config.prefix ++ "-saved-views") ]
        [ form [ Html.Attributes.class "flex flex-wrap items-end gap-2", onSubmit config.saveClicked ]
            [ Ui.fieldLabel "Saved view"
                [ Ui.textInput [ type_ "text", placeholder "View name", value config.nameValue, onInput config.nameChanged, testId (config.prefix ++ "-saved-view-name") ] ]
            , Ui.secondaryButton [ type_ "submit", testId (config.prefix ++ "-save-view") ] "Save"
            ]
        , if List.isEmpty config.views then
            p [ Html.Attributes.class "text-xs text-slate-500", testId (config.prefix ++ "-saved-views-empty") ] [ text "No saved views." ]

          else
            div [ Html.Attributes.class "flex flex-wrap gap-2", testId (config.prefix ++ "-saved-view-list") ]
                (List.map
                    (\savedView ->
                        Ui.secondaryButton [ type_ "button", onClick (config.applyClicked savedView.name), testId (config.prefix ++ "-saved-view") ] (queueViewLabel savedView)
                    )
                    config.views
                )
        ]


queueViewLabel : QueueView -> String
queueViewLabel savedView =
    String.join " · "
        (savedView.name
            :: List.filter
                (\part -> String.trim part /= "")
                [ queueViewStateLabel savedView.stateFilter
                , queueViewTypeLabel savedView.typeFilter
                , sortLabel savedView.sort
                ]
        )


queueViewStateLabel : String -> String
queueViewStateLabel value =
    case value of
        "review" ->
            "Review"

        "ready" ->
            "Ready"

        "assigned" ->
            "Assigned"

        "draft" ->
            "Draft"

        "open" ->
            "Open"

        "closed" ->
            "Closed"

        "cancelled" ->
            "Cancelled"

        _ ->
            ""


sortLabel : String -> String
sortLabel value =
    case value of
        "newest" ->
            "Newest"

        "oldest" ->
            "Oldest"

        "title_asc" ->
            "Title A-Z"

        "title_desc" ->
            "Title Z-A"

        "reward_desc" ->
            "Reward high"

        "reward_asc" ->
            "Reward low"

        _ ->
            ""


queueViewTypeLabel : String -> String
queueViewTypeLabel value =
    if String.trim value == "" then
        ""

    else
        taskTypeLabel value


filterTeamWork : String -> String -> List Task.TaskListItemResponse -> List Task.TaskListItemResponse
filterTeamWork teamId tag tasks =
    case tag of
        "review" ->
            List.filter (\item -> item.reviewerAction /= "none") tasks

        "ready" ->
            List.filter teamCanActOnTask tasks

        "assigned" ->
            List.filter (\item -> item.activeAssigneeID == teamId) tasks

        "" ->
            tasks

        _ ->
            []


teamCanActOnTask : Task.TaskListItemResponse -> Bool
teamCanActOnTask item =
    case item.viewerAction of
        Task.TaskViewerActionSubmit ->
            True

        Task.TaskViewerActionReserve ->
            True

        Task.TaskViewerActionRequestApproval ->
            True

        _ ->
            False


teamWorkSection : String -> String -> String -> String -> List Task.TaskListItemResponse -> Html Msg
teamWorkSection subjectId title identifier emptyMessage tasks =
    div [ Html.Attributes.class "space-y-2", testId identifier ]
        [ Ui.sectionTitleWithCount title (List.length tasks) (identifier ++ "-heading")
        , if List.isEmpty tasks then
            p [ Html.Attributes.class "text-sm text-slate-500", testId (identifier ++ "-empty") ] [ text emptyMessage ]

          else
            div [ Html.Attributes.class "divide-y divide-slate-100" ] (List.map (taskRow subjectId) tasks)
        ]


collectiblesHoldingsList : String -> List Collectible.CollectibleResponse -> Html Msg
collectiblesHoldingsList idPrefix collectibles =
    if List.isEmpty collectibles then
        p [ Html.Attributes.class "text-sm text-slate-500", testId (idPrefix ++ "-empty") ] [ text "No collectibles yet." ]

    else
        div [ Html.Attributes.class "divide-y divide-slate-100", testId idPrefix ]
            (List.map collectibleHoldingRow collectibles)


collectibleHoldingRow : Collectible.CollectibleResponse -> Html Msg
collectibleHoldingRow c =
    div [ Html.Attributes.class "flex items-center gap-2 py-2", testId "collectible-holding-row" ]
        [ Sprites.pixel c.art 5
        , span [ Html.Attributes.class "text-sm font-medium" ] [ text c.name ]
        , Ui.badge (collectibleKindLabel c.kind)
        ]


collectibleDetailView : String -> LoggedInModel -> Html Msg
collectibleDetailView collectibleId state =
    Ui.card
        [ a [ href "#/collectibles", Html.Attributes.class Ui.secondaryButtonClass, testId "back-collectibles" ] [ text "Back to collectibles" ]
        , case List.filter (\collectible -> collectible.id == collectibleId) state.collectibles of
            collectible :: _ ->
                div [ Html.Attributes.class "mt-3 space-y-2", testId "collectible-detail" ]
                    [ Sprites.pixel collectible.art 10
                    , p [ Html.Attributes.class "text-2xl font-semibold", testId "collectible-detail-name" ] [ text collectible.name ]
                    , Ui.label_ ("Collectible " ++ collectible.id)
                    , p [ Html.Attributes.class "text-sm" ] [ text ("Kind: " ++ collectibleKindLabel collectible.kind) ]
                    , p [ Html.Attributes.class "text-sm" ] [ text ("State: " ++ collectibleStateLabel collectible.state) ]
                    , p [ Html.Attributes.class "text-sm" ] [ text ("Transfer policy: " ++ collectiblePolicyLabel collectible.transferPolicy) ]
                    , case collectible.transferPolicy of
                        Collectible.CollectibleTransferPolicyTransferableBetweenUsers ->
                            tradeControls collectible state

                        Collectible.CollectibleTransferPolicyTransferableWithinOrganization ->
                            tradeControls collectible state

                        Collectible.CollectibleTransferPolicyNonTransferableExceptPayout ->
                            tradeUnavailableNote "This collectible's policy only allows it to move as a task payout, so it cannot be traded directly."

                        Collectible.CollectibleTransferPolicyIssuerControlled ->
                            tradeUnavailableNote "This collectible's policy only allows its issuer to move it, so it cannot be traded directly."
                    ]

            [] ->
                p [ Html.Attributes.class "mt-3 text-sm text-slate-500", testId "collectible-detail-missing" ] [ text "This collectible is no longer in your holdings." ]

        -- Rendered at the card level so a successful trade's confirmation persists
        -- even after the traded collectible leaves your holdings.
        , maybeNote state.transferMessage "transfer-message"
        ]


tradeControls : Collectible.CollectibleResponse -> LoggedInModel -> Html Msg
tradeControls collectible state =
    div [ Html.Attributes.class "mt-3 space-y-2" ]
        [ Ui.label_ "Trade to another user"
        , userPicker "transfer-recipient-id" state.transferRecipientId state.userDirectoryQuery TransferRecipientIdChanged "Choose user" state.userDirectory state.userDirectoryOffset
        , Ui.primaryButton [ type_ "button", onClick (TransferCollectibleClicked collectible.id), testId "transfer-collectible" ] "Trade"
        ]


tradeUnavailableNote : String -> Html Msg
tradeUnavailableNote reason =
    p [ Html.Attributes.class "mt-3 text-sm text-slate-500", testId "transfer-unavailable" ] [ text reason ]


{-| The Series section embedded on the Tasks hub (see `tasksView`) — content
only, no outer card or heading, since the hub wraps this in its own
`Ui.disclosure "tasks-series"` with "Series" as the disclosure title.
-}
seriesSection : LoggedInModel -> List (Html Msg)
seriesSection state =
    [ p [ Html.Attributes.class "text-sm text-slate-600" ] [ text "Group related tasks into an ordered series with its own discussion thread." ]
    , form [ Html.Attributes.class "mt-3 space-y-2", onSubmit CreateSeriesClicked ]
        [ Ui.fieldLabel "Title"
            [ Ui.textInput [ type_ "text", placeholder "Series title", value state.createSeriesTitle, onInput CreateSeriesTitleChanged, testId "series-create-title" ] ]
        , Ui.fieldLabel "Description"
            [ Ui.textarea_ [ placeholder "What is this series about?", value state.createSeriesDescription, onInput CreateSeriesDescriptionChanged, testId "series-create-description" ] ]
        , Ui.primaryButton [ type_ "submit", testId "create-series" ] "Create series"
        , maybeNote state.seriesMessage "series-message"
        ]
    , Ui.sectionTitle "Your series"
    , if List.isEmpty state.seriesList then
        p [ Html.Attributes.class "text-sm text-slate-500", testId "series-empty" ] [ text "No series yet." ]

      else
        div [ Html.Attributes.class "divide-y divide-slate-100", testId "series" ] (List.map seriesRow state.seriesList)
    ]


seriesRow : TaskSeries.TaskSeriesResponse -> Html Msg
seriesRow series =
    div [ Html.Attributes.class "flex flex-wrap items-center justify-between gap-2 py-2", testId "series-row" ]
        [ div [ Html.Attributes.class "flex flex-wrap items-center gap-2" ]
            [ p [ Html.Attributes.class "text-sm font-medium" ] [ text series.title ]
            , seriesStateBadge series.state
            ]
        , a [ href ("#/series/" ++ series.id), Html.Attributes.class Ui.secondaryButtonClass, testId "open-series" ] [ text "Open" ]
        ]


seriesDetailView : String -> LoggedInModel -> Html Msg
seriesDetailView seriesId state =
    Ui.card
        [ a [ href "#/tasks", Html.Attributes.class Ui.secondaryButtonClass, testId "back-series" ] [ text "Back to tasks" ]
        , case state.seriesDetail of
            Just data ->
                let
                    isCreator =
                        data.series.createdBy == state.subjectId
                in
                div [ Html.Attributes.class "mt-3 space-y-4", testId "series-detail" ]
                    [ div [ Html.Attributes.class "space-y-2" ]
                        [ p [ Html.Attributes.class "text-2xl font-semibold", testId "series-detail-title" ] [ text data.series.title ]
                        , seriesStateBadge data.series.state |> wrapBadge "series-state"
                        , p [ Html.Attributes.class "text-sm text-slate-700" ] [ text data.series.description ]
                        ]
                    , seriesTasksSection seriesId isCreator data
                    , if isCreator then
                        Ui.disclosure "series-creator-controls" False "Manage series" (seriesCreatorControls data.series state)

                      else
                        text ""
                    , Ui.disclosure "series-comments-section" False "Discussion" (seriesCommentsSection seriesId state data)
                    , maybeNote state.seriesMessage "series-message"
                    ]

            Nothing ->
                case state.seriesDetailError of
                    Just message ->
                        p [ Html.Attributes.class "mt-3 text-sm text-slate-700", testId "series-detail-missing" ] [ text ("Could not load this series: " ++ message) ]

                    Nothing ->
                        p [ Html.Attributes.class "mt-3 text-sm text-slate-500", testId "series-detail-missing" ] [ text ("Loading series " ++ seriesId ++ "…") ]
        ]


wrapBadge : String -> Html Msg -> Html Msg
wrapBadge identifier badge =
    span [ testId identifier ] [ badge ]


{-| Status badges below convey their state via more than color (the label
text still names the state); the tone just adds a faster-to-scan signal for
sighted users, e.g. skimming a long task list for anything that needs review.
-}
taskStateBadge : Task.TaskState -> Html Msg
taskStateBadge state =
    let
        ( tone, icon ) =
            case state of
                Task.TaskStateOpen ->
                    ( "success", "●" )

                Task.TaskStateDraft ->
                    ( "neutral", "○" )

                Task.TaskStateClosed ->
                    ( "info", "✓" )

                Task.TaskStateCancelled ->
                    ( "danger", "✕" )

                Task.TaskStateExpired ->
                    ( "warning", "⏳" )
    in
    Ui.badgeVariantWithIcon tone icon (taskStateLabel state)


{-| The task's reward as its own small badge, so a scannable list surfaces
"what's it worth" at a glance rather than as muted trailing text. Skipped
entirely for a "none" reward - there's nothing to highlight.
-}
taskRewardBadge : String -> Int -> Int -> Html Msg
taskRewardBadge rewardKind rewardCreditAmount rewardCollectibleCount =
    if rewardKind == "none" then
        text ""

    else
        Ui.badgeVariantWithIcon "reward" "◆" (rewardLabel rewardKind rewardCreditAmount rewardCollectibleCount)


submissionStateBadge : Submission.SubmissionState -> Html Msg
submissionStateBadge state =
    let
        tone =
            case state of
                Submission.SubmissionStateSubmitted ->
                    "neutral"

                Submission.SubmissionStateInvalid ->
                    "danger"

                Submission.SubmissionStateAccepted ->
                    "success"

                Submission.SubmissionStateRejected ->
                    "danger"

                Submission.SubmissionStateChangesRequested ->
                    "warning"
    in
    Ui.badgeVariant tone (submissionStateLabel state)


collectibleStateBadge : Collectible.CollectibleState -> Html Msg
collectibleStateBadge state =
    let
        tone =
            case state of
                Collectible.CollectibleStateMinted ->
                    "neutral"

                Collectible.CollectibleStateEscrowed ->
                    "warning"

                Collectible.CollectibleStateAwarded ->
                    "success"
    in
    Ui.badgeVariant tone (collectibleStateLabel state)


seriesStateBadge : String -> Html Msg
seriesStateBadge state =
    let
        tone =
            case state of
                "draft" ->
                    "neutral"

                "published" ->
                    "success"

                "closed" ->
                    "danger"

                _ ->
                    "neutral"
    in
    Ui.badgeVariant tone state


privacyRequestStatusBadge : String -> Html Msg
privacyRequestStatusBadge status =
    let
        tone =
            case status of
                "queued" ->
                    "warning"

                "resolved" ->
                    "success"

                _ ->
                    "neutral"
    in
    Ui.badgeVariant tone status


moderationReportStateBadge : String -> Html Msg
moderationReportStateBadge state =
    let
        tone =
            case state of
                "open" ->
                    "warning"

                "resolved" ->
                    "success"

                "dismissed" ->
                    "neutral"

                _ ->
                    "neutral"
    in
    Ui.badgeVariant tone state


seriesTasksSection : String -> Bool -> SeriesDetailData -> Html Msg
seriesTasksSection seriesId isCreator data =
    div [ Html.Attributes.class "space-y-2" ]
        [ Ui.sectionTitle "Tasks"
        , if List.isEmpty data.tasks then
            p [ Html.Attributes.class "text-sm text-slate-500", testId "series-tasks-empty" ] [ text "No tasks in this series yet." ]

          else
            div [ Html.Attributes.class "divide-y divide-slate-100", testId "series-tasks" ]
                (List.map (seriesTaskRow seriesId isCreator) data.tasks)
        ]


seriesTaskRow : String -> Bool -> SeriesTaskEntry -> Html Msg
seriesTaskRow seriesId isCreator entry =
    div [ Html.Attributes.class "flex flex-wrap items-center justify-between gap-2 py-2", testId "series-task-row" ]
        [ a [ href ("#/tasks/" ++ entry.id), Html.Attributes.class "w-full text-sm underline break-words sm:w-auto", testId "series-task-link" ] [ text (entry.title ++ " · " ++ entry.state) ]
        , if isCreator then
            div [ Html.Attributes.class "flex flex-wrap gap-2" ]
                [ Ui.secondaryButton [ type_ "button", onClick (MoveSeriesTaskUpClicked seriesId entry.id), testId "series-task-up" ] "Up"
                , Ui.secondaryButton [ type_ "button", onClick (MoveSeriesTaskDownClicked seriesId entry.id), testId "series-task-down" ] "Down"
                , Ui.secondaryButton [ type_ "button", onClick (RemoveSeriesTaskClicked seriesId entry.id), testId "series-remove-task" ] "Remove"
                ]

          else
            text ""
        ]


seriesCreatorControls : TaskSeries.TaskSeriesResponse -> LoggedInModel -> List (Html Msg)
seriesCreatorControls series state =
    [ form [ Html.Attributes.class "space-y-2", onSubmit (UpdateSeriesClicked series.id) ]
        [ Ui.fieldLabel "Title"
            [ Ui.textInput [ type_ "text", placeholder "Series title", value state.seriesRenameTitle, onInput SeriesRenameTitleChanged, testId "series-rename-title" ] ]
        , Ui.fieldLabel "Description"
            [ Ui.textarea_ [ placeholder "Description", value state.seriesRenameDescription, onInput SeriesRenameDescriptionChanged, testId "series-rename-description" ] ]
        , Ui.primaryButton [ type_ "submit", testId "series-update" ] "Save changes"
        ]
    , div [ Html.Attributes.class "flex flex-wrap gap-2" ] (seriesStateButtons series)
    , form [ Html.Attributes.class "flex flex-wrap items-end gap-2", onSubmit (AddSeriesTaskClicked series.id) ]
        [ Ui.fieldLabel "Add task"
            [ taskPicker "series-add-task-id" state.addSeriesTaskId AddSeriesTaskIdChanged state.tasks ]
        , Ui.primaryButton [ type_ "submit", disabled (state.addSeriesTaskId == ""), testId "series-add-task" ] "Add task"
        ]
    ]


seriesStateButtons : TaskSeries.TaskSeriesResponse -> List (Html Msg)
seriesStateButtons series =
    if series.state == "draft" then
        [ Ui.secondaryButton [ type_ "button", onClick (PublishSeriesClicked series.id), testId "series-publish" ] "Publish" ]

    else if series.state == "published" then
        [ Ui.secondaryButton [ type_ "button", onClick (UnpublishSeriesClicked series.id), testId "series-unpublish" ] "Unpublish"
        , Ui.secondaryButton [ type_ "button", onClick (CloseSeriesClicked series.id), testId "series-close" ] "Close"
        ]

    else if series.state == "closed" then
        [ Ui.secondaryButton [ type_ "button", onClick (ReopenSeriesClicked series.id), testId "series-reopen" ] "Reopen" ]

    else
        []


seriesCommentsSection : String -> LoggedInModel -> SeriesDetailData -> List (Html Msg)
seriesCommentsSection seriesId state data =
    [ if List.isEmpty data.comments then
        p [ Html.Attributes.class "text-sm text-slate-500", testId "series-comments-empty" ] [ text "No comments yet." ]

      else
        div [ Html.Attributes.class "space-y-2", testId "series-comments" ] (List.map seriesCommentRow data.comments)
    , form [ Html.Attributes.class "space-y-2", onSubmit (AddSeriesCommentClicked seriesId) ]
        [ Ui.textarea_ [ placeholder "Add a comment", value state.seriesCommentBody, onInput SeriesCommentBodyChanged, testId "series-comment-body" ]
        , Ui.primaryButton [ type_ "submit", testId "add-series-comment" ] "Comment"
        ]
    ]


seriesCommentRow : TaskSeries.SeriesCommentResponse -> Html Msg
seriesCommentRow comment =
    div [ Html.Attributes.class "rounded-md border border-slate-200 bg-white p-3", testId "series-comment" ]
        [ a [ href ("#/users/" ++ comment.authorUserID), Html.Attributes.class "text-xs font-medium text-slate-600 underline" ] [ text comment.authorUserID ]
        , p [ Html.Attributes.class "text-sm text-slate-700 break-words" ] [ text comment.body ]
        ]


userTaskListView : String -> String -> String -> List Task.TaskListItemResponse -> Html Msg
userTaskListView heading identifier userId tasks =
    Ui.card
        [ a [ href ("#/users/" ++ userId), Html.Attributes.class Ui.secondaryButtonClass, testId "back-user" ] [ text "Back to profile" ]
        , Ui.sectionTitle heading
        , if List.isEmpty tasks then
            p [ Html.Attributes.class "text-sm text-slate-500", testId (identifier ++ "-empty") ] [ text "Nothing to show." ]

          else
            div [ Html.Attributes.class "divide-y divide-slate-100", testId identifier ]
                (List.map
                    (\item ->
                        a [ href ("#/tasks/" ++ item.id), Html.Attributes.class "block py-2 text-sm underline", testId (identifier ++ "-row") ]
                            [ p [ Html.Attributes.class "font-medium break-words" ] [ text item.title ]
                            , p [ Html.Attributes.class "text-xs text-slate-500 break-words" ]
                                [ text (taskStateLabel item.state ++ " · " ++ rewardLabel item.rewardKind item.rewardCreditAmount item.rewardCollectibleCount ++ activeAssigneeSuffix item) ]
                            ]
                    )
                    tasks
                )
        ]


{-| The standalone `/users/<id>/submissions` route, still reachable from the
Profile page's "Submissions" link (`user-submissions-link` in
`userDetailView`) — kept as its own page (unlike Discovery/Series) since
that link explicitly targets its own URL.
-}
userSubmissionsView : String -> LoggedInModel -> Html Msg
userSubmissionsView userId state =
    Ui.card
        (a [ href ("#/users/" ++ userId), Html.Attributes.class Ui.secondaryButtonClass, testId "back-user" ] [ text "Back to profile" ]
            :: userSubmissionsSection state
        )


{-| The submissions section embedded on the Tasks hub (see `tasksView`) —
same content as the standalone page, minus the "Back to profile" link, which
doesn't make sense inline on a hub page.
-}
userSubmissionsSection : LoggedInModel -> List (Html Msg)
userSubmissionsSection state =
    let
        submissions =
            state.userSubmissions

        revisionItems =
            List.filter isRevisionSubmission submissions
    in
    [ Ui.sectionTitleWithCount "Revision inbox" (List.length revisionItems) "revision-inbox-heading"
    , if List.isEmpty revisionItems then
        p [ Html.Attributes.class "text-sm text-slate-500", testId "revision-inbox-empty" ] [ text "No requested revisions." ]

      else
        div [ Html.Attributes.class "divide-y divide-slate-100", testId "revision-inbox" ]
            (List.map revisionSubmissionRow revisionItems)
    , Ui.disclosure "user-submissions-all"
        False
        ("All submissions (" ++ String.fromInt (List.length submissions) ++ ")")
        [ if List.isEmpty submissions then
            p [ Html.Attributes.class "text-sm text-slate-500", testId "user-submissions-empty" ] [ text "No submissions." ]

          else
            div [ Html.Attributes.class "divide-y divide-slate-100", testId "user-submissions" ]
                (List.map userSubmissionRow submissions)
        , paginationControls "user-submissions-page" PreviousUserSubmissionsPageClicked NextUserSubmissionsPageClicked state.userSubmissionsOffset (List.length state.userSubmissions)
        ]
    , revisionTimelineView submissions
    ]


revisionSubmissionRow : Submission.SubmissionResponse -> Html Msg
revisionSubmissionRow item =
    div [ Html.Attributes.class "space-y-2 py-2", testId "revision-submission-row" ]
        [ userSubmissionRow item
        , Ui.primaryButton [ type_ "button", onClick (StartRevisionClicked item.taskID item.responseJSON), testId "revision-resubmit" ] "Revise"
        ]


userSubmissionRow : Submission.SubmissionResponse -> Html Msg
userSubmissionRow item =
    div [ Html.Attributes.class "space-y-1 py-2", testId "user-submission-row" ]
        [ a [ href ("#/tasks/" ++ item.taskID), Html.Attributes.class "text-sm underline", testId "user-submission-task-link" ] [ text ("Task " ++ item.taskID) ]
        , p [ Html.Attributes.class "text-xs text-slate-600" ] [ text (submissionStateLabel item.state) ]
        , reviewNoteView item.reviewNote
        , Ui.codeBlock [ testId "user-submission-response" ] item.responseJSON
        , validationErrorsView item.validationErrors
        , sensitiveFieldsView item.sensitiveFields
        ]


revisionTimelineView : List Submission.SubmissionResponse -> Html Msg
revisionTimelineView submissions =
    div [ Html.Attributes.class "space-y-2", testId "revision-timeline" ]
        [ Ui.sectionTitleWithCount "Revision timeline" (List.length submissions) "revision-timeline-heading"
        , if List.isEmpty submissions then
            p [ Html.Attributes.class "text-sm text-slate-500", testId "revision-timeline-empty" ] [ text "No submission history." ]

          else
            div [ Html.Attributes.class "divide-y divide-slate-100" ] (List.map revisionTimelineRow submissions)
        ]


revisionTimelineRow : Submission.SubmissionResponse -> Html Msg
revisionTimelineRow item =
    div [ Html.Attributes.class "space-y-1 py-2", testId "revision-timeline-row" ]
        [ div [ Html.Attributes.class "flex flex-wrap items-center gap-2" ]
            [ submissionStateBadge item.state
            , a [ href ("#/tasks/" ++ item.taskID), Html.Attributes.class "text-sm underline", testId "revision-timeline-task-link" ] [ text ("Task " ++ item.taskID) ]
            ]
        , reviewNoteView item.reviewNote
        , validationErrorsView item.validationErrors
        , sensitiveFieldsView item.sensitiveFields
        ]


isRevisionSubmission : Submission.SubmissionResponse -> Bool
isRevisionSubmission submission =
    submission.state == Submission.SubmissionStateChangesRequested


userDetailView : String -> String -> LoggedInModel -> Html Msg
userDetailView origin userId state =
    div [ Html.Attributes.class "space-y-6" ]
        (Ui.card
            [ Ui.sectionTitle "User"
            , p [ Html.Attributes.class "text-sm font-medium", testId "user-id" ] [ text userId ]
            , div [ Html.Attributes.class "flex flex-wrap gap-2" ]
                (a [ href ("#/users/" ++ userId ++ "/work"), Html.Attributes.class Ui.secondaryButtonClass, testId "user-work-link" ] [ text "Currently working on" ]
                    -- The submissions API rejects any viewer who isn't the
                    -- submitter themselves, so this link 404s/403s on anyone
                    -- else's profile — only show it on your own.
                    :: (if userId == state.subjectId then
                            [ a [ href ("#/users/" ++ userId ++ "/submissions"), Html.Attributes.class Ui.secondaryButtonClass, testId "user-submissions-link" ] [ text "My submissions" ] ]

                        else
                            []
                       )
                )
            , Ui.sectionTitle "Tasks posted"
            , case state.userProfile of
                Just profile ->
                    if List.isEmpty profile.tasks then
                        p [ Html.Attributes.class "text-sm text-slate-500", testId "user-tasks-empty" ] [ text "No public tasks." ]

                    else
                        div [ Html.Attributes.class "divide-y divide-slate-100", testId "user-tasks" ]
                            (List.map
                                (\item ->
                                    a [ href ("#/tasks/" ++ item.id), Html.Attributes.class "block py-2 text-sm underline", testId "user-task-row" ] [ text item.title ]
                                )
                                profile.tasks
                            )

                Nothing ->
                    case state.userProfileError of
                        Just message ->
                            p [ Html.Attributes.class "text-sm text-slate-700", testId "user-profile-error" ] [ text ("Could not load this user: " ++ message) ]

                        Nothing ->
                            p [ Html.Attributes.class "text-sm text-slate-500" ] [ text "Loading…" ]
            ]
            :: (if userId == state.subjectId then
                    [ accountSettingsCard state
                    , userAgentAccessCard origin state
                    ]

                else
                    []
               )
        )


accountSettingsCard : LoggedInModel -> Html Msg
accountSettingsCard state =
    Ui.card
        [ Ui.sectionTitle "Account settings"
        , form [ Html.Attributes.class "space-y-2", onSubmit UpdateProfileClicked ]
            [ Ui.fieldLabel "Email" [ Ui.textInput [ type_ "email", placeholder "person@example.com", value state.accountEmail, onInput AccountEmailChanged, testId "account-email" ] ]
            , Ui.primaryButton [ type_ "submit", testId "update-profile" ] "Save profile"
            ]
        , Ui.disclosure "account-email-verification"
            False
            "Email verification"
            [ Ui.secondaryButton [ type_ "button", onClick RequestEmailVerificationClicked, testId "request-email-verification" ] "Create verification token"
            , Ui.textInput [ type_ "text", placeholder "Verification token", value state.emailVerificationInput, onInput EmailVerificationInputChanged, testId "email-verification-token" ]
            , Ui.secondaryButton [ type_ "button", onClick ConfirmEmailVerificationClicked, testId "confirm-email-verification" ] "Verify email"
            ]
        , Ui.disclosure "account-password"
            False
            "Change password"
            [ form [ Html.Attributes.class "space-y-2", onSubmit ChangePasswordClicked ]
                [ Ui.fieldLabel "Current password" [ Ui.textInput [ type_ "password", value state.currentPassword, onInput CurrentPasswordChanged, testId "current-password" ] ]
                , Ui.fieldLabel "New password" [ Ui.textInput [ type_ "password", value state.newPassword, onInput NewPasswordChanged, testId "new-password" ] ]
                , Ui.primaryButton [ type_ "submit", testId "change-password" ] "Change password"
                ]
            ]
        , Ui.disclosure "account-privacy"
            False
            "Privacy requests"
            [ div [ Html.Attributes.class "flex flex-wrap gap-2" ]
                [ Ui.secondaryButton [ type_ "button", onClick (PrivacyRequestClicked Privacy.PrivacyRequestKindDataExport), testId "request-data-export" ] "Request data export"
                , Ui.secondaryButton [ type_ "button", onClick (PrivacyRequestClicked Privacy.PrivacyRequestKindSensitiveFieldDeletion), testId "request-sensitive-deletion" ] "Request sensitive-field deletion"
                ]
            , myPrivacyRequestsList state.myPrivacyRequests
            ]
        , Ui.disclosure "account-deactivate"
            False
            "Deactivate account"
            (if state.deactivateConfirming then
                [ p [ Html.Attributes.class "text-sm text-red-700", testId "deactivate-confirm-warning" ]
                    [ text "This permanently deactivates your account: your password and tokens are revoked and your email is anonymized. It cannot be undone." ]
                , div [ Html.Attributes.class "flex flex-wrap gap-2" ]
                    [ Ui.dangerButton [ type_ "button", onClick ConfirmDeactivateAccountClicked, testId "confirm-deactivate-account" ] "Yes, deactivate my account"
                    , Ui.secondaryButton [ type_ "button", onClick CancelDeactivateAccountClicked, testId "cancel-deactivate-account" ] "Keep my account"
                    ]
                ]

             else
                [ Ui.dangerButton [ type_ "button", onClick DeactivateAccountClicked, testId "deactivate-account" ] "Deactivate account" ]
            )
        , maybeNote state.accountMessage "account-message"
        ]


-- myPrivacyRequestsList shows the caller's own privacy requests with their
-- status: previously a queued request was a dead end (only platform admins
-- could ever see its state or the export result).
myPrivacyRequestsList : List Privacy.PrivacyRequestResponse -> Html Msg
myPrivacyRequestsList requests =
    if List.isEmpty requests then
        p [ Html.Attributes.class "text-sm text-slate-500", testId "my-privacy-empty" ] [ text "No privacy requests yet." ]

    else
        div [ Html.Attributes.class "space-y-2", testId "my-privacy-requests" ] (List.map myPrivacyRequestRow requests)


privacyRequestKindText : String -> String
privacyRequestKindText kind =
    case kind of
        "data_export" ->
            "Data export"

        "sensitive_field_deletion" ->
            "Sensitive-field deletion"

        other ->
            other


myPrivacyRequestRow : Privacy.PrivacyRequestResponse -> Html Msg
myPrivacyRequestRow request =
    div [ Html.Attributes.class "space-y-1 rounded-md bg-slate-50 p-2 text-sm", testId "my-privacy-request" ]
        ([ div [ Html.Attributes.class "flex flex-wrap items-center gap-2" ]
            [ span [ Html.Attributes.class "font-medium" ] [ text (privacyRequestKindText request.kind) ]
            , Ui.badge request.status
            , span [ Html.Attributes.class "text-xs text-slate-500" ] [ text request.createdAt ]
            ]
         ]
            ++ (if request.resolutionNote /= "" then
                    [ p [ Html.Attributes.class "text-xs text-slate-600" ] [ text ("Resolution: " ++ request.resolutionNote) ] ]

                else
                    []
               )
            ++ (if request.exportJSON /= "" then
                    [ Ui.codeBlock [ testId "my-privacy-export" ] request.exportJSON ]

                else
                    []
               )
        )


userAgentAccessCard : String -> LoggedInModel -> Html Msg
userAgentAccessCard origin state =
    Ui.card
        (Ui.sectionTitle "Your agent access"
            :: p [ Html.Attributes.class "text-sm text-slate-700" ] [ text "A personal agent token lets you drive Sharecrop from an agent (over MCP) or the API. Only you can see it here. Treat it like a password." ]
            :: (case state.userAgentToken of
                    Nothing ->
                        [ Ui.primaryButton [ type_ "button", onClick MintUserTokenClicked, testId "mint-user-token" ] "Create agent token" ]

                    Just token ->
                        [ Ui.label_ "Agent token"
                        , Ui.codeBlock [ testId "user-token" ] token
                        , copyButton token
                        , Ui.secondaryButton [ type_ "button", onClick MintUserTokenClicked, testId "mint-user-token" ] "Rotate token"
                        , Ui.label_ "Install the MCP"
                        , integrationEntry "Claude Code — add the Sharecrop MCP server:" "user-mcp-install" (mcpClaudeInstall origin token)
                        , integrationEntry "Claude Code — update the server (e.g. after rotating the token):" "user-mcp-update" (mcpClaudeUpdate origin token)
                        , integrationEntry "Or add it to your MCP client config (.mcp.json, Codex, Claude Desktop):" "user-mcp-config" (mcpConfig origin token)
                        ]
               )
        )


mcpClaudeInstall : String -> String -> String
mcpClaudeInstall origin token =
    "claude mcp add --transport http sharecrop " ++ origin ++ "/mcp --header \"Authorization: Bearer " ++ token ++ "\""


mcpClaudeUpdate : String -> String -> String
mcpClaudeUpdate origin token =
    "claude mcp remove sharecrop && " ++ mcpClaudeInstall origin token


overviewView : LoggedInModel -> Html Msg
overviewView state =
    div [ Html.Attributes.class "space-y-6", testId "overview" ]
        [ Ui.sectionTitle "Credit account"
        , balanceView state.balance
        , ledgerView state.entries state.ledgerOffset
        ]


ownerChooser : LoggedInModel -> Html Msg
ownerChooser state =
    if List.isEmpty state.organizations then
        text ""

    else
        div []
            [ Ui.label_ "Owner"
            , div [ Html.Attributes.class "flex flex-wrap gap-2", testId "create-owner" ]
                (Ui.chooserButton (state.createTaskOwner == "") (CreateTaskOwnerChanged "") "create-owner-me" "Me"
                    :: List.map (ownerButton state.createTaskOwner) state.organizations
                )
            ]


ownerButton : String -> Organization.OrganizationResponse -> Html Msg
ownerButton selected organization =
    Ui.chooserButton (selected == organization.id)
        (CreateTaskOwnerChanged organization.id)
        ("create-owner-" ++ organization.id)
        organization.name


organizationsView : LoggedInModel -> Html Msg
organizationsView state =
    Ui.card
        [ p [ Html.Attributes.class "text-sm text-slate-600" ] [ text "Organizations you belong to. Create one to own tasks and credits as a team." ]
        , organizationsList state
        , form [ Html.Attributes.class "mt-3 flex flex-wrap items-end gap-2", onSubmit CreateOrgClicked ]
            [ Ui.fieldLabel "New organization"
                [ Ui.textInput [ type_ "text", placeholder "Organization name", value state.createOrgName, onInput CreateOrgNameChanged, testId "create-org-name" ] ]
            , Ui.primaryButton [ type_ "submit", testId "create-org" ] "Create organization"
            ]
        , maybeNote state.orgMessage "org-message"

        -- Standalone teams previously existed only inside <select> pickers:
        -- there was no way to create one in the browser and no link to a
        -- team's page outside organization pages.
        , Ui.sectionTitle "Teams"
        , p [ Html.Attributes.class "text-sm text-slate-600" ] [ text "Personal teams for sharing tasks with a fixed group, outside any organization." ]
        , standaloneTeamsList state.standaloneTeams
        , form [ Html.Attributes.class "mt-3 flex flex-wrap items-end gap-2", onSubmit CreateTeamClicked ]
            [ Ui.fieldLabel "New team"
                [ Ui.textInput [ type_ "text", placeholder "Team name", value state.createTeamName, onInput CreateTeamNameChanged, testId "create-team-name" ] ]
            , Ui.primaryButton [ type_ "submit", testId "create-team" ] "Create team"
            ]
        , maybeNote state.createTeamMessage "create-team-message"
        ]


standaloneTeamsList : List Team.TeamResponse -> Html Msg
standaloneTeamsList teams =
    if List.isEmpty teams then
        p [ Html.Attributes.class "text-sm text-slate-500", testId "standalone-teams-empty" ] [ text "No teams yet." ]

    else
        div [ Html.Attributes.class "divide-y divide-slate-100", testId "standalone-teams" ]
            (List.map
                (\team ->
                    div [ Html.Attributes.class "flex items-center justify-between gap-2 py-2" ]
                        [ span [ Html.Attributes.class "text-sm font-medium" ] [ text team.name ]
                        , a [ href ("#/teams/" ++ team.id), Html.Attributes.class Ui.secondaryButtonClass, testId "standalone-team-open" ] [ text "Open" ]
                        ]
                )
                teams
            )


organizationDetailView : LoggedInModel -> Html Msg
organizationDetailView state =
    let
        name =
            state.organizations
                |> List.filter (\organization -> organization.id == state.activeOrgId)
                |> List.head
                |> Maybe.map .name
                |> Maybe.withDefault state.activeOrgId
    in
    Ui.card
        [ a [ href "#/organizations", Html.Attributes.class Ui.secondaryButtonClass, testId "back-organizations" ] [ text "Back to organizations" ]
        , Ui.sectionTitle name
        , activeOrganizationView state
        ]


organizationsList : LoggedInModel -> Html Msg
organizationsList state =
    if List.isEmpty state.organizations then
        p [ Html.Attributes.class "text-sm text-slate-500", testId "organizations-empty" ] [ text "You do not belong to any organizations yet." ]

    else
        div [ Html.Attributes.class "divide-y divide-slate-100", testId "organizations" ] (List.map organizationRow state.organizations)


activeOrganizationView : LoggedInModel -> Html Msg
activeOrganizationView state =
    if state.activeOrgId == "" then
        text ""

    else
        div [ Html.Attributes.class "mt-4 space-y-4 rounded-md bg-slate-50 p-4", testId "active-organization" ]
            (Ui.label_ ("Spendable balance: " ++ balanceLabel state.orgBalance)
                :: allocatedLine state.orgBalance
                ++ [ organizationOperationsDashboard state
            , Ui.sectionTitleWithCount "Organization tasks" (List.length state.orgTasks) "org-tasks-heading"
            , Ui.disclosure "org-task-filters" False "Filters" [ orgTaskControls state ]
            , tasksListSimple "org-tasks" state.orgTasks
            , maybeNote state.orgTaskMessage "org-task-message"
            , Ui.disclosure "org-teams-section" False ("Teams (" ++ String.fromInt (List.length state.orgTeams) ++ ")") <|
                [ orgTeamsList state.orgTeams
                , form [ Html.Attributes.class "flex flex-wrap items-end gap-2", onSubmit CreateOrgTeamClicked ]
                    [ Ui.fieldLabel "New team"
                        [ Ui.textInput [ type_ "text", placeholder "Team name", value state.createOrgTeamName, onInput CreateOrgTeamNameChanged, testId "create-org-team-name" ] ]
                    , Ui.primaryButton [ type_ "submit", testId "create-org-team" ] "Create team"
                    ]
                , maybeNote state.orgTeamMessage "org-team-message"
                ]
            , Ui.disclosure "org-members-section" False ("Members (" ++ String.fromInt (List.length state.orgMembers) ++ ")") <|
                [ orgMembersList state.orgMembers
                , Ui.sectionTitle "Provision a member"
                , form [ Html.Attributes.class "flex flex-wrap items-end gap-2", onSubmit ProvisionMemberClicked ]
                    [ Ui.fieldLabel "Member email"
                        [ Ui.textInput [ type_ "email", placeholder "person@example.com", value state.provisionMemberEmail, onInput ProvisionMemberEmailChanged, testId "provision-member-email" ] ]
                    , provisionRolePicker state.provisionMemberRoles
                    , Ui.primaryButton [ type_ "submit", testId "provision-member" ] "Provision member"
                    ]
                , maybeNote state.provisionMemberMessage "provision-member-message"
                ]
            , Ui.disclosure "org-collectibles-section" False ("Collectibles (" ++ String.fromInt (List.length state.orgCollectibles) ++ ")") <|
                [ collectiblesHoldingsList "org-collectibles" state.orgCollectibles
                , maybeNote state.orgCollectiblesMessage "org-collectibles-message"
                , if List.isEmpty state.orgCollectibles then
                    text ""

                  else
                    div [ Html.Attributes.class "mt-3 space-y-2" ]
                        [ Ui.label_ "Award a collectible to a member"
                        , orgMemberPicker "award-org-collectible-recipient" state.awardOrgCollectibleRecipientId AwardOrgCollectibleRecipientIdChanged state.orgMembers
                        , div [ Html.Attributes.class "divide-y divide-slate-100", testId "org-collectible-award-rows" ] (List.map (orgCollectibleAwardRow state.awardOrgCollectibleRecipientId) state.orgCollectibles)
                        ]
                , -- Kept outside the isEmpty branch above: a successful award empties
                  -- state.orgCollectibles when it was the org's last one, which would
                  -- otherwise make this confirmation disappear the instant it's shown.
                  maybeNote state.awardOrgCollectibleMessage "award-org-collectible-message"
                ]
            , Ui.disclosure "org-credentials-section" False ("Credentials (" ++ String.fromInt (List.length state.orgCredentials) ++ ")") <|
                [ orgCredentialsList state.orgCredentials
                , orgNewCredentialView state.newOrgCredential
                , form [ Html.Attributes.class "mt-3 space-y-3", onSubmit CreateOrgCredentialClicked ]
                    [ Ui.textInput [ type_ "text", placeholder "Credential label", value state.orgCredentialLabel, onInput OrgCredentialLabelChanged, testId "org-credential-label" ]
                    , div [ Html.Attributes.class "space-y-1" ] (List.map (orgScopeCheckbox state.orgCredentialScopes) allScopes)
                    , Ui.fieldLabel "Expires in (hours, blank for never)" [ Ui.textInput [ type_ "number", placeholder "never", value state.orgCredentialExpiresHours, onInput OrgCredentialExpiresHoursChanged, testId "org-credential-expires-hours" ] ]
                    , Ui.primaryButton [ type_ "submit", testId "create-org-credential" ] "Create credential"
                    , maybeNote state.orgCredentialMessage "org-credential-message"
                    ]
                ]
            ]
            )


orgScopeCheckbox : List Agent.AgentScope -> Agent.AgentScope -> Html Msg
orgScopeCheckbox selected scope =
    label [ Html.Attributes.class "flex min-h-[44px] items-center gap-2 text-sm" ]
        [ Html.input
            [ type_ "checkbox"
            , Html.Attributes.class Ui.checkboxClass
            , checked (List.member scope selected)
            , onCheck (\_ -> ToggleOrgCredentialScope scope)
            , testId ("org-scope-" ++ scopeTag scope)
            ]
            []
        , span [] [ text (scopeLabel scope ++ " (" ++ scopeTag scope ++ ")") ]
        ]


orgNewCredentialView : Maybe Agent.OrgCredentialCreatedResponse -> Html Msg
orgNewCredentialView created =
    case created of
        Just credential ->
            div [ Html.Attributes.class "mt-4 space-y-3 rounded-md bg-slate-50 p-4" ]
                [ Ui.label_ "New organization token (shown once)"
                , Ui.codeBlock [ testId "org-credential-secret" ] credential.secret
                ]

        Nothing ->
            text ""


orgCredentialsList : List Agent.OrgCredentialResponse -> Html Msg
orgCredentialsList credentials =
    if List.isEmpty credentials then
        text ""

    else
        div [ Html.Attributes.class "mt-2 divide-y divide-slate-100", testId "org-credentials" ] (List.map orgCredentialRow credentials)


orgCredentialRow : Agent.OrgCredentialResponse -> Html Msg
orgCredentialRow credential =
    div [ Html.Attributes.class "flex items-center justify-between py-2", testId "org-credential-row" ]
        [ div []
            [ p [ Html.Attributes.class "font-medium" ] [ text credential.label ]
            , p [ Html.Attributes.class "text-xs text-slate-500" ] [ text (credentialStateLabel credential.state ++ expiryNote credential.expiresAt ++ " · " ++ String.join ", " (List.map scopeLabel credential.scopes)) ]
            ]
        , orgRevokeButton credential
        ]


orgRevokeButton : Agent.OrgCredentialResponse -> Html Msg
orgRevokeButton credential =
    case credential.state of
        Agent.AgentCredentialStateActive ->
            Ui.secondaryButton [ onClick (RevokeOrgCredentialClicked credential.id), testId "revoke-org-credential" ] "Revoke"

        Agent.AgentCredentialStateRevoked ->
            span [ Html.Attributes.class "text-xs text-slate-600" ] [ text "revoked" ]


organizationOperationsDashboard : LoggedInModel -> Html Msg
organizationOperationsDashboard state =
    div [ Html.Attributes.class "space-y-3 rounded-md border border-slate-200 bg-white p-3", testId "org-operations-dashboard" ]
        [ Ui.sectionTitle "Operations"
        , div [ Html.Attributes.class "grid gap-2 sm:grid-cols-2" ]
            [ operationMetric "Spendable" (balanceLabel state.orgBalance) "org-ops-balance"
            , operationMetric "Allocated" (allocatedLabel state.orgBalance) "org-ops-allocated"
            , operationMetric "Teams" (String.fromInt (List.length state.orgTeams)) "org-ops-teams"
            , operationMetric "Active members" (String.fromInt (countMembers Organization.MembershipStatusActive state.orgMembers)) "org-ops-members-active"
            , operationMetric "Inactive members" (String.fromInt (inactiveMemberCount state.orgMembers)) "org-ops-members-inactive"
            , operationMetric "Collectibles" (String.fromInt (List.length state.orgCollectibles)) "org-ops-collectibles"
            , operationMetric "Draft tasks" (String.fromInt (countTasks Task.TaskStateDraft state.orgTasks)) "org-ops-tasks-draft"
            , operationMetric "Open tasks" (String.fromInt (countTasks Task.TaskStateOpen state.orgTasks)) "org-ops-tasks-open"
            , operationMetric "Closed tasks" (String.fromInt (countTasks Task.TaskStateClosed state.orgTasks)) "org-ops-tasks-closed"
            ]
        , orgLedgerPanel state.orgLedger state.orgLedgerOffset
        , orgAuditPanel state.orgAuditEvents state.orgAuditMessage
        ]


orgLedgerPanel : List Ledger.LedgerEntryResponse -> Int -> Html Msg
orgLedgerPanel entries offset =
    div [ Html.Attributes.class "space-y-2", testId "org-ledger-panel" ]
        [ h3 [ Html.Attributes.class "text-sm font-semibold text-slate-900" ] [ text "Organization ledger" ]
        , if List.isEmpty entries then
            p [ Html.Attributes.class "text-sm text-slate-500", testId "org-ledger-empty" ] [ text "No ledger entries." ]

          else
            table [ Html.Attributes.class "w-full text-left text-sm" ]
                [ tbody [ testId "org-ledger" ] (List.map ledgerRow entries)
                ]
        , paginationControls "org-ledger-page" PreviousOrgLedgerPageClicked NextOrgLedgerPageClicked offset (List.length entries)
        ]


orgAuditPanel : List Admin.AuditEventResponse -> Maybe Note -> Html Msg
orgAuditPanel events message =
    div [ Html.Attributes.class "space-y-2", testId "org-audit-panel" ]
        [ h3 [ Html.Attributes.class "text-sm font-semibold text-slate-900" ] [ text "Organization audit" ]
        , if List.isEmpty events then
            p [ Html.Attributes.class "text-sm text-slate-500", testId "org-audit-empty" ] [ text "No audit events." ]

          else
            div [ Html.Attributes.class "space-y-2", testId "org-audit-events" ]
                (List.map orgAuditEventRow events)
        , maybeNote message "org-audit-message"
        ]


orgAuditEventRow : Admin.AuditEventResponse -> Html Msg
orgAuditEventRow event =
    div [ Html.Attributes.class "rounded-md bg-slate-50 p-2 text-sm", testId "org-audit-event" ]
        [ p [ Html.Attributes.class "font-medium text-slate-900" ] [ text event.action ]
        , p [ Html.Attributes.class "text-xs text-slate-500" ] [ text (event.subjectKind ++ " · " ++ event.createdAt) ]
        ]


operationMetric : String -> String -> String -> Html Msg
operationMetric labelText valueText identifier =
    div [ Html.Attributes.class "rounded-md bg-slate-50 p-3", testId identifier ]
        [ p [ Html.Attributes.class "text-xs uppercase text-slate-500" ] [ text labelText ]
        , p [ Html.Attributes.class "text-lg font-semibold text-slate-900" ] [ text valueText ]
        ]


countMembers : Organization.MembershipStatus -> List Organization.OrganizationMemberResponse -> Int
countMembers status members =
    List.length (List.filter (\member -> member.status == status) members)


inactiveMemberCount : List Organization.OrganizationMemberResponse -> Int
inactiveMemberCount members =
    List.length members - countMembers Organization.MembershipStatusActive members


countTasks : Task.TaskState -> List Task.TaskListItemResponse -> Int
countTasks state tasks =
    List.length (List.filter (\task -> task.state == state) tasks)


orgTaskControls : LoggedInModel -> Html Msg
orgTaskControls state =
    div [ Html.Attributes.class "space-y-2" ]
        [ Ui.fieldLabel "Search organization tasks"
            [ Ui.textInput [ type_ "search", placeholder "Task title or ID", value state.orgTaskQuery, onInput OrgTaskQueryChanged, testId "org-task-query" ] ]
        , taskTypeFilterSelect "org-task-type" state.orgTaskTypeFilter OrgTaskTypeFilterChanged
        , taskSortSelect "org-task-sort" state.orgTaskSort OrgTaskSortChanged
        , div [ Html.Attributes.class "flex flex-wrap gap-2" ]
            [ Ui.secondaryButton [ type_ "button", onClick SearchOrgTasksClicked, testId "org-task-search" ] "Search"
            ]
        , paginationControls "org-tasks-page" PreviousOrgTasksPageClicked NextOrgTasksPageClicked state.orgTaskOffset (List.length state.orgTasks)
        , div [ Html.Attributes.class "flex flex-wrap gap-2", testId "org-task-filter" ]
            (List.map (orgTaskFilterButton state.orgTaskFilter) orgTaskFilterOptions)
        , queueSavedViews
            { nameValue = state.orgTaskSavedViewName
            , nameChanged = OrgTaskSavedViewNameChanged
            , saveClicked = SaveOrgTaskViewClicked
            , applyClicked = ApplyOrgTaskViewClicked
            , views = state.orgTaskSavedViews
            , prefix = "org-task"
            }
        ]


orgTaskFilterOptions : List ( String, String )
orgTaskFilterOptions =
    [ ( "", "All" )
    , ( "open", "Open" )
    , ( "draft", "Draft" )
    , ( "closed", "Closed" )
    ]


orgTaskFilterButton : String -> ( String, String ) -> Html Msg
orgTaskFilterButton selected ( tag, labelText ) =
    Ui.chooserButton (selected == tag)
        (OrgTaskFilterChanged tag)
        ("org-task-filter-"
            ++ (if tag == "" then
                    "all"

                else
                    tag
               )
        )
        labelText


orgTeamsList : List Team.TeamResponse -> Html Msg
orgTeamsList teams =
    if List.isEmpty teams then
        p [ Html.Attributes.class "text-sm text-slate-500", testId "org-teams-empty" ] [ text "No teams yet." ]

    else
        div [ Html.Attributes.class "divide-y divide-slate-100", testId "org-teams" ]
            (List.map (\team -> a [ href ("#/teams/" ++ team.id), Html.Attributes.class "block py-1 text-sm underline", testId "org-team-row" ] [ text team.name ]) teams)


provisionRolePicker : List String -> Html Msg
provisionRolePicker selected =
    div [ Html.Attributes.class "flex flex-wrap gap-2" ]
        (List.map (roleCheckbox selected) provisionableRoles)


provisionableRoles : List String
provisionableRoles =
    [ "member", "reviewer", "public_publisher", "billing", "admin" ]


roleCheckbox : List String -> String -> Html Msg
roleCheckbox selected role =
    Ui.checkbox [ checked (List.member role selected), onCheck (\_ -> ToggleProvisionMemberRole role), testId ("provision-role-" ++ role) ] (roleLabel role)


roleLabel : String -> String
roleLabel role =
    case role of
        "public_publisher" ->
            "public publisher"

        _ ->
            role


orgMembersList : List Organization.OrganizationMemberResponse -> Html Msg
orgMembersList members =
    if List.isEmpty members then
        p [ Html.Attributes.class "text-sm text-slate-500", testId "org-members-empty" ] [ text "No members yet." ]

    else
        div [ Html.Attributes.class "divide-y divide-slate-100", testId "org-members" ] (List.map orgMemberRow members)


orgMemberRow : Organization.OrganizationMemberResponse -> Html Msg
orgMemberRow member =
    let
        roles =
            if List.isEmpty member.roles then
                "no roles"

            else
                String.join ", " (List.map organizationRoleText member.roles)
    in
    div [ Html.Attributes.class "flex flex-wrap items-center justify-between gap-2 py-2", testId "org-member-row" ]
        [ div []
            [ a [ href ("#/users/" ++ member.userID), Html.Attributes.class "text-sm font-medium underline", testId "org-member-link" ] [ text member.userID ]
            , p [ Html.Attributes.class "text-xs text-slate-600" ] [ text (roles ++ " · " ++ membershipStatusText member.status) ]
            ]
        , case member.status of
            Organization.MembershipStatusActive ->
                div [ Html.Attributes.class "flex flex-wrap gap-2" ]
                    [ Ui.secondaryButton [ type_ "button", onClick (UpdateMemberRolesClicked member.userID [ "member" ]), testId "member-role-member" ] "Member"
                    , Ui.secondaryButton [ type_ "button", onClick (UpdateMemberRolesClicked member.userID [ "member", "reviewer" ]), testId "member-role-reviewer" ] "Reviewer"
                    , Ui.secondaryButton [ type_ "button", onClick (UpdateMemberRolesClicked member.userID [ "admin" ]), testId "member-role-admin" ] "Admin"
                    , Ui.secondaryButton [ type_ "button", onClick (DeactivateMemberClicked member.userID), testId "deactivate-member" ] "Deactivate"
                    ]

            Organization.MembershipStatusDeactivated ->
                -- Role changes and deactivation require an active membership,
                -- so these buttons only produce server rejections here. The
                -- API has no reactivation path, so none is offered.
                p [ Html.Attributes.class "text-xs text-slate-500", testId "member-deactivated-note" ]
                    [ text "Deactivated." ]

            Organization.MembershipStatusRemoved ->
                text ""
        ]


membershipStatusText : Organization.MembershipStatus -> String
membershipStatusText status =
    case status of
        Organization.MembershipStatusActive ->
            "active"

        Organization.MembershipStatusDeactivated ->
            "deactivated"

        Organization.MembershipStatusRemoved ->
            "removed"


organizationRoleText : Organization.OrganizationRole -> String
organizationRoleText role =
    case role of
        Organization.OrganizationRoleOwner ->
            "owner"

        Organization.OrganizationRoleAdmin ->
            "admin"

        Organization.OrganizationRoleMember ->
            "member"

        Organization.OrganizationRoleBilling ->
            "billing"

        Organization.OrganizationRoleReviewer ->
            "reviewer"

        Organization.OrganizationRolePublicPublisher ->
            "public publisher"


tasksListSimple : String -> List Task.TaskListItemResponse -> Html Msg
tasksListSimple identifier tasks =
    if List.isEmpty tasks then
        p [ Html.Attributes.class "text-sm text-slate-500", testId (identifier ++ "-empty") ] [ text "No tasks yet." ]

    else
        div [ Html.Attributes.class "divide-y divide-slate-100", testId identifier ]
            (List.map (\item -> p [ Html.Attributes.class "py-1 text-sm", testId (identifier ++ "-row") ] [ text (item.title ++ " · " ++ taskStateLabel item.state) ]) tasks)


organizationRow : Organization.OrganizationResponse -> Html Msg
organizationRow organization =
    div [ Html.Attributes.class "flex items-center justify-between py-2", testId "organization-row" ]
        [ p [ Html.Attributes.class "font-medium" ] [ text organization.name ]
        , a [ href ("#/organizations/" ++ organization.id), Html.Attributes.class Ui.secondaryButtonClass, testId "select-organization" ] [ text "Open" ]
        ]


balanceView : Maybe Wallet -> Html Msg
balanceView balance =
    Ui.card
        ([ Ui.label_ "Spendable balance"
         , p [ Html.Attributes.class "text-3xl font-semibold", testId "balance" ] [ text (balanceLabel balance) ]
         ]
            ++ allocatedLine balance
        )


-- balanceLabel shows the SPENDABLE credits (what the account can spend or use
-- to fund tasks). Credits allocated to tasks are shown separately by
-- allocatedLine - they are locked until the task finishes or is refunded, so
-- they are deliberately not part of this figure.
balanceLabel : Maybe Wallet -> String
balanceLabel balance =
    case balance of
        Just wallet ->
            String.fromInt wallet.spendable ++ " credits"

        Nothing ->
            "Loading…"


-- allocatedLabel shows the credits currently locked to tasks (the allocated
-- wallet section), for the org operations dashboard's compact metric grid.
allocatedLabel : Maybe Wallet -> String
allocatedLabel balance =
    case balance of
        Just wallet ->
            String.fromInt wallet.allocated ++ " credits"

        Nothing ->
            "Loading…"


allocatedLine : Maybe Wallet -> List (Html Msg)
allocatedLine balance =
    case balance of
        Just wallet ->
            if wallet.allocated > 0 then
                [ p [ Html.Attributes.class "text-sm text-slate-600", testId "allocated-balance" ]
                    [ text (String.fromInt wallet.allocated ++ " credits allocated to tasks (locked until each task finishes or is refunded)") ]
                ]

            else
                []

        Nothing ->
            []


createTaskView : LoggedInModel -> Html Msg
createTaskView state =
    form [ Html.Attributes.class "space-y-4 rounded-lg border border-slate-200 bg-white p-6 shadow-sm", onSubmit CreateTaskClicked ]
        [ Ui.sectionTitle "Create a task"
        , Ui.fieldLabel "Title *"
            (Ui.textInputToned state.createTitleInvalid [ type_ "text", placeholder "Short, descriptive title", value state.createTitle, onInput CreateTitleChanged, testId "create-title" ]
                :: (if state.createTitleInvalid then
                        [ Ui.fieldError "Title is required" ]

                    else
                        []
                   )
            )
        , Ui.fieldLabel "Template" [ taskTypeSelect state.createTaskType ]
        , Ui.fieldLabel "Description *"
            (Ui.textareaToned state.createDescriptionInvalid [ placeholder "What the worker should do", value state.createDescription, onInput CreateDescriptionChanged, Html.Attributes.rows 3, testId "create-description" ]
                :: (if state.createDescriptionInvalid then
                        [ Ui.fieldError "Description is required" ]

                    else
                        []
                   )
            )
        , if state.createTaskType == "general" then
            schemaDesignerView state

          else
            p [ Html.Attributes.class "text-xs text-slate-600", testId "template-schema-note" ]
                [ text ("The " ++ taskTypeLabel state.createTaskType ++ " template prefilled the description and response schema; open Advanced options below to review or edit the schema.") ]
        , Ui.disclosure "create-advanced-options"
            False
            "Advanced options"
            [ Ui.fieldLabel "Reference URL (optional, e.g. a pull request)" [ Ui.textInput [ type_ "text", placeholder "https://github.com/org/repo/pull/123", value state.createReferenceURL, onInput CreateReferenceURLChanged, testId "create-reference-url" ] ]
            , Ui.fieldLabel "Response schema (JSON, advanced)" [ Ui.textarea_ [ placeholder "{\"kind\":\"freeform\"}", value state.createResponseSchema, onInput CreateResponseSchemaChanged, Html.Attributes.rows 3, testId "create-response-schema" ] ]
            , Ui.fieldLabel "Task input (JSON, optional)" [ Ui.textarea_ [ placeholder "Embed any data the worker needs, or leave blank", value state.createPayloadJson, onInput CreatePayloadChanged, Html.Attributes.rows 3, testId "create-payload" ] ]
            , selectedAttachmentsView "Attachments" state.createAttachments PickCreateAttachmentClicked RemoveCreateAttachmentClicked "create-attachments"
            ]
        , Ui.label_ "Reward"
        , div [ Html.Attributes.class "flex flex-wrap gap-2" ] (List.map (rewardKindButton state.createRewardKind) allRewardKinds)
        , rewardAmountField state
        , rewardCollectibleField state
        , Ui.label_ "Visibility"
        , div [ Html.Attributes.class "flex flex-wrap gap-2" ] (List.map (visibilityButton state.createVisibility) allVisibilityTags)
        , visibilityScopeField state
        , Ui.disclosure "create-task-ownership"
            False
            "Ownership & access"
            [ ownerChooser state
            , Ui.label_ "Participation"
            , div [ Html.Attributes.class "flex flex-wrap gap-2" ] (List.map (participationButton state.createParticipationPolicy) allParticipationPolicies)
            , if participationUsesReservation state.createParticipationPolicy then
                Ui.fieldLabel "Reservation expiry (hours)" [ Ui.textInput [ type_ "number", placeholder "48", value state.createReservationHours, onInput CreateReservationHoursChanged, testId "create-reservation-hours" ] ]

              else
                text ""
            , Ui.label_ "Assignee"
            , div [ Html.Attributes.class "flex flex-wrap gap-2" ] (List.map (assigneeScopeButton state.createAssigneeScope) allAssigneeScopes)
            ]
        , Ui.primaryButton [ type_ "submit", testId "create-task" ] "Create task"
        , maybeNote state.createMessage "create-message"
        ]


allRewardKinds : List String
allRewardKinds =
    [ "none", "credit", "collectible", "bundle" ]


rewardKindButton : String -> String -> Html Msg
rewardKindButton selected kind =
    Ui.chooserButton (selected == kind)
        (CreateRewardKindChanged kind)
        ("create-reward-kind-" ++ kind)
        (rewardKindLabel kind)


rewardKindLabel : String -> String
rewardKindLabel kind =
    case kind of
        "credit" ->
            "Credits"

        "collectible" ->
            "Collectible"

        "bundle" ->
            "Bundle"

        _ ->
            "No reward"


rewardAmountField : LoggedInModel -> Html Msg
rewardAmountField state =
    if state.createRewardKind == "credit" || state.createRewardKind == "bundle" then
        Ui.fieldLabel "Credit amount *"
            (Ui.textInputToned state.createRewardAmountInvalid [ type_ "number", placeholder "Amount in credits", value state.createRewardAmount, onInput CreateRewardAmountChanged, testId "create-reward" ]
                :: (if state.createRewardAmountInvalid then
                        [ Ui.fieldError "Enter a credit amount of at least 1" ]

                    else
                        []
                   )
            )

    else
        text ""


rewardCollectibleField : LoggedInModel -> Html Msg
rewardCollectibleField state =
    if state.createRewardKind == "collectible" || state.createRewardKind == "bundle" then
        let
            available =
                List.filter (\collectible -> collectible.state == Collectible.CollectibleStateMinted) state.collectibles
        in
        div [ Html.Attributes.class "space-y-2", testId "create-reward-collectibles" ]
            [ Ui.label_ "Collectibles"
            , if List.isEmpty available then
                p [ Html.Attributes.class "text-sm text-slate-500" ] [ text "No minted collectibles available." ]

              else
                div [ Html.Attributes.class "space-y-1" ]
                    (List.map
                        (\collectible ->
                            Ui.checkbox
                                [ checked (List.member collectible.id state.createRewardCollectibleIds)
                                , onCheck (\_ -> ToggleCreateRewardCollectible collectible.id)
                                , testId ("create-reward-collectible-" ++ collectible.id)
                                ]
                                (collectible.name ++ " · " ++ collectibleKindLabel collectible.kind)
                        )
                        available
                    )
            ]

    else
        text ""


schemaFromFields : List SchemaFieldDraft -> String
schemaFromFields fields =
    let
        named =
            List.filter (\field -> String.trim field.name /= "") fields
    in
    if List.isEmpty named then
        "{\"kind\":\"freeform\"}"

    else
        Encode.encode 0
            (Encode.object
                [ ( "kind", Encode.string "object" )
                , ( "fields", Encode.list encodeSchemaField named )
                ]
            )


encodeSchemaField : SchemaFieldDraft -> Encode.Value
encodeSchemaField field =
    Encode.object
        [ ( "name", Encode.string (String.trim field.name) )
        , ( "presence"
          , Encode.string
                (if field.required then
                    "required"

                 else
                    "may_omit"
                )
          )
        , ( "schema", encodeFieldSchema field )
        ]


encodeFieldSchema : SchemaFieldDraft -> Encode.Value
encodeFieldSchema field =
    case field.kind of
        "enum" ->
            Encode.object
                [ ( "kind", Encode.string "enum" )
                , ( "values", Encode.list Encode.string (enumValueList field.enumValues) )
                ]

        "array" ->
            Encode.object
                [ ( "kind", Encode.string "array" )
                , ( "item", Encode.object [ ( "kind", Encode.string field.itemKind ) ] )
                ]

        other ->
            Encode.object [ ( "kind", Encode.string other ) ]


enumValueList : String -> List String
enumValueList raw =
    raw
        |> String.split ","
        |> List.map String.trim
        |> List.filter (\value -> value /= "")


schemaFieldKinds : List String
schemaFieldKinds =
    [ "string", "integer", "decimal_string", "enum", "array", "freeform" ]


schemaItemKinds : List String
schemaItemKinds =
    [ "string", "integer", "decimal_string" ]


schemaDesignerView : LoggedInModel -> Html Msg
schemaDesignerView state =
    div [ Html.Attributes.class "space-y-3 rounded-md border border-slate-200 bg-slate-50 p-4" ]
        [ Ui.label_ "Response schema designer"
        , p [ Html.Attributes.class "text-xs text-slate-600" ]
            [ text "Add fields to build an object schema without writing JSON. Pick a type per field — enum and array prompt for their values. With no fields the schema is freeform." ]
        , div [ Html.Attributes.class "space-y-2" ]
            (List.indexedMap schemaFieldRow state.createSchemaFields)
        , Ui.secondaryButton [ type_ "button", onClick AddSchemaFieldClicked, testId "schema-add-field" ] "Add field"
        ]


schemaFieldRow : Int -> SchemaFieldDraft -> Html Msg
schemaFieldRow index field =
    div [ Html.Attributes.class "space-y-2 rounded-md border border-slate-200 bg-white p-3" ]
        [ div [ Html.Attributes.class "flex flex-col gap-2 sm:flex-row sm:items-end" ]
            [ div [ Html.Attributes.class "w-full sm:flex-1" ]
                [ Ui.fieldLabel "Field name"
                    [ Ui.textInput
                        [ type_ "text"
                        , placeholder "summary"
                        , value field.name
                        , onInput (SchemaFieldNameChanged index)
                        , testId "schema-field-name"
                        ]
                    ]
                ]
            , div [ Html.Attributes.class "w-full sm:w-auto" ]
                [ Ui.fieldLabel "Type"
                    [ select
                        [ Html.Attributes.class Ui.fieldClass
                        , value field.kind
                        , onInput (SchemaFieldKindChanged index)
                        , testId "schema-field-kind"
                        ]
                        (List.map (schemaKindOption field.kind) schemaFieldKinds)
                    ]
                ]
            , label [ Html.Attributes.class "flex min-h-[44px] w-full items-center gap-2 text-sm text-slate-700 sm:w-auto" ]
                [ Html.input
                    [ type_ "checkbox"
                    , Html.Attributes.class Ui.checkboxClass
                    , checked field.required
                    , onCheck (SchemaFieldRequiredChanged index)
                    , testId "schema-field-required"
                    ]
                    []
                , text "Required"
                ]
            , Ui.secondaryButton
                [ type_ "button", onClick (RemoveSchemaFieldClicked index), testId "schema-field-remove", Html.Attributes.class "w-full sm:w-auto" ]
                "Remove"
            ]
        , schemaFieldDetail index field
        ]


schemaFieldDetail : Int -> SchemaFieldDraft -> Html Msg
schemaFieldDetail index field =
    case field.kind of
        "enum" ->
            div [ Html.Attributes.class "w-full" ]
                [ Ui.fieldLabel "Allowed values (comma-separated)"
                    [ Ui.textInput
                        [ type_ "text"
                        , placeholder "low, medium, high"
                        , value field.enumValues
                        , onInput (SchemaFieldEnumValuesChanged index)
                        , testId "schema-field-enum-values"
                        ]
                    ]
                ]

        "array" ->
            div [ Html.Attributes.class "w-full sm:w-auto" ]
                [ Ui.fieldLabel "Item type"
                    [ select
                        [ Html.Attributes.class Ui.fieldClass
                        , value field.itemKind
                        , onInput (SchemaFieldItemKindChanged index)
                        , testId "schema-field-item-kind"
                        ]
                        (List.map (schemaKindOption field.itemKind) schemaItemKinds)
                    ]
                ]

        _ ->
            text ""


schemaKindOption : String -> String -> Html Msg
schemaKindOption selectedKind kind =
    option [ value kind, selected (kind == selectedKind) ] [ text kind ]


allTaskTypes : List String
allTaskTypes =
    [ "general", "code_review", "security_review", "product_review", "ui_ux_review", "qa_testing" ]


taskTypeLabel : String -> String
taskTypeLabel tag =
    case tag of
        "code_review" ->
            "Code review"

        "security_review" ->
            "Security review"

        "product_review" ->
            "Product review"

        "ui_ux_review" ->
            "UI/UX review"

        "qa_testing" ->
            "QA testing"

        _ ->
            "General"


taskTypeSelect : String -> Html Msg
taskTypeSelect selectedType =
    select
        [ Html.Attributes.class Ui.fieldClass
        , value selectedType
        , onInput CreateTaskTypeChanged
        , testId "create-task-type"
        ]
        (List.map (taskTypeOption selectedType) allTaskTypes)


taskTypeOption : String -> String -> Html Msg
taskTypeOption selectedType tag =
    let
        optionLabel =
            if tag == "general" then
                "Freeform (no template)"

            else
                taskTypeLabel tag
    in
    option [ value tag, selected (selectedType == tag) ] [ text optionLabel ]


taskTemplate : String -> Maybe { description : String, schema : String }
taskTemplate taskType =
    case taskType of
        "code_review" ->
            Just
                { description = "Review the linked pull request. Identify correctness, design, and style issues, then give an overall verdict."
                , schema = "{\"kind\":\"object\",\"fields\":[{\"name\":\"summary\",\"presence\":\"required\",\"schema\":{\"kind\":\"string\"}},{\"name\":\"issues\",\"presence\":\"required\",\"schema\":{\"kind\":\"array\",\"item\":{\"kind\":\"string\"}}},{\"name\":\"verdict\",\"presence\":\"required\",\"schema\":{\"kind\":\"enum\",\"values\":[\"approve\",\"request_changes\",\"comment\"]}}]}"
                }

        "security_review" ->
            Just
                { description = "Perform a security review of the linked code. List vulnerabilities with remediation and an overall severity."
                , schema = "{\"kind\":\"object\",\"fields\":[{\"name\":\"summary\",\"presence\":\"required\",\"schema\":{\"kind\":\"string\"}},{\"name\":\"findings\",\"presence\":\"required\",\"schema\":{\"kind\":\"array\",\"item\":{\"kind\":\"string\"}}},{\"name\":\"severity\",\"presence\":\"required\",\"schema\":{\"kind\":\"enum\",\"values\":[\"none\",\"low\",\"medium\",\"high\",\"critical\"]}}]}"
                }

        "product_review" ->
            Just
                { description = "Review the linked product or feature. Assess clarity, value, and gaps, then recommend next steps."
                , schema = "{\"kind\":\"object\",\"fields\":[{\"name\":\"summary\",\"presence\":\"required\",\"schema\":{\"kind\":\"string\"}},{\"name\":\"strengths\",\"presence\":\"required\",\"schema\":{\"kind\":\"array\",\"item\":{\"kind\":\"string\"}}},{\"name\":\"recommendations\",\"presence\":\"required\",\"schema\":{\"kind\":\"array\",\"item\":{\"kind\":\"string\"}}}]}"
                }

        "ui_ux_review" ->
            Just
                { description = "Review the linked UI/UX. Check usability, accessibility, and visual consistency, then list issues."
                , schema = "{\"kind\":\"object\",\"fields\":[{\"name\":\"summary\",\"presence\":\"required\",\"schema\":{\"kind\":\"string\"}},{\"name\":\"issues\",\"presence\":\"required\",\"schema\":{\"kind\":\"array\",\"item\":{\"kind\":\"string\"}}},{\"name\":\"accessibility\",\"presence\":\"required\",\"schema\":{\"kind\":\"enum\",\"values\":[\"pass\",\"fail\"]}}]}"
                }

        "qa_testing" ->
            Just
                { description = "Test the linked build against its requirements. Report the cases you ran and the overall result."
                , schema = "{\"kind\":\"object\",\"fields\":[{\"name\":\"summary\",\"presence\":\"required\",\"schema\":{\"kind\":\"string\"}},{\"name\":\"cases\",\"presence\":\"required\",\"schema\":{\"kind\":\"array\",\"item\":{\"kind\":\"string\"}}},{\"name\":\"result\",\"presence\":\"required\",\"schema\":{\"kind\":\"enum\",\"values\":[\"pass\",\"fail\"]}}]}"
                }

        _ ->
            Nothing


visibilityButton : String -> String -> Html Msg
visibilityButton selected tag =
    Ui.chooserButton (selected == tag)
        (CreateVisibilityChanged tag)
        ("create-visibility-" ++ tag)
        (visibilityLabel tag)


allAssigneeScopes : List Task.TaskAssigneeScope
allAssigneeScopes =
    [ Task.TaskAssigneeScopeUser, Task.TaskAssigneeScopeOrganizationTeam, Task.TaskAssigneeScopeTeam ]


assigneeScopeButton : Task.TaskAssigneeScope -> Task.TaskAssigneeScope -> Html Msg
assigneeScopeButton selected scope =
    Ui.chooserButton (selected == scope)
        (CreateAssigneeScopeChosen scope)
        ("create-assignee-" ++ assigneeScopeTag scope)
        (assigneeScopeLabel scope)


visibilityScopeField : LoggedInModel -> Html Msg
visibilityScopeField state =
    if state.createVisibility == visibilityUserTag then
        Ui.fieldLabel "Share with user"
            [ userPicker "create-scope-user" state.createScopeUserId state.userDirectoryQuery CreateScopeUserIdChanged "Choose user" state.userDirectory state.userDirectoryOffset ]

    else if state.createVisibility == visibilityTeamTag then
        Ui.fieldLabel "Share with team"
            [ teamPicker "create-scope-team" state.createScopeTeamId state.standaloneTeamQuery CreateScopeTeamIdChanged StandaloneTeamQueryChanged SearchStandaloneTeamsClicked PreviousStandaloneTeamsPageClicked NextStandaloneTeamsPageClicked "Choose team" state.standaloneTeams state.standaloneTeamOffset ]

    else if state.createVisibility == visibilityOrganizationTag then
        Ui.fieldLabel "Share with organization"
            [ organizationPicker "create-scope-organization" state.createScopeOrganizationId state.organizationQuery CreateScopeOrganizationIdChanged OrganizationQueryChanged SearchOrganizationsClicked PreviousOrganizationsPageClicked NextOrganizationsPageClicked "Choose organization" state.organizations state.organizationOffset ]

    else
        text ""


participationButton : String -> Task.TaskParticipationPolicy -> Html Msg
participationButton selectedPolicy policy =
    Ui.chooserButton (selectedPolicy == participationPolicyTag policy)
        (CreateParticipationChanged (participationPolicyTag policy))
        ("create-participation-" ++ participationPolicyTag policy)
        (participationPolicyLabel policy)


ledgerView : List Ledger.LedgerEntryResponse -> Int -> Html Msg
ledgerView entries offset =
    Ui.card
        [ Ui.sectionTitle "Ledger"
        , table [ Html.Attributes.class "w-full text-left text-sm" ]
            [ thead []
                [ tr [ Html.Attributes.class "text-slate-500" ]
                    [ th [ Html.Attributes.class "pb-2" ] [ text "Entry" ]
                    , th [ Html.Attributes.class "pb-2 text-right" ] [ text "Amount" ]
                    ]
                ]
            , tbody [ testId "ledger" ] (List.map ledgerRow entries)
            ]
        , paginationControls "ledger-page" PreviousLedgerPageClicked NextLedgerPageClicked offset (List.length entries)
        ]


ledgerRow : Ledger.LedgerEntryResponse -> Html Msg
ledgerRow entry =
    let
        amountClass =
            if entry.amount < 0 then
                "py-2 text-right tabular-nums text-red-700"

            else
                "py-2 text-right tabular-nums text-green-700"

        amountText =
            if entry.amount > 0 then
                "+" ++ String.fromInt entry.amount

            else
                String.fromInt entry.amount
    in
    tr [ Html.Attributes.class "border-t border-slate-100", testId "ledger-entry" ]
        [ td [ Html.Attributes.class "py-2" ] [ text (kindLabel entry.kind) ]
        , td [ Html.Attributes.class amountClass ] [ text amountText ]
        ]


fundingView : LoggedInModel -> Html Msg
fundingView state =
    form [ Html.Attributes.class "space-y-4 rounded-lg border border-slate-200 bg-white p-6 shadow-sm", onSubmit FundClicked ]
        [ Ui.sectionTitle "Fund a task"
        , p [ Html.Attributes.class "text-sm text-slate-600" ] [ text "Only draft tasks can be funded. Unpublish an open task first to change its funding." ]

        -- The backend only accepts funding for draft tasks, so offering the
        -- rest of the list here just produces avoidable rejections.
        , taskPicker "fund-task-id" state.fundTaskId FundTaskIdChanged (List.filter (\item -> item.state == Task.TaskStateDraft) state.tasks)
        , Ui.textInput [ type_ "number", placeholder "Amount in credits", value state.fundAmount, onInput FundAmountChanged, testId "fund-amount" ]
        , organizationPicker "fund-organization" state.fundOrganizationId state.organizationQuery FundOrganizationIdChanged OrganizationQueryChanged SearchOrganizationsClicked PreviousOrganizationsPageClicked NextOrganizationsPageClicked "Personal balance" state.organizations state.organizationOffset
        , Ui.primaryButton [ type_ "submit", disabled (state.fundTaskId == ""), testId "fund" ] "Fund task"
        , maybeNote state.fundMessage "fund-message"
        ]


taskPicker : String -> String -> (String -> Msg) -> List Task.TaskListItemResponse -> Html Msg
taskPicker identifier selectedTaskId change tasks =
    select
        [ Html.Attributes.class Ui.fieldClass
        , value selectedTaskId
        , onInput change
        , testId identifier
        ]
        (blankOption "Select task" :: List.map (taskOption selectedTaskId) tasks)


taskOption : String -> Task.TaskListItemResponse -> Html Msg
taskOption selectedTaskId item =
    option [ value item.id, selected (selectedTaskId == item.id) ]
        [ text (item.title ++ " · " ++ taskStateLabel item.state ++ " · " ++ rewardLabel item.rewardKind item.rewardCreditAmount item.rewardCollectibleCount) ]


orgMemberPicker : String -> String -> (String -> Msg) -> List Organization.OrganizationMemberResponse -> Html Msg
orgMemberPicker identifier selectedUserId change members =
    select
        [ Html.Attributes.class Ui.fieldClass
        , value selectedUserId
        , onInput change
        , testId identifier
        ]
        (blankOption "Select member" :: List.map (orgMemberOption selectedUserId) members)


orgMemberOption : String -> Organization.OrganizationMemberResponse -> Html Msg
orgMemberOption selectedUserId member =
    option [ value member.userID, selected (selectedUserId == member.userID) ] [ text member.userID ]


orgCollectibleAwardRow : String -> Collectible.CollectibleResponse -> Html Msg
orgCollectibleAwardRow recipientId collectible =
    div [ Html.Attributes.class "flex flex-wrap items-center justify-between gap-2 py-2", testId "org-collectible-award-row" ]
        [ div [ Html.Attributes.class "flex min-w-0 flex-wrap items-center gap-2" ]
            [ Sprites.pixel collectible.art 5
            , span [ Html.Attributes.class "text-sm font-medium break-words" ] [ text collectible.name ]
            ]
        , Ui.secondaryButton [ type_ "button", onClick (AwardOrgCollectibleClicked collectible.id), disabled (recipientId == ""), testId "award-org-collectible" ] "Award to member"
        ]


{-| The Tasks hub: the single destination for everything task-related.
Merges what used to be four separate nav-reachable pages/menu items (Tasks,
Discovery, the Work menu's Submissions and Series) into one page, plus a
"+ New task" button in place of a dedicated top-level nav entry (Create Task
itself stays its own route/page — see `createTaskView` — this is just a
shortcut into it). My tasks and Discover public tasks stay always-expanded
(both are primary, equally-common things to do here); My submissions and
Series collapse into disclosures since they're reached far less often (per
Playwright-usage counts: the old `nav-submissions` link was never clicked by
any spec, and `nav-series-list` only once).
-}
tasksView : String -> LoggedInModel -> Html Msg
tasksView origin state =
    let
        visibleTasks =
            filterTasksByQuery state.taskListQuery state.tasks
    in
    Ui.card
        ([ a [ href ("#" ++ pageToPath CreateTaskPage), Html.Attributes.class Ui.primaryButtonClass, testId "new-task-button" ] [ text "+ New task" ]
         , Ui.sectionTitle "My tasks"
         , Ui.disclosure "tasks-filters"
            False
            "Filters"
            [ Ui.label_ "Filter by state (select any number)"
            , div [ Html.Attributes.class "flex flex-wrap gap-2", testId "task-filter" ] (List.map (taskFilterChip state.taskStateFilter) taskStateFilterOptions)
            , Ui.fieldLabel "Search loaded tasks"
                [ Ui.textInput [ type_ "search", placeholder "Task title or ID", value state.taskListQuery, onInput TaskListQueryChanged, testId "tasks-query" ] ]
            , taskTypeFilterSelect "tasks-type" state.taskListTypeFilter TaskListTypeFilterChanged
            , taskSortSelect "tasks-sort" state.taskListSort TaskListSortChanged
            ]
         , paginationControls "tasks-page" PreviousTasksPageClicked NextTasksPageClicked state.taskListOffset (List.length state.tasks)
         , tasksList state.subjectId visibleTasks
         ]
            ++ discoverySection state
            ++ [ Ui.disclosure "tasks-submissions" False "My submissions" (userSubmissionsSection state)
               , Ui.disclosure "tasks-series" False "Series" (seriesSection state)
               ]
        )


taskStateFilterOptions : List ( String, String )
taskStateFilterOptions =
    [ ( "open", "Open" )
    , ( "draft", "Draft" )
    , ( "closed", "Closed" )
    , ( "expired", "Expired" )
    , ( "cancelled", "Cancelled" )
    ]


{-| Multiple chips can be active at once (e.g. Open + Closed), unlike the
single-select buttons this replaced - no chip selected means no filter
("All"), rather than "All" being its own selectable option.
-}
taskFilterChip : List String -> ( String, String ) -> Html Msg
taskFilterChip selected ( tag, labelText ) =
    Ui.chooserButton (List.member tag selected) (TaskStateFilterToggled tag) ("task-filter-" ++ tag) labelText


taskTypeFilterSelect : String -> String -> (String -> Msg) -> Html Msg
taskTypeFilterSelect identifier selectedType change =
    Ui.fieldLabel "Task type"
        [ select [ Html.Attributes.class Ui.fieldClass, value selectedType, onInput change, testId identifier ]
            (List.map (stringOption selectedType) taskTypeFilterOptions)
        ]


taskTypeFilterOptions : List ( String, String )
taskTypeFilterOptions =
    [ ( "", "All types" )
    , ( "general", "General" )
    , ( "code_review", "Code review" )
    , ( "security_review", "Security review" )
    , ( "product_review", "Product review" )
    , ( "ui_ux_review", "UI/UX review" )
    , ( "qa_testing", "QA testing" )
    ]


taskSortSelect : String -> String -> (String -> Msg) -> Html Msg
taskSortSelect identifier selectedSort change =
    Ui.fieldLabel "Sort"
        [ select [ Html.Attributes.class Ui.fieldClass, value selectedSort, onInput change, testId identifier ]
            (List.map (stringOption selectedSort) taskSortOptions)
        ]


taskSortOptions : List ( String, String )
taskSortOptions =
    [ ( "newest", "Newest" )
    , ( "oldest", "Oldest" )
    , ( "title_asc", "Title A-Z" )
    , ( "title_desc", "Title Z-A" )
    , ( "reward_desc", "Reward high-low" )
    , ( "reward_asc", "Reward low-high" )
    ]


stringOption : String -> ( String, String ) -> Html Msg
stringOption selectedValue ( optionValue, labelText ) =
    option [ value optionValue, selected (selectedValue == optionValue) ] [ text labelText ]


blankOption : String -> Html Msg
blankOption labelText =
    option [ Html.Attributes.attribute "value" "" ] [ text labelText ]


organizationPicker : String -> String -> String -> (String -> Msg) -> (String -> Msg) -> Msg -> Msg -> Msg -> String -> List Organization.OrganizationResponse -> Int -> Html Msg
organizationPicker identifier selectedOrganizationId query change queryChange search previous next blankLabel organizations offset =
    div [ Html.Attributes.class "space-y-2" ]
        [ selectorSearchControls identifier "Search organizations" query queryChange search previous next offset (List.length organizations)
        , select
            [ Html.Attributes.class Ui.fieldClass, value selectedOrganizationId, onInput change, testId identifier ]
            (blankOption blankLabel
                :: List.map (\organization -> option [ value organization.id, selected (selectedOrganizationId == organization.id) ] [ text organization.name ]) organizations
            )
        ]


userPicker : String -> String -> String -> (String -> Msg) -> String -> List UserDirectoryEntry -> Int -> Html Msg
userPicker identifier selectedUserId query change blankLabel users offset =
    div [ Html.Attributes.class "space-y-2" ]
        [ selectorSearchControls identifier "Search users" query UserDirectoryQueryChanged SearchUserDirectoryClicked PreviousUserDirectoryPageClicked NextUserDirectoryPageClicked offset (List.length users)
        , select
            [ Html.Attributes.class Ui.fieldClass, value selectedUserId, onInput change, testId identifier ]
            (blankOption blankLabel
                :: List.map (\user -> option [ value user.id, selected (selectedUserId == user.id) ] [ text user.email ]) users
            )
        ]


teamPicker : String -> String -> String -> (String -> Msg) -> (String -> Msg) -> Msg -> Msg -> Msg -> String -> List Team.TeamResponse -> Int -> Html Msg
teamPicker identifier selectedTeamId query change queryChange search previous next blankLabel teams offset =
    div [ Html.Attributes.class "space-y-2" ]
        [ selectorSearchControls identifier "Search teams" query queryChange search previous next offset (List.length teams)
        , select
            [ Html.Attributes.class Ui.fieldClass, value selectedTeamId, onInput change, testId identifier ]
            (blankOption blankLabel
                :: List.map (\team -> option [ value team.id, selected (selectedTeamId == team.id) ] [ text team.name ]) teams
            )
        ]


selectorSearchControls : String -> String -> String -> (String -> Msg) -> Msg -> Msg -> Msg -> Int -> Int -> Html Msg
selectorSearchControls identifier placeholderText query queryChange search previous next offset shownCount =
    div [ Html.Attributes.class "space-y-2" ]
        [ div [ Html.Attributes.class "flex flex-wrap gap-2" ]
            [ Ui.textInput [ type_ "search", placeholder placeholderText, value query, onInput queryChange, testId (identifier ++ "-query") ]
            , Ui.secondaryButton [ type_ "button", onClick search, testId (identifier ++ "-search") ] "Search"
            ]
        , div [ Html.Attributes.class "flex flex-wrap items-center gap-2 text-xs text-slate-500" ]
            [ Ui.secondaryButton [ type_ "button", disabled (offset == 0), onClick previous, testId (identifier ++ "-previous") ] "Previous"
            , span [ testId (identifier ++ "-offset") ] [ text ("Offset " ++ String.fromInt offset) ]
            , Ui.secondaryButton [ type_ "button", disabled (shownCount < pageSize), onClick next, testId (identifier ++ "-next") ] "Next"
            ]
        ]


activeAssigneeSuffix : Task.TaskListItemResponse -> String
activeAssigneeSuffix item =
    if item.activeAssigneeID == "" then
        ""

    else
        " · reserved by " ++ item.activeAssigneeID


filterTasksByQuery : String -> List Task.TaskListItemResponse -> List Task.TaskListItemResponse
filterTasksByQuery query tasks =
    let
        normalized =
            String.toLower (String.trim query)
    in
    if normalized == "" then
        tasks

    else
        List.filter
            (\item ->
                String.contains normalized (String.toLower item.title)
                    || String.contains normalized (String.toLower item.id)
            )
            tasks


tasksList : String -> List Task.TaskListItemResponse -> Html Msg
tasksList subjectId tasks =
    if List.isEmpty tasks then
        p [ Html.Attributes.class "text-sm text-slate-500", testId "tasks-empty" ] [ text "No tasks yet." ]

    else
        div [ Html.Attributes.class "divide-y divide-slate-100", testId "tasks" ] (List.map (taskRow subjectId) tasks)


{-| A task is "mine" if I created it, or if I'm the active assignee (reserved
or submitted) on it - either way it's something I'm personally involved in,
not just something I happen to be looking at.
-}
isMyTask : String -> Task.TaskListItemResponse -> Bool
isMyTask subjectId item =
    item.createdBy == subjectId || (item.activeAssigneeKind == "user" && item.activeAssigneeID == subjectId)


taskRow : String -> Task.TaskListItemResponse -> Html Msg
taskRow subjectId item =
    let
        mine =
            isMyTask subjectId item
    in
    div
        [ Html.Attributes.class
            ("flex items-center justify-between gap-3 py-2"
                ++ (if mine then
                        " border-l-2 border-blue-300 pl-3 -ml-3.5"

                    else
                        ""
                   )
            )
        , testId "task-row"
        ]
        [ div [ Html.Attributes.class "min-w-0" ]
            [ p [ Html.Attributes.class "flex flex-wrap items-center gap-2 font-medium break-words" ]
                (text item.title
                    :: (if mine then
                            [ span [ Html.Attributes.class "rounded border border-blue-200 bg-blue-50 px-1.5 py-0.5 text-[10px] font-semibold tracking-wide text-blue-700", testId "mine-flag" ] [ text "MINE" ] ]

                        else
                            []
                       )
                )
            , p [ Html.Attributes.class "flex flex-wrap items-center gap-1.5 text-xs text-slate-500 break-words" ]
                [ taskStateBadge item.state
                , taskRewardBadge item.rewardKind item.rewardCreditAmount item.rewardCollectibleCount
                , text (activeAssigneeSuffix item)
                ]
            ]
        , div [ Html.Attributes.class "flex shrink-0 gap-2" ]
            -- Funding lives only on the task's own detail page (its "Fund
            -- this task" panel), not as a shortcut from the list row.
            [ a [ href ("#/tasks/" ++ item.id), Html.Attributes.class Ui.secondaryButtonClass, testId "view-task" ] [ text "View" ] ]
        ]


agentsView : String -> LoggedInModel -> Html Msg
agentsView origin state =
    Ui.card
        [ Ui.sectionTitle "Agent setup"
        , p [ Html.Attributes.class "text-sm text-slate-600" ] [ text "Create a scoped credential for a local MCP agent." ]
        , form [ Html.Attributes.class "mt-3 space-y-3", onSubmit CreateAgentClicked ]
            [ Ui.textInput [ type_ "text", placeholder "Agent label", value state.agentLabel, onInput AgentLabelChanged, testId "agent-label" ]
            , div [ Html.Attributes.class "space-y-1" ] (List.map (scopeCheckbox state.agentScopes) allScopes)
            , Ui.fieldLabel "Expires in (hours, blank for never)" [ Ui.textInput [ type_ "number", placeholder "never", value state.agentExpiresHours, onInput AgentExpiresHoursChanged, testId "agent-expires-hours" ] ]
            , Ui.primaryButton [ type_ "submit", testId "create-agent" ] "Create credential"
            , maybeNote state.agentMessage "agent-message"
            ]
        , newCredentialView origin state.newCredential
        , credentialsList state.credentials
        ]


scopeCheckbox : List Agent.AgentScope -> Agent.AgentScope -> Html Msg
scopeCheckbox selected scope =
    label [ Html.Attributes.class "flex min-h-[44px] items-center gap-2 text-sm" ]
        [ Html.input
            [ type_ "checkbox"
            , Html.Attributes.class Ui.checkboxClass
            , checked (List.member scope selected)
            , onCheck (\_ -> ToggleScope scope)
            , testId ("scope-" ++ scopeTag scope)
            ]
            []
        , span [] [ text (scopeLabel scope ++ " (" ++ scopeTag scope ++ ")") ]
        ]


newCredentialView : String -> Maybe Agent.AgentCredentialCreatedResponse -> Html Msg
newCredentialView origin created =
    case created of
        Just credential ->
            div [ Html.Attributes.class "mt-4 space-y-3 rounded-md bg-slate-50 p-4" ]
                [ Ui.label_ "New agent token (shown once)"
                , Ui.codeBlock [ testId "agent-secret" ] credential.secret
                , Ui.label_ "MCP client configuration"
                , Ui.codeBlock [ testId "mcp-config" ] (mcpConfig origin credential.secret)
                ]

        Nothing ->
            text ""


credentialsList : List Agent.AgentCredentialResponse -> Html Msg
credentialsList credentials =
    if List.isEmpty credentials then
        text ""

    else
        div [ Html.Attributes.class "mt-4 divide-y divide-slate-100", testId "credentials" ] (List.map credentialRow credentials)


credentialRow : Agent.AgentCredentialResponse -> Html Msg
credentialRow credential =
    div [ Html.Attributes.class "flex items-center justify-between py-2", testId "credential-row" ]
        [ div []
            [ p [ Html.Attributes.class "font-medium" ] [ text credential.label ]
            , p [ Html.Attributes.class "text-xs text-slate-500" ] [ text (credentialStateLabel credential.state ++ expiryNote credential.expiresAt ++ " · " ++ String.join ", " (List.map scopeLabel credential.scopes)) ]
            ]
        , revokeButton credential
        ]


expiryNote : String -> String
expiryNote expiresAt =
    if expiresAt == "" then
        ""

    else
        " · expires " ++ expiresAt


revokeButton : Agent.AgentCredentialResponse -> Html Msg
revokeButton credential =
    case credential.state of
        Agent.AgentCredentialStateActive ->
            Ui.secondaryButton [ onClick (RevokeClicked credential.id), testId "revoke-credential" ] "Revoke"

        Agent.AgentCredentialStateRevoked ->
            span [ Html.Attributes.class "text-xs text-slate-600" ] [ text "revoked" ]



-- Collectibles panel


collectiblesView : LoggedInModel -> Html Msg
collectiblesView state =
    Ui.card
        [ p [ Html.Attributes.class "text-sm text-slate-600" ] [ text "Mint your own collectibles, award default collectibles to users, teams, or organizations, and trade collectibles to other users." ]
        , Ui.disclosure "collectibles-mint" False "Mint a collectible" [ mintForm state ]
        , Ui.disclosure "collectibles-award-task" False "Award a collectible to a task" [ awardForm state ]
        , if state.isAdmin then
            Ui.disclosure "award-default-section" False "Admin: award a default collectible" (awardRecipientControl state)

          else
            text ""
        , catalogGallery state
        , collectiblesList state
        ]


awardRecipientControl : LoggedInModel -> List (Html Msg)
awardRecipientControl state =
    [ p [ Html.Attributes.class "text-xs text-slate-600", testId "award-admin-note" ] [ text "Awarding default collectibles requires a platform administrator (enabled in the demo)." ]
    , div [ Html.Attributes.class "flex flex-wrap gap-2" ]
        [ Ui.chooserButton (state.awardRecipientKind == "user") (AwardRecipientKindChanged "user") "award-kind-user" "User"
        , Ui.chooserButton (state.awardRecipientKind == "team") (AwardRecipientKindChanged "team") "award-kind-team" "Team"
        , Ui.chooserButton (state.awardRecipientKind == "organization") (AwardRecipientKindChanged "organization") "award-kind-organization" "Organization"
        ]
    , awardRecipientPicker state
    , maybeNote state.awardDefaultMessage "award-default-message"
    ]


awardRecipientPicker : LoggedInModel -> Html Msg
awardRecipientPicker state =
    if state.awardRecipientKind == "organization" then
        organizationPicker "award-recipient-id" state.awardRecipientId state.organizationQuery AwardRecipientIdChanged OrganizationQueryChanged SearchOrganizationsClicked PreviousOrganizationsPageClicked NextOrganizationsPageClicked "Choose organization" state.organizations state.organizationOffset

    else if state.awardRecipientKind == "team" then
        teamPicker "award-recipient-id" state.awardRecipientId state.standaloneTeamQuery AwardRecipientIdChanged StandaloneTeamQueryChanged SearchStandaloneTeamsClicked PreviousStandaloneTeamsPageClicked NextStandaloneTeamsPageClicked "Choose team" state.standaloneTeams state.standaloneTeamOffset

    else
        userPicker "award-recipient-id" state.awardRecipientId state.userDirectoryQuery AwardRecipientIdChanged "Choose user" state.userDirectory state.userDirectoryOffset


catalogGallery : LoggedInModel -> Html Msg
catalogGallery state =
    div [ Html.Attributes.class "mt-3 grid grid-cols-2 gap-3 sm:grid-cols-3", testId "catalog" ]
        (List.map (catalogEntry state.isAdmin state.awardRecipientId) state.collectibleCatalog)


catalogEntry : Bool -> String -> Collectible.CollectibleCatalogEntry -> Html Msg
catalogEntry isAdmin recipientId entry =
    div [ Html.Attributes.class "flex flex-col items-center gap-1 rounded-md border border-slate-200 p-2 text-center", testId "catalog-entry" ]
        [ Sprites.pixel entry.art 6
        , span [ Html.Attributes.class "text-xs font-medium break-words" ] [ text entry.name ]
        , Ui.badge (collectibleKindLabel entry.kind)
        , if isAdmin then
            Ui.secondaryButton [ type_ "button", onClick (AwardDefaultClicked entry.slug), disabled (String.trim recipientId == ""), testId "catalog-award" ] "Award"

          else
            text ""
        ]


mintForm : LoggedInModel -> Html Msg
mintForm state =
    form [ Html.Attributes.class "mt-3 space-y-3", onSubmit MintClicked ]
        [ Ui.textInput [ type_ "text", placeholder "Collectible name", value state.collectibleName, onInput CollectibleNameChanged, testId "collectible-name" ]
        , Ui.label_ "Kind"
        , div [ Html.Attributes.class "flex gap-2" ] (List.map (kindButton state.collectibleKind) allKinds)
        , Ui.label_ "Transfer policy"
        , div [ Html.Attributes.class "flex flex-wrap gap-2" ] (List.map (policyButton state.collectiblePolicy) allPolicies)
        , Ui.primaryButton [ type_ "submit", testId "mint-collectible" ] "Mint collectible"
        , maybeNote state.collectibleMessage "collectible-message"
        ]


kindButton : Collectible.CollectibleKind -> Collectible.CollectibleKind -> Html Msg
kindButton selected kind =
    Ui.chooserButton (selected == kind)
        (CollectibleKindChosen kind)
        ("collectible-kind-" ++ collectibleKindTag kind)
        (collectibleKindLabel kind)


policyButton : Collectible.CollectibleTransferPolicy -> Collectible.CollectibleTransferPolicy -> Html Msg
policyButton selected policy =
    Ui.chooserButton (selected == policy)
        (CollectiblePolicyChosen policy)
        ("collectible-policy-" ++ collectiblePolicyTag policy)
        (collectiblePolicyLabel policy)


awardForm : LoggedInModel -> Html Msg
awardForm state =
    div [ Html.Attributes.class "mt-4 space-y-3" ]
        -- Collectible reward funding is draft-only on the backend, like
        -- credit funding.
        [ taskPicker "award-task-id" state.awardTaskId AwardTaskIdChanged (List.filter (\item -> item.state == Task.TaskStateDraft) state.tasks)
        , p [ Html.Attributes.class "text-xs text-slate-500" ] [ text "Choose a draft task here, then press Award next to a collectible below." ]
        , maybeNote state.awardMessage "award-message"
        ]


collectiblesList : LoggedInModel -> Html Msg
collectiblesList state =
    if List.isEmpty state.collectibles then
        p [ Html.Attributes.class "mt-4 text-sm text-slate-500", testId "collectibles-empty" ] [ text "No collectibles yet." ]

    else
        div [ Html.Attributes.class "mt-4 divide-y divide-slate-100", testId "collectibles" ] (List.map (collectibleRow state.awardTaskId) state.collectibles)


collectibleRow : String -> Collectible.CollectibleResponse -> Html Msg
collectibleRow awardTaskId collectible =
    div [ Html.Attributes.class "flex flex-wrap items-center justify-between gap-2 py-2", testId "collectible-row" ]
        [ div [ Html.Attributes.class "flex min-w-0 flex-wrap items-center gap-2" ]
            [ Sprites.pixel collectible.art 5
            , a [ href ("#/collectibles/" ++ collectible.id), Html.Attributes.class "font-medium underline break-words", testId "collectible-link" ] [ text collectible.name ]
            , collectibleStateBadge collectible.state
            , span [ Html.Attributes.class "text-xs text-slate-500" ] [ text (collectibleKindLabel collectible.kind) ]
            ]
        , awardCollectibleButton awardTaskId collectible
        ]


awardCollectibleButton : String -> Collectible.CollectibleResponse -> Html Msg
awardCollectibleButton awardTaskId collectible =
    case collectible.state of
        Collectible.CollectibleStateMinted ->
            Ui.secondaryButton [ type_ "button", onClick (AwardClicked collectible.id), disabled (awardTaskId == ""), testId "award-collectible" ] "Award to selected task"

        _ ->
            text ""



-- Discovery page


{-| The Discover-public-tasks section embedded on the Tasks hub (see
`tasksView`) — content only, no outer card (Discovery no longer has its own
route; this is the only place it renders).
-}
discoverySection : LoggedInModel -> List (Html Msg)
discoverySection state =
    let
        visibleTasks =
            filterTasksByQuery state.discoveryQuery state.discoveryTasks
    in
    [ Ui.sectionTitle "Discover public tasks"
    , Ui.disclosure "discovery-filters"
        False
        "Filters"
        [ Ui.checkbox [ checked state.discoveryIncludeReserved, onClick (DiscoveryIncludeReservedChanged (not state.discoveryIncludeReserved)), testId "include-reserved" ] "Include reserved"
        , Ui.fieldLabel "Search loaded discovery"
            [ Ui.textInput [ type_ "search", placeholder "Task title or ID", value state.discoveryQuery, onInput DiscoveryQueryChanged, testId "discovery-query" ] ]
        ]
    , paginationControls "discovery-page" PreviousDiscoveryPageClicked NextDiscoveryPageClicked state.discoveryOffset (List.length state.discoveryTasks)
    , discoveryList state.subjectId visibleTasks
    ]


discoveryList : String -> List Task.TaskListItemResponse -> Html Msg
discoveryList subjectId tasks =
    if List.isEmpty tasks then
        p [ Html.Attributes.class "text-sm text-slate-500", testId "discovery-empty" ] [ text "No public tasks available." ]

    else
        div [ Html.Attributes.class "divide-y divide-slate-100", testId "discovery-tasks" ] (List.map (discoveryRow subjectId) tasks)


discoveryRow : String -> Task.TaskListItemResponse -> Html Msg
discoveryRow subjectId item =
    let
        mine =
            isMyTask subjectId item
    in
    div
        [ Html.Attributes.class
            ("flex items-center justify-between gap-3 py-2"
                ++ (if mine then
                        " border-l-2 border-blue-300 pl-3 -ml-3.5"

                    else
                        ""
                   )
            )
        , testId "discovery-task-row"
        ]
        [ div [ Html.Attributes.class "min-w-0" ]
            [ p [ Html.Attributes.class "flex flex-wrap items-center gap-2 font-medium break-words" ]
                (text item.title
                    :: (if mine then
                            [ span [ Html.Attributes.class "rounded border border-blue-200 bg-blue-50 px-1.5 py-0.5 text-[10px] font-semibold tracking-wide text-blue-700", testId "mine-flag" ] [ text "MINE" ] ]

                        else
                            []
                       )
                )
            , p [ Html.Attributes.class "flex flex-wrap items-center gap-1.5 text-xs text-slate-500 break-words" ]
                [ taskStateBadge item.state
                , taskRewardBadge item.rewardKind item.rewardCreditAmount item.rewardCollectibleCount
                , text ("· " ++ participationPolicyLabel item.participationPolicy ++ activeAssigneeSuffix item)
                ]
            ]
        , div [ Html.Attributes.class "shrink-0" ] [ Ui.secondaryButton [ onClick (DiscoveryViewClicked item.id), testId "discovery-view" ] "View" ]
        ]


-- paginationControls takes the number of items on the current page so Next
-- can be disabled on the last (short) page instead of paging into blank
-- pages that read as misleading empty states. A page of exactly pageSize
-- items keeps Next enabled even when it happens to be the final one - the
-- client cannot tell without a total count the API does not return.
paginationControls : String -> Msg -> Msg -> Int -> Int -> Html Msg
paginationControls identifier previous next offset shownCount =
    div [ Html.Attributes.class "flex flex-wrap items-center gap-2 text-xs text-slate-600", testId identifier ]
        [ Ui.secondaryButton [ type_ "button", disabled (offset == 0), onClick previous, testId (identifier ++ "-previous") ] "Previous"
        , span [ testId (identifier ++ "-offset") ] [ text ("Offset " ++ String.fromInt offset) ]
        , Ui.secondaryButton [ type_ "button", disabled (shownCount < pageSize), onClick next, testId (identifier ++ "-next") ] "Next"
        ]



-- Task detail page


taskDetailPageView : String -> LoggedInModel -> Html Msg
taskDetailPageView origin state =
    let
        isOwner =
            state.detail |> Maybe.map (\detail -> detail.createdBy == state.subjectId) |> Maybe.withDefault False

        canReview =
            state.detail |> Maybe.map (\detail -> detail.reviewerAction == "review") |> Maybe.withDefault False
    in
    div [ Html.Attributes.class "space-y-6" ]
        ([ a [ href "#/tasks", Html.Attributes.class Ui.secondaryButtonClass, testId "detail-back" ] [ text "Back" ]
         , detailCard origin state

         -- Shown to every viewer, not just workers: "who has this reserved"
         -- is useful to the owner too, and the reserve/request-approval
         -- action form only renders for viewers whose server-computed
         -- viewerAction actually allows it (owners' own viewerAction is
         -- never Reserve/RequestApproval), so this needs no isOwner/
         -- canReview gating of its own. Placed right after the task's own
         -- details, above any role-specific controls, since "can I reserve
         -- this / who has it" is usually the first thing a visitor wants to
         -- know — previously buried below owner/reviewer controls entirely
         -- (i.e. an owner had no way to see or act on reservation requests
         -- through the browser at all).
         , reservationCard state
         ]
            ++ (if isOwner then
                    [ ownerControlsCard state, submissionsCard state ]

                else if canReview then
                    [ submissionsCard state ]

                else
                    [ submitCard state, mySubmissionsCard state ]
               )
            ++ [ taskCommentsCard state, moderationReportCard state ]
        )


taskCommentsCard : LoggedInModel -> Html Msg
taskCommentsCard state =
    case state.detail of
        Just detail ->
            Ui.card
                [ Ui.sectionTitle "Discussion"
                , if List.isEmpty state.taskComments then
                    p [ Html.Attributes.class "text-sm text-slate-500", testId "task-comments-empty" ] [ text "No comments yet." ]

                  else
                    div [ Html.Attributes.class "space-y-2", testId "task-comments" ] (List.map taskCommentRow state.taskComments)
                , form [ Html.Attributes.class "space-y-2", onSubmit (AddTaskCommentClicked detail.id) ]
                    [ Ui.textarea_ [ placeholder "Add a comment", value state.taskCommentBody, onInput TaskCommentBodyChanged, testId "task-comment-body" ]
                    , Ui.primaryButton [ type_ "submit", testId "add-task-comment" ] "Comment"
                    , maybeNote state.taskCommentMessage "task-comment-message"
                    ]
                ]

        Nothing ->
            text ""


taskCommentRow : Task.TaskCommentResponse -> Html Msg
taskCommentRow comment =
    div [ Html.Attributes.class "rounded-md border border-slate-200 bg-white p-3", testId "task-comment" ]
        [ a [ href ("#/users/" ++ comment.authorUserID), Html.Attributes.class "text-xs font-medium text-slate-600 underline" ] [ text comment.authorUserID ]
        , p [ Html.Attributes.class "text-sm text-slate-700 break-words" ] [ text comment.body ]
        ]


moderationReportCard : LoggedInModel -> Html Msg
moderationReportCard state =
    case state.detail of
        Just detail ->
            Ui.card
                [ Ui.disclosure "moderation-report-panel"
                    False
                    "Report task"
                    [ div [ Html.Attributes.class "flex flex-wrap gap-2", testId "moderation-reasons" ]
                        (List.map (moderationReasonButton state.moderationReason) moderationReasonOptions)
                    , Ui.textarea_
                        [ placeholder "Describe the issue"
                        , value state.moderationDetails
                        , onInput ModerationDetailsChanged
                        , Html.Attributes.rows 4
                        , testId "moderation-details"
                        ]
                    , Ui.secondaryButton [ type_ "button", onClick (ReportTaskClicked detail.id), testId "report-task" ] "Submit report"
                    , maybeNote state.moderationMessage "moderation-message"
                    ]
                ]

        Nothing ->
            text ""


moderationReasonOptions : List ( Moderation.ModerationReason, String )
moderationReasonOptions =
    [ ( Moderation.ModerationReasonPolicy, "Policy" )
    , ( Moderation.ModerationReasonSpam, "Spam" )
    , ( Moderation.ModerationReasonAbuse, "Abuse" )
    , ( Moderation.ModerationReasonPII, "PII" )
    , ( Moderation.ModerationReasonOther, "Other" )
    ]


moderationReasonButton : Moderation.ModerationReason -> ( Moderation.ModerationReason, String ) -> Html Msg
moderationReasonButton selectedReason ( reason, labelText ) =
    let
        selectedClass =
            if selectedReason == reason then
                " ring-2 ring-slate-900"

            else
                ""
    in
    button [ type_ "button", onClick (ModerationReasonChanged reason), Html.Attributes.class (Ui.secondaryButtonClass ++ selectedClass), testId ("moderation-reason-" ++ String.toLower labelText) ] [ text labelText ]


seriesLinkBlock : TaskDetail -> List (Html Msg)
seriesLinkBlock detail =
    if detail.seriesID == "" then
        []

    else
        [ a [ href ("#/series/" ++ detail.seriesID), Html.Attributes.class "text-sm underline", testId "task-series-link" ] [ text "Part of a series" ] ]


taskTypeBadge : TaskDetail -> List (Html Msg)
taskTypeBadge detail =
    if detail.taskType == "" || detail.taskType == "general" then
        []

    else
        [ span [ testId "detail-type" ] [ Ui.badge (taskTypeLabel detail.taskType) ] ]


referenceBlock : TaskDetail -> List (Html Msg)
referenceBlock detail =
    if detail.referenceURL == "" then
        []

    else
        [ Ui.label_ "Reference"
        , a [ href detail.referenceURL, Html.Attributes.target "_blank", Html.Attributes.rel "noopener noreferrer", Html.Attributes.class "text-sm underline break-all", testId "detail-reference" ] [ text detail.referenceURL ]
        ]


taskInputBlock : TaskDetail -> List (Html Msg)
taskInputBlock detail =
    if detail.payloadKind == "json" && detail.payloadJson /= "" then
        [ Ui.label_ "Task input", Ui.codeBlock [ testId "detail-input" ] detail.payloadJson ]

    else
        []


taskAttachmentsBlock : TaskDetail -> List (Html Msg)
taskAttachmentsBlock detail =
    if List.isEmpty detail.attachments then
        []

    else
        [ Ui.label_ "Attachments"
        , div [ Html.Attributes.class "flex flex-wrap gap-2", testId "detail-attachments" ]
            (List.map taskAttachmentLink detail.attachments)
        ]


taskAttachmentLink : Task.TaskAttachmentResponse -> Html Msg
taskAttachmentLink attachment =
    attachmentLink attachment.name attachment.contentType attachment.sizeBytes attachment.dataURL


submissionAttachmentsView : List Submission.SubmissionAttachmentResponse -> Html Msg
submissionAttachmentsView attachments =
    if List.isEmpty attachments then
        text ""

    else
        div [ Html.Attributes.class "flex flex-wrap gap-2", testId "submission-attachments" ]
            (List.map submissionAttachmentLink attachments)


submissionAttachmentLink : Submission.SubmissionAttachmentResponse -> Html Msg
submissionAttachmentLink attachment =
    attachmentLink attachment.name attachment.contentType attachment.sizeBytes attachment.dataURL


attachmentLink : String -> String -> Int -> String -> Html Msg
attachmentLink name contentType sizeBytes dataURL =
    a
        [ href dataURL
        , Html.Attributes.download name
        , Html.Attributes.class "rounded border border-slate-200 px-2 py-1 text-xs text-slate-700 underline"
        , testId "attachment-link"
        ]
        [ text (name ++ " · " ++ contentType ++ " · " ++ String.fromInt sizeBytes ++ " bytes") ]


selectedAttachmentsView : String -> List SelectedAttachment -> Msg -> (Int -> Msg) -> String -> Html Msg
selectedAttachmentsView labelText attachments pickMsg removeMsg id =
    div [ Html.Attributes.class "space-y-2", testId id ]
        [ div [ Html.Attributes.class "flex flex-wrap items-center gap-2" ]
            [ Ui.label_ labelText
            , Ui.secondaryButton [ type_ "button", onClick pickMsg, testId (id ++ "-pick") ] "Add file"
            ]
        , if List.isEmpty attachments then
            p [ Html.Attributes.class "text-xs text-slate-500" ] [ text "No files attached." ]

          else
            div [ Html.Attributes.class "space-y-1" ]
                (List.indexedMap (selectedAttachmentRow removeMsg) attachments)
        ]


selectedAttachmentRow : (Int -> Msg) -> Int -> SelectedAttachment -> Html Msg
selectedAttachmentRow removeMsg index attachment =
    div [ Html.Attributes.class "flex flex-wrap items-center justify-between gap-2 rounded border border-slate-200 px-3 py-2 text-sm", testId "selected-attachment" ]
        [ span [ Html.Attributes.class "break-all text-slate-700" ] [ text (attachment.name ++ " · " ++ attachment.contentType ++ " · " ++ String.fromInt attachment.sizeBytes ++ " bytes") ]
        , Ui.secondaryButton [ type_ "button", onClick (removeMsg index), testId "remove-attachment" ] "Remove"
        ]


{-| A reward-return action (owner "Reclaim" or worker "Refund") paired with a
small info toggle that explains what it does. Both roles reach the same
endpoints; the label, test id, and explanation carry the role-specific meaning.
-}
rewardReturnControl : Msg -> String -> String -> String -> String -> Html Msg
rewardReturnControl clickMsg label buttonTestId infoTestId explanation =
    div [ Html.Attributes.class "space-y-1" ]
        [ Ui.secondaryButton [ type_ "button", onClick clickMsg, testId buttonTestId ] label
        , Ui.explainToggle infoTestId explanation
        ]


{-| Owner-side wording: the owner takes their own allocated reward back. -}
ownerReclaimExplanation : String
ownerReclaimExplanation =
    "Reclaim moves the reward you allocated to this task back to your wallet's spendable balance and cancels the task. You can only reclaim before the task is awarded to a worker."


{-| Worker-side wording: the active implementor hands the reward back to the
requester. Same effect as a reclaim, but named from the worker's side. -}
workerRefundExplanation : String
workerRefundExplanation =
    "Refund returns the reward to the requester and cancels the task. Use it if you have reserved this task but cannot complete the work. You can only refund before the task is awarded."


ownerControlsCard : LoggedInModel -> Html Msg
ownerControlsCard state =
    case state.detail of
        Just detail ->
            let
                draftOrOpen =
                    detail.state == Task.TaskStateDraft || detail.state == Task.TaskStateOpen

                -- A draft task is always fundable by its creator, regardless
                -- of the reward kind it was created with - funding a
                -- none/collectible-only task adds a credit component
                -- (transitioning it to credit/bundle server-side). Once a
                -- task is open, a credit/bundle reward is always already
                -- funded (the backend requires funding before opening), so
                -- funding is draft-only.
                canFund =
                    detail.state == Task.TaskStateDraft

                -- A brand-new, unfunded draft is the one case where funding
                -- is genuinely the next step, not just an available option -
                -- surfaced as an open-by-default callout instead of a
                -- same-weight collapsed disclosure next to unrelated
                -- housekeeping sections (API & MCP, Report task).
                -- A draft that holds no allocated reward yet is where funding
                -- is the genuine next step - whether it declared no reward or
                -- declared a credit/bundle/collectible reward it hasn't funded.
                -- A declared credit/bundle reward must be funded before the
                -- task can open, so surfacing the fund panel prominently
                -- (rather than as a collapsed disclosure) is what stops the
                -- owner from clicking Open straight into an invariant error.
                needsFundingGuidance =
                    canFund && not (holdsCredits || holdsCollectibles)

                -- Whether the task currently holds any allocated reward (live
                -- funding state from the detail response), as opposed to a
                -- merely *declared* reward that was never funded. Only a task
                -- that actually holds a reward can be refunded, so this gates
                -- the Refund button - a declared-but-unfunded draft no longer
                -- shows a Refund button that would just report "nothing to
                -- refund".
                holdsCredits =
                    detail.allocatedCredits > 0

                holdsCollectibles =
                    not (List.isEmpty detail.allocatedCollectibleIDs)

                -- The owner's reward-return action is a *reclaim*: it pulls the
                -- reward the owner allocated to this task back to their own
                -- spendable balance and cancels the task. This is deliberately
                -- worded differently from a worker's *refund* (see
                -- workerRefundControl in the reservation card) even though both
                -- hit the same endpoint - the endpoint returns the reward to the
                -- funder (always the owner) and cancels the task, but "reclaim"
                -- names it from the owner's side ("take my reward back") and
                -- "refund" from the worker's side ("give the reward back").
                --
                -- The credit refund endpoint (/refund) is the unified path: it
                -- returns held credits AND held collectibles together (so it
                -- handles bundle rewards in one shot). The collectible-refund
                -- endpoint only handles collectible-only tasks (it 409s on
                -- bundle). So: any held credits (credit or bundle) -> /refund;
                -- collectible-only holdings -> /collectible-refund.
                reclaimControl =
                    if draftOrOpen && holdsCredits && holdsCollectibles then
                        Just (rewardReturnControl (RefundTaskClicked detail.id) "Reclaim reward" "refund-task" "refund-task-info" ownerReclaimExplanation)

                    else if draftOrOpen && holdsCredits then
                        Just (rewardReturnControl (RefundTaskClicked detail.id) "Reclaim credits" "refund-task" "refund-task-info" ownerReclaimExplanation)

                    else if draftOrOpen && holdsCollectibles then
                        Just (rewardReturnControl (RefundCollectibleRewardClicked detail.id) "Reclaim collectible" "refund-collectible" "refund-collectible-info" ownerReclaimExplanation)

                    else
                        Nothing

                buttons =
                    List.filterMap identity
                        [ if detail.state == Task.TaskStateDraft then
                            Just (Ui.secondaryButton [ type_ "button", onClick (OpenTaskClicked detail.id), testId "open-task" ] "Open")

                          else
                            Nothing
                        , -- Moves an open task back to draft - the escape hatch for a task
                          -- that reached "open" without its declared reward actually being
                          -- funded (e.g. a backend that didn't enforce that invariant): back
                          -- to draft, fund it through the normal draft funding panel, then
                          -- reopen. Also just generally useful to un-publish a mistake.
                          if detail.state == Task.TaskStateOpen then
                            Just (Ui.secondaryButton [ type_ "button", onClick (UnpublishTaskClicked detail.id), testId "unpublish-task" ] "Unpublish")

                          else
                            Nothing
                        , -- Cancel ends an unfunded task. Reward-bearing tasks are ended via
                          -- Reclaim instead, which is the clearer label for "take my
                          -- allocated reward back and close the task" - though the backend now
                          -- settles the funds either way (cancelling a funded task returns
                          -- its allocated credits to the owner's spendable balance too).
                          if detail.state == Task.TaskStateDraft || (detail.state == Task.TaskStateOpen && detail.rewardKind == "none") then
                            Just (Ui.secondaryButton [ type_ "button", onClick (CancelTaskClicked detail.id), testId "cancel-task" ] "Cancel")

                          else
                            Nothing
                        ]
            in
            Ui.card
                [ Ui.sectionTitle "Owner controls"
                , p [ Html.Attributes.class "rounded-md bg-slate-100 px-3 py-2 text-sm text-slate-700", testId "task-guidance" ] [ text (taskStateGuidance detail.state) ]
                , taskFundingStatus holdsCredits holdsCollectibles detail
                , div [ Html.Attributes.class "flex flex-wrap gap-2" ] buttons
                , Maybe.withDefault (text "") reclaimControl
                , maybeNote state.taskActionMessage "task-action-message"
                , if needsFundingGuidance then
                    -- state.fundTaskId is kept synced to the currently-viewed
                    -- task on entering this page (see enterPage), so this
                    -- reuses the exact same FundClicked/fund-* plumbing as
                    -- the standalone Funding page without a separate Msg.
                    -- A brand-new unfunded draft gets an open-by-default
                    -- callout instead of a collapsed disclosure, since
                    -- funding is genuinely the next step here.
                    div [ Html.Attributes.class "space-y-3 rounded-md border border-blue-200 bg-blue-50 p-4", testId "fund-task-callout" ]
                        [ p [ Html.Attributes.class "text-xs font-semibold tracking-wide text-blue-700" ] [ text "BEFORE YOU OPEN THIS TASK" ]
                        , p [ Html.Attributes.class "text-sm text-blue-900" ]
                            [ text
                                (if detail.rewardKind == "none" then
                                    "This draft has no reward yet. Fund it with credits, a collectible, or both - or open it unfunded if that's intentional."

                                 else
                                    "This draft declares a reward but hasn't been funded yet. A declared reward must be funded before the task can open."
                                )
                            ]
                        , Ui.textInput [ type_ "number", placeholder "Amount in credits", value state.fundAmount, onInput FundAmountChanged, testId "fund-amount" ]
                        , organizationPicker "fund-organization" state.fundOrganizationId state.organizationQuery FundOrganizationIdChanged OrganizationQueryChanged SearchOrganizationsClicked PreviousOrganizationsPageClicked NextOrganizationsPageClicked "Personal balance" state.organizations state.organizationOffset
                        , Ui.primaryButton [ type_ "button", onClick FundClicked, testId "fund" ] "Fund task"
                        , maybeNote state.fundMessage "fund-message"
                        ]

                  else if canFund then
                    Ui.disclosure "fund-task-panel"
                        False
                        "Fund this task"
                        [ Ui.textInput [ type_ "number", placeholder "Amount in credits", value state.fundAmount, onInput FundAmountChanged, testId "fund-amount" ]
                        , organizationPicker "fund-organization" state.fundOrganizationId state.organizationQuery FundOrganizationIdChanged OrganizationQueryChanged SearchOrganizationsClicked PreviousOrganizationsPageClicked NextOrganizationsPageClicked "Personal balance" state.organizations state.organizationOffset
                        , Ui.primaryButton [ type_ "button", onClick FundClicked, testId "fund" ] "Fund task"
                        , maybeNote state.fundMessage "fund-message"
                        ]

                  else
                    text ""
                , if canFund then
                    -- state.awardTaskId is kept synced to the currently-viewed
                    -- task on entering this page (see enterPage), so this
                    -- reuses the exact same AwardClicked/award-* plumbing as
                    -- the Collectibles page without a separate Msg.
                    Ui.disclosure "add-collectible-reward-panel"
                        False
                        "Add a collectible to this task's reward"
                        [ collectiblesList state
                        , maybeNote state.awardMessage "award-message"
                        ]

                  else
                    text ""
                ]

        Nothing ->
            text ""


detailCard : String -> LoggedInModel -> Html Msg
detailCard origin state =
    case state.detail of
        Just detail ->
            Ui.card
                ([ p [ Html.Attributes.class "text-2xl font-semibold", testId "detail-title" ] [ text detail.title ]
                 , div [ Html.Attributes.class "flex flex-wrap items-center gap-2" ]
                    ([ taskStateBadge detail.state
                     , Ui.badge (availabilityKindLabel detail.availabilityKind)
                     , Ui.badge (participationPolicyLabel detail.participationPolicy)
                     ]
                        ++ taskTypeBadge detail
                    )
                , p [ Html.Attributes.class "text-xs text-slate-500" ]
                    [ text "Posted by "
                    , a [ href ("#/users/" ++ detail.createdBy), Html.Attributes.class "underline", testId "detail-created-by-link" ] [ text detail.createdBy ]
                    ]
                , p [ Html.Attributes.class "text-sm font-medium" ] [ text ("Reward: " ++ rewardLabel detail.rewardKind detail.rewardCreditAmount detail.rewardCollectibleCount) ]
                , p [ Html.Attributes.class "text-sm text-slate-700" ] [ text detail.description ]
                ]
                    ++ referenceBlock detail
                    ++ seriesLinkBlock detail
                    ++ taskInputBlock detail
                    ++ taskAttachmentsBlock detail
                    ++ [ Ui.label_ "Response schema"
                       , Ui.codeBlock [ testId "detail-schema" ] detail.responseSchemaJson
                       , taskIntegration origin detail.id state
                       ]
                )

        Nothing ->
            case state.detailError of
                Just message ->
                    Ui.card [ p [ Html.Attributes.class "text-sm text-slate-700", testId "detail-error" ] [ text ("Could not load this task: " ++ message) ] ]

                Nothing ->
                    Ui.card [ p [ Html.Attributes.class "text-sm text-slate-500" ] [ text "Loading task…" ] ]


reservationCard : LoggedInModel -> Html Msg
reservationCard state =
    case state.detail of
        Just detail ->
            let
                isOwner =
                    detail.createdBy == state.subjectId
            in
            Ui.card
                [ Ui.sectionTitle "Reservation"

                -- Spell these out: the raw enum badges ("submit user",
                -- "wait user") read as noise to anyone who has not read the
                -- API reference.
                , p [ Html.Attributes.class "text-sm text-slate-600", testId "reservation-summary" ]
                    [ text ("You can: " ++ viewerActionSentence detail.viewerAction ++ " · Assignee scope: " ++ assigneeScopeLabel detail.assigneeScope) ]

                -- The server's viewerAction doesn't rule out an owner
                -- reserving their own task, but that's not a real workflow
                -- (previously invisible entirely, since owners never saw
                -- this card at all) — the display below (existing
                -- reservations, approve/decline/cancel) is still shown to
                -- owners; only the "go claim this yourself" action is
                -- owner-gated.
                , if isOwner then
                    text ""

                  else
                    reservationAction state detail
                , reservationsList isOwner state.subjectId state.reservations
                , workerRefundControl state detail isOwner

                -- The refund confirmation lives in taskActionMessage, which the
                -- owner sees in the owner-controls card. A worker never sees that
                -- card, so echo it here (worker-only) - otherwise the worker's
                -- Refund click would report nothing back.
                , if isOwner then
                    text ""

                  else
                    maybeNote state.taskActionMessage "worker-task-action-message"
                , reservationSecretView state.reservationSecret
                , maybeNote state.reservationMessage "reservation-message"
                ]

        Nothing ->
            text ""


{-| The active implementor's counterpart to the owner's Reclaim: a worker who
holds the active reservation on a still-fundable task can hand the reward back
to the requester and cancel the task (the backend authorizes the active
implementor for the same /refund path). Only shown when the viewer actually
holds an active reservation and the task still holds a reward - so it never
appears as a button that would just fail server-side. Owners use Reclaim in the
owner controls instead, so this is worker-only.
-}
workerRefundControl : LoggedInModel -> PublicTaskDetail -> Bool -> Html Msg
workerRefundControl state detail isOwner =
    let
        holdsActiveReservation =
            List.any
                (\reservation ->
                    reservation.state
                        == Task.TaskReservationStateActive
                        && reservation.assigneeKind
                        == Task.TaskAssigneeScopeUser
                        && reservation.assigneeID
                        == state.subjectId
                )
                state.reservations

        draftOrOpen =
            detail.state == Task.TaskStateDraft || detail.state == Task.TaskStateOpen

        holdsCredits =
            detail.allocatedCredits > 0

        holdsCollectibles =
            not (List.isEmpty detail.allocatedCollectibleIDs)
    in
    if isOwner || not holdsActiveReservation || not draftOrOpen then
        text ""

    else if holdsCredits then
        -- Any held credits (credit or bundle) go through /refund, which returns
        -- credits and collectibles together - matching the owner's Reclaim.
        rewardReturnControl (RefundTaskClicked detail.id) "Refund reward" "worker-refund-task" "worker-refund-task-info" workerRefundExplanation

    else if holdsCollectibles then
        rewardReturnControl (RefundCollectibleRewardClicked detail.id) "Refund collectible" "worker-refund-collectible" "worker-refund-collectible-info" workerRefundExplanation

    else
        text ""


viewerActionSentence : Task.TaskViewerAction -> String
viewerActionSentence action =
    case action of
        Task.TaskViewerActionSubmit ->
            "submit a response now"

        Task.TaskViewerActionReserve ->
            "reserve this task"

        Task.TaskViewerActionRequestApproval ->
            "request a reservation (the owner must approve it)"

        Task.TaskViewerActionWait ->
            -- viewerAction is computed without looking at who is asking, so
            -- the active reservation may well be the viewer's own (listed
            -- below) - do not claim it belongs to someone else.
            "wait - this task already has an active reservation"

        Task.TaskViewerActionNone ->
            "view this task (no worker action available)"


reservationSecretView : Maybe String -> Html Msg
reservationSecretView secret =
    case secret of
        Just token ->
            div [ Html.Attributes.class "mt-4 space-y-3 rounded-md bg-slate-50 p-4" ]
                [ Ui.label_ "Agent token for this task (shown once)"
                , Ui.codeBlock [ testId "reservation-agent-secret" ] token
                ]

        Nothing ->
            text ""


reservationAction : LoggedInModel -> PublicTaskDetail -> Html Msg
reservationAction state detail =
    case detail.viewerAction of
        Task.TaskViewerActionReserve ->
            reservationActionForm state detail "Reserve" "reserve-task"

        Task.TaskViewerActionRequestApproval ->
            reservationActionForm state detail "Request approval" "request-approval"

        _ ->
            text ""


reservationActionForm : LoggedInModel -> PublicTaskDetail -> String -> String -> Html Msg
reservationActionForm state detail label id =
    div [ Html.Attributes.class "space-y-3" ]
        [ organizationTeamReservationFields state detail
        , Ui.primaryButton [ type_ "button", onClick (ReserveClicked detail.id), testId id ] label
        ]


organizationTeamReservationFields : LoggedInModel -> PublicTaskDetail -> Html Msg
organizationTeamReservationFields state detail =
    case detail.assigneeScope of
        Task.TaskAssigneeScopeOrganizationTeam ->
            div [ Html.Attributes.class "grid gap-3 md:grid-cols-2" ]
                [ Ui.fieldLabel "Organization" [ organizationPicker "reservation-organization-id" state.reservationOrganizationId state.organizationQuery ReservationOrganizationIdChanged OrganizationQueryChanged SearchOrganizationsClicked PreviousOrganizationsPageClicked NextOrganizationsPageClicked "Choose organization" state.organizations state.organizationOffset ]
                , Ui.fieldLabel "Team" [ teamPicker "reservation-team-id" state.reservationTeamId state.orgTeamQuery ReservationTeamIdChanged OrgTeamQueryChanged SearchOrgTeamsClicked PreviousOrgTeamsPageClicked NextOrgTeamsPageClicked "Choose team" state.orgTeams state.orgTeamOffset ]
                ]

        Task.TaskAssigneeScopeTeam ->
            Ui.fieldLabel "Team" [ teamPicker "reservation-team-id" state.reservationTeamId state.standaloneTeamQuery ReservationTeamIdChanged StandaloneTeamQueryChanged SearchStandaloneTeamsClicked PreviousStandaloneTeamsPageClicked NextStandaloneTeamsPageClicked "Choose team" state.standaloneTeams state.standaloneTeamOffset ]

        _ ->
            text ""


reservationsList : Bool -> String -> List Task.TaskReservationResponse -> Html Msg
reservationsList isOwner subjectId reservations =
    if List.isEmpty reservations then
        text ""

    else
        div [ Html.Attributes.class "divide-y divide-slate-100", testId "reservations" ] (List.map (reservationRow isOwner subjectId) reservations)


reservationRow : Bool -> String -> Task.TaskReservationResponse -> Html Msg
reservationRow isOwner subjectId reservation =
    div [ Html.Attributes.class "flex items-center justify-between gap-3 py-2", testId "reservation-row" ]
        [ div []
            [ p [ Html.Attributes.class "text-sm font-medium" ]
                [ assigneeIdentityLink reservation.assigneeKind reservation.assigneeID
                , text (" · " ++ assigneeScopeLabel reservation.assigneeKind)
                ]
            , p [ Html.Attributes.class "text-xs text-slate-500" ] [ text (reservationStateLabel reservation.state) ]
            ]
        , div [ Html.Attributes.class "flex flex-wrap gap-2" ] (reservationButtons isOwner subjectId reservation)
        ]


{-| Only a user-scoped assignee ID is a valid `/users/<id>` profile — team and
organization-team IDs aren't, so those render as plain text.
-}
assigneeIdentityLink : Task.TaskAssigneeScope -> String -> Html Msg
assigneeIdentityLink assigneeKind assigneeID =
    if assigneeKind == Task.TaskAssigneeScopeUser then
        a [ href ("#/users/" ++ assigneeID), Html.Attributes.class "underline", testId "reservation-assignee-link" ] [ text assigneeID ]

    else
        text assigneeID


{-| Only show actions the current viewer is actually entitled to take —
approving/declining a request is owner-only, and cancelling an active
reservation is either the owner (force-cancel) or the person who holds it.
Every other viewer (e.g. a different worker browsing the task) sees the
reservation but no buttons that would just fail server-side if clicked.
-}
reservationButtons : Bool -> String -> Task.TaskReservationResponse -> List (Html Msg)
reservationButtons isOwner subjectId reservation =
    let
        isHolder =
            reservation.assigneeKind == Task.TaskAssigneeScopeUser && reservation.assigneeID == subjectId
    in
    case reservation.state of
        Task.TaskReservationStateRequested ->
            if isOwner then
                [ Ui.primaryButton [ type_ "button", onClick (ApproveReservationClicked reservation.id), testId "approve-reservation" ] "Approve"
                , Ui.secondaryButton [ type_ "button", onClick (DeclineReservationClicked reservation.id), testId "decline-reservation" ] "Decline"
                ]

            else
                []

        Task.TaskReservationStateActive ->
            if isOwner || isHolder then
                [ Ui.secondaryButton [ type_ "button", onClick (CancelReservationClicked reservation.id), testId "cancel-reservation" ] "Cancel" ]

            else
                []

        _ ->
            []


submitCard : LoggedInModel -> Html Msg
submitCard state =
    case state.detail of
        Nothing ->
            text ""

        Just detail ->
            -- Only render the submit form when the task can actually accept a
            -- submission: it must be open. Eligibility (reservation/approval) is
            -- enforced by the server; viewerAction is viewer-independent so it is
            -- not a reliable "can this viewer submit" signal. Hiding on non-open
            -- states removes the dead form on closed/cancelled/refunded tasks.
            if detail.state == Task.TaskStateOpen then
                submitCardForm state

            else
                text ""


submitCardForm : LoggedInModel -> Html Msg
submitCardForm state =
    let
        schemaFields =
            if state.submitRawMode then
                Nothing

            else
                state.detail
                    |> Maybe.andThen (\detail -> ResponseSchema.parse detail.responseSchemaJson)
                    |> Maybe.andThen ResponseSchema.formFields

        editor =
            case schemaFields of
                Just fields ->
                    -- The task declared a structured response schema, so
                    -- render one typed input per field instead of making the
                    -- worker hand-write the whole JSON document.
                    div [ Html.Attributes.class "space-y-3", testId "submit-schema-form" ]
                        (List.map (schemaFieldInput state.submitFieldValues) fields
                            ++ [ Ui.checkbox [ checked False, onClick (SubmitRawModeToggled True), testId "submit-raw-toggle" ] "Edit as raw JSON instead" ]
                        )

                Nothing ->
                    div [ Html.Attributes.class "space-y-3" ]
                        (Ui.textarea_
                            [ placeholder "{}"
                            , value state.submitInput
                            , onInput SubmitInputChanged
                            , Html.Attributes.rows 6
                            , testId "detail-submit-input"
                            ]
                            :: (if state.submitRawMode then
                                    [ Ui.checkbox [ checked True, onClick (SubmitRawModeToggled False), testId "submit-raw-toggle" ] "Edit as raw JSON instead" ]

                                else
                                    []
                               )
                        )
    in
    form [ Html.Attributes.class "space-y-4 rounded-lg border border-slate-200 bg-white p-6 shadow-sm", onSubmit SubmitClicked ]
        [ Ui.sectionTitle "Submit a response"
        , editor
        , selectedAttachmentsView "Attachments" state.submitAttachments PickSubmitAttachmentClicked RemoveSubmitAttachmentClicked "submit-attachments"
        , Ui.primaryButton [ type_ "submit", testId "detail-submit" ] "Submit response"
        , maybeNote state.submitMessage "detail-submit-message"
        ]


schemaFieldInput : Dict.Dict String String -> ResponseSchema.FormField -> Html Msg
schemaFieldInput values field =
    let
        raw =
            Dict.get field.name values |> Maybe.withDefault ""

        label =
            field.name
                ++ (if field.required then
                        " *"

                    else
                        " (optional)"
                   )

        fieldTestId =
            testId ("submit-field-" ++ field.name)
    in
    case field.input of
        ResponseSchema.TextInput ->
            Ui.fieldLabel label [ Ui.textInput [ type_ "text", value raw, onInput (SubmitFieldChanged field.name), fieldTestId ] ]

        ResponseSchema.IntegerInput ->
            Ui.fieldLabel label [ Ui.textInput [ type_ "number", value raw, onInput (SubmitFieldChanged field.name), fieldTestId ] ]

        ResponseSchema.DecimalInput ->
            Ui.fieldLabel (label ++ " - decimal number") [ Ui.textInput [ type_ "text", value raw, onInput (SubmitFieldChanged field.name), fieldTestId ] ]

        ResponseSchema.EnumSelect options ->
            Ui.fieldLabel label
                [ select [ Html.Attributes.class Ui.fieldClass, value raw, onInput (SubmitFieldChanged field.name), fieldTestId ]
                    (blankOption "Choose a value" :: List.map (\option_ -> option [ value option_, selected (raw == option_) ] [ text option_ ]) options)
                ]

        ResponseSchema.FixedLiteral literal ->
            Ui.fieldLabel (field.name ++ " - fixed value, included automatically")
                [ p [ Html.Attributes.class "text-sm text-slate-600", fieldTestId ] [ text literal ] ]

        ResponseSchema.LinesArray ->
            Ui.fieldLabel (label ++ " - one item per line")
                [ Ui.textarea_ [ value raw, onInput (SubmitFieldChanged field.name), Html.Attributes.rows 4, fieldTestId ] ]

        ResponseSchema.JsonArrayInput ->
            Ui.fieldLabel (label ++ " - JSON array")
                [ Ui.textarea_ [ placeholder "[ ]", value raw, onInput (SubmitFieldChanged field.name), Html.Attributes.rows 4, fieldTestId ] ]

        ResponseSchema.JsonInput ->
            Ui.fieldLabel (label ++ " - JSON")
                [ Ui.textarea_ [ placeholder "{}", value raw, onInput (SubmitFieldChanged field.name), Html.Attributes.rows 4, fieldTestId ] ]


mySubmissionsCard : LoggedInModel -> Html Msg
mySubmissionsCard state =
    case state.detail of
        Nothing ->
            text ""

        Just detail ->
            let
                mine =
                    List.filter (\submission -> submission.taskID == detail.id) state.userSubmissions
            in
            if List.isEmpty mine then
                text ""

            else
                Ui.card
                    [ Ui.sectionTitle "My submissions"
                    , div [ Html.Attributes.class "divide-y divide-slate-100", testId "my-submissions" ]
                        (List.map (mySubmissionRow state) mine)
                    ]


mySubmissionRow : LoggedInModel -> Submission.SubmissionResponse -> Html Msg
mySubmissionRow state submission =
    div [ Html.Attributes.class "space-y-2 py-3", testId "my-submission-row" ]
        [ div [ Html.Attributes.class "flex items-center justify-between gap-2" ]
            [ submissionStateBadge submission.state
            , Ui.secondaryButton [ type_ "button", onClick (OpenSubmissionComments submission.id), testId "my-submission-comments-toggle" ] (discussionButtonLabel state submission.id)
            ]
        , reviewNoteView submission.reviewNote
        , Ui.codeBlock [ testId "my-submission-response" ] submission.responseJSON
        , submissionAttachmentsView submission.attachments
        , validationErrorsView submission.validationErrors
        , sensitiveFieldsView submission.sensitiveFields
        , submissionCommentsThread state submission
        ]


submissionsCard : LoggedInModel -> Html Msg
submissionsCard state =
    Ui.card
        [ Ui.sectionTitle "Submissions"
        , if List.isEmpty state.submissions then
            text ""

          else
            reviewControls state
        , submissionsList state
        , maybeNote state.reviewMessage "review-message"
        ]


reviewControls : LoggedInModel -> Html Msg
reviewControls state =
    div [ Html.Attributes.class "mb-3 grid gap-3 rounded border border-slate-200 p-3 text-sm" ]
        [ Ui.fieldLabel "Review note"
            [ Ui.textarea_ [ Html.Attributes.class "min-h-20", Html.Attributes.rows 3, value state.reviewNote, onInput ReviewNoteChanged, testId "review-note" ] ]
        , div [ Html.Attributes.class "grid gap-2 sm:grid-cols-3" ]
            [ Ui.fieldLabel "Partial payout"
                [ Ui.textInput [ type_ "number", value state.reviewPartialCredit, onInput ReviewPartialCreditChanged, testId "review-partial-credit" ] ]
            , Ui.fieldLabel "Tip"
                [ Ui.textInput [ type_ "number", value state.reviewTip, onInput ReviewTipChanged, testId "review-tip" ] ]
            , div [ Html.Attributes.class "pt-6" ]
                [ Ui.checkbox [ checked state.reviewBan, onCheck ReviewBanChanged, testId "review-ban" ] "Ban implementor" ]
            ]
        , Ui.fieldLabel "Tip a collectible (optional)"
            [ select
                [ Html.Attributes.class Ui.fieldClass
                , value state.reviewTipCollectibleId
                , onInput ReviewTipCollectibleChanged
                , testId "review-tip-collectible"
                ]
                (blankOption "No collectible tip"
                    :: List.map
                        (\c -> option [ value c.id, selected (state.reviewTipCollectibleId == c.id) ]
                            [ text (c.name ++ " · " ++ collectibleKindLabel c.kind) ]
                        )
                        -- Only minted holdings can be tipped; offering an
                        -- escrowed or awarded collectible here just produces
                        -- a server rejection on Accept.
                        (List.filter (\c -> c.state == Collectible.CollectibleStateMinted) state.collectibles)
                )
            ]
        ]


submissionsList : LoggedInModel -> Html Msg
submissionsList state =
    if List.isEmpty state.submissions then
        p [ Html.Attributes.class "text-sm text-slate-500", testId "submissions-empty" ] [ text "No submissions to review." ]

    else
        div [ Html.Attributes.class "divide-y divide-slate-100", testId "submissions" ]
            (List.map (submissionRow state) state.submissions)


submissionRow : LoggedInModel -> Submission.SubmissionResponse -> Html Msg
submissionRow state submission =
    div [ Html.Attributes.class "space-y-2 py-3", testId "submission-row" ]
        [ div [ Html.Attributes.class "flex items-center justify-between gap-2" ]
            [ submissionStateBadge submission.state
            , reviewButtons state submission
            ]
        , p [ Html.Attributes.class "text-xs text-slate-500" ]
            [ text "Submitter: "
            , a [ href ("#/users/" ++ submission.submitterID), Html.Attributes.class "underline", testId "submission-submitter-link" ] [ text submission.submitterID ]
            ]
        , reviewNoteView submission.reviewNote
        , Ui.codeBlock [ testId "submission-response" ] submission.responseJSON
        , submissionAttachmentsView submission.attachments
        , validationErrorsView submission.validationErrors
        , Ui.secondaryButton [ type_ "button", onClick (OpenSubmissionComments submission.id), testId "submission-comments-toggle" ] (discussionButtonLabel state submission.id)
        , submissionCommentsThread state submission
        ]


submissionCommentsThread : LoggedInModel -> Submission.SubmissionResponse -> Html Msg
submissionCommentsThread state submission =
    if state.activeSubmissionCommentsID == Just submission.id then
        div [ Html.Attributes.class "space-y-2 rounded-md bg-slate-50 p-3", testId "submission-comments-thread" ]
            [ if List.isEmpty state.submissionComments then
                p [ Html.Attributes.class "text-sm text-slate-500", testId "submission-comments-empty" ] [ text "No comments yet." ]

              else
                div [ Html.Attributes.class "space-y-2", testId "submission-comments" ] (List.map submissionCommentRow state.submissionComments)
            , form [ Html.Attributes.class "space-y-2", onSubmit (AddSubmissionCommentClicked submission.id) ]
                [ Ui.textarea_ [ placeholder "Add a comment", value state.submissionCommentBody, onInput SubmissionCommentBodyChanged, testId "submission-comment-body" ]
                , Ui.primaryButton [ type_ "submit", testId "add-submission-comment" ] "Comment"
                , maybeNote state.submissionCommentMessage "submission-comment-message"
                ]
            ]

    else
        text ""

discussionButtonLabel : LoggedInModel -> String -> String
discussionButtonLabel state submissionId =
    if state.activeSubmissionCommentsID == Just submissionId then
        "Discussion open"

    else
        "Discuss"


submissionCommentRow : Submission.SubmissionCommentResponse -> Html Msg
submissionCommentRow comment =
    div [ Html.Attributes.class "rounded-md border border-slate-200 bg-white p-3", testId "submission-comment" ]
        [ a [ href ("#/users/" ++ comment.authorUserID), Html.Attributes.class "text-xs font-medium text-slate-600 underline" ] [ text comment.authorUserID ]
        , p [ Html.Attributes.class "text-sm text-slate-700 break-words" ] [ text comment.body ]
        ]


reviewButtons : LoggedInModel -> Submission.SubmissionResponse -> Html Msg
reviewButtons state submission =
    case submission.state of
        Submission.SubmissionStateSubmitted ->
            div [ Html.Attributes.class "flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:justify-end" ]
                [ Ui.secondaryButton [ type_ "button", onClick (RequestChangesClicked submission.id), disabled (String.trim state.reviewNote == ""), testId "request-changes" ] "Request changes"
                , Ui.secondaryButton [ type_ "button", onClick (RejectClicked submission.id), disabled (String.trim state.reviewNote == ""), testId "reject-submission" ] "Reject"
                , Ui.primaryButton [ type_ "button", onClick (AcceptClicked submission.id), testId "accept-submission" ] "Accept"
                ]

        _ ->
            text ""


reviewNoteView : String -> Html Msg
reviewNoteView note =
    if String.isEmpty (String.trim note) then
        text ""

    else
        p [ Html.Attributes.class "rounded border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-900", testId "submission-review-note" ] [ text note ]


validationErrorsView : List Submission.SubmissionValidationErrorResponse -> Html Msg
validationErrorsView errors =
    if List.isEmpty errors then
        text ""

    else
        div [ Html.Attributes.class "space-y-1" ] (List.map validationErrorView errors)


validationErrorView : Submission.SubmissionValidationErrorResponse -> Html Msg
validationErrorView item =
    p [ Html.Attributes.class "text-xs text-red-700" ] [ text (item.path ++ ": " ++ item.message) ]


sensitiveFieldsView : List Submission.SubmissionSensitiveFieldResponse -> Html Msg
sensitiveFieldsView fields =
    if List.isEmpty fields then
        text ""

    else
        div [ Html.Attributes.class "space-y-1 rounded border border-slate-200 bg-slate-50 px-3 py-2", testId "submission-sensitive-fields" ]
            (p [ Html.Attributes.class "text-xs font-semibold text-slate-700" ] [ text "Sensitive fields" ]
                :: List.map sensitiveFieldView fields
            )


sensitiveFieldView : Submission.SubmissionSensitiveFieldResponse -> Html Msg
sensitiveFieldView field =
    p [ Html.Attributes.class "text-xs text-slate-600", testId "submission-sensitive-field" ]
        [ text (field.path ++ " · " ++ field.category ++ " · " ++ field.retention ++ " · " ++ field.redaction ++ " · " ++ field.state ++ redactedAtSuffix field.redactedAt) ]


redactedAtSuffix : String -> String
redactedAtSuffix value =
    if String.trim value == "" then
        ""

    else
        " at " ++ value



-- Labels and helpers


maybeError : Maybe String -> String -> Html Msg
maybeError message identifier =
    case message of
        Just value ->
            Ui.errorText identifier value

        Nothing ->
            text ""


maybeNote : Maybe Note -> String -> Html Msg
maybeNote message identifier =
    case message of
        Just (SuccessNote value) ->
            Ui.successText identifier value

        Just (FailureNote value) ->
            Ui.errorText identifier value

        Nothing ->
            text ""


mcpConfig : String -> String -> String
mcpConfig origin secret =
    "{\n  \"mcpServers\": {\n    \"sharecrop\": {\n      \"url\": \""
        ++ origin
        ++ "/mcp\",\n      \"headers\": { \"Authorization\": \"Bearer "
        ++ secret
    ++ "\" }\n    }\n  }\n}"


copyButton : String -> Html Msg
copyButton clipboardText =
    Ui.secondaryButton [ onClick (CopyClicked clipboardText), testId "copy-command", Html.Attributes.class "w-full sm:w-auto" ] "Copy"


taskIntegration : String -> String -> LoggedInModel -> Html Msg
taskIntegration origin taskId state =
    Ui.disclosure "toggle-integration"
        False
        "API & MCP"
        [ taskIntegrationBody origin taskId state ]


taskIntegrationBody : String -> String -> LoggedInModel -> Html Msg
taskIntegrationBody origin taskId state =
    case state.taskAgentToken of
        Nothing ->
            div [ Html.Attributes.class "space-y-2" ]
                [ p [ Html.Attributes.class "text-sm text-slate-700" ] [ text "Create an agent token to get runnable, copy-paste commands for this task." ]
                , Ui.primaryButton [ onClick MintTaskTokenClicked, testId "mint-task-token" ] "Create agent token"
                ]

        Just token ->
            div [ Html.Attributes.class "space-y-4" ]
                [ div [ Html.Attributes.class "space-y-2" ]
                    [ Ui.label_ "Agent token"
                    , Ui.codeBlock [ testId "integration-token" ] token
                    , copyButton token
                    , p [ Html.Attributes.class "text-sm text-slate-700" ] [ text "Use this as the Bearer token below. Treat it like a password." ]
                    , Ui.secondaryButton [ onClick MintTaskTokenClicked, testId "mint-task-token" ] "Rotate"
                    ]
                , Ui.label_ "MCP"
                , integrationEntry "Install the MCP server (add to your .mcp.json or Claude config):" "integration-mcp-config" (mcpConfig origin token)
                , integrationEntry "Fetch the response schema your submission must match:" "integration-mcp-schema" (mcpSchemaBody taskId)
                , integrationEntry "Submit your response to this task:" "integration-mcp-submit" (mcpSubmitBody taskId)
                , Ui.label_ "REST API"
                , integrationEntry "Get this task over REST:" "integration-rest-get" (restGetCurl origin taskId token)
                , integrationEntry "Reserve this task:" "integration-rest-reserve" (restReserveCurl origin taskId token)
                , integrationEntry "Submit your response:" "integration-rest-submit" (restSubmitCurl origin taskId token)
                ]


integrationEntry : String -> String -> String -> Html Msg
integrationEntry description identifier command =
    div [ Html.Attributes.class "space-y-2" ]
        [ p [ Html.Attributes.class "text-sm text-slate-700" ] [ text description ]
        , Ui.codeBlock [ testId identifier ] command
        , copyButton command
        ]


mcpSchemaBody : String -> String
mcpSchemaBody taskId =
    "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"sharecrop.get_task_schema\",\"arguments\":{\"task_id\":\""
        ++ taskId
        ++ "\"}}}"


mcpSubmitBody : String -> String
mcpSubmitBody taskId =
    "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"sharecrop.submit_response\",\"arguments\":{\"task_id\":\""
        ++ taskId
        ++ "\",\"response_json\":\"{}\"}}}"


restGetCurl : String -> String -> String -> String
restGetCurl origin taskId token =
    "curl "
        ++ origin
        ++ "/api/tasks/"
        ++ taskId
        ++ " -H \"Authorization: Bearer "
        ++ token
        ++ "\""


restReserveCurl : String -> String -> String -> String
restReserveCurl origin taskId token =
    "curl -X POST "
        ++ origin
        ++ "/api/tasks/"
        ++ taskId
        ++ "/reservations -H \"Authorization: Bearer "
        ++ token
        ++ "\""


restSubmitCurl : String -> String -> String -> String
restSubmitCurl origin taskId token =
    "curl -X POST "
        ++ origin
        ++ "/api/tasks/"
        ++ taskId
        ++ "/submissions -H \"Authorization: Bearer "
        ++ token
        ++ "\" -H \"Content-Type: application/json\" -d '{\"response_json\":\"{}\"}'"


-- taskFundingStatus shows what reward is actually allocated (locked) to the
-- task right now, drawn from the live allocated fields on the detail response
-- rather than the declared reward. Renders nothing when the task holds no
-- reward, so an unfunded draft stays quiet.
taskFundingStatus : Bool -> Bool -> TaskDetail -> Html Msg
taskFundingStatus holdsCredits holdsCollectibles detail =
    let
        parts =
            (if holdsCredits then
                [ String.fromInt detail.allocatedCredits ++ " credits" ]

             else
                []
            )
                ++ (if holdsCollectibles then
                        [ pluralizeCollectibles (List.length detail.allocatedCollectibleIDs) ]

                    else
                        []
                   )
    in
    if List.isEmpty parts then
        text ""

    else
        p [ Html.Attributes.class "text-sm text-slate-700", testId "task-funding-status" ]
            [ text ("Allocated to this task: " ++ String.join " and " parts ++ ".") ]


pluralizeCollectibles : Int -> String
pluralizeCollectibles count =
    if count == 1 then
        "1 collectible"

    else
        String.fromInt count ++ " collectibles"


fundSuccessLabel : Ledger.TaskFundResponse -> String
fundSuccessLabel fund =
    "Allocated " ++ String.fromInt fund.creditAmount ++ " credits to this task."


submitSuccessLabel : Submission.SubmissionCreatedResponse -> String
submitSuccessLabel created =
    let
        base =
            "Submission " ++ created.submission.id ++ " (" ++ submissionStateLabel created.submission.state ++ ")."
    in
    if List.isEmpty created.submission.validationErrors then
        base

    else
        base ++ " " ++ String.join "; " (List.map (\error -> error.path ++ ": " ++ error.message) created.submission.validationErrors)


mintSuccessLabel : Collectible.CollectibleResponse -> String
mintSuccessLabel collectible =
    "Minted " ++ collectible.name ++ " (" ++ collectibleStateLabel collectible.state ++ ")."


awardSuccessLabel : Collectible.CollectibleResponse -> String
awardSuccessLabel collectible =
    "Awarded " ++ collectible.name ++ " (" ++ collectibleStateLabel collectible.state ++ ")."


reservationSuccessLabel : Task.TaskReservationResponse -> String
reservationSuccessLabel reservation =
    "Reservation " ++ reservationStateLabel reservation.state ++ "."


allKinds : List Collectible.CollectibleKind
allKinds =
    [ Collectible.CollectibleKindUnique
    , Collectible.CollectibleKindEdition
    , Collectible.CollectibleKindBadge
    ]


allPolicies : List Collectible.CollectibleTransferPolicy
allPolicies =
    [ Collectible.CollectibleTransferPolicyNonTransferableExceptPayout
    , Collectible.CollectibleTransferPolicyTransferableBetweenUsers
    , Collectible.CollectibleTransferPolicyTransferableWithinOrganization
    , Collectible.CollectibleTransferPolicyIssuerControlled
    ]


allParticipationPolicies : List Task.TaskParticipationPolicy
allParticipationPolicies =
    [ Task.TaskParticipationPolicyOpen
    , Task.TaskParticipationPolicyReservationRequired
    , Task.TaskParticipationPolicyApprovalRequired
    ]
