module Sharecrop.View exposing (..)

import Browser
import Html exposing (Html, a, div, form, label, main_, option, p, select, span, table, tbody, td, text, th, thead, tr)
import Html.Attributes exposing (checked, disabled, href, placeholder, selected, type_, value)
import Html.Events exposing (onCheck, onClick, onInput, onSubmit)
import Json.Decode as Decode
import Json.Encode as Encode
import Sharecrop.Generated.Agent as Agent
import Sharecrop.Generated.Collectible as Collectible
import Sharecrop.Generated.Ledger as Ledger
import Sharecrop.Generated.Notification as Notification
import Sharecrop.Generated.Organization as Organization
import Sharecrop.Generated.Privacy as Privacy
import Sharecrop.Generated.Submission as Submission
import Sharecrop.Generated.Task as Task
import Sharecrop.Generated.TaskSeries as TaskSeries
import Sharecrop.Generated.Team as Team
import Sharecrop.Sprites as Sprites
import Sharecrop.Labels exposing (allScopes, assigneeScopeLabel, assigneeScopeTag, availabilityKindLabel, collectibleKindLabel, collectibleKindTag, collectiblePolicyLabel, collectiblePolicyTag, collectibleStateLabel, credentialStateLabel, escrowStateLabel, kindLabel, participationPolicyLabel, participationPolicyTag, participationUsesReservation, reservationStateLabel, rewardLabel, scopeLabel, scopeTag, submissionStateLabel, taskStateGuidance, taskStateLabel, viewerActionLabel)
import Sharecrop.Types exposing (..)
import Sharecrop.Ui as Ui exposing (testId)


view : Model -> Browser.Document Msg
view model =
    { title = "Sharecrop"
    , body =
        [ main_ [ Html.Attributes.class "min-h-screen bg-slate-50 p-4 text-slate-950 sm:p-8" ]
            [ div [ Html.Attributes.class "mx-auto max-w-3xl space-y-6" ]
                [ Ui.pageTitle "Sharecrop"
                , sessionView model
                ]
            ]
        ]
    }


sessionView : Model -> Html Msg
sessionView model =
    case model.session of
        LoggedOut ->
            authView model

        LoggedIn state ->
            loggedInView model state


authView : Model -> Html Msg
authView model =
    form
        [ Html.Attributes.class "space-y-4 rounded-lg border border-slate-200 bg-white p-6 shadow-sm", onSubmit LoginClicked ]
        [ p [ Html.Attributes.class "text-slate-600" ] [ text "Sign in or create an account to view your credit ledger and set up agents." ]
        , Ui.textInput [ type_ "email", placeholder "Email", value model.email, onInput EmailChanged, testId "email" ]
        , Ui.textInput [ type_ "password", placeholder "Password", value model.password, onInput PasswordChanged, testId "password" ]
        , div [ Html.Attributes.class "flex gap-3" ]
            [ Ui.primaryButton [ type_ "submit", testId "login" ] "Log in"
            , Ui.secondaryButton [ type_ "button", onClick RegisterClicked, testId "register" ] "Register"
            , Ui.secondaryButton [ type_ "button", onClick GuestClicked, testId "guest-login" ] "Continue as guest"
            ]
        , div [ Html.Attributes.class "space-y-2 border-t border-slate-100 pt-4" ]
            [ Ui.label_ "Password reset"
            , Ui.textInput [ type_ "email", placeholder "Account email", value model.resetEmail, onInput PasswordResetEmailChanged, testId "reset-email" ]
            , div [ Html.Attributes.class "flex flex-wrap gap-2" ]
                [ Ui.secondaryButton [ type_ "button", onClick RequestPasswordResetClicked, testId "request-password-reset" ] "Create reset token" ]
            , Ui.textInput [ type_ "text", placeholder "Reset token", value model.resetToken, onInput PasswordResetTokenChanged, testId "reset-token" ]
            , Ui.textInput [ type_ "password", placeholder "New password", value model.resetPassword, onInput PasswordResetPasswordChanged, testId "reset-password" ]
            , Ui.secondaryButton [ type_ "button", onClick ConfirmPasswordResetClicked, testId "confirm-password-reset" ] "Reset password"
            ]
        , maybeError model.authError "auth-error"
        ]


loggedInView : Model -> LoggedInModel -> Html Msg
loggedInView model state =
    div [ Html.Attributes.class "space-y-6" ]
        [ navBar model.demo state.page state.subjectId state.isAdmin
        , pageView model.origin state
        ]


