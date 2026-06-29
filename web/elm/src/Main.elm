port module Main exposing (main)

import Browser
import Browser.Navigation as Nav
import Http
import Sharecrop.Api as Api
import Sharecrop.Generated.Agent as Agent
import Sharecrop.Generated.Auth as Auth
import Sharecrop.Generated.Collectible as Collectible
import Sharecrop.Generated.Notification as Notification
import Sharecrop.Generated.Organization as Organization
import Sharecrop.Generated.Privacy as Privacy
import Sharecrop.Generated.SavedQueueViews as SavedQueueViews
import Sharecrop.Generated.Task as Task
import Sharecrop.Labels exposing (httpErrorLabel, participationPolicyTag)
import Sharecrop.Types exposing (..)
import Sharecrop.View as View
import Url exposing (Url)




main : Program Flags Model Msg
main =
    Browser.application
        { init = \flags url key -> ( initialModel flags key url, Api.postRefresh )
        , update = update
        , subscriptions = \_ -> Sub.none
        , view = View.view
        , onUrlRequest = LinkClicked
        , onUrlChange = UrlChanged
        }


port copyToClipboard : String -> Cmd msg


port reloadDemo : () -> Cmd msg


initialModel : Flags -> Nav.Key -> Url -> Model
initialModel flags key url =
    { origin = flags.origin
    , demo = flags.demo
    , key = key
    , route = pageFromUrl url
    , email = ""
    , password = ""
    , resetEmail = ""
    , resetToken = ""
    , resetPassword = ""
    , authError = Nothing
    , session = LoggedOut
    }


emptyLoggedIn : Auth.AuthResponse -> LoggedInModel
emptyLoggedIn response =
    { accessToken = response.accessToken
    , subjectId = response.subjectID
    , isAdmin = response.role == "admin"
    , page = OverviewPage
    , balance = Nothing
    , entries = []
    , createTitle = ""
    , createDescription = ""
    , createResponseSchema = "{\"kind\":\"freeform\"}"
    , createSchemaFields = []
    , createPayloadJson = ""
    , createRewardKind = "none"
    , createRewardAmount = ""
    , createRewardCollectibleIds = []
    , createVisibility = visibilityDefaultTag
    , createScopeUserId = ""
    , createScopeTeamId = ""
    , createScopeOrganizationId = ""
    , createAssigneeScope = Task.TaskAssigneeScopeUser
    , createParticipationPolicy = participationPolicyTag Task.TaskParticipationPolicyOpen
    , createReservationHours = "48"
    , createMessage = Nothing
    , fundTaskId = ""
    , fundAmount = ""
    , fundOrganizationId = ""
    , fundMessage = Nothing
    , fundNonce = 0
    , tasks = []
    , taskStateFilter = ""
    , taskListOffset = 0
    , taskListQuery = ""
    , taskListTypeFilter = ""
    , taskListSort = "newest"
    , agentLabel = ""
    , agentScopes = [ Agent.AgentScopeTasksRead, Agent.AgentScopeSubmissionsWrite ]
    , credentials = []
    , newCredential = Nothing
    , agentMessage = Nothing
    , discoveryTasks = []
    , discoveryIncludeReserved = False
    , discoveryOffset = 0
    , discoveryQuery = ""
    , detail = Nothing
    , detailError = Nothing
    , reservations = []
    , reservationOrganizationId = ""
    , reservationTeamId = ""
    , reservationMessage = Nothing
    , submissions = []
    , submitInput = ""
    , submitMessage = Nothing
    , reviewNote = ""
    , reviewPartialCredit = ""
    , reviewTip = ""
    , reviewTipCollectibleId = ""
    , reviewBan = False
    , reviewMessage = Nothing
    , collectibles = []
    , collectibleName = ""
    , collectibleKind = Collectible.CollectibleKindBadge
    , collectiblePolicy = Collectible.CollectibleTransferPolicyNonTransferableExceptPayout
    , collectibleMessage = Nothing
    , awardTaskId = ""
    , awardMessage = Nothing
    , awardDefaultMessage = Nothing
    , collectibleCatalog = []
    , awardRecipientKind = "user"
    , awardRecipientId = ""
    , transferRecipientId = ""
    , transferMessage = Nothing
    , organizations = []
    , createOrgName = ""
    , orgMessage = Nothing
    , activeOrgId = ""
    , orgBalance = Nothing
    , orgLedger = []
    , orgAuditEvents = []
    , orgTeams = []
    , standaloneTeams = []
    , orgMembers = []
    , orgTasks = []
    , orgTaskQuery = ""
    , orgTaskFilter = ""
    , orgTaskTypeFilter = ""
    , orgTaskSort = "newest"
    , orgTaskOffset = 0
    , orgTaskMessage = Nothing
    , orgTaskSavedViewName = ""
    , orgTaskSavedViews = []
    , orgCollectibles = []
    , orgCollectiblesMessage = Nothing
    , teamCollectibles = []
    , teamCollectiblesMessage = Nothing
    , userProfile = Nothing
    , userProfileError = Nothing
    , userWork = []
    , userSubmissions = []
    , pendingRevisionTaskID = Nothing
    , pendingRevisionResponse = ""
    , seriesDetail = Nothing
    , seriesDetailError = Nothing
    , seriesList = []
    , createSeriesTitle = ""
    , createSeriesDescription = ""
    , seriesMessage = Nothing
    , addSeriesTaskId = ""
    , seriesCommentBody = ""
    , seriesRenameTitle = ""
    , seriesRenameDescription = ""
    , teamDetail = Nothing
    , teamDetailError = Nothing
    , teamWork = []
    , teamWorkQuery = ""
    , teamWorkFilter = ""
    , teamWorkTypeFilter = ""
    , teamWorkSort = "newest"
    , teamWorkOffset = 0
    , teamWorkMessage = Nothing
    , teamWorkSavedViewName = ""
    , teamWorkSavedViews = []
    , teamMemberEmail = ""
    , teamMemberMessage = Nothing
    , createOrgTeamName = ""
    , orgTeamMessage = Nothing
    , provisionMemberEmail = ""
    , provisionMemberRoles = [ "member" ]
    , provisionMemberMessage = Nothing
    , createTaskOwner = ""
    , createTaskType = "general"
    , createReferenceURL = ""
    , taskComments = []
    , taskCommentBody = ""
    , taskCommentMessage = Nothing
    , submissionComments = []
    , activeSubmissionCommentsID = Nothing
    , submissionCommentBody = ""
    , submissionCommentMessage = Nothing
    , taskAgentToken = Nothing
    , taskIntegrationOpen = False
    , taskActionMessage = Nothing
    , userAgentToken = Nothing
    , accountEmail = ""
    , currentPassword = ""
    , newPassword = ""
    , emailVerificationToken = ""
    , emailVerificationInput = ""
    , accountMessage = Nothing
    , userDirectory = []
    , userDirectoryQuery = ""
    , userDirectoryOffset = 0
    , organizationQuery = ""
    , organizationOffset = 0
    , standaloneTeamQuery = ""
    , standaloneTeamOffset = 0
    , orgTeamQuery = ""
    , orgTeamOffset = 0
    , operations = Nothing
    , auditEvents = []
    , adminPrivacyRequests = []
    , adminPrivacyResolutionNote = ""
    , auditActionFilter = ""
    , auditSubjectKindFilter = ""
    , auditSubjectIDFilter = ""
    , adminMessage = Nothing
    , notifications = []
    , inboxMessage = Nothing
    }


updateFieldAt : Int -> (SchemaFieldDraft -> SchemaFieldDraft) -> List SchemaFieldDraft -> List SchemaFieldDraft
updateFieldAt index transform fields =
    List.indexedMap
        (\i field ->
            if i == index then
                transform field

            else
                field
        )
        fields


applySchemaFields : (List SchemaFieldDraft -> List SchemaFieldDraft) -> LoggedInModel -> LoggedInModel
applySchemaFields transform state =
    let
        nextFields =
            transform state.createSchemaFields
    in
    { state
        | createSchemaFields = nextFields
        , createResponseSchema = View.schemaFromFields nextFields
    }


replaceNotification : Notification.NotificationResponse -> List Notification.NotificationResponse -> List Notification.NotificationResponse
replaceNotification replacement notifications =
    List.map
        (\notification ->
            if notification.id == replacement.id then
                replacement

            else
                notification
        )
        notifications


replacePrivacyRequest : Privacy.PrivacyRequestResponse -> List Privacy.PrivacyRequestResponse -> List Privacy.PrivacyRequestResponse
replacePrivacyRequest replacement requests =
    List.map
        (\request ->
            if request.id == replacement.id then
                replacement

            else
                request
        )
        requests


teamWorkSavedViewScope : String
teamWorkSavedViewScope =
    "team_work"


orgTaskSavedViewScope : String
orgTaskSavedViewScope =
    "organization_tasks"


queueViewFromResponse : SavedQueueViews.SavedQueueViewResponse -> QueueView
queueViewFromResponse response =
    { name = response.name
    , query = response.query
    , stateFilter = response.stateFilter
    , typeFilter = response.typeFilter
    , sort = response.sort
    }


saveQueueView : QueueView -> List QueueView -> List QueueView
saveQueueView view views =
    view :: List.filter (\existing -> existing.name /= view.name) views


queueViewByName : String -> List QueueView -> Maybe QueueView
queueViewByName name views =
    views
        |> List.filter (\view -> view.name == name)
        |> List.head


orgTeamSearchOrganizationID : LoggedInModel -> String
orgTeamSearchOrganizationID state =
    if state.reservationOrganizationId /= "" then
        state.reservationOrganizationId

    else
        state.activeOrgId


