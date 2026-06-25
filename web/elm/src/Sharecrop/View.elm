module Sharecrop.View exposing (..)

import Browser
import Html exposing (Html, a, div, form, label, main_, option, p, select, span, table, tbody, td, text, th, thead, tr)
import Html.Attributes exposing (checked, disabled, href, placeholder, selected, type_, value)
import Html.Events exposing (onCheck, onClick, onInput, onSubmit)
import Json.Encode as Encode
import Sharecrop.Generated.Agent as Agent
import Sharecrop.Generated.Collectible as Collectible
import Sharecrop.Generated.Ledger as Ledger
import Sharecrop.Generated.Organization as Organization
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
            loggedInView model.origin state


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
            ]
        , maybeError model.authError "auth-error"
        ]


loggedInView : String -> LoggedInModel -> Html Msg
loggedInView origin state =
    div [ Html.Attributes.class "space-y-6" ]
        [ navBar state.page state.subjectId
        , pageView origin state
        ]


navBar : Page -> String -> Html Msg
navBar current subjectId =
    div [ Html.Attributes.class "flex flex-wrap items-center gap-2" ]
        [ navLink current OverviewPage "overview" "Overview"
        , navLink current TasksPage "tasks" "Tasks"
        , navLink current CreateTaskPage "create-task" "New task"
        , navLink current DiscoveryPage "discovery" "Discovery"
        , navLink current FundingPage "funding" "Funding"
        , navLink current AgentsPage "agents" "Agents"
        , navLink current CollectiblesPage "collectibles" "Collectibles"
        , navLink current SeriesListPage "series-list" "Series"
        , navLink current OrganizationsPage "organizations" "Organizations"
        , a [ href ("/users/" ++ subjectId), Html.Attributes.class Ui.secondaryButtonClass, testId "nav-profile" ] [ text "Profile" ]
        , Ui.secondaryButton [ type_ "button", onClick LogoutClicked, testId "logout" ] "Log out"
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
    a [ href (pageToPath target), Html.Attributes.class styleClass, testId ("nav-" ++ identifier) ] [ text labelText ]


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
            userSubmissionsView userId state.userSubmissions

        CollectibleDetailPage collectibleId ->
            collectibleDetailView collectibleId state

        SeriesListPage ->
            seriesListView state

        SeriesDetailPage seriesId ->
            seriesDetailView seriesId state

        TeamDetailPage teamId ->
            teamDetailView teamId state


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
                            (List.map (\memberId -> a [ href ("/users/" ++ memberId), Html.Attributes.class "block py-2 text-sm underline", testId "team-member-row" ] [ text memberId ]) detail.members)
                    , if detail.team.ownerKind == "user" && detail.team.ownerUserID == state.subjectId then
                        form [ Html.Attributes.class "flex flex-wrap items-end gap-2", onSubmit (AddTeamMemberClicked detail.team.id) ]
                            [ Ui.fieldLabel "Add member by email"
                                [ Ui.textInput [ type_ "email", placeholder "person@example.com", value state.teamMemberEmail, onInput TeamMemberEmailChanged, testId "team-member-email" ] ]
                            , Ui.primaryButton [ type_ "submit", testId "add-team-member" ] "Add member"
                            , maybeNote state.teamMemberMessage "team-member-message"
                            ]

                      else
                        text ""
                    , Ui.sectionTitle "Collectibles"
                    , collectiblesHoldingsList "team-collectibles" state.teamCollectibles
                    ]

            Nothing ->
                p [ Html.Attributes.class "text-sm text-slate-500", testId "team-detail-missing" ] [ text ("Loading team " ++ teamId ++ "…") ]
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
        [ a [ href "/collectibles", Html.Attributes.class Ui.secondaryButtonClass, testId "back-collectibles" ] [ text "Back to collectibles" ]
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
                        , Ui.textInput [ type_ "text", placeholder "Recipient user id", value state.transferRecipientId, onInput TransferRecipientIdChanged, testId "transfer-recipient-id" ]
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
        , a [ href ("/series/" ++ series.id), Html.Attributes.class Ui.secondaryButtonClass, testId "open-series" ] [ text "Open" ]
        ]