navBar : Bool -> Page -> String -> Bool -> Html Msg
navBar demo current subjectId isAdmin =
    div [ Html.Attributes.class "flex flex-wrap items-center gap-2" ]
        [ navLink current OverviewPage "overview" "Overview"
        , navLink current TasksPage "tasks" "Tasks"
        , navLink current CreateTaskPage "create-task" "New task"
        , navLink current DiscoveryPage "discovery" "Discovery"
        , navLink current FundingPage "funding" "Funding"
        , navLink current AgentsPage "agents" "Agents"
        , navLink current CollectiblesPage "collectibles" "Collectibles"
        , navLink current InboxPage "inbox" "Inbox"
        , navLink current SeriesListPage "series-list" "Series"
        , navLink current OrganizationsPage "organizations" "Organizations"
        , if isAdmin then
            navLink current AdminPage "admin" "Admin"

          else
            text ""
        , a [ href ("#/users/" ++ subjectId), Html.Attributes.class Ui.secondaryButtonClass, testId "nav-profile" ] [ text "Profile" ]
        , Ui.secondaryButton [ type_ "button", onClick LogoutClicked, testId "logout" ] "Log out"
        , if demo then
            Ui.secondaryButton [ type_ "button", onClick ResetDemoClicked, testId "reset-demo" ] "Reset demo"

          else
            text ""
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
    a [ href ("#" ++ pageToPath target), Html.Attributes.class styleClass, testId ("nav-" ++ identifier) ] [ text labelText ]


adminView : LoggedInModel -> Html Msg
adminView state =
    Ui.card
        [ Ui.sectionTitle "Operations"
        , case state.operations of
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
        , Ui.sectionTitle "Audit events"
        , div [ Html.Attributes.class "grid gap-3 sm:grid-cols-3" ]
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
        , if List.isEmpty state.auditEvents then
            p [ Html.Attributes.class "text-sm text-slate-500", testId "admin-audit-empty" ] [ text "No audit events." ]

          else
            div [ Html.Attributes.class "divide-y divide-slate-100", testId "admin-audit-events" ]
                (List.map auditEventRow state.auditEvents)
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


inboxView : LoggedInModel -> Html Msg
inboxView state =
    Ui.card
        [ Ui.sectionTitle "Inbox"
        , if List.isEmpty state.notifications then
            p [ Html.Attributes.class "text-sm text-slate-500", testId "inbox-empty" ] [ text "No notifications." ]

          else
            div [ Html.Attributes.class "divide-y divide-slate-100", testId "inbox-list" ]
                (List.map notificationRow state.notifications)
        , maybeNote state.inboxMessage "inbox-message"
        ]


notificationRow : Notification.NotificationResponse -> Html Msg
notificationRow notification =
    div [ Html.Attributes.class "space-y-2 py-3 text-sm", testId "notification-row" ]
        [ div [ Html.Attributes.class "flex flex-wrap items-center justify-between gap-2" ]
            [ p [ Html.Attributes.class "font-medium text-slate-900" ] [ text (notification.kind ++ " on " ++ notification.subjectKind) ]
            , span [ Html.Attributes.class (notificationStateClass notification.state), testId "notification-state" ] [ text notification.state ]
            ]
        , p [ Html.Attributes.class "break-words text-xs text-slate-500" ] [ text ("Subject " ++ notification.subjectID ++ " · actor " ++ notification.actorUserID ++ " · " ++ notification.createdAt) ]
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


pageView : String -> LoggedInModel -> Html Msg
pageView origin state =
    case state.page of
        OverviewPage ->
            overviewView state

        TasksPage ->
            tasksView origin state

        CreateTaskPage ->
            createTaskView state

        TaskDetailPage _ ->
            taskDetailPageView origin state

        DiscoveryPage ->
            discoveryView state

        FundingPage ->
            fundingView state

        AgentsPage ->
            agentsView origin state

        CollectiblesPage ->
            collectiblesView state

        OrganizationsPage ->
            organizationsView state

        OrganizationDetailPage _ ->
            organizationDetailView state

        UserDetailPage userId ->
            userDetailView origin userId state

        UserWorkPage userId ->
            userTaskListView "Public work" "user-work" userId state.userWork

        UserSubmissionsPage userId ->
            userSubmissionsView userId state

        CollectibleDetailPage collectibleId ->
            collectibleDetailView collectibleId state

        SeriesListPage ->
            seriesListView state

        SeriesDetailPage seriesId ->
            seriesDetailView seriesId state

        TeamDetailPage teamId ->
            teamDetailView teamId state

        AdminPage ->
            adminView state

        InboxPage ->
            inboxView state

        NotFoundPage ->
            Ui.card
                [ Ui.sectionTitle "Page not found"
                , p [ Html.Attributes.class "text-sm text-slate-600" ] [ text "That page does not exist." ]
                , a [ href "#/", Html.Attributes.class Ui.secondaryButtonClass, testId "not-found-home" ] [ text "Go to overview" ]
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
                    , if detail.team.ownerKind == "user" && detail.team.ownerUserID == state.subjectId then
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
        [ Ui.fieldLabel "Search team work"
            [ Ui.textInput [ type_ "search", placeholder "Task title or ID", value state.teamWorkQuery, onInput TeamWorkQueryChanged, testId "team-work-query" ] ]
        , taskTypeFilterSelect "team-work-type" state.teamWorkTypeFilter TeamWorkTypeFilterChanged
        , taskSortSelect "team-work-sort" state.teamWorkSort TeamWorkSortChanged
        , div [ Html.Attributes.class "flex flex-wrap gap-2" ]
            [ Ui.secondaryButton [ type_ "button", onClick SearchTeamWorkClicked, testId "team-work-search" ] "Search"
            ]
        , paginationControls "team-work-page" PreviousTeamWorkPageClicked NextTeamWorkPageClicked state.teamWorkOffset
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
        , teamWorkSection "Review queue" "team-review-queue" "No submissions waiting for team review." reviewTasks
        , teamWorkSection "Ready for team" "team-ready-work" "No team-visible tasks are ready for action." readyForTeam
        , teamWorkSection "Assigned to team" "team-assigned-work" "No tasks are currently assigned to this team." assignedToTeam
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
    chooserButton (selected == tag)
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
    savedView.name
        ++ " · "
        ++ queuePart "query" savedView.query
        ++ " · "
        ++ queuePart "state" savedView.stateFilter
        ++ " · "
        ++ queuePart "type" savedView.typeFilter
        ++ " · "
        ++ savedView.sort


queuePart : String -> String -> String
queuePart name valueText =
    if String.trim valueText == "" then
        name ++ ": any"

    else
        name ++ ": " ++ valueText


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


teamWorkSection : String -> String -> String -> List Task.TaskListItemResponse -> Html Msg
teamWorkSection title identifier emptyMessage tasks =
    div [ Html.Attributes.class "space-y-2", testId identifier ]
        [ Ui.sectionTitle title
        , if List.isEmpty tasks then
            p [ Html.Attributes.class "text-sm text-slate-500", testId (identifier ++ "-empty") ] [ text emptyMessage ]

          else
            div [ Html.Attributes.class "divide-y divide-slate-100" ] (List.map taskRow tasks)
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
                    , div [ Html.Attributes.class "mt-3 space-y-2" ]
                        [ Ui.label_ "Trade to another user"
                        , userPicker "transfer-recipient-id" state.transferRecipientId state.userDirectoryQuery TransferRecipientIdChanged "Choose user" state.userDirectory state.userDirectoryOffset
                        , Ui.primaryButton [ type_ "button", onClick (TransferCollectibleClicked collectible.id), testId "transfer-collectible" ] "Trade"
                        ]
                    ]

            [] ->
                p [ Html.Attributes.class "mt-3 text-sm text-slate-500", testId "collectible-detail-missing" ] [ text "This collectible is no longer in your holdings." ]

        -- Rendered at the card level so a successful trade's confirmation persists
        -- even after the traded collectible leaves your holdings.
        , maybeNote state.transferMessage "transfer-message"
        ]


seriesListView : LoggedInModel -> Html Msg
seriesListView state =
    Ui.card
        [ Ui.sectionTitle "Task series"
        , p [ Html.Attributes.class "text-sm text-slate-600" ] [ text "Group related tasks into an ordered series with its own discussion thread." ]
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
            , Ui.badge series.state
            ]
        , a [ href ("#/series/" ++ series.id), Html.Attributes.class Ui.secondaryButtonClass, testId "open-series" ] [ text "Open" ]
        ]


seriesDetailView : String -> LoggedInModel -> Html Msg
seriesDetailView seriesId state =
    Ui.card
        [ a [ href "#/series", Html.Attributes.class Ui.secondaryButtonClass, testId "back-series" ] [ text "Back to series" ]
        , case state.seriesDetail of
            Just data ->
                let
                    isCreator =
                        data.series.createdBy == state.subjectId
                in
                div [ Html.Attributes.class "mt-3 space-y-4", testId "series-detail" ]
                    [ div [ Html.Attributes.class "space-y-2" ]
                        [ p [ Html.Attributes.class "text-2xl font-semibold", testId "series-detail-title" ] [ text data.series.title ]
                        , Ui.badge data.series.state |> wrapBadge "series-state"
                        , p [ Html.Attributes.class "text-sm text-slate-700" ] [ text data.series.description ]
                        ]
                    , seriesTasksSection seriesId isCreator data
                    , if isCreator then
                        seriesCreatorControls data.series state

                      else
                        text ""
                    , seriesCommentsSection seriesId state data
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


seriesCreatorControls : TaskSeries.TaskSeriesResponse -> LoggedInModel -> Html Msg
seriesCreatorControls series state =
    div [ Html.Attributes.class "space-y-3 rounded-md bg-slate-50 p-4", testId "series-creator-controls" ]
        [ Ui.sectionTitle "Manage series"
        , form [ Html.Attributes.class "space-y-2", onSubmit (UpdateSeriesClicked series.id) ]
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


seriesCommentsSection : String -> LoggedInModel -> SeriesDetailData -> Html Msg
seriesCommentsSection seriesId state data =
    div [ Html.Attributes.class "space-y-2" ]
        [ Ui.sectionTitle "Discussion"
        , if List.isEmpty data.comments then
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
                (List.map (\item -> a [ href ("#/tasks/" ++ item.id), Html.Attributes.class "block py-2 text-sm underline", testId (identifier ++ "-row") ] [ text (item.title ++ " · " ++ taskStateLabel item.state) ]) tasks)
        ]


userSubmissionsView : String -> LoggedInModel -> Html Msg
userSubmissionsView userId state =
    let
        submissions =
            state.userSubmissions

        revisionItems =
            List.filter isRevisionSubmission submissions
    in
    Ui.card
        [ a [ href ("#/users/" ++ userId), Html.Attributes.class Ui.secondaryButtonClass, testId "back-user" ] [ text "Back to profile" ]
        , Ui.sectionTitle "Submissions"
        , Ui.sectionTitle "Revision inbox"
        , if List.isEmpty revisionItems then
            p [ Html.Attributes.class "text-sm text-slate-500", testId "revision-inbox-empty" ] [ text "No requested revisions." ]

          else
            div [ Html.Attributes.class "divide-y divide-slate-100", testId "revision-inbox" ]
                (List.map revisionSubmissionRow revisionItems)
        , if List.isEmpty submissions then
            p [ Html.Attributes.class "text-sm text-slate-500", testId "user-submissions-empty" ] [ text "No submissions." ]

          else
            div [ Html.Attributes.class "divide-y divide-slate-100", testId "user-submissions" ]
                (List.map userSubmissionRow submissions)
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
        [ Ui.sectionTitle "Revision timeline"
        , if List.isEmpty submissions then
            p [ Html.Attributes.class "text-sm text-slate-500", testId "revision-timeline-empty" ] [ text "No submission history." ]

          else
            div [ Html.Attributes.class "divide-y divide-slate-100" ] (List.map revisionTimelineRow submissions)
        ]


revisionTimelineRow : Submission.SubmissionResponse -> Html Msg
revisionTimelineRow item =
    div [ Html.Attributes.class "space-y-1 py-2", testId "revision-timeline-row" ]
        [ div [ Html.Attributes.class "flex flex-wrap items-center gap-2" ]
            [ Ui.badge (submissionStateLabel item.state)
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
                [ a [ href ("#/users/" ++ userId ++ "/work"), Html.Attributes.class Ui.secondaryButtonClass, testId "user-work-link" ] [ text "Public work" ]
                , a [ href ("#/users/" ++ userId ++ "/submissions"), Html.Attributes.class Ui.secondaryButtonClass, testId "user-submissions-link" ] [ text "Submissions" ]
                ]
            , Ui.sectionTitle "Public tasks"
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
        , div [ Html.Attributes.class "space-y-2" ]
            [ Ui.label_ "Email verification"
            , Ui.secondaryButton [ type_ "button", onClick RequestEmailVerificationClicked, testId "request-email-verification" ] "Create verification token"
            , Ui.textInput [ type_ "text", placeholder "Verification token", value state.emailVerificationInput, onInput EmailVerificationInputChanged, testId "email-verification-token" ]
            , Ui.secondaryButton [ type_ "button", onClick ConfirmEmailVerificationClicked, testId "confirm-email-verification" ] "Verify email"
            ]
        , form [ Html.Attributes.class "space-y-2", onSubmit ChangePasswordClicked ]
            [ Ui.fieldLabel "Current password" [ Ui.textInput [ type_ "password", value state.currentPassword, onInput CurrentPasswordChanged, testId "current-password" ] ]
            , Ui.fieldLabel "New password" [ Ui.textInput [ type_ "password", value state.newPassword, onInput NewPasswordChanged, testId "new-password" ] ]
            , Ui.primaryButton [ type_ "submit", testId "change-password" ] "Change password"
            ]
        , div [ Html.Attributes.class "space-y-2" ]
            [ Ui.label_ "Privacy requests"
            , div [ Html.Attributes.class "flex flex-wrap gap-2" ]
                [ Ui.secondaryButton [ type_ "button", onClick (PrivacyRequestClicked Privacy.PrivacyRequestKindDataExport), testId "request-data-export" ] "Request data export"
                , Ui.secondaryButton [ type_ "button", onClick (PrivacyRequestClicked Privacy.PrivacyRequestKindSensitiveFieldDeletion), testId "request-sensitive-deletion" ] "Request sensitive-field deletion"
                ]
            ]
        , Ui.dangerButton [ type_ "button", onClick DeactivateAccountClicked, testId "deactivate-account" ] "Deactivate account"
        , maybeNote state.accountMessage "account-message"
        ]


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
        , ledgerView state.entries
        ]


ownerChooser : LoggedInModel -> Html Msg
ownerChooser state =
    if List.isEmpty state.organizations then
        text ""

    else
        div []
            [ Ui.label_ "Owner"
            , div [ Html.Attributes.class "flex flex-wrap gap-2", testId "create-owner" ]
                (chooserButton (state.createTaskOwner == "") (CreateTaskOwnerChanged "") "create-owner-me" "Me"
                    :: List.map (ownerButton state.createTaskOwner) state.organizations
                )
            ]


ownerButton : String -> Organization.OrganizationResponse -> Html Msg
ownerButton selected organization =
    chooserButton (selected == organization.id)
        (CreateTaskOwnerChanged organization.id)
        ("create-owner-" ++ organization.id)
        organization.name


organizationsView : LoggedInModel -> Html Msg
organizationsView state =
    Ui.card
        [ Ui.sectionTitle "Organizations"
        , p [ Html.Attributes.class "text-sm text-slate-600" ] [ text "Organizations you belong to. Create one to own tasks and credits as a team." ]
        , organizationsList state
        , form [ Html.Attributes.class "mt-3 flex flex-wrap items-end gap-2", onSubmit CreateOrgClicked ]
            [ Ui.fieldLabel "New organization"
                [ Ui.textInput [ type_ "text", placeholder "Organization name", value state.createOrgName, onInput CreateOrgNameChanged, testId "create-org-name" ] ]
            , Ui.primaryButton [ type_ "submit", testId "create-org" ] "Create organization"
            ]
        , maybeNote state.orgMessage "org-message"
        ]


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
            [ Ui.label_ ("Balance: " ++ balanceLabel state.orgBalance)
            , organizationOperationsDashboard state
            , Ui.sectionTitle "Organization tasks"
            , orgTaskControls state
            , tasksListSimple "org-tasks" state.orgTasks
            , maybeNote state.orgTaskMessage "org-task-message"
            , Ui.sectionTitle "Teams"
            , orgTeamsList state.orgTeams
            , form [ Html.Attributes.class "flex flex-wrap items-end gap-2", onSubmit CreateOrgTeamClicked ]
                [ Ui.fieldLabel "New team"
                    [ Ui.textInput [ type_ "text", placeholder "Team name", value state.createOrgTeamName, onInput CreateOrgTeamNameChanged, testId "create-org-team-name" ] ]
                , Ui.primaryButton [ type_ "submit", testId "create-org-team" ] "Create team"
                ]
            , maybeNote state.orgTeamMessage "org-team-message"
            , Ui.sectionTitle "Members"
            , orgMembersList state.orgMembers
            , Ui.sectionTitle "Provision a member"
            , form [ Html.Attributes.class "flex flex-wrap items-end gap-2", onSubmit ProvisionMemberClicked ]
                [ Ui.fieldLabel "Member email"
                    [ Ui.textInput [ type_ "email", placeholder "person@example.com", value state.provisionMemberEmail, onInput ProvisionMemberEmailChanged, testId "provision-member-email" ] ]
                , provisionRolePicker state.provisionMemberRoles
                , Ui.primaryButton [ type_ "submit", testId "provision-member" ] "Provision member"
                ]
            , maybeNote state.provisionMemberMessage "provision-member-message"
            , Ui.sectionTitle "Collectibles"
            , collectiblesHoldingsList "org-collectibles" state.orgCollectibles
            , maybeNote state.orgCollectiblesMessage "org-collectibles-message"
            ]


organizationOperationsDashboard : LoggedInModel -> Html Msg
organizationOperationsDashboard state =
    div [ Html.Attributes.class "space-y-3 rounded-md border border-slate-200 bg-white p-3", testId "org-operations-dashboard" ]
        [ Ui.sectionTitle "Operations"
        , div [ Html.Attributes.class "grid gap-2 sm:grid-cols-2" ]
            [ operationMetric "Balance" (balanceLabel state.orgBalance) "org-ops-balance"
            , operationMetric "Teams" (String.fromInt (List.length state.orgTeams)) "org-ops-teams"
            , operationMetric "Active members" (String.fromInt (countMembers Organization.MembershipStatusActive state.orgMembers)) "org-ops-members-active"
            , operationMetric "Inactive members" (String.fromInt (inactiveMemberCount state.orgMembers)) "org-ops-members-inactive"
            , operationMetric "Collectibles" (String.fromInt (List.length state.orgCollectibles)) "org-ops-collectibles"
            , operationMetric "Draft tasks" (String.fromInt (countTasks Task.TaskStateDraft state.orgTasks)) "org-ops-tasks-draft"
            , operationMetric "Open tasks" (String.fromInt (countTasks Task.TaskStateOpen state.orgTasks)) "org-ops-tasks-open"
            , operationMetric "Closed tasks" (String.fromInt (countTasks Task.TaskStateClosed state.orgTasks)) "org-ops-tasks-closed"
            ]
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
        , paginationControls "org-tasks-page" PreviousOrgTasksPageClicked NextOrgTasksPageClicked state.orgTaskOffset
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
    chooserButton (selected == tag)
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
        , div [ Html.Attributes.class "flex flex-wrap gap-2" ]
            [ Ui.secondaryButton [ type_ "button", onClick (UpdateMemberRolesClicked member.userID [ "member" ]), testId "member-role-member" ] "Member"
            , Ui.secondaryButton [ type_ "button", onClick (UpdateMemberRolesClicked member.userID [ "member", "reviewer" ]), testId "member-role-reviewer" ] "Reviewer"
            , Ui.secondaryButton [ type_ "button", onClick (UpdateMemberRolesClicked member.userID [ "admin" ]), testId "member-role-admin" ] "Admin"
            , Ui.secondaryButton [ type_ "button", onClick (DeactivateMemberClicked member.userID), testId "deactivate-member" ] "Deactivate"
            ]
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


balanceView : Maybe Int -> Html Msg
balanceView balance =
    Ui.card
        [ Ui.label_ "Balance"
        , p [ Html.Attributes.class "text-3xl font-semibold", testId "balance" ] [ text (balanceLabel balance) ]
        ]


balanceLabel : Maybe Int -> String
balanceLabel balance =
    case balance of
        Just amount ->
            String.fromInt amount ++ " credits"

        Nothing ->
            "Loading…"


createTaskView : LoggedInModel -> Html Msg
createTaskView state =
    form [ Html.Attributes.class "space-y-4 rounded-lg border border-slate-200 bg-white p-6 shadow-sm", onSubmit CreateTaskClicked ]
        [ Ui.sectionTitle "Create a task"
        , Ui.fieldLabel "Title" [ Ui.textInput [ type_ "text", placeholder "Short, descriptive title", value state.createTitle, onInput CreateTitleChanged, testId "create-title" ] ]
        , Ui.fieldLabel "Template" [ taskTypeSelect state.createTaskType ]
        , Ui.fieldLabel "Reference URL (optional, e.g. a pull request)" [ Ui.textInput [ type_ "text", placeholder "https://github.com/org/repo/pull/123", value state.createReferenceURL, onInput CreateReferenceURLChanged, testId "create-reference-url" ] ]
        , Ui.fieldLabel "Description" [ Ui.textarea_ [ placeholder "What the worker should do", value state.createDescription, onInput CreateDescriptionChanged, Html.Attributes.rows 3, testId "create-description" ] ]
        , if state.createTaskType == "general" then
            schemaDesignerView state

          else
            p [ Html.Attributes.class "text-xs text-slate-600", testId "template-schema-note" ]
                [ text ("The " ++ taskTypeLabel state.createTaskType ++ " template prefilled the description and response schema below — edit them if you need to.") ]
        , Ui.fieldLabel "Response schema (JSON, advanced)" [ Ui.textarea_ [ placeholder "{\"kind\":\"freeform\"}", value state.createResponseSchema, onInput CreateResponseSchemaChanged, Html.Attributes.rows 3, testId "create-response-schema" ] ]
        , Ui.fieldLabel "Task input (JSON, optional)" [ Ui.textarea_ [ placeholder "Embed any data the worker needs, or leave blank", value state.createPayloadJson, onInput CreatePayloadChanged, Html.Attributes.rows 3, testId "create-payload" ] ]
        , Ui.label_ "Reward"
        , div [ Html.Attributes.class "flex flex-wrap gap-2" ] (List.map (rewardKindButton state.createRewardKind) allRewardKinds)
        , rewardAmountField state
        , rewardCollectibleField state
        , ownerChooser state
        , Ui.label_ "Participation"
        , div [ Html.Attributes.class "flex flex-wrap gap-2" ] (List.map (participationButton state.createParticipationPolicy) allParticipationPolicies)
        , if participationUsesReservation state.createParticipationPolicy then
            Ui.fieldLabel "Reservation expiry (hours)" [ Ui.textInput [ type_ "number", placeholder "48", value state.createReservationHours, onInput CreateReservationHoursChanged, testId "create-reservation-hours" ] ]

          else
            text ""
        , Ui.label_ "Visibility"
        , div [ Html.Attributes.class "flex flex-wrap gap-2" ] (List.map (visibilityButton state.createVisibility) allVisibilityTags)
        , visibilityScopeField state
        , Ui.label_ "Assignee"
        , div [ Html.Attributes.class "flex flex-wrap gap-2" ] (List.map (assigneeScopeButton state.createAssigneeScope) allAssigneeScopes)
        , Ui.primaryButton [ type_ "submit", testId "create-task" ] "Create task"
        , maybeNote state.createMessage "create-message"
        ]


allRewardKinds : List String
allRewardKinds =
    [ "none", "credit", "collectible", "bundle" ]


rewardKindButton : String -> String -> Html Msg
rewardKindButton selected kind =
    chooserButton (selected == kind)
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
        Ui.fieldLabel "Credit amount" [ Ui.textInput [ type_ "number", placeholder "Amount in credits", value state.createRewardAmount, onInput CreateRewardAmountChanged, testId "create-reward" ] ]

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
    chooserButton (selected == tag)
        (CreateVisibilityChanged tag)
        ("create-visibility-" ++ tag)
        (visibilityLabel tag)


allAssigneeScopes : List Task.TaskAssigneeScope
allAssigneeScopes =
    [ Task.TaskAssigneeScopeUser, Task.TaskAssigneeScopeOrganizationTeam ]


assigneeScopeButton : Task.TaskAssigneeScope -> Task.TaskAssigneeScope -> Html Msg
assigneeScopeButton selected scope =
    chooserButton (selected == scope)
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
    chooserButton (selectedPolicy == participationPolicyTag policy)
        (CreateParticipationChanged (participationPolicyTag policy))
        ("create-participation-" ++ participationPolicyTag policy)
        (participationPolicyLabel policy)


ledgerView : List Ledger.LedgerEntryResponse -> Html Msg
ledgerView entries =
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
        , taskPicker "fund-task-id" state.fundTaskId FundTaskIdChanged state.tasks
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
        (option [ value "" ] [ text "Select task" ] :: List.map (taskOption selectedTaskId) tasks)


taskOption : String -> Task.TaskListItemResponse -> Html Msg
taskOption selectedTaskId item =
    option [ value item.id, selected (selectedTaskId == item.id) ]
        [ text (item.title ++ " · " ++ taskStateLabel item.state ++ " · " ++ rewardLabel item.rewardKind item.rewardCreditAmount item.rewardCollectibleCount) ]


tasksView : String -> LoggedInModel -> Html Msg
tasksView origin state =
    let
        visibleTasks =
            filterTasksByQuery state.taskListQuery state.tasks
    in
    Ui.card
        [ Ui.sectionTitle "My tasks"
        , Ui.label_ "Filter by state"
        , div [ Html.Attributes.class "flex flex-wrap gap-2", testId "task-filter" ] (List.map (taskFilterButton state.taskStateFilter) taskStateFilterOptions)
        , Ui.fieldLabel "Search loaded tasks"
            [ Ui.textInput [ type_ "search", placeholder "Task title or ID", value state.taskListQuery, onInput TaskListQueryChanged, testId "tasks-query" ] ]
        , taskTypeFilterSelect "tasks-type" state.taskListTypeFilter TaskListTypeFilterChanged
        , taskSortSelect "tasks-sort" state.taskListSort TaskListSortChanged
        , paginationControls "tasks-page" PreviousTasksPageClicked NextTasksPageClicked state.taskListOffset
        , tasksList visibleTasks
        ]


taskStateFilterOptions : List ( String, String )
taskStateFilterOptions =
    [ ( "", "All" )
    , ( "open", "Open" )
    , ( "draft", "Draft" )
    , ( "closed", "Closed" )
    ]


taskFilterButton : String -> ( String, String ) -> Html Msg
taskFilterButton selected ( tag, labelText ) =
    chooserButton (selected == tag)
        (TaskStateFilterChanged tag)
        ("task-filter-"
            ++ (if tag == "" then
                    "all"

                else
                    tag
               )
        )
        labelText


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


organizationPicker : String -> String -> String -> (String -> Msg) -> (String -> Msg) -> Msg -> Msg -> Msg -> String -> List Organization.OrganizationResponse -> Int -> Html Msg
organizationPicker identifier selectedOrganizationId query change queryChange search previous next blankLabel organizations offset =
    div [ Html.Attributes.class "space-y-2" ]
        [ selectorSearchControls identifier "Search organizations" query queryChange search previous next offset
        , select
            [ Html.Attributes.class Ui.fieldClass, value selectedOrganizationId, onInput change, testId identifier ]
            (option [ value "" ] [ text blankLabel ]
                :: List.map (\organization -> option [ value organization.id, selected (selectedOrganizationId == organization.id) ] [ text organization.name ]) organizations
            )
        ]


userPicker : String -> String -> String -> (String -> Msg) -> String -> List UserDirectoryEntry -> Int -> Html Msg
userPicker identifier selectedUserId query change blankLabel users offset =
    div [ Html.Attributes.class "space-y-2" ]
        [ selectorSearchControls identifier "Search users" query UserDirectoryQueryChanged SearchUserDirectoryClicked PreviousUserDirectoryPageClicked NextUserDirectoryPageClicked offset
        , select
            [ Html.Attributes.class Ui.fieldClass, value selectedUserId, onInput change, testId identifier ]
            (option [ value "" ] [ text blankLabel ]
                :: List.map (\user -> option [ value user.id, selected (selectedUserId == user.id) ] [ text user.email ]) users
            )
        ]


teamPicker : String -> String -> String -> (String -> Msg) -> (String -> Msg) -> Msg -> Msg -> Msg -> String -> List Team.TeamResponse -> Int -> Html Msg
teamPicker identifier selectedTeamId query change queryChange search previous next blankLabel teams offset =
    div [ Html.Attributes.class "space-y-2" ]
        [ selectorSearchControls identifier "Search teams" query queryChange search previous next offset
        , select
            [ Html.Attributes.class Ui.fieldClass, value selectedTeamId, onInput change, testId identifier ]
            (option [ value "" ] [ text blankLabel ]
                :: List.map (\team -> option [ value team.id, selected (selectedTeamId == team.id) ] [ text team.name ]) teams
            )
        ]


selectorSearchControls : String -> String -> String -> (String -> Msg) -> Msg -> Msg -> Msg -> Int -> Html Msg
selectorSearchControls identifier placeholderText query queryChange search previous next offset =
    div [ Html.Attributes.class "space-y-2" ]
        [ div [ Html.Attributes.class "flex flex-wrap gap-2" ]
            [ Ui.textInput [ type_ "search", placeholder placeholderText, value query, onInput queryChange, testId (identifier ++ "-query") ]
            , Ui.secondaryButton [ type_ "button", onClick search, testId (identifier ++ "-search") ] "Search"
            ]
        , div [ Html.Attributes.class "flex flex-wrap items-center gap-2 text-xs text-slate-500" ]
            [ Ui.secondaryButton [ type_ "button", disabled (offset == 0), onClick previous, testId (identifier ++ "-previous") ] "Previous"
            , span [ testId (identifier ++ "-offset") ] [ text ("Offset " ++ String.fromInt offset) ]
            , Ui.secondaryButton [ type_ "button", onClick next, testId (identifier ++ "-next") ] "Next"
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


tasksList : List Task.TaskListItemResponse -> Html Msg
tasksList tasks =
    if List.isEmpty tasks then
        p [ Html.Attributes.class "text-sm text-slate-500", testId "tasks-empty" ] [ text "No tasks yet." ]

    else
        div [ Html.Attributes.class "divide-y divide-slate-100", testId "tasks" ] (List.map taskRow tasks)


taskRow : Task.TaskListItemResponse -> Html Msg
taskRow item =
    div [ Html.Attributes.class "flex items-center justify-between gap-3 py-2", testId "task-row" ]
        [ div [ Html.Attributes.class "min-w-0" ]
            [ p [ Html.Attributes.class "font-medium break-words" ] [ text item.title ]
            , p [ Html.Attributes.class "text-xs text-slate-500 break-words" ] [ text (taskStateLabel item.state ++ " · " ++ rewardLabel item.rewardKind item.rewardCreditAmount item.rewardCollectibleCount ++ activeAssigneeSuffix item) ]
            ]
        , a [ href ("#/tasks/" ++ item.id), Html.Attributes.class (Ui.secondaryButtonClass ++ " shrink-0"), testId "view-task" ] [ text "View" ]
        ]


agentsView : String -> LoggedInModel -> Html Msg
agentsView origin state =
    Ui.card
        [ Ui.sectionTitle "Agent setup"
        , p [ Html.Attributes.class "text-sm text-slate-600" ] [ text "Create a scoped credential for a local MCP agent." ]
        , form [ Html.Attributes.class "mt-3 space-y-3", onSubmit CreateAgentClicked ]
            [ Ui.textInput [ type_ "text", placeholder "Agent label", value state.agentLabel, onInput AgentLabelChanged, testId "agent-label" ]
            , div [ Html.Attributes.class "space-y-1" ] (List.map (scopeCheckbox state.agentScopes) allScopes)
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
            , p [ Html.Attributes.class "text-xs text-slate-500" ] [ text (credentialStateLabel credential.state ++ " · " ++ String.join ", " (List.map scopeLabel credential.scopes)) ]
            ]
        , revokeButton credential
        ]


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
        [ Ui.sectionTitle "Collectibles"
        , p [ Html.Attributes.class "text-sm text-slate-600" ] [ text "Mint your own collectibles, award default collectibles to users, teams, or organizations, and trade collectibles to other users." ]
        , mintForm state
        , awardForm state
        , if state.isAdmin then
            awardRecipientControl state

          else
            text ""
        , catalogGallery state
        , collectiblesList state
        ]


awardRecipientControl : LoggedInModel -> Html Msg
awardRecipientControl state =
    div [ Html.Attributes.class "mt-4 space-y-3" ]
        [ Ui.label_ "Admin: award a default collectible"
        , p [ Html.Attributes.class "text-xs text-slate-600", testId "award-admin-note" ] [ text "Awarding default collectibles requires a platform administrator (enabled in the demo)." ]
        , div [ Html.Attributes.class "flex flex-wrap gap-2" ]
            [ chooserButton (state.awardRecipientKind == "user") (AwardRecipientKindChanged "user") "award-kind-user" "User"
            , chooserButton (state.awardRecipientKind == "team") (AwardRecipientKindChanged "team") "award-kind-team" "Team"
            , chooserButton (state.awardRecipientKind == "organization") (AwardRecipientKindChanged "organization") "award-kind-organization" "Organization"
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
    chooserButton (selected == kind)
        (CollectibleKindChosen kind)
        ("collectible-kind-" ++ collectibleKindTag kind)
        (collectibleKindLabel kind)


policyButton : Collectible.CollectibleTransferPolicy -> Collectible.CollectibleTransferPolicy -> Html Msg
policyButton selected policy =
    chooserButton (selected == policy)
        (CollectiblePolicyChosen policy)
        ("collectible-policy-" ++ collectiblePolicyTag policy)
        (collectiblePolicyLabel policy)


chooserButton : Bool -> Msg -> String -> String -> Html Msg
chooserButton isSelected msg identifier labelText =
    if isSelected then
        Ui.primaryButton [ type_ "button", onClick msg, Html.Attributes.attribute "aria-pressed" "true", testId identifier ] labelText

    else
        Ui.secondaryButton [ type_ "button", onClick msg, Html.Attributes.attribute "aria-pressed" "false", testId identifier ] labelText


awardForm : LoggedInModel -> Html Msg
awardForm state =
    div [ Html.Attributes.class "mt-4 space-y-3" ]
        [ Ui.label_ "Award a collectible to a task"
        , taskPicker "award-task-id" state.awardTaskId AwardTaskIdChanged state.tasks
        , p [ Html.Attributes.class "text-xs text-slate-500" ] [ text "Choose the task here, then press Award next to a collectible below." ]
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
            , Ui.badge (collectibleStateLabel collectible.state)
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


discoveryView : LoggedInModel -> Html Msg
discoveryView state =
    let
        visibleTasks =
            filterTasksByQuery state.discoveryQuery state.discoveryTasks
    in
    Ui.card
        [ Ui.sectionTitle "Discover public tasks"
        , Ui.checkbox [ checked state.discoveryIncludeReserved, onClick (DiscoveryIncludeReservedChanged (not state.discoveryIncludeReserved)), testId "include-reserved" ] "Include reserved"
        , Ui.fieldLabel "Search loaded discovery"
            [ Ui.textInput [ type_ "search", placeholder "Task title or ID", value state.discoveryQuery, onInput DiscoveryQueryChanged, testId "discovery-query" ] ]
        , paginationControls "discovery-page" PreviousDiscoveryPageClicked NextDiscoveryPageClicked state.discoveryOffset
        , discoveryList visibleTasks
        ]


discoveryList : List Task.TaskListItemResponse -> Html Msg
discoveryList tasks =
    if List.isEmpty tasks then
        p [ Html.Attributes.class "text-sm text-slate-500", testId "discovery-empty" ] [ text "No public tasks available." ]

    else
        div [ Html.Attributes.class "divide-y divide-slate-100", testId "discovery-tasks" ] (List.map discoveryRow tasks)


discoveryRow : Task.TaskListItemResponse -> Html Msg
discoveryRow item =
    div [ Html.Attributes.class "flex items-center justify-between gap-3 py-2", testId "discovery-task-row" ]
        [ div [ Html.Attributes.class "min-w-0" ]
            [ p [ Html.Attributes.class "font-medium break-words" ] [ text item.title ]
            , p [ Html.Attributes.class "text-xs text-slate-500 break-words" ] [ text (taskStateLabel item.state ++ " · " ++ rewardLabel item.rewardKind item.rewardCreditAmount item.rewardCollectibleCount ++ " · " ++ participationPolicyLabel item.participationPolicy ++ activeAssigneeSuffix item) ]
            ]
        , div [ Html.Attributes.class "shrink-0" ] [ Ui.secondaryButton [ onClick (DiscoveryViewClicked item.id), testId "discovery-view" ] "View" ]
        ]


paginationControls : String -> Msg -> Msg -> Int -> Html Msg
paginationControls identifier previous next offset =
    div [ Html.Attributes.class "flex flex-wrap items-center gap-2 text-xs text-slate-600", testId identifier ]
        [ Ui.secondaryButton [ type_ "button", disabled (offset == 0), onClick previous, testId (identifier ++ "-previous") ] "Previous"
        , span [ testId (identifier ++ "-offset") ] [ text ("Offset " ++ String.fromInt offset) ]
        , Ui.secondaryButton [ type_ "button", onClick next, testId (identifier ++ "-next") ] "Next"
        ]



-- Task detail page


taskDetailPageView : String -> LoggedInModel -> Html Msg
taskDetailPageView origin state =
    let
        isOwner =
            state.detail |> Maybe.map (\detail -> detail.createdBy == state.subjectId) |> Maybe.withDefault False

        canReview =
            state.detail |> Maybe.map (\detail -> detail.reviewerAction == "review") |> Maybe.withDefault False

        backHref =
            if isOwner then
                "#/tasks"

            else
                "#/discovery"
    in
    div [ Html.Attributes.class "space-y-6" ]
        ([ a [ href backHref, Html.Attributes.class Ui.secondaryButtonClass, testId "detail-back" ] [ text "Back" ]
         , detailCard origin state
         ]
            ++ (if isOwner then
                    [ ownerControlsCard state, submissionsCard state ]

                else if canReview then
                    [ submissionsCard state ]

                else
                    [ reservationCard state, submitCard state, mySubmissionsCard state ]
               )
            ++ [ taskCommentsCard state ]
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
    if (detail.payloadKind == "inline" || detail.payloadKind == "json") && detail.payloadJson /= "" then
        [ Ui.label_ "Task input", Ui.codeBlock [ testId "detail-input" ] detail.payloadJson ]

    else
        []


ownerControlsCard : LoggedInModel -> Html Msg
ownerControlsCard state =
    case state.detail of
        Just detail ->
            let
                draftOrOpen =
                    detail.state == Task.TaskStateDraft || detail.state == Task.TaskStateOpen

                -- The credit refund endpoint (/refund) is the unified refund: it
                -- returns held credits AND held collectibles together (so it
                -- handles bundle rewards in one shot). The collectible-refund
                -- endpoint only handles collectible-only tasks (it 409s on bundle).
                -- So: credit and bundle -> /refund; collectible-only -> /collectible-refund.
                refundButton =
                    if draftOrOpen && detail.rewardKind == "credit" then
                        Just (Ui.secondaryButton [ type_ "button", onClick (RefundTaskClicked detail.id), testId "refund-task" ] "Refund credits")

                    else if draftOrOpen && detail.rewardKind == "bundle" then
                        Just (Ui.secondaryButton [ type_ "button", onClick (RefundTaskClicked detail.id), testId "refund-task" ] "Refund reward")

                    else if draftOrOpen && detail.rewardKind == "collectible" then
                        Just (Ui.secondaryButton [ type_ "button", onClick (RefundCollectibleRewardClicked detail.id), testId "refund-collectible" ] "Refund collectible")

                    else
                        Nothing

                buttons =
                    List.filterMap identity
                        [ if detail.state == Task.TaskStateDraft then
                            Just (Ui.secondaryButton [ type_ "button", onClick (OpenTaskClicked detail.id), testId "open-task" ] "Open")

                          else
                            Nothing
                        , -- Cancel ends the task. Offered for drafts (any reward) and for
                          -- open tasks with no reward. Reward-bearing OPEN tasks are ended
                          -- via Refund instead, since Cancel of a funded task would orphan
                          -- the held escrow (the backend's Refund returns escrow; Cancel does not).
                          if detail.state == Task.TaskStateDraft || (detail.state == Task.TaskStateOpen && detail.rewardKind == "none") then
                            Just (Ui.secondaryButton [ type_ "button", onClick (CancelTaskClicked detail.id), testId "cancel-task" ] "Cancel")

                          else
                            Nothing
                        , refundButton
                        ]
            in
            Ui.card
                [ Ui.sectionTitle "Owner controls"
                , p [ Html.Attributes.class "rounded-md bg-slate-100 px-3 py-2 text-sm text-slate-700", testId "task-guidance" ] [ text (taskStateGuidance detail.state) ]
                , div [ Html.Attributes.class "flex flex-wrap gap-2" ] buttons
                , maybeNote state.taskActionMessage "task-action-message"
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
                    ([ Ui.badge (taskStateLabel detail.state)
                     , Ui.badge (availabilityKindLabel detail.availabilityKind)
                     , Ui.badge (participationPolicyLabel detail.participationPolicy)
                     ]
                        ++ taskTypeBadge detail
                    )
                , p [ Html.Attributes.class "text-sm font-medium" ] [ text ("Reward: " ++ rewardLabel detail.rewardKind detail.rewardCreditAmount detail.rewardCollectibleCount) ]
                , p [ Html.Attributes.class "text-sm text-slate-700" ] [ text detail.description ]
                ]
                    ++ referenceBlock detail
                    ++ seriesLinkBlock detail
                    ++ taskInputBlock detail
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
            Ui.card
                [ Ui.sectionTitle "Reservation"
                , div [ Html.Attributes.class "flex flex-wrap items-center gap-2" ]
                    [ Ui.badge (viewerActionLabel detail.viewerAction)
                    , Ui.badge (assigneeScopeLabel detail.assigneeScope)
                    ]
                , reservationAction state detail
                , reservationsList state.reservations
                , maybeNote state.reservationMessage "reservation-message"
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

        _ ->
            text ""


reservationsList : List Task.TaskReservationResponse -> Html Msg
reservationsList reservations =
    if List.isEmpty reservations then
        text ""

    else
        div [ Html.Attributes.class "divide-y divide-slate-100", testId "reservations" ] (List.map reservationRow reservations)


reservationRow : Task.TaskReservationResponse -> Html Msg
reservationRow reservation =
    div [ Html.Attributes.class "flex items-center justify-between gap-3 py-2", testId "reservation-row" ]
        [ div []
            [ p [ Html.Attributes.class "text-sm font-medium" ] [ text (reservation.assigneeID ++ " · " ++ assigneeScopeLabel reservation.assigneeKind) ]
            , p [ Html.Attributes.class "text-xs text-slate-500" ] [ text (reservationStateLabel reservation.state) ]
            ]
        , div [ Html.Attributes.class "flex flex-wrap gap-2" ] (reservationButtons reservation)
        ]


reservationButtons : Task.TaskReservationResponse -> List (Html Msg)
reservationButtons reservation =
    case reservation.state of
        Task.TaskReservationStateRequested ->
            [ Ui.primaryButton [ type_ "button", onClick (ApproveReservationClicked reservation.id), testId "approve-reservation" ] "Approve"
            , Ui.secondaryButton [ type_ "button", onClick (DeclineReservationClicked reservation.id), testId "decline-reservation" ] "Decline"
            ]

        Task.TaskReservationStateActive ->
            [ Ui.secondaryButton [ type_ "button", onClick (CancelReservationClicked reservation.id), testId "cancel-reservation" ] "Cancel" ]

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
    form [ Html.Attributes.class "space-y-4 rounded-lg border border-slate-200 bg-white p-6 shadow-sm", onSubmit SubmitClicked ]
        [ Ui.sectionTitle "Submit a response"
        , Ui.textarea_
            [ placeholder "{}"
            , value state.submitInput
            , onInput SubmitInputChanged
            , Html.Attributes.rows 6
            , testId "detail-submit-input"
            ]
        , Ui.primaryButton [ type_ "submit", testId "detail-submit" ] "Submit response"
        , maybeNote state.submitMessage "detail-submit-message"
        ]


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
            [ Ui.badge (submissionStateLabel submission.state)
            , Ui.secondaryButton [ type_ "button", onClick (OpenSubmissionComments submission.id), testId "my-submission-comments-toggle" ] (discussionButtonLabel state submission.id)
            ]
        , reviewNoteView submission.reviewNote
        , Ui.codeBlock [ testId "my-submission-response" ] submission.responseJSON
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
        [ label [ Html.Attributes.class "grid gap-1" ]
            [ span [ Html.Attributes.class "text-xs font-semibold text-slate-600" ] [ text "Review note" ]
            , Html.textarea
                [ Html.Attributes.class "min-h-20 rounded border border-slate-300 px-3 py-2 text-sm"
                , Html.Attributes.rows 3
                , value state.reviewNote
                , onInput ReviewNoteChanged
                , testId "review-note"
                ]
                []
            ]
        , div [ Html.Attributes.class "grid gap-2 sm:grid-cols-3" ]
            [ label [ Html.Attributes.class "grid gap-1" ]
                [ span [ Html.Attributes.class "text-xs font-semibold text-slate-600" ] [ text "Partial payout" ]
                , Html.input [ Html.Attributes.class "rounded border border-slate-300 px-3 py-2 text-sm", type_ "number", value state.reviewPartialCredit, onInput ReviewPartialCreditChanged, testId "review-partial-credit" ] []
                ]
            , label [ Html.Attributes.class "grid gap-1" ]
                [ span [ Html.Attributes.class "text-xs font-semibold text-slate-600" ] [ text "Tip" ]
                , Html.input [ Html.Attributes.class "rounded border border-slate-300 px-3 py-2 text-sm", type_ "number", value state.reviewTip, onInput ReviewTipChanged, testId "review-tip" ] []
                ]
            , div [ Html.Attributes.class "pt-6" ]
                [ Ui.checkbox [ checked state.reviewBan, onCheck ReviewBanChanged, testId "review-ban" ] "Ban implementor" ]
            ]
        , label [ Html.Attributes.class "grid gap-1" ]
            [ span [ Html.Attributes.class "text-xs font-semibold text-slate-600" ] [ text "Tip a collectible (optional)" ]
            , select
                [ Html.Attributes.class Ui.fieldClass
                , value state.reviewTipCollectibleId
                , onInput ReviewTipCollectibleChanged
                , testId "review-tip-collectible"
                ]
                (option [ value "" ] [ text "No collectible tip" ]
                    :: List.map
                        (\c -> option [ value c.id, selected (state.reviewTipCollectibleId == c.id) ]
                            [ text (c.name ++ " · " ++ collectibleKindLabel c.kind) ]
                        )
                        state.collectibles
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
            [ Ui.badge (submissionStateLabel submission.state)
            , reviewButtons state submission
            ]
        , p [ Html.Attributes.class "text-xs text-slate-500" ] [ text ("Submitter: " ++ submission.submitterID) ]
        , reviewNoteView submission.reviewNote
        , Ui.codeBlock [ testId "submission-response" ] submission.responseJSON
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
                [ Ui.secondaryButton [ onClick (RequestChangesClicked submission.id), disabled (String.trim state.reviewNote == ""), testId "request-changes" ] "Request changes"
                , Ui.secondaryButton [ onClick (RejectClicked submission.id), disabled (String.trim state.reviewNote == ""), testId "reject-submission" ] "Reject"
                , Ui.primaryButton [ onClick (AcceptClicked submission.id), testId "accept-submission" ] "Accept"
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
        [ text (field.path ++ " · " ++ field.category ++ " · " ++ field.retention ++ " · " ++ field.redaction) ]



-- Labels and helpers


maybeError : Maybe String -> String -> Html Msg
maybeError message identifier =
    case message of
        Just value ->
            Ui.errorText identifier value

        Nothing ->
            text ""


maybeNote : Maybe String -> String -> Html Msg
maybeNote message identifier =
    case message of
        Just value ->
            Ui.noteText identifier value

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
    let
        indicator =
            if state.taskIntegrationOpen then
                " ▾"

            else
                " ▸"

        body =
            if state.taskIntegrationOpen then
                taskIntegrationBody origin taskId state

            else
                text ""
    in
    div [ Html.Attributes.class "space-y-3", testId "task-instructions" ]
        [ Ui.secondaryButton [ onClick ToggleTaskIntegration, testId "toggle-integration" ] ("API & MCP" ++ indicator)
        , body
        ]


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


fundSuccessLabel : Ledger.TaskEscrowResponse -> String
fundSuccessLabel escrow =
    "Escrowed " ++ String.fromInt escrow.amount ++ " credits (" ++ escrowStateLabel escrow.state ++ ")."


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