loggedInForPage : Auth.AuthResponse -> Page -> LoggedInModel
loggedInForPage response page =
    let
        state =
            emptyLoggedIn response
    in
    { state | page = page }


pageFromUrl : Url -> Page
pageFromUrl url =
    let
        fragment =
            Maybe.withDefault "" url.fragment
    in
    case String.split "/" (String.dropLeft 1 fragment) of
        [ "" ] ->
            OverviewPage

        [ "tasks" ] ->
            TasksPage

        [ "tasks", "new" ] ->
            CreateTaskPage

        [ "tasks", taskId ] ->
            TaskDetailPage taskId

        [ "discovery" ] ->
            DiscoveryPage

        [ "funding" ] ->
            FundingPage

        [ "agents" ] ->
            AgentsPage

        [ "collectibles" ] ->
            CollectiblesPage

        [ "collectibles", collectibleId ] ->
            CollectibleDetailPage collectibleId

        [ "series" ] ->
            SeriesListPage

        [ "series", seriesId ] ->
            SeriesDetailPage seriesId

        [ "teams", teamId ] ->
            TeamDetailPage teamId

        [ "admin" ] ->
            AdminPage

        [ "inbox" ] ->
            InboxPage

        [ "organizations" ] ->
            OrganizationsPage

        [ "organizations", organizationId ] ->
            OrganizationDetailPage organizationId

        [ "users", userId ] ->
            UserDetailPage userId

        [ "users", userId, "work" ] ->
            UserWorkPage userId

        [ "users", userId, "submissions" ] ->
            UserSubmissionsPage userId

        _ ->
            NotFoundPage


-- enterPage applies any per-page state a route needs when it becomes active, so
-- a deep link or back/forward leaves the model consistent with the URL.
enterPage : Page -> LoggedInModel -> LoggedInModel
enterPage page state =
    case page of
        TasksPage ->
            { state | page = page, taskStateFilter = "", taskListOffset = 0, taskListQuery = "", taskListTypeFilter = "", taskListSort = "newest" }

        DiscoveryPage ->
            { state | page = page, discoveryIncludeReserved = False, discoveryOffset = 0, discoveryQuery = "" }

        OrganizationDetailPage organizationId ->
            { state | page = page, activeOrgId = organizationId, orgBalance = Nothing, orgLedger = [], orgAuditEvents = [], orgTeams = [], orgMembers = [], orgTasks = [], orgTaskQuery = "", orgTaskFilter = "", orgTaskTypeFilter = "", orgTaskSort = "newest", orgTaskOffset = 0, orgTaskMessage = Nothing, orgCollectibles = [], orgCollectiblesMessage = Nothing, orgTeamMessage = Nothing, provisionMemberRoles = [ "member" ], provisionMemberMessage = Nothing }

        UserDetailPage _ ->
            { state | page = page, userProfile = Nothing, userProfileError = Nothing }

        UserWorkPage _ ->
            { state | page = page, userWork = [] }

        UserSubmissionsPage _ ->
            { state | page = page, userSubmissions = [] }

        SeriesListPage ->
            { state | page = page, seriesMessage = Nothing }

        SeriesDetailPage _ ->
            { state | page = page, seriesDetail = Nothing, seriesDetailError = Nothing, seriesMessage = Nothing, addSeriesTaskId = "", seriesCommentBody = "", seriesRenameTitle = "", seriesRenameDescription = "" }

        TeamDetailPage _ ->
            { state | page = page, teamDetail = Nothing, teamDetailError = Nothing, teamWork = [], teamWorkQuery = "", teamWorkFilter = "", teamWorkTypeFilter = "", teamWorkSort = "newest", teamWorkOffset = 0, teamWorkMessage = Nothing, teamCollectibles = [], teamCollectiblesMessage = Nothing, teamMemberEmail = "", teamMemberMessage = Nothing }

        AdminPage ->
            { state | page = page, operations = Nothing, auditEvents = [], adminPrivacyRequests = [], adminPrivacyResolutionNote = "", auditActionFilter = "", auditSubjectKindFilter = "", auditSubjectIDFilter = "", adminMessage = Nothing }

        InboxPage ->
            { state | page = page, notifications = [], inboxMessage = Nothing }

        CollectibleDetailPage _ ->
            { state | page = page, transferMessage = Nothing, transferRecipientId = "" }

        TaskDetailPage taskId ->
            -- Clear the previous task's detail substate so a task->task link does
            -- not briefly show the prior task's badges, submissions, or comments.
            -- Review form fields are reset here too so the prior submission's
            -- note / partial credit / tip / ban does not carry over to the next.
            { state | page = page, detail = Nothing, detailError = Nothing, reservations = [], reservationOrganizationId = "", reservationTeamId = "", reservationMessage = Nothing, submissions = [], submitInput = revisionDraftFor taskId state, submitMessage = Nothing, reviewNote = "", reviewPartialCredit = "", reviewTip = "", reviewTipCollectibleId = "", reviewBan = False, reviewMessage = Nothing, taskComments = [], taskCommentBody = "", taskCommentMessage = Nothing, submissionComments = [], activeSubmissionCommentsID = Nothing, submissionCommentBody = "", submissionCommentMessage = Nothing, taskAgentToken = Nothing, taskIntegrationOpen = False, taskActionMessage = Nothing, pendingRevisionTaskID = Nothing, pendingRevisionResponse = "" }

        CollectiblesPage ->
            -- Reset the award / mint / transfer messages and drafts so a stale
            -- "Awarded" note or prefilled recipient does not reappear on return.
            { state | page = page, awardMessage = Nothing, awardDefaultMessage = Nothing, collectibleMessage = Nothing, transferMessage = Nothing, collectibleName = "", awardRecipientId = "", awardTaskId = "" }

        CreateTaskPage ->
            -- Clear a half-finished draft and any stale create message on entry.
            { state | page = page, createTitle = "", createDescription = "", createResponseSchema = "{\"kind\":\"freeform\"}", createSchemaFields = [], createPayloadJson = "", createRewardKind = "none", createRewardAmount = "", createRewardCollectibleIds = [], createMessage = Nothing, createTaskType = "general", createReferenceURL = "", createParticipationPolicy = participationPolicyTag Task.TaskParticipationPolicyOpen, createReservationHours = "48" }

        FundingPage ->
            { state | page = page, fundMessage = Nothing }

        _ ->
            { state | page = page }