seriesDetailView : String -> LoggedInModel -> Html Msg
seriesDetailView seriesId state =
    Ui.card
        [ a [ href "/series", Html.Attributes.class Ui.secondaryButtonClass, testId "back-series" ] [ text "Back to series" ]
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
        [ a [ href ("/tasks/" ++ entry.id), Html.Attributes.class "w-full text-sm underline break-words sm:w-auto", testId "series-task-link" ] [ text (entry.title ++ " · " ++ entry.state) ]
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
            [ Ui.fieldLabel "Add task by ID"
                [ Ui.textInput [ type_ "text", placeholder "task id", value state.addSeriesTaskId, onInput AddSeriesTaskIdChanged, testId "series-add-task-id" ] ]
            , Ui.primaryButton [ type_ "submit", testId "series-add-task" ] "Add task"
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
        [ a [ href ("/users/" ++ comment.authorUserID), Html.Attributes.class "text-xs font-medium text-slate-600 underline" ] [ text comment.authorUserID ]
        , p [ Html.Attributes.class "text-sm text-slate-700 break-words" ] [ text comment.body ]
        ]


userTaskListView : String -> String -> String -> List Task.TaskListItemResponse -> Html Msg
userTaskListView heading identifier userId tasks =
    Ui.card
        [ a [ href ("/users/" ++ userId), Html.Attributes.class Ui.secondaryButtonClass, testId "back-user" ] [ text "Back to profile" ]
        , Ui.sectionTitle heading
        , if List.isEmpty tasks then
            p [ Html.Attributes.class "text-sm text-slate-500", testId (identifier ++ "-empty") ] [ text "Nothing to show." ]

          else
            div [ Html.Attributes.class "divide-y divide-slate-100", testId identifier ]
                (List.map (\item -> a [ href ("/tasks/" ++ item.id), Html.Attributes.class "block py-2 text-sm underline", testId (identifier ++ "-row") ] [ text (item.title ++ " · " ++ taskStateLabel item.state) ]) tasks)
        ]


userSubmissionsView : String -> List Submission.SubmissionResponse -> Html Msg
userSubmissionsView userId submissions =
    Ui.card
        [ a [ href ("/users/" ++ userId), Html.Attributes.class Ui.secondaryButtonClass, testId "back-user" ] [ text "Back to profile" ]
        , Ui.sectionTitle "Submissions"
        , if List.isEmpty submissions then
            p [ Html.Attributes.class "text-sm text-slate-500", testId "user-submissions-empty" ] [ text "No submissions." ]

          else
            div [ Html.Attributes.class "divide-y divide-slate-100", testId "user-submissions" ]
                (List.map
                    (\item ->
                        div [ Html.Attributes.class "space-y-1 py-2", testId "user-submission-row" ]
                            [ a [ href ("/tasks/" ++ item.taskID), Html.Attributes.class "text-sm underline" ] [ text ("Task " ++ item.taskID) ]
                            , p [ Html.Attributes.class "text-xs text-slate-600" ] [ text (submissionStateLabel item.state) ]
                            ]
                    )
                    submissions
                )
        ]


userDetailView : String -> String -> LoggedInModel -> Html Msg
userDetailView origin userId state =
    div [ Html.Attributes.class "space-y-6" ]
        (Ui.card
            [ Ui.sectionTitle "User"
            , p [ Html.Attributes.class "text-sm font-medium", testId "user-id" ] [ text userId ]
            , div [ Html.Attributes.class "flex flex-wrap gap-2" ]
                [ a [ href ("/users/" ++ userId ++ "/work"), Html.Attributes.class Ui.secondaryButtonClass, testId "user-work-link" ] [ text "Public work" ]
                , a [ href ("/users/" ++ userId ++ "/submissions"), Html.Attributes.class Ui.secondaryButtonClass, testId "user-submissions-link" ] [ text "Submissions" ]
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
                                    a [ href ("/tasks/" ++ item.id), Html.Attributes.class "block py-2 text-sm underline", testId "user-task-row" ] [ text item.title ]
                                )
                                profile.tasks
                            )

                Nothing ->
                    p [ Html.Attributes.class "text-sm text-slate-500" ] [ text "Loading…" ]
            ]
            :: (if userId == state.subjectId then
                    [ userAgentAccessCard origin state ]

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
        [ a [ href "/organizations", Html.Attributes.class Ui.secondaryButtonClass, testId "back-organizations" ] [ text "Back to organizations" ]
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
            , Ui.sectionTitle "Organization tasks"
            , tasksListSimple "org-tasks" state.orgTasks
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
                , Ui.primaryButton [ type_ "submit", testId "provision-member" ] "Provision member"
                ]
            , maybeNote state.provisionMemberMessage "provision-member-message"
            , Ui.sectionTitle "Collectibles"
            , collectiblesHoldingsList "org-collectibles" state.orgCollectibles
            ]


orgTeamsList : List Team.TeamResponse -> Html Msg
orgTeamsList teams =
    if List.isEmpty teams then
        p [ Html.Attributes.class "text-sm text-slate-500", testId "org-teams-empty" ] [ text "No teams yet." ]

    else
        div [ Html.Attributes.class "divide-y divide-slate-100", testId "org-teams" ]
            (List.map (\team -> a [ href ("/teams/" ++ team.id), Html.Attributes.class "block py-1 text-sm underline", testId "org-team-row" ] [ text team.name ]) teams)


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
    div [ Html.Attributes.class "flex items-center justify-between gap-2 py-2", testId "org-member-row" ]
        [ a [ href ("/users/" ++ member.userID), Html.Attributes.class "text-sm font-medium underline", testId "org-member-link" ] [ text member.userID ]
        , p [ Html.Attributes.class "text-xs text-slate-600" ] [ text (roles ++ " · " ++ membershipStatusText member.status) ]
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
        , a [ href ("/organizations/" ++ organization.id), Html.Attributes.class Ui.secondaryButtonClass, testId "select-organization" ] [ text "Open" ]
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
        , Ui.fieldLabel "Credit reward" [ Ui.textInput [ type_ "number", placeholder "Blank for no reward", value state.createRewardAmount, onInput CreateRewardAmountChanged, testId "create-reward" ] ]
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
        Ui.fieldLabel "Share with user ID"
            [ Ui.textInput [ type_ "text", placeholder "User ID to grant access", value state.createScopeUserId, onInput CreateScopeUserIdChanged, testId "create-scope-user" ] ]

    else if state.createVisibility == visibilityTeamTag then
        Ui.fieldLabel "Share with team ID"
            [ Ui.textInput [ type_ "text", placeholder "Team ID (standalone or organization team)", value state.createScopeTeamId, onInput CreateScopeTeamIdChanged, testId "create-scope-team" ] ]

    else if state.createVisibility == visibilityOrganizationTag then
        Ui.fieldLabel "Share with organization ID"
            [ Ui.textInput [ type_ "text", placeholder "Organization ID", value state.createScopeOrganizationId, onInput CreateScopeOrganizationIdChanged, testId "create-scope-organization" ] ]

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
        , Ui.textInput [ type_ "text", placeholder "Organization ID (optional — fund from org credits)", value state.fundOrganizationId, onInput FundOrganizationIdChanged, testId "fund-organization" ]
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
    Ui.card
        [ Ui.sectionTitle "My tasks"
        , Ui.label_ "Filter by state"
        , div [ Html.Attributes.class "flex flex-wrap gap-2", testId "task-filter" ] (List.map (taskFilterButton state.taskStateFilter) taskStateFilterOptions)
        , tasksList state.tasks
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


activeAssigneeSuffix : Task.TaskListItemResponse -> String
activeAssigneeSuffix item =
    if item.activeAssigneeID == "" then
        ""

    else
        " · reserved by " ++ item.activeAssigneeID


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
        , a [ href ("/tasks/" ++ item.id), Html.Attributes.class (Ui.secondaryButtonClass ++ " shrink-0"), testId "view-task" ] [ text "View" ]
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
        , awardRecipientControl state
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
        , Ui.textInput [ type_ "text", placeholder "Recipient id", value state.awardRecipientId, onInput AwardRecipientIdChanged, testId "award-recipient-id" ]
        , p [ Html.Attributes.class "text-xs text-slate-500" ] [ text "Pick a recipient, then Award a collectible below." ]
        , maybeNote state.awardDefaultMessage "award-default-message"
        ]


catalogGallery : LoggedInModel -> Html Msg
catalogGallery state =
    div [ Html.Attributes.class "mt-3 grid grid-cols-2 gap-3 sm:grid-cols-3", testId "catalog" ]
        (List.map (catalogEntry state.awardRecipientId) state.collectibleCatalog)


catalogEntry : String -> Collectible.CollectibleCatalogEntry -> Html Msg
catalogEntry recipientId entry =
    div [ Html.Attributes.class "flex flex-col items-center gap-1 rounded-md border border-slate-200 p-2 text-center", testId "catalog-entry" ]
        [ Sprites.pixel entry.art 6
        , span [ Html.Attributes.class "text-xs font-medium break-words" ] [ text entry.name ]
        , Ui.badge (collectibleKindLabel entry.kind)
        , Ui.secondaryButton [ type_ "button", onClick (AwardDefaultClicked entry.slug), disabled (String.trim recipientId == ""), testId "catalog-award" ] "Award"
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
            , a [ href ("/collectibles/" ++ collectible.id), Html.Attributes.class "font-medium underline break-words", testId "collectible-link" ] [ text collectible.name ]
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
    Ui.card
        [ Ui.sectionTitle "Discover public tasks"
        , Ui.checkbox [ checked state.discoveryIncludeReserved, onClick (DiscoveryIncludeReservedChanged (not state.discoveryIncludeReserved)), testId "include-reserved" ] "Include reserved"
        , discoveryList state.discoveryTasks
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



-- Task detail page


taskDetailPageView : String -> LoggedInModel -> Html Msg
taskDetailPageView origin state =
    let
        isOwner =
            state.detail |> Maybe.map (\detail -> detail.createdBy == state.subjectId) |> Maybe.withDefault False

        backHref =
            if isOwner then
                "/tasks"

            else
                "/discovery"
    in
    div [ Html.Attributes.class "space-y-6" ]
        ([ a [ href backHref, Html.Attributes.class Ui.secondaryButtonClass, testId "detail-back" ] [ text "Back" ]
         , detailCard origin state
         ]
            ++ (if isOwner then
                    [ ownerControlsCard state, submissionsCard state ]

                else
                    [ reservationCard state, submitCard state ]
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
        [ a [ href ("/users/" ++ comment.authorUserID), Html.Attributes.class "text-xs font-medium text-slate-600 underline" ] [ text comment.authorUserID ]
        , p [ Html.Attributes.class "text-sm text-slate-700 break-words" ] [ text comment.body ]
        ]


seriesLinkBlock : TaskDetail -> List (Html Msg)
seriesLinkBlock detail =
    if detail.seriesID == "" then
        []

    else
        [ a [ href ("/series/" ++ detail.seriesID), Html.Attributes.class "text-sm underline", testId "task-series-link" ] [ text "Part of a series" ] ]


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
            Ui.card
                [ Ui.sectionTitle "Owner controls"
                , p [ Html.Attributes.class "rounded-md bg-slate-100 px-3 py-2 text-sm text-slate-700", testId "task-guidance" ] [ text (taskStateGuidance detail.state) ]
                , div [ Html.Attributes.class "flex gap-2" ]
                    [ Ui.secondaryButton [ type_ "button", onClick (OpenTaskClicked detail.id), testId "open-task" ] "Open"
                    , Ui.secondaryButton [ type_ "button", onClick (RefundTaskClicked detail.id), testId "refund-task" ] "Refund"
                    ]
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
                , reservationAction detail
                , reservationsList state.reservations
                , maybeNote state.reservationMessage "reservation-message"
                ]

        Nothing ->
            text ""


reservationAction : PublicTaskDetail -> Html Msg
reservationAction detail =
    case detail.viewerAction of
        Task.TaskViewerActionReserve ->
            Ui.primaryButton [ type_ "button", onClick (ReserveClicked detail.id), testId "reserve-task" ] "Reserve"

        Task.TaskViewerActionRequestApproval ->
            Ui.primaryButton [ type_ "button", onClick (ReserveClicked detail.id), testId "request-approval" ] "Request approval"

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


submissionsCard : LoggedInModel -> Html Msg
submissionsCard state =
    Ui.card
        [ Ui.sectionTitle "Submissions"
        , reviewControls state
        , submissionsList state
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
        , maybeNote state.reviewMessage "review-message"
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
        , Ui.secondaryButton [ type_ "button", onClick (OpenSubmissionComments submission.id), testId "submission-comments-toggle" ] "Comments"
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
                , Ui.primaryButton [ type_ "button", onClick (AddSubmissionCommentClicked submission.id), testId "add-submission-comment" ] "Comment"
                , maybeNote state.submissionCommentMessage "submission-comment-message"
                ]
            ]

    else
        text ""


submissionCommentRow : Submission.SubmissionCommentResponse -> Html Msg
submissionCommentRow comment =
    div [ Html.Attributes.class "rounded-md border border-slate-200 bg-white p-3", testId "submission-comment" ]
        [ a [ href ("/users/" ++ comment.authorUserID), Html.Attributes.class "text-xs font-medium text-slate-600 underline" ] [ text comment.authorUserID ]
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