revisionDraftFor : String -> LoggedInModel -> String
revisionDraftFor taskId state =
    if state.pendingRevisionTaskID == Just taskId then
        state.pendingRevisionResponse

    else
        ""


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        EmailChanged value ->
            ( { model | email = value }, Cmd.none )

        PasswordChanged value ->
            ( { model | password = value }, Cmd.none )

        RegisterClicked ->
            ( { model | authError = Nothing }, Api.postAuth "/api/auth/register" model )

        LoginClicked ->
            ( { model | authError = Nothing }, Api.postAuth "/api/auth/login" model )

        GuestClicked ->
            ( { model | authError = Nothing }, Api.postGuest )

        AuthReceived (Ok response) ->
            let
                state =
                    loggedInForPage response model.route
            in
            ( { model | password = "", authError = Nothing, session = LoggedIn { state | accountEmail = model.email } }
            , Api.loadAfterAuth response.accessToken
            )

        AuthReceived (Err error) ->
            ( { model | authError = Just (httpErrorLabel error) }, Cmd.none )

        RefreshReceived (Ok response) ->
            ( { model | session = LoggedIn (loggedInForPage response model.route) }
            , Cmd.batch [ Api.loadAfterAuth response.accessToken, Api.routeLoadCmd response.accessToken response.subjectID model.route ]
            )

        RefreshReceived (Err _) ->
            ( model, Cmd.none )

        PasswordResetEmailChanged value ->
            ( { model | resetEmail = value }, Cmd.none )

        PasswordResetTokenChanged value ->
            ( { model | resetToken = value }, Cmd.none )

        PasswordResetPasswordChanged value ->
            ( { model | resetPassword = value }, Cmd.none )

        RequestPasswordResetClicked ->
            ( { model | authError = Nothing }, Api.requestPasswordReset model )

        ConfirmPasswordResetClicked ->
            ( { model | authError = Nothing }, Api.confirmPasswordReset model )

        PasswordResetRequested (Ok token) ->
            if token == "" then
                ( { model | authError = Just "Password reset instructions sent." }, Cmd.none )

            else
                ( { model | resetToken = token, authError = Just "Password reset token created." }, Cmd.none )

        PasswordResetRequested (Err error) ->
            ( { model | authError = Just (httpErrorLabel error) }, Cmd.none )

        PasswordResetConfirmed (Ok ()) ->
            ( { model | resetPassword = "", resetToken = "", authError = Just "Password reset. Log in with the new password." }, Cmd.none )

        PasswordResetConfirmed (Err error) ->
            ( { model | authError = Just (httpErrorLabel error) }, Cmd.none )

        BalanceReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | balance = Api.balanceFromResult result }), Cmd.none )

        LedgerReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | entries = Api.entriesFromResult result }), Cmd.none )

        TasksReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | tasks = Api.tasksFromResult result }), Cmd.none )

        TaskStateFilterChanged value ->
            let
                updated =
                    Api.updateLoggedIn model (\state -> { state | taskStateFilter = value, taskListOffset = 0 })
            in
            Api.withSession updated (\state -> ( updated, Api.fetchTasks state.accessToken value state.taskListTypeFilter state.taskListSort 0 ))

        TaskListQueryChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | taskListQuery = value }), Cmd.none )

        TaskListTypeFilterChanged value ->
            let
                updated =
                    Api.updateLoggedIn model (\state -> { state | taskListTypeFilter = value, taskListOffset = 0 })
            in
            Api.withSession updated (\state -> ( updated, Api.fetchTasks state.accessToken state.taskStateFilter value state.taskListSort 0 ))

        TaskListSortChanged value ->
            let
                updated =
                    Api.updateLoggedIn model (\state -> { state | taskListSort = value, taskListOffset = 0 })
            in
            Api.withSession updated (\state -> ( updated, Api.fetchTasks state.accessToken state.taskStateFilter state.taskListTypeFilter value 0 ))

        PreviousTasksPageClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            max 0 (state.taskListOffset - Api.selectorPageSize)
                    in
                    ( Api.updateLoggedIn model (\current -> { current | taskListOffset = offset }), Api.fetchTasks state.accessToken state.taskStateFilter state.taskListTypeFilter state.taskListSort offset )
                )

        NextTasksPageClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            state.taskListOffset + Api.selectorPageSize
                    in
                    ( Api.updateLoggedIn model (\current -> { current | taskListOffset = offset }), Api.fetchTasks state.accessToken state.taskStateFilter state.taskListTypeFilter state.taskListSort offset )
                )

        CreateTitleChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | createTitle = value }), Cmd.none )

        CreateDescriptionChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | createDescription = value }), Cmd.none )

        CreateResponseSchemaChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | createResponseSchema = value }), Cmd.none )

        AddSchemaFieldClicked ->
            ( Api.updateLoggedIn model
                (applySchemaFields
                    (\fields -> fields ++ [ { name = "", kind = "string", required = True, itemKind = "string", enumValues = "" } ])
                )
            , Cmd.none
            )

        RemoveSchemaFieldClicked index ->
            ( Api.updateLoggedIn model
                (applySchemaFields
                    (\fields ->
                        List.indexedMap Tuple.pair fields
                            |> List.filter (\( i, _ ) -> i /= index)
                            |> List.map Tuple.second
                    )
                )
            , Cmd.none
            )

        SchemaFieldNameChanged index value ->
            ( Api.updateLoggedIn model
                (applySchemaFields (updateFieldAt index (\field -> { field | name = value })))
            , Cmd.none
            )

        SchemaFieldKindChanged index value ->
            ( Api.updateLoggedIn model
                (applySchemaFields (updateFieldAt index (\field -> { field | kind = value })))
            , Cmd.none
            )

        SchemaFieldRequiredChanged index value ->
            ( Api.updateLoggedIn model
                (applySchemaFields (updateFieldAt index (\field -> { field | required = value })))
            , Cmd.none
            )

        SchemaFieldItemKindChanged index value ->
            ( Api.updateLoggedIn model
                (applySchemaFields (updateFieldAt index (\field -> { field | itemKind = value })))
            , Cmd.none
            )

        SchemaFieldEnumValuesChanged index value ->
            ( Api.updateLoggedIn model
                (applySchemaFields (updateFieldAt index (\field -> { field | enumValues = value })))
            , Cmd.none
            )

        CreatePayloadChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | createPayloadJson = value }), Cmd.none )

        CreateRewardKindChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | createRewardKind = value, createRewardCollectibleIds = [] }), Cmd.none )

        CreateRewardAmountChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | createRewardAmount = value }), Cmd.none )

        ToggleCreateRewardCollectible collectibleId ->
            ( Api.updateLoggedIn model (\state -> { state | createRewardCollectibleIds = toggleString collectibleId state.createRewardCollectibleIds }), Cmd.none )

        CreateVisibilityChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | createVisibility = value, createScopeUserId = "", createScopeTeamId = "", createScopeOrganizationId = "" }), Cmd.none )

        CreateScopeUserIdChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | createScopeUserId = value }), Cmd.none )

        CreateScopeTeamIdChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | createScopeTeamId = value }), Cmd.none )

        CreateScopeOrganizationIdChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | createScopeOrganizationId = value }), Cmd.none )

        CreateAssigneeScopeChosen scope ->
            ( Api.updateLoggedIn model (\state -> { state | createAssigneeScope = scope }), Cmd.none )

        CreateParticipationChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | createParticipationPolicy = value }), Cmd.none )

        CreateReservationHoursChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | createReservationHours = value }), Cmd.none )

        CreateTaskClicked ->
            Api.withSession model (\state -> Api.createTaskCommand model state)

        CreateTaskReceived (Ok created) ->
            ( Api.updateLoggedIn model
                (\state ->
                    { state
                        | createTitle = ""
                        , createDescription = ""
                        , createResponseSchema = "{\"kind\":\"freeform\"}"
                        , createSchemaFields = []
                        , createPayloadJson = ""
                        , createTaskType = "general"
                        , createReferenceURL = ""
                        , createRewardCollectibleIds = []
                        , createParticipationPolicy = participationPolicyTag Task.TaskParticipationPolicyOpen
                        , createReservationHours = "48"
                        , createMessage = Just ("Created task " ++ created.id)
                        , fundTaskId = created.id
                        , fundAmount =
                            if created.rewardKind == "credit" then
                                String.fromInt created.rewardCreditAmount

                            else
                                state.fundAmount
                    }
                )
            , Api.refreshTasksAndLedger model
            )

        CreateTaskReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | createMessage = Just (httpErrorLabel error) }), Cmd.none )

        CredentialsReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | credentials = Api.credentialsFromResult result }), Cmd.none )

        FundTaskIdChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | fundTaskId = value }), Cmd.none )

        FundAmountChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | fundAmount = value }), Cmd.none )

        FundOrganizationIdChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | fundOrganizationId = value }), Cmd.none )

        FundClicked ->
            let
                bumped =
                    Api.updateLoggedIn model (\state -> { state | fundNonce = state.fundNonce + 1 })
            in
            Api.withSession bumped (\state -> Api.fundTaskCommand bumped state)

        FundReceived (Ok escrow) ->
            ( Api.updateLoggedIn model (\state -> { state | fundMessage = Just (View.fundSuccessLabel escrow) }), Api.refreshLedger model )

        FundReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | fundMessage = Just (httpErrorLabel error) }), Cmd.none )

        OpenTaskClicked taskId ->
            Api.withSession model (\state -> ( model, Api.postOpenTask state.accessToken taskId ))

        OpenTaskReceived (Ok detail) ->
            ( Api.updateLoggedIn model (\state -> { state | detail = Just detail, taskActionMessage = Just "Task opened." })
            , Api.refreshTasksAndDiscovery model
            )

        OpenTaskReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | taskActionMessage = Just (httpErrorLabel error) }), Cmd.none )

        RefundTaskClicked taskId ->
            Api.withSession model (\state -> ( model, Api.postRefundTask state.accessToken taskId ))

        RefundTaskReceived (Ok _) ->
            ( Api.updateLoggedIn model (\state -> { state | taskActionMessage = Just "Task refunded and cancelled." })
            , Cmd.batch [ Api.refreshTasksAndLedger model, Api.refreshAfterAccept model ]
            )

        RefundTaskReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | taskActionMessage = Just (httpErrorLabel error) }), Cmd.none )

        CancelTaskClicked taskId ->
            Api.withSession model (\state -> ( model, Api.postCancelTask state.accessToken taskId ))

        CancelTaskReceived (Ok detail) ->
            ( Api.updateLoggedIn model (\state -> { state | detail = Just detail, taskActionMessage = Just "Task cancelled." })
            , Api.refreshTasksAndDiscovery model
            )

        CancelTaskReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | taskActionMessage = Just (httpErrorLabel error) }), Cmd.none )

        RefundCollectibleRewardClicked taskId ->
            Api.withSession model (\state -> ( model, Api.postRefundCollectibleReward state.accessToken taskId ))

        RefundCollectibleRewardReceived (Ok _) ->
            ( Api.updateLoggedIn model (\state -> { state | taskActionMessage = Just "Collectible reward refunded." })
            , Cmd.batch [ Api.refreshAfterAccept model, Api.refreshCollectibles model ]
            )

        RefundCollectibleRewardReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | taskActionMessage = Just (httpErrorLabel error) }), Cmd.none )

        AgentLabelChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | agentLabel = value }), Cmd.none )

        ToggleScope scope ->
            ( Api.updateLoggedIn model (\state -> { state | agentScopes = Api.toggleScope scope state.agentScopes }), Cmd.none )

        CreateAgentClicked ->
            Api.withSession model (\state -> Api.createAgentCommand model state)

        AgentCreated (Ok created) ->
            ( Api.updateLoggedIn model (\state -> { state | newCredential = Just created, agentMessage = Nothing }), Api.refreshCredentials model )

        AgentCreated (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | agentMessage = Just (httpErrorLabel error) }), Cmd.none )

        ToggleTaskIntegration ->
            ( Api.updateLoggedIn model (\state -> { state | taskIntegrationOpen = not state.taskIntegrationOpen }), Cmd.none )

        MintTaskTokenClicked ->
            Api.withSession model (\state -> ( model, Api.mintTaskToken state.accessToken ))

        TaskTokenMinted (Ok created) ->
            ( Api.updateLoggedIn model (\state -> { state | taskAgentToken = Just created.secret }), Cmd.none )

        TaskTokenMinted (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | taskActionMessage = Just ("Could not create agent token: " ++ httpErrorLabel error) }), Cmd.none )

        MintUserTokenClicked ->
            Api.withSession model (\state -> ( model, Api.mintUserToken state.accessToken ))

        UserTokenMinted (Ok created) ->
            ( Api.updateLoggedIn model (\state -> { state | userAgentToken = Just created.secret }), Cmd.none )

        UserTokenMinted (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | taskActionMessage = Just ("Could not create agent token: " ++ httpErrorLabel error) }), Cmd.none )

        CopyClicked clipboardText ->
            ( model, copyToClipboard clipboardText )

        RevokeClicked credentialId ->
            Api.withSession model (\state -> ( model, Api.revokeAgent state.accessToken credentialId ))

        AgentRevoked _ ->
            ( model, Api.refreshCredentials model )

        LogoutClicked ->
            ( { model | session = LoggedOut, email = "", password = "" }
            , Cmd.batch [ Api.postLogout, Nav.pushUrl model.key "#/" ]
            )

        LogoutReceived _ ->
            ( model, Cmd.none )

        DiscoveryIncludeReservedChanged value ->
            Api.withSession model
                (\state ->
                    let
                        nextState =
                            { state | discoveryIncludeReserved = value, discoveryOffset = 0 }
                    in
                    ( Api.updateLoggedIn model (\_ -> nextState), Api.fetchDiscovery state.accessToken value 0 )
                )

        DiscoveryQueryChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | discoveryQuery = value }), Cmd.none )

        PreviousDiscoveryPageClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            max 0 (state.discoveryOffset - Api.selectorPageSize)
                    in
                    ( Api.updateLoggedIn model (\current -> { current | discoveryOffset = offset }), Api.fetchDiscovery state.accessToken state.discoveryIncludeReserved offset )
                )

        NextDiscoveryPageClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            state.discoveryOffset + Api.selectorPageSize
                    in
                    ( Api.updateLoggedIn model (\current -> { current | discoveryOffset = offset }), Api.fetchDiscovery state.accessToken state.discoveryIncludeReserved offset )
                )

        DiscoveryReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | discoveryTasks = Api.tasksFromResult result }), Cmd.none )

        DiscoveryViewClicked taskId ->
            ( Api.updateLoggedIn model
                (\s ->
                    { s
                        | detail = Nothing
                        , detailError = Nothing
                        , reservations = []
                        , reservationMessage = Nothing
                        , submissions = []
                        , submitInput = ""
                        , submitMessage = Nothing
                        , reviewNote = ""
                        , reviewPartialCredit = ""
                        , reviewTip = ""
                        , reviewTipCollectibleId = ""
                        , reviewBan = False
                        , reviewMessage = Nothing
                        , taskActionMessage = Nothing
                        , taskComments = []
                        , taskCommentBody = ""
                        , submissionCommentBody = ""
                        , activeSubmissionCommentsID = Nothing
                        , taskAgentToken = Nothing
                        , taskIntegrationOpen = False
                    }
                )
            , Nav.pushUrl model.key ("#/tasks/" ++ taskId)
            )

        DetailReceived (Ok detail) ->
            ( Api.updateLoggedIn model (\state -> { state | detail = Just detail, detailError = Nothing }), Cmd.none )

        DetailReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | detailError = Just (httpErrorLabel error) }), Cmd.none )

        ReserveClicked taskId ->
            Api.withSession model (\state -> ( Api.updateLoggedIn model (\current -> { current | reservationMessage = Nothing }), Api.postReservation state taskId ))

        ReservationOrganizationIdChanged value ->
            Api.withSession model
                (\state ->
                    ( Api.updateLoggedIn model (\current -> { current | reservationOrganizationId = value, reservationTeamId = "", orgTeams = [], orgTeamQuery = "", orgTeamOffset = 0 })
                    , if value == "" then
                        Cmd.none

                      else
                        Api.fetchOrgTeams state.accessToken value
                    )
                )

        ReservationTeamIdChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | reservationTeamId = value }), Cmd.none )

        ReservationReceived (Ok reservation) ->
            ( Api.updateLoggedIn model (\state -> { state | reservationMessage = Just (View.reservationSuccessLabel reservation) })
            , Api.refreshDetailReservations model
            )

        ReservationReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | reservationMessage = Just (httpErrorLabel error) }), Cmd.none )

        ReservationsReceived (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | reservations = response.reservations }), Cmd.none )

        ReservationsReceived (Err _) ->
            ( Api.updateLoggedIn model (\state -> { state | reservations = [] }), Cmd.none )

        ApproveReservationClicked reservationId ->
            Api.withSession model (\state -> Api.reservationChangeCommand model state reservationId "approve")

        DeclineReservationClicked reservationId ->
            Api.withSession model (\state -> Api.reservationChangeCommand model state reservationId "decline")

        CancelReservationClicked reservationId ->
            Api.withSession model (\state -> Api.reservationChangeCommand model state reservationId "cancel")

        ReservationChangeReceived (Ok reservation) ->
            ( Api.updateLoggedIn model (\state -> { state | reservationMessage = Just (View.reservationSuccessLabel reservation) })
            , Api.refreshDetailReservations model
            )

        ReservationChangeReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | reservationMessage = Just (httpErrorLabel error) }), Cmd.none )

        SubmissionsReceived (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | submissions = response.submissions }), Cmd.none )

        SubmissionsReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | submissions = [], reviewMessage = Just (httpErrorLabel error) }), Cmd.none )

        SubmitInputChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | submitInput = value }), Cmd.none )

        SubmitClicked ->
            Api.withSession model (\state -> Api.submitCommand model state)

        SubmitReceived (Ok created) ->
            Api.withSession model
                (\state ->
                    ( Api.updateLoggedIn model (\current -> { current | submitMessage = Just (View.submitSuccessLabel created), activeSubmissionCommentsID = Just created.submission.id, submissionComments = [], submissionCommentMessage = Nothing })
                    , Cmd.batch
                        [ Api.refreshDetailSubmissions model
                        , Api.fetchSubmissionComments state.accessToken created.submission.id
                        ]
                    )
                )

        SubmitReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | submitMessage = Just (httpErrorLabel error) }), Cmd.none )

        ReviewNoteChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | reviewNote = value }), Cmd.none )

        ReviewPartialCreditChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | reviewPartialCredit = value }), Cmd.none )

        ReviewTipChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | reviewTip = value }), Cmd.none )

        ReviewTipCollectibleChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | reviewTipCollectibleId = value }), Cmd.none )

        ReviewBanChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | reviewBan = value }), Cmd.none )

        AcceptClicked submissionId ->
            Api.withSession model (\state -> Api.acceptCommand model state submissionId)

        RequestChangesClicked submissionId ->
            Api.withSession model (\state -> Api.requestChangesCommand model state submissionId)

        RejectClicked submissionId ->
            Api.withSession model (\state -> Api.rejectCommand model state submissionId)

        ReviewActionReceived submissionId (Ok _) ->
            -- Clear the review form so the next submission in the list does not
            -- inherit the previous one's note / partial credit / tip / collectible tip / ban.
            Api.withSession model
                (\state ->
                    ( Api.updateLoggedIn model (\current -> { current | reviewMessage = Just "Review saved.", reviewNote = "", reviewPartialCredit = "", reviewTip = "", reviewTipCollectibleId = "", reviewBan = False, activeSubmissionCommentsID = Just submissionId, submissionComments = [], submissionCommentMessage = Nothing })
                    , Cmd.batch
                        [ Api.refreshAfterAccept model
                        , Api.fetchSubmissionComments state.accessToken submissionId
                        ]
                    )
                )

        ReviewActionReceived _ (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | reviewMessage = Just (httpErrorLabel error) }), Cmd.none )

        CollectibleNameChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | collectibleName = value }), Cmd.none )

        CollectibleKindChosen kind ->
            ( Api.updateLoggedIn model (\state -> { state | collectibleKind = kind }), Cmd.none )

        CollectiblePolicyChosen policy ->
            ( Api.updateLoggedIn model (\state -> { state | collectiblePolicy = policy }), Cmd.none )

        MintClicked ->
            Api.withSession model (\state -> Api.mintCommand model state)

        MintReceived (Ok collectible) ->
            ( Api.updateLoggedIn model
                (\state ->
                    { state
                        | collectibleName = ""
                        , collectibleMessage = Just (View.mintSuccessLabel collectible)
                    }
                )
            , Api.refreshCollectibles model
            )

        MintReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | collectibleMessage = Just (httpErrorLabel error) }), Cmd.none )

        CollectiblesReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | collectibles = Api.collectiblesFromResult result }), Cmd.none )

        AwardTaskIdChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | awardTaskId = value }), Cmd.none )

        AwardClicked collectibleId ->
            Api.withSession model (\state -> Api.awardCommand model state collectibleId)

        AwardReceived (Ok collectible) ->
            let
                updated =
                    Api.updateLoggedIn model (\state -> { state | awardMessage = Just (View.awardSuccessLabel collectible) })
            in
            Api.withSession updated (\state -> ( updated, Cmd.batch [ Api.fetchCollectibles state.accessToken, Api.fetchTasks state.accessToken state.taskStateFilter state.taskListTypeFilter state.taskListSort state.taskListOffset ] ))

        AwardReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | awardMessage = Just (httpErrorLabel error) }), Cmd.none )

        CollectibleCatalogReceived (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | collectibleCatalog = response.entries }), Cmd.none )

        CollectibleCatalogReceived (Err _) ->
            ( model, Cmd.none )

        AwardRecipientKindChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | awardRecipientKind = value, awardRecipientId = "" }), Cmd.none )

        AwardRecipientIdChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | awardRecipientId = value }), Cmd.none )

        AwardDefaultClicked slug ->
            Api.withSession model
                (\state ->
                    if String.trim state.awardRecipientId == "" then
                        ( Api.updateLoggedIn model (\current -> { current | awardDefaultMessage = Just "Enter a recipient id first." }), Cmd.none )

                    else
                        ( model, Api.awardDefaultCollectible state.accessToken slug state.awardRecipientKind state.awardRecipientId )
                )

        AwardDefaultReceived (Ok _) ->
            let
                updated =
                    Api.updateLoggedIn model (\state -> { state | awardDefaultMessage = Just "Awarded the collectible." })
            in
            ( updated, Api.refreshCollectibles updated )

        AwardDefaultReceived (Err _) ->
            ( Api.updateLoggedIn model (\state -> { state | awardDefaultMessage = Just "Only platform admins can award default collectibles." }), Cmd.none )

        TransferRecipientIdChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | transferRecipientId = value }), Cmd.none )

        TransferCollectibleClicked collectibleId ->
            Api.withSession model
                (\state ->
                    if String.trim state.transferRecipientId == "" then
                        ( Api.updateLoggedIn model (\current -> { current | transferMessage = Just "Enter a recipient id first." }), Cmd.none )

                    else
                        ( model, Api.transferCollectible state.accessToken collectibleId state.transferRecipientId )
                )

        TransferCollectibleReceived (Ok _) ->
            let
                updated =
                    Api.updateLoggedIn model (\state -> { state | transferMessage = Just "Transferred." })
            in
            ( updated, Api.refreshCollectibles updated )

        TransferCollectibleReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | transferMessage = Just (httpErrorLabel error) }), Cmd.none )

        OrganizationsReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | organizations = Api.organizationsFromResult result }), Cmd.none )

        CreateOrgNameChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | createOrgName = value }), Cmd.none )

        CreateOrgClicked ->
            Api.withSession model (\state -> Api.createOrgCommand model state)

        CreateOrgReceived (Ok organization) ->
            ( Api.updateLoggedIn model (\state -> { state | createOrgName = "", orgMessage = Just ("Created organization " ++ organization.name) }), Api.refreshOrganizations model )

        CreateOrgReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | orgMessage = Just (httpErrorLabel error) }), Cmd.none )

        OrgBalanceReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | orgBalance = Api.balanceFromResult result }), Cmd.none )

        OrgLedgerReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | orgLedger = Api.entriesFromResult result }), Cmd.none )

        OrgAuditEventsReceived (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | orgAuditEvents = response.events }), Cmd.none )

        OrgAuditEventsReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | orgAuditEvents = [], orgTaskMessage = Just (httpErrorLabel error) }), Cmd.none )

        OrgTeamsReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | orgTeams = Api.teamsFromResult result }), Cmd.none )

        StandaloneTeamsReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | standaloneTeams = Api.teamsFromResult result }), Cmd.none )

        UserDirectoryReceived result ->
            ( Api.updateLoggedIn model
                (\state ->
                    case result of
                        Ok users ->
                            { state | userDirectory = users }

                        Err _ ->
                            { state | userDirectory = [] }
                )
            , Cmd.none
            )

        UserDirectoryQueryChanged value ->
            Api.withSession model
                (\state ->
                    ( Api.updateLoggedIn model (\current -> { current | userDirectoryQuery = value, userDirectoryOffset = 0 }), Api.fetchUserDirectoryPage state.accessToken value 0 )
                )

        SearchUserDirectoryClicked ->
            Api.withSession model
                (\state ->
                    ( Api.updateLoggedIn model (\current -> { current | userDirectoryOffset = 0 }), Api.fetchUserDirectoryPage state.accessToken state.userDirectoryQuery 0 )
                )

        PreviousUserDirectoryPageClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            max 0 (state.userDirectoryOffset - Api.selectorPageSize)
                    in
                    ( Api.updateLoggedIn model (\current -> { current | userDirectoryOffset = offset }), Api.fetchUserDirectoryPage state.accessToken state.userDirectoryQuery offset )
                )

        NextUserDirectoryPageClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            state.userDirectoryOffset + Api.selectorPageSize
                    in
                    ( Api.updateLoggedIn model (\current -> { current | userDirectoryOffset = offset }), Api.fetchUserDirectoryPage state.accessToken state.userDirectoryQuery offset )
                )

        OrganizationQueryChanged value ->
            Api.withSession model
                (\state ->
                    ( Api.updateLoggedIn model (\current -> { current | organizationQuery = value, organizationOffset = 0 }), Api.fetchOrganizationsPage state.accessToken value 0 )
                )

        SearchOrganizationsClicked ->
            Api.withSession model
                (\state ->
                    ( Api.updateLoggedIn model (\current -> { current | organizationOffset = 0 }), Api.fetchOrganizationsPage state.accessToken state.organizationQuery 0 )
                )

        PreviousOrganizationsPageClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            max 0 (state.organizationOffset - Api.selectorPageSize)
                    in
                    ( Api.updateLoggedIn model (\current -> { current | organizationOffset = offset }), Api.fetchOrganizationsPage state.accessToken state.organizationQuery offset )
                )

        NextOrganizationsPageClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            state.organizationOffset + Api.selectorPageSize
                    in
                    ( Api.updateLoggedIn model (\current -> { current | organizationOffset = offset }), Api.fetchOrganizationsPage state.accessToken state.organizationQuery offset )
                )

        StandaloneTeamQueryChanged value ->
            Api.withSession model
                (\state ->
                    ( Api.updateLoggedIn model (\current -> { current | standaloneTeamQuery = value, standaloneTeamOffset = 0 }), Api.fetchStandaloneTeamsPage state.accessToken value 0 )
                )

        SearchStandaloneTeamsClicked ->
            Api.withSession model
                (\state ->
                    ( Api.updateLoggedIn model (\current -> { current | standaloneTeamOffset = 0 }), Api.fetchStandaloneTeamsPage state.accessToken state.standaloneTeamQuery 0 )
                )

        PreviousStandaloneTeamsPageClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            max 0 (state.standaloneTeamOffset - Api.selectorPageSize)
                    in
                    ( Api.updateLoggedIn model (\current -> { current | standaloneTeamOffset = offset }), Api.fetchStandaloneTeamsPage state.accessToken state.standaloneTeamQuery offset )
                )

        NextStandaloneTeamsPageClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            state.standaloneTeamOffset + Api.selectorPageSize
                    in
                    ( Api.updateLoggedIn model (\current -> { current | standaloneTeamOffset = offset }), Api.fetchStandaloneTeamsPage state.accessToken state.standaloneTeamQuery offset )
                )

        OrgTeamQueryChanged value ->
            Api.withSession model
                (\state ->
                    ( Api.updateLoggedIn model (\current -> { current | orgTeamQuery = value, orgTeamOffset = 0 }), Api.fetchOrgTeamsPage state.accessToken (orgTeamSearchOrganizationID state) value 0 )
                )

        SearchOrgTeamsClicked ->
            Api.withSession model
                (\state ->
                    ( Api.updateLoggedIn model (\current -> { current | orgTeamOffset = 0 }), Api.fetchOrgTeamsPage state.accessToken (orgTeamSearchOrganizationID state) state.orgTeamQuery 0 )
                )

        PreviousOrgTeamsPageClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            max 0 (state.orgTeamOffset - Api.selectorPageSize)
                    in
                    ( Api.updateLoggedIn model (\current -> { current | orgTeamOffset = offset }), Api.fetchOrgTeamsPage state.accessToken (orgTeamSearchOrganizationID state) state.orgTeamQuery offset )
                )

        NextOrgTeamsPageClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            state.orgTeamOffset + Api.selectorPageSize
                    in
                    ( Api.updateLoggedIn model (\current -> { current | orgTeamOffset = offset }), Api.fetchOrgTeamsPage state.accessToken (orgTeamSearchOrganizationID state) state.orgTeamQuery offset )
                )

        OrgMembersReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | orgMembers = Api.membersFromResult result }), Cmd.none )

        UserProfileReceived result ->
            ( Api.updateLoggedIn model
                (\state ->
                    case result of
                        Ok profile ->
                            { state | userProfile = Just profile, userProfileError = Nothing }

                        Err error ->
                            { state | userProfile = Nothing, userProfileError = Just (httpErrorLabel error) }
                )
            , Cmd.none
            )

        UserWorkReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | userWork = Api.tasksFromResult result }), Cmd.none )

        UserSubmissionsReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | userSubmissions = Api.submissionsFromResult result }), Cmd.none )

        StartRevisionClicked taskId responseJson ->
            ( Api.updateLoggedIn model (\state -> { state | pendingRevisionTaskID = Just taskId, pendingRevisionResponse = responseJson })
            , Nav.pushUrl model.key ("#/tasks/" ++ taskId)
            )

        SeriesListReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | seriesList = Api.seriesFromResult result }), Cmd.none )

        CreateSeriesTitleChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | createSeriesTitle = value }), Cmd.none )

        CreateSeriesDescriptionChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | createSeriesDescription = value }), Cmd.none )

        CreateSeriesClicked ->
            Api.withSession model (\state -> Api.createSeriesCommand model state)

        SeriesDetailReceived result ->
            ( Api.updateLoggedIn model
                (\state ->
                    case result of
                        Ok data ->
                            { state | seriesDetail = Just data, seriesDetailError = Nothing, seriesRenameTitle = data.series.title, seriesRenameDescription = data.series.description }

                        Err error ->
                            { state | seriesDetail = Nothing, seriesDetailError = Just (httpErrorLabel error) }
                )
            , Cmd.none
            )

        SeriesMutationReceived (Ok data) ->
            ( Api.updateLoggedIn model
                (\state ->
                    { state
                        | seriesDetail = Just data
                        , createSeriesTitle = ""
                        , createSeriesDescription = ""
                        , addSeriesTaskId = ""
                        , seriesRenameTitle = data.series.title
                        , seriesRenameDescription = data.series.description
                        , seriesMessage = Just "Series saved."
                    }
                )
            , seriesListRefresh model
            )

        SeriesMutationReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | seriesMessage = Just (httpErrorLabel error) }), Cmd.none )

        PublishSeriesClicked seriesId ->
            Api.withSession model (\state -> ( model, Api.seriesStateCommand state.accessToken seriesId "publish" ))

        UnpublishSeriesClicked seriesId ->
            Api.withSession model (\state -> ( model, Api.seriesStateCommand state.accessToken seriesId "unpublish" ))

        CloseSeriesClicked seriesId ->
            Api.withSession model (\state -> ( model, Api.seriesStateCommand state.accessToken seriesId "close" ))

        ReopenSeriesClicked seriesId ->
            Api.withSession model (\state -> ( model, Api.seriesStateCommand state.accessToken seriesId "reopen" ))

        AddSeriesTaskIdChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | addSeriesTaskId = value }), Cmd.none )

        AddSeriesTaskClicked seriesId ->
            Api.withSession model (\state -> Api.addSeriesTaskCommand model state seriesId)

        RemoveSeriesTaskClicked seriesId taskId ->
            Api.withSession model (\state -> ( model, Api.removeSeriesTaskCommand state.accessToken seriesId taskId ))

        MoveSeriesTaskUpClicked seriesId taskId ->
            seriesReorder model seriesId taskId True

        MoveSeriesTaskDownClicked seriesId taskId ->
            seriesReorder model seriesId taskId False

        SeriesCommentBodyChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | seriesCommentBody = value }), Cmd.none )

        AddSeriesCommentClicked seriesId ->
            Api.withSession model (\state -> Api.addSeriesCommentCommand model state seriesId)

        SeriesCommentReceived (Ok comment) ->
            ( Api.updateLoggedIn model
                (\state ->
                    { state
                        | seriesCommentBody = ""
                        , seriesDetail = Maybe.map (\data -> { data | comments = data.comments ++ [ comment ] }) state.seriesDetail
                    }
                )
            , Cmd.none
            )

        SeriesCommentReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | seriesMessage = Just (httpErrorLabel error) }), Cmd.none )

        SeriesRenameTitleChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | seriesRenameTitle = value }), Cmd.none )

        SeriesRenameDescriptionChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | seriesRenameDescription = value }), Cmd.none )

        UpdateSeriesClicked seriesId ->
            Api.withSession model (\state -> Api.updateSeriesCommand model state seriesId)

        TeamDetailReceived result ->
            ( Api.updateLoggedIn model
                (\state ->
                    case result of
                        Ok detail ->
                            { state | teamDetail = Just detail, teamDetailError = Nothing }

                        Err error ->
                            { state | teamDetail = Nothing, teamDetailError = Just (httpErrorLabel error) }
                )
            , Cmd.none
            )

        TeamWorkReceived result ->
            ( Api.updateLoggedIn model
                (\state ->
                    case result of
                        Ok response ->
                            { state | teamWork = response.tasks, teamWorkMessage = Nothing }

                        Err error ->
                            { state | teamWork = [], teamWorkMessage = Just (httpErrorLabel error) }
                )
            , Cmd.none
            )

        TeamWorkQueryChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | teamWorkQuery = value }), Cmd.none )

        TeamWorkFilterChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | teamWorkFilter = value }), Cmd.none )

        TeamWorkTypeFilterChanged value ->
            let
                updated =
                    Api.updateLoggedIn model (\state -> { state | teamWorkTypeFilter = value, teamWorkOffset = 0 })
            in
            Api.withSession updated
                (\state ->
                    case state.teamDetail of
                        Just detail ->
                            ( updated, Api.fetchTeamWork state.accessToken detail.team.id state.teamWorkQuery value state.teamWorkSort 0 )

                        Nothing ->
                            ( updated, Cmd.none )
                )

        TeamWorkSortChanged value ->
            let
                updated =
                    Api.updateLoggedIn model (\state -> { state | teamWorkSort = value, teamWorkOffset = 0 })
            in
            Api.withSession updated
                (\state ->
                    case state.teamDetail of
                        Just detail ->
                            ( updated, Api.fetchTeamWork state.accessToken detail.team.id state.teamWorkQuery state.teamWorkTypeFilter value 0 )

                        Nothing ->
                            ( updated, Cmd.none )
                )

        TeamWorkSavedViewNameChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | teamWorkSavedViewName = value }), Cmd.none )

        SaveTeamWorkViewClicked ->
            Api.withSession model
                (\state ->
                    let
                        name =
                            String.trim state.teamWorkSavedViewName
                    in
                    if name == "" then
                        ( Api.updateLoggedIn model (\current -> { current | teamWorkMessage = Just "A saved view name is required." }), Cmd.none )

                    else
                        let
                            view =
                                { name = name
                                , query = state.teamWorkQuery
                                , stateFilter = state.teamWorkFilter
                                , typeFilter = state.teamWorkTypeFilter
                                , sort = state.teamWorkSort
                                }
                        in
                        ( Api.updateLoggedIn model (\current -> { current | teamWorkMessage = Nothing }), Api.saveSavedQueueView state.accessToken teamWorkSavedViewScope view )
                )

        ApplyTeamWorkViewClicked name ->
            Api.withSession model
                (\state ->
                    case ( state.teamDetail, queueViewByName name state.teamWorkSavedViews ) of
                        ( Just detail, Just view ) ->
                            ( Api.updateLoggedIn model
                                (\current ->
                                    { current
                                        | teamWorkQuery = view.query
                                        , teamWorkFilter = view.stateFilter
                                        , teamWorkTypeFilter = view.typeFilter
                                        , teamWorkSort = view.sort
                                        , teamWorkOffset = 0
                                        , teamWorkMessage = Just ("Applied view: " ++ view.name)
                                    }
                                )
                            , Api.fetchTeamWork state.accessToken detail.team.id view.query view.typeFilter view.sort 0
                            )

                        ( _, Nothing ) ->
                            ( Api.updateLoggedIn model (\current -> { current | teamWorkMessage = Just "Saved view was not found." }), Cmd.none )

                        ( Nothing, _ ) ->
                            ( model, Cmd.none )
                )

        SearchTeamWorkClicked ->
            Api.withSession model
                (\state ->
                    case state.teamDetail of
                        Just detail ->
                            let
                                offset =
                                    0
                            in
                            ( Api.updateLoggedIn model (\current -> { current | teamWorkOffset = offset }), Api.fetchTeamWork state.accessToken detail.team.id state.teamWorkQuery state.teamWorkTypeFilter state.teamWorkSort offset )

                        Nothing ->
                            ( model, Cmd.none )
                )

        PreviousTeamWorkPageClicked ->
            Api.withSession model
                (\state ->
                    case state.teamDetail of
                        Just detail ->
                            let
                                offset =
                                    max 0 (state.teamWorkOffset - Api.selectorPageSize)
                            in
                            ( Api.updateLoggedIn model (\current -> { current | teamWorkOffset = offset }), Api.fetchTeamWork state.accessToken detail.team.id state.teamWorkQuery state.teamWorkTypeFilter state.teamWorkSort offset )

                        Nothing ->
                            ( model, Cmd.none )
                )

        NextTeamWorkPageClicked ->
            Api.withSession model
                (\state ->
                    case state.teamDetail of
                        Just detail ->
                            let
                                offset =
                                    state.teamWorkOffset + Api.selectorPageSize
                            in
                            ( Api.updateLoggedIn model (\current -> { current | teamWorkOffset = offset }), Api.fetchTeamWork state.accessToken detail.team.id state.teamWorkQuery state.teamWorkTypeFilter state.teamWorkSort offset )

                        Nothing ->
                            ( model, Cmd.none )
                )

        TeamMemberEmailChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | teamMemberEmail = value }), Cmd.none )

        AddTeamMemberClicked teamId ->
            Api.withSession model (\state -> ( model, Api.postAddTeamMember state.accessToken teamId state.teamMemberEmail ))

        AddTeamMemberReceived (Ok detail) ->
            ( Api.updateLoggedIn model (\state -> { state | teamDetail = Just detail, teamMemberEmail = "", teamMemberMessage = Just "Member added." }), Cmd.none )

        AddTeamMemberReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | teamMemberMessage = Just (httpErrorLabel error) }), Cmd.none )

        OrgTasksReceived (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | orgTasks = response.tasks, orgTaskMessage = Nothing }), Cmd.none )

        OrgTasksReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | orgTasks = [], orgTaskMessage = Just (httpErrorLabel error) }), Cmd.none )

        OrgTaskQueryChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | orgTaskQuery = value }), Cmd.none )

        OrgTaskFilterChanged value ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            0
                    in
                    ( Api.updateLoggedIn model (\current -> { current | orgTaskFilter = value, orgTaskOffset = offset }), Api.fetchOrgTasksPage state.accessToken state.activeOrgId state.orgTaskQuery value state.orgTaskTypeFilter state.orgTaskSort offset )
                )

        OrgTaskTypeFilterChanged value ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            0
                    in
                    ( Api.updateLoggedIn model (\current -> { current | orgTaskTypeFilter = value, orgTaskOffset = offset }), Api.fetchOrgTasksPage state.accessToken state.activeOrgId state.orgTaskQuery state.orgTaskFilter value state.orgTaskSort offset )
                )

        OrgTaskSortChanged value ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            0
                    in
                    ( Api.updateLoggedIn model (\current -> { current | orgTaskSort = value, orgTaskOffset = offset }), Api.fetchOrgTasksPage state.accessToken state.activeOrgId state.orgTaskQuery state.orgTaskFilter state.orgTaskTypeFilter value offset )
                )

        OrgTaskSavedViewNameChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | orgTaskSavedViewName = value }), Cmd.none )

        SaveOrgTaskViewClicked ->
            Api.withSession model
                (\state ->
                    let
                        name =
                            String.trim state.orgTaskSavedViewName
                    in
                    if name == "" then
                        ( Api.updateLoggedIn model (\current -> { current | orgTaskMessage = Just "A saved view name is required." }), Cmd.none )

                    else
                        let
                            view =
                                { name = name
                                , query = state.orgTaskQuery
                                , stateFilter = state.orgTaskFilter
                                , typeFilter = state.orgTaskTypeFilter
                                , sort = state.orgTaskSort
                                }
                        in
                        ( Api.updateLoggedIn model (\current -> { current | orgTaskMessage = Nothing }), Api.saveSavedQueueView state.accessToken orgTaskSavedViewScope view )
                )

        ApplyOrgTaskViewClicked name ->
            Api.withSession model
                (\state ->
                    case queueViewByName name state.orgTaskSavedViews of
                        Just view ->
                            ( Api.updateLoggedIn model
                                (\current ->
                                    { current
                                        | orgTaskQuery = view.query
                                        , orgTaskFilter = view.stateFilter
                                        , orgTaskTypeFilter = view.typeFilter
                                        , orgTaskSort = view.sort
                                        , orgTaskOffset = 0
                                        , orgTaskMessage = Just ("Applied view: " ++ view.name)
                                    }
                                )
                            , Api.fetchOrgTasksPage state.accessToken state.activeOrgId view.query view.stateFilter view.typeFilter view.sort 0
                            )

                        Nothing ->
                            ( Api.updateLoggedIn model (\current -> { current | orgTaskMessage = Just "Saved view was not found." }), Cmd.none )
                )

        SearchOrgTasksClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            0
                    in
                    ( Api.updateLoggedIn model (\current -> { current | orgTaskOffset = offset }), Api.fetchOrgTasksPage state.accessToken state.activeOrgId state.orgTaskQuery state.orgTaskFilter state.orgTaskTypeFilter state.orgTaskSort offset )
                )

        PreviousOrgTasksPageClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            max 0 (state.orgTaskOffset - Api.selectorPageSize)
                    in
                    ( Api.updateLoggedIn model (\current -> { current | orgTaskOffset = offset }), Api.fetchOrgTasksPage state.accessToken state.activeOrgId state.orgTaskQuery state.orgTaskFilter state.orgTaskTypeFilter state.orgTaskSort offset )
                )

        NextOrgTasksPageClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            state.orgTaskOffset + Api.selectorPageSize
                    in
                    ( Api.updateLoggedIn model (\current -> { current | orgTaskOffset = offset }), Api.fetchOrgTasksPage state.accessToken state.activeOrgId state.orgTaskQuery state.orgTaskFilter state.orgTaskTypeFilter state.orgTaskSort offset )
                )

        OrgCollectiblesReceived (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | orgCollectibles = response.collectibles, orgCollectiblesMessage = Nothing }), Cmd.none )

        OrgCollectiblesReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | orgCollectibles = [], orgCollectiblesMessage = Just (httpErrorLabel error) }), Cmd.none )

        TeamCollectiblesReceived (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | teamCollectibles = response.collectibles, teamCollectiblesMessage = Nothing }), Cmd.none )

        TeamCollectiblesReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | teamCollectibles = [], teamCollectiblesMessage = Just (httpErrorLabel error) }), Cmd.none )

        CreateOrgTeamNameChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | createOrgTeamName = value }), Cmd.none )

        CreateOrgTeamClicked ->
            Api.withSession model (\state -> Api.createOrgTeamCommand model state)

        CreateOrgTeamReceived (Ok team) ->
            let
                updated =
                    Api.updateLoggedIn model (\state -> { state | createOrgTeamName = "", orgTeamMessage = Just ("Created team " ++ team.name) })
            in
            Api.withSession updated (\state -> ( updated, Api.fetchOrgTeams state.accessToken state.activeOrgId ))

        CreateOrgTeamReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | orgTeamMessage = Just (httpErrorLabel error) }), Cmd.none )

        ProvisionMemberEmailChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | provisionMemberEmail = value }), Cmd.none )

        ToggleProvisionMemberRole role ->
            ( Api.updateLoggedIn model (\state -> { state | provisionMemberRoles = toggleString role state.provisionMemberRoles }), Cmd.none )

        ProvisionMemberClicked ->
            Api.withSession model (\state -> Api.provisionMemberCommand model state)

        ProvisionMemberReceived (Ok ()) ->
            let
                updated =
                    Api.updateLoggedIn model (\state -> { state | provisionMemberEmail = "", provisionMemberMessage = Just "Member provisioned." })
            in
            Api.withSession updated (\state -> ( updated, Api.authorizedRequest "GET" state.accessToken ("/api/organizations/" ++ state.activeOrgId ++ "/members") Http.emptyBody (Http.expectJson OrgMembersReceived Organization.organizationMembersResponseDecoder) ))

        ProvisionMemberReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | provisionMemberMessage = Just (httpErrorLabel error) }), Cmd.none )

        UpdateMemberRolesClicked userId roles ->
            Api.withSession model (\state -> Api.updateMemberRolesCommand model state userId roles)

        UpdateMemberRolesReceived (Ok _) ->
            let
                updated =
                    Api.updateLoggedIn model (\state -> { state | provisionMemberMessage = Just "Member roles updated." })
            in
            Api.withSession updated (\state -> ( updated, Api.authorizedRequest "GET" state.accessToken ("/api/organizations/" ++ state.activeOrgId ++ "/members") Http.emptyBody (Http.expectJson OrgMembersReceived Organization.organizationMembersResponseDecoder) ))

        UpdateMemberRolesReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | provisionMemberMessage = Just (httpErrorLabel error) }), Cmd.none )

        DeactivateMemberClicked userId ->
            Api.withSession model (\state -> Api.deactivateMemberCommand model state userId)

        DeactivateMemberReceived (Ok _) ->
            let
                updated =
                    Api.updateLoggedIn model (\state -> { state | provisionMemberMessage = Just "Member deactivated." })
            in
            Api.withSession updated (\state -> ( updated, Api.authorizedRequest "GET" state.accessToken ("/api/organizations/" ++ state.activeOrgId ++ "/members") Http.emptyBody (Http.expectJson OrgMembersReceived Organization.organizationMembersResponseDecoder) ))

        DeactivateMemberReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | provisionMemberMessage = Just (httpErrorLabel error) }), Cmd.none )

        CreateTaskOwnerChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | createTaskOwner = value }), Cmd.none )

        CreateTaskTypeChanged value ->
            ( Api.updateLoggedIn model
                (\state ->
                    case View.taskTemplate value of
                        Just template ->
                            -- The template owns the schema; clear the designer
                            -- fields so a later designer edit can't silently
                            -- overwrite the prefilled schema.
                            { state | createTaskType = value, createDescription = template.description, createResponseSchema = template.schema, createSchemaFields = [] }

                        Nothing ->
                            -- Freeform: hand the schema back to the designer.
                            { state | createTaskType = value, createResponseSchema = "{\"kind\":\"freeform\"}", createSchemaFields = [] }
                )
            , Cmd.none
            )

        CreateReferenceURLChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | createReferenceURL = value }), Cmd.none )

        TaskCommentBodyChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | taskCommentBody = value }), Cmd.none )

        AddTaskCommentClicked taskId ->
            Api.withSession model
                (\state ->
                    if String.trim state.taskCommentBody == "" then
                        ( Api.updateLoggedIn model (\current -> { current | taskCommentMessage = Just "Write a comment first." }), Cmd.none )

                    else
                        ( Api.updateLoggedIn model (\current -> { current | taskCommentMessage = Nothing })
                        , Api.postTaskComment state.accessToken taskId (String.trim state.taskCommentBody)
                        )
                )

        TaskCommentReceived (Ok comment) ->
            ( Api.updateLoggedIn model (\state -> { state | taskComments = state.taskComments ++ [ comment ], taskCommentBody = "", taskCommentMessage = Nothing }), Cmd.none )

        TaskCommentReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | taskCommentMessage = Just (httpErrorLabel error) }), Cmd.none )

        TaskCommentsReceived (Ok comments) ->
            ( Api.updateLoggedIn model (\state -> { state | taskComments = comments }), Cmd.none )

        TaskCommentsReceived (Err _) ->
            ( Api.updateLoggedIn model (\state -> { state | taskComments = [] }), Cmd.none )

        OpenSubmissionComments submissionId ->
            Api.withSession model
                (\state ->
                    ( Api.updateLoggedIn model (\current -> { current | activeSubmissionCommentsID = Just submissionId, submissionComments = [], submissionCommentMessage = Nothing })
                    , Api.fetchSubmissionComments state.accessToken submissionId
                    )
                )

        SubmissionCommentsReceived (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | submissionComments = response.comments }), Cmd.none )

        SubmissionCommentsReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | submissionCommentMessage = Just (httpErrorLabel error) }), Cmd.none )

        SubmissionCommentBodyChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | submissionCommentBody = value }), Cmd.none )

        AddSubmissionCommentClicked submissionId ->
            Api.withSession model
                (\state ->
                    if String.trim state.submissionCommentBody == "" then
                        ( Api.updateLoggedIn model (\current -> { current | submissionCommentMessage = Just "Write a comment first." }), Cmd.none )

                    else
                        ( Api.updateLoggedIn model (\current -> { current | submissionCommentMessage = Nothing })
                        , Api.addSubmissionComment state.accessToken submissionId (String.trim state.submissionCommentBody)
                        )
                )

        SubmissionCommentAdded (Ok _) ->
            Api.withSession model
                (\state ->
                    ( Api.updateLoggedIn model (\current -> { current | submissionCommentBody = "" })
                    , case state.activeSubmissionCommentsID of
                        Just submissionId ->
                            Api.fetchSubmissionComments state.accessToken submissionId

                        Nothing ->
                            Cmd.none
                    )
                )

        SubmissionCommentAdded (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | submissionCommentMessage = Just (httpErrorLabel error) }), Cmd.none )

        AccountEmailChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | accountEmail = value }), Cmd.none )

        CurrentPasswordChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | currentPassword = value }), Cmd.none )

        NewPasswordChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | newPassword = value }), Cmd.none )

        EmailVerificationInputChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | emailVerificationInput = value }), Cmd.none )

        RequestEmailVerificationClicked ->
            Api.withSession model (\state -> ( Api.updateLoggedIn model (\current -> { current | accountMessage = Nothing }), Api.requestEmailVerification state.accessToken ))

        ConfirmEmailVerificationClicked ->
            Api.withSession model (\state -> ( Api.updateLoggedIn model (\current -> { current | accountMessage = Nothing }), Api.confirmEmailVerification state.accessToken state.emailVerificationInput ))

        UpdateProfileClicked ->
            Api.withSession model (\state -> ( Api.updateLoggedIn model (\current -> { current | accountMessage = Nothing }), Api.updateProfile state.accessToken state.accountEmail ))

        ChangePasswordClicked ->
            Api.withSession model (\state -> ( Api.updateLoggedIn model (\current -> { current | accountMessage = Nothing }), Api.changePassword state.accessToken state.currentPassword state.newPassword ))

        DeactivateAccountClicked ->
            Api.withSession model (\state -> ( Api.updateLoggedIn model (\current -> { current | accountMessage = Nothing }), Api.deactivateAccount state.accessToken ))

        PrivacyRequestClicked kind ->
            Api.withSession model (\state -> ( Api.updateLoggedIn model (\current -> { current | accountMessage = Nothing }), Api.requestPrivacy state.accessToken kind ))

        EmailVerificationRequested (Ok token) ->
            if token == "" then
                ( Api.updateLoggedIn model (\state -> { state | accountMessage = Just "Verification instructions sent." }), Cmd.none )

            else
                ( Api.updateLoggedIn model (\state -> { state | emailVerificationToken = token, emailVerificationInput = token, accountMessage = Just "Verification token created." }), Cmd.none )

        EmailVerificationRequested (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | accountMessage = Just (httpErrorLabel error) }), Cmd.none )

        AccountActionReceived (Ok ()) ->
            ( Api.updateLoggedIn model (\state -> { state | currentPassword = "", newPassword = "", emailVerificationInput = "", accountMessage = Just "Account updated." }), Cmd.none )

        AccountActionReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | accountMessage = Just (httpErrorLabel error) }), Cmd.none )

        DeactivateAccountReceived (Ok ()) ->
            ( { model | session = LoggedOut, email = "", password = "" }
            , Nav.pushUrl model.key "#/"
            )

        DeactivateAccountReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | accountMessage = Just (httpErrorLabel error) }), Cmd.none )

        PrivacyRequestReceived (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | accountMessage = Just ("Privacy request queued: " ++ response.kind) }), Cmd.none )

        PrivacyRequestReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | accountMessage = Just (httpErrorLabel error) }), Cmd.none )

        SavedQueueViewsReceived (Ok response) ->
            let
                teamViews =
                    response.views
                        |> List.filter (\view -> view.scope == teamWorkSavedViewScope)
                        |> List.map queueViewFromResponse

                orgViews =
                    response.views
                        |> List.filter (\view -> view.scope == orgTaskSavedViewScope)
                        |> List.map queueViewFromResponse
            in
            ( Api.updateLoggedIn model (\state -> { state | teamWorkSavedViews = teamViews, orgTaskSavedViews = orgViews }), Cmd.none )

        SavedQueueViewsReceived (Err _) ->
            ( model, Cmd.none )

        SavedQueueViewSaved (Ok response) ->
            let
                view =
                    queueViewFromResponse response
            in
            if response.scope == teamWorkSavedViewScope then
                ( Api.updateLoggedIn model (\state -> { state | teamWorkSavedViews = saveQueueView view state.teamWorkSavedViews, teamWorkSavedViewName = "", teamWorkMessage = Just ("Saved view: " ++ view.name) }), Cmd.none )

            else if response.scope == orgTaskSavedViewScope then
                ( Api.updateLoggedIn model (\state -> { state | orgTaskSavedViews = saveQueueView view state.orgTaskSavedViews, orgTaskSavedViewName = "", orgTaskMessage = Just ("Saved view: " ++ view.name) }), Cmd.none )

            else
                ( model, Cmd.none )

        SavedQueueViewSaved (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | teamWorkMessage = Just (httpErrorLabel error), orgTaskMessage = Just (httpErrorLabel error) }), Cmd.none )

        OperationsReceived (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | operations = Just response, adminMessage = Nothing }), Cmd.none )

        OperationsReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | operations = Nothing, adminMessage = Just (httpErrorLabel error) }), Cmd.none )

        AuditEventsReceived (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | auditEvents = response.events, adminMessage = Nothing }), Cmd.none )

        AuditEventsReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | auditEvents = [], adminMessage = Just (httpErrorLabel error) }), Cmd.none )

        AdminPrivacyRequestsReceived (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | adminPrivacyRequests = response.requests, adminMessage = Nothing }), Cmd.none )

        AdminPrivacyRequestsReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | adminPrivacyRequests = [], adminMessage = Just (httpErrorLabel error) }), Cmd.none )

        AdminPrivacyResolutionNoteChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | adminPrivacyResolutionNote = value }), Cmd.none )

        ResolveAdminPrivacyRequestClicked requestId ->
            Api.withSession model (\state -> ( Api.updateLoggedIn model (\current -> { current | adminMessage = Nothing }), Api.resolveAdminPrivacyRequest state.accessToken requestId state.adminPrivacyResolutionNote ))

        AdminPrivacyRequestResolved (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | adminPrivacyRequests = replacePrivacyRequest response state.adminPrivacyRequests, adminPrivacyResolutionNote = "", adminMessage = Just "Privacy request resolved." }), Cmd.none )

        AdminPrivacyRequestResolved (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | adminMessage = Just (httpErrorLabel error) }), Cmd.none )

        AuditActionFilterChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | auditActionFilter = value }), Cmd.none )

        AuditSubjectKindFilterChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | auditSubjectKindFilter = value }), Cmd.none )

        AuditSubjectIDFilterChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | auditSubjectIDFilter = value }), Cmd.none )

        SearchAuditEventsClicked ->
            Api.withSession model (\state -> ( model, Api.fetchAuditEvents state.accessToken state.auditActionFilter state.auditSubjectKindFilter state.auditSubjectIDFilter ))

        NotificationsReceived (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | notifications = response.notifications, inboxMessage = Nothing }), Cmd.none )

        NotificationsReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | notifications = [], inboxMessage = Just (httpErrorLabel error) }), Cmd.none )

        MarkNotificationReadClicked notificationId ->
            Api.withSession model (\state -> ( model, Api.markNotificationRead state.accessToken notificationId ))

        NotificationReadReceived (Ok notification) ->
            ( Api.updateLoggedIn model (\state -> { state | notifications = replaceNotification notification state.notifications, inboxMessage = Nothing }), Cmd.none )

        NotificationReadReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | inboxMessage = Just (httpErrorLabel error) }), Cmd.none )

        LinkClicked request ->
            case request of
                Browser.Internal url ->
                    ( model, Nav.pushUrl model.key (Url.toString url) )

                Browser.External href ->
                    ( model, Nav.load href )

        UrlChanged url ->
            let
                page =
                    pageFromUrl url
            in
            case model.session of
                LoggedIn state ->
                    ( { model | route = page, session = LoggedIn (enterPage page state) }
                    , Api.routeLoadCmd state.accessToken state.subjectId page
                    )

                LoggedOut ->
                    ( { model | route = page }, Cmd.none )

        ResetDemoClicked ->
            ( model, reloadDemo () )


seriesListRefresh : Model -> Cmd Msg
seriesListRefresh model =
    case model.session of
        LoggedIn state ->
            if state.page == SeriesListPage then
                Api.fetchSeriesList state.accessToken

            else
                Cmd.none

        LoggedOut ->
            Cmd.none


seriesReorder : Model -> String -> String -> Bool -> ( Model, Cmd Msg )
seriesReorder model seriesId taskId up =
    Api.withSession model
        (\state ->
            case state.seriesDetail of
                Just data ->
                    ( model, Api.reorderSeriesCommand state.accessToken seriesId (Api.moveSeriesTaskOrder up taskId data.tasks) )

                Nothing ->
                    ( model, Cmd.none )
        )


toggleString : String -> List String -> List String
toggleString value values =
    if List.member value values then
        List.filter (\existing -> existing /= value) values

    else
        value :: values
