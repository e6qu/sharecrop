module Main exposing (main)

import Browser
import Browser.Navigation as Nav
import Html exposing (Html, a, div, form, label, main_, option, p, select, span, table, tbody, td, text, th, thead, tr)
import Html.Attributes exposing (checked, disabled, href, placeholder, selected, type_, value)
import Html.Events exposing (onCheck, onClick, onInput, onSubmit)
import Http
import Json.Decode as Decode
import Json.Encode as Encode
import Sharecrop.Generated.Agent as Agent
import Sharecrop.Generated.Auth as Auth
import Sharecrop.Generated.Collectible as Collectible
import Sharecrop.Generated.Ledger as Ledger
import Sharecrop.Generated.Organization as Organization
import Sharecrop.Generated.Submission as Submission
import Sharecrop.Generated.Task as Task
import Sharecrop.Generated.TaskSeries as TaskSeries
import Sharecrop.Generated.Team as Team
import Sharecrop.Labels
    exposing
        ( allScopes
        , assigneeScopeLabel
        , assigneeScopeTag
        , availabilityKindLabel
        , collectibleKindLabel
        , collectibleKindTag
        , collectiblePolicyLabel
        , collectiblePolicyTag
        , collectibleStateLabel
        , credentialStateLabel
        , escrowStateLabel
        , httpErrorLabel
        , kindLabel
        , participationPolicyLabel
        , participationPolicyTag
        , reservationStateLabel
        , rewardLabel
        , scopeTag
        , submissionStateLabel
        , taskStateGuidance
        , taskStateLabel
        , viewerActionLabel
        )
import Sharecrop.Types exposing (..)
import Sharecrop.View as View
import Sharecrop.Ui as Ui exposing (testId)
import Url exposing (Url)




main : Program Flags Model Msg
main =
    Browser.application
        { init = \flags url key -> ( initialModel flags key url, postRefresh )
        , update = update
        , subscriptions = \_ -> Sub.none
        , view = View.view
        , onUrlRequest = LinkClicked
        , onUrlChange = UrlChanged
        }


initialModel : Flags -> Nav.Key -> Url -> Model
initialModel flags key url =
    { origin = flags.origin
    , key = key
    , route = pageFromUrl url
    , email = ""
    , password = ""
    , authError = Nothing
    , session = LoggedOut
    }


emptyLoggedIn : Auth.AuthResponse -> LoggedInModel
emptyLoggedIn response =
    { accessToken = response.accessToken
    , subjectId = response.subjectID
    , page = OverviewPage
    , balance = Nothing
    , entries = []
    , createTitle = ""
    , createDescription = ""
    , createRewardAmount = ""
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
    , tasks = []
    , taskStateFilter = ""
    , agentLabel = ""
    , agentScopes = [ Agent.AgentScopeTasksRead, Agent.AgentScopeSubmissionsWrite ]
    , credentials = []
    , newCredential = Nothing
    , agentMessage = Nothing
    , discoveryTasks = []
    , discoveryIncludeReserved = False
    , detail = Nothing
    , reservations = []
    , reservationMessage = Nothing
    , submissions = []
    , submitInput = ""
    , submitMessage = Nothing
    , reviewNote = ""
    , reviewPartialCredit = ""
    , reviewTip = ""
    , reviewBan = False
    , reviewMessage = Nothing
    , collectibles = []
    , collectibleName = ""
    , collectibleKind = Collectible.CollectibleKindBadge
    , collectiblePolicy = Collectible.CollectibleTransferPolicyNonTransferableExceptPayout
    , collectibleMessage = Nothing
    , awardTaskId = ""
    , awardMessage = Nothing
    , organizations = []
    , createOrgName = ""
    , orgMessage = Nothing
    , activeOrgId = ""
    , orgBalance = Nothing
    , orgTeams = []
    , orgMembers = []
    , orgTasks = []
    , userProfile = Nothing
    , userWork = []
    , userSubmissions = []
    , seriesDetail = Nothing
    , teamDetail = Nothing
    , teamMemberEmail = ""
    , teamMemberMessage = Nothing
    , createOrgTeamName = ""
    , orgTeamMessage = Nothing
    , provisionMemberEmail = ""
    , provisionMemberMessage = Nothing
    , createTaskOwner = ""
    }


loggedInForPage : Auth.AuthResponse -> Page -> LoggedInModel
loggedInForPage response page =
    let
        state =
            emptyLoggedIn response
    in
    { state | page = page }


pageFromUrl : Url -> Page
pageFromUrl url =
    case String.split "/" (String.dropLeft 1 url.path) of
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

        [ "series", seriesId ] ->
            SeriesDetailPage seriesId

        [ "teams", teamId ] ->
            TeamDetailPage teamId

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
            OverviewPage


-- enterPage applies any per-page state a route needs when it becomes active, so
-- a deep link or back/forward leaves the model consistent with the URL.
enterPage : Page -> LoggedInModel -> LoggedInModel
enterPage page state =
    case page of
        OrganizationDetailPage organizationId ->
            { state | page = page, activeOrgId = organizationId, orgBalance = Nothing, orgTeams = [], orgMembers = [], orgTasks = [], orgTeamMessage = Nothing, provisionMemberMessage = Nothing }

        UserDetailPage _ ->
            { state | page = page, userProfile = Nothing }

        UserWorkPage _ ->
            { state | page = page, userWork = [] }

        UserSubmissionsPage _ ->
            { state | page = page, userSubmissions = [] }

        SeriesDetailPage _ ->
            { state | page = page, seriesDetail = Nothing }

        TeamDetailPage _ ->
            { state | page = page, teamDetail = Nothing, teamMemberEmail = "", teamMemberMessage = Nothing }

        _ ->
            { state | page = page }


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        EmailChanged value ->
            ( { model | email = value }, Cmd.none )

        PasswordChanged value ->
            ( { model | password = value }, Cmd.none )

        RegisterClicked ->
            ( { model | authError = Nothing }, postAuth "/api/auth/register" model )

        LoginClicked ->
            ( { model | authError = Nothing }, postAuth "/api/auth/login" model )

        AuthReceived (Ok response) ->
            ( { model | password = "", authError = Nothing, session = LoggedIn (loggedInForPage response model.route) }
            , loadAfterAuth response.accessToken
            )

        AuthReceived (Err error) ->
            ( { model | authError = Just (httpErrorLabel error) }, Cmd.none )

        RefreshReceived (Ok response) ->
            ( { model | session = LoggedIn (loggedInForPage response model.route) }
            , Cmd.batch [ loadAfterAuth response.accessToken, routeLoadCmd response.accessToken model.route ]
            )

        RefreshReceived (Err _) ->
            ( model, Cmd.none )

        BalanceReceived result ->
            ( updateLoggedIn model (\state -> { state | balance = balanceFromResult result }), Cmd.none )

        LedgerReceived result ->
            ( updateLoggedIn model (\state -> { state | entries = entriesFromResult result }), Cmd.none )

        TasksReceived result ->
            ( updateLoggedIn model (\state -> { state | tasks = tasksFromResult result }), Cmd.none )

        TaskStateFilterChanged value ->
            let
                updated =
                    updateLoggedIn model (\state -> { state | taskStateFilter = value })
            in
            withSession updated (\state -> ( updated, fetchTasks state.accessToken value ))

        CreateTitleChanged value ->
            ( updateLoggedIn model (\state -> { state | createTitle = value }), Cmd.none )

        CreateDescriptionChanged value ->
            ( updateLoggedIn model (\state -> { state | createDescription = value }), Cmd.none )

        CreateRewardAmountChanged value ->
            ( updateLoggedIn model (\state -> { state | createRewardAmount = value }), Cmd.none )

        CreateVisibilityChanged value ->
            ( updateLoggedIn model (\state -> { state | createVisibility = value }), Cmd.none )

        CreateScopeUserIdChanged value ->
            ( updateLoggedIn model (\state -> { state | createScopeUserId = value }), Cmd.none )

        CreateScopeTeamIdChanged value ->
            ( updateLoggedIn model (\state -> { state | createScopeTeamId = value }), Cmd.none )

        CreateScopeOrganizationIdChanged value ->
            ( updateLoggedIn model (\state -> { state | createScopeOrganizationId = value }), Cmd.none )

        CreateAssigneeScopeChosen scope ->
            ( updateLoggedIn model (\state -> { state | createAssigneeScope = scope }), Cmd.none )

        CreateParticipationChanged value ->
            ( updateLoggedIn model (\state -> { state | createParticipationPolicy = value }), Cmd.none )

        CreateReservationHoursChanged value ->
            ( updateLoggedIn model (\state -> { state | createReservationHours = value }), Cmd.none )

        CreateTaskClicked ->
            withSession model (\state -> createTaskCommand model state)

        CreateTaskReceived (Ok created) ->
            ( updateLoggedIn model
                (\state ->
                    { state
                        | createTitle = ""
                        , createDescription = ""
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
            , refreshTasksAndLedger model
            )

        CreateTaskReceived (Err error) ->
            ( updateLoggedIn model (\state -> { state | createMessage = Just (httpErrorLabel error) }), Cmd.none )

        CredentialsReceived result ->
            ( updateLoggedIn model (\state -> { state | credentials = credentialsFromResult result }), Cmd.none )

        FundTaskIdChanged value ->
            ( updateLoggedIn model (\state -> { state | fundTaskId = value }), Cmd.none )

        FundAmountChanged value ->
            ( updateLoggedIn model (\state -> { state | fundAmount = value }), Cmd.none )

        FundOrganizationIdChanged value ->
            ( updateLoggedIn model (\state -> { state | fundOrganizationId = value }), Cmd.none )

        FundClicked ->
            withSession model (\state -> fundTaskCommand model state)

        FundReceived (Ok escrow) ->
            ( updateLoggedIn model (\state -> { state | fundMessage = Just (View.fundSuccessLabel escrow) }), refreshLedger model )

        FundReceived (Err error) ->
            ( updateLoggedIn model (\state -> { state | fundMessage = Just (httpErrorLabel error) }), Cmd.none )

        OpenTaskClicked taskId ->
            withSession model (\state -> ( model, postOpenTask state.accessToken taskId ))

        OpenTaskReceived (Ok detail) ->
            ( updateLoggedIn model (\state -> { state | detail = Just detail, createMessage = Just "Task opened." })
            , refreshTasksAndDiscovery model
            )

        OpenTaskReceived (Err error) ->
            ( updateLoggedIn model (\state -> { state | createMessage = Just (httpErrorLabel error) }), Cmd.none )

        RefundTaskClicked taskId ->
            withSession model (\state -> ( model, postRefundTask state.accessToken taskId ))

        RefundTaskReceived (Ok _) ->
            ( updateLoggedIn model (\state -> { state | createMessage = Just "Task refunded and cancelled." }), refreshTasksAndLedger model )

        RefundTaskReceived (Err error) ->
            ( updateLoggedIn model (\state -> { state | createMessage = Just (httpErrorLabel error) }), Cmd.none )

        AgentLabelChanged value ->
            ( updateLoggedIn model (\state -> { state | agentLabel = value }), Cmd.none )

        ToggleScope scope ->
            ( updateLoggedIn model (\state -> { state | agentScopes = toggleScope scope state.agentScopes }), Cmd.none )

        CreateAgentClicked ->
            withSession model (\state -> createAgentCommand model state)

        AgentCreated (Ok created) ->
            ( updateLoggedIn model (\state -> { state | newCredential = Just created, agentMessage = Nothing }), refreshCredentials model )

        AgentCreated (Err error) ->
            ( updateLoggedIn model (\state -> { state | agentMessage = Just (httpErrorLabel error) }), Cmd.none )

        RevokeClicked credentialId ->
            withSession model (\state -> ( model, revokeAgent state.accessToken credentialId ))

        AgentRevoked _ ->
            ( model, refreshCredentials model )

        LogoutClicked ->
            ( { model | session = LoggedOut, email = "", password = "" }
            , Cmd.batch [ postLogout, Nav.pushUrl model.key "/" ]
            )

        LogoutReceived _ ->
            ( model, Cmd.none )

        DiscoveryIncludeReservedChanged value ->
            withSession model
                (\state ->
                    let
                        nextState =
                            { state | discoveryIncludeReserved = value }
                    in
                    ( updateLoggedIn model (\_ -> nextState), fetchDiscovery state.accessToken value )
                )

        DiscoveryReceived result ->
            ( updateLoggedIn model (\state -> { state | discoveryTasks = tasksFromResult result }), Cmd.none )

        DiscoveryViewClicked taskId ->
            ( updateLoggedIn model
                (\s ->
                    { s
                        | detail = Nothing
                        , reservations = []
                        , reservationMessage = Nothing
                        , submissions = []
                        , submitInput = ""
                        , submitMessage = Nothing
                    }
                )
            , Nav.pushUrl model.key ("/tasks/" ++ taskId)
            )

        DetailReceived (Ok detail) ->
            ( updateLoggedIn model (\state -> { state | detail = Just detail }), Cmd.none )

        DetailReceived (Err error) ->
            ( updateLoggedIn model (\state -> { state | submitMessage = Just (httpErrorLabel error) }), Cmd.none )

        ReserveClicked taskId ->
            withSession model (\state -> ( updateLoggedIn model (\current -> { current | reservationMessage = Nothing }), postReservation state.accessToken taskId ))

        ReservationReceived (Ok reservation) ->
            ( updateLoggedIn model (\state -> { state | reservationMessage = Just (View.reservationSuccessLabel reservation) })
            , refreshDetailReservations model
            )

        ReservationReceived (Err error) ->
            ( updateLoggedIn model (\state -> { state | reservationMessage = Just (httpErrorLabel error) }), Cmd.none )

        ReservationsReceived (Ok response) ->
            ( updateLoggedIn model (\state -> { state | reservations = response.reservations }), Cmd.none )

        ReservationsReceived (Err _) ->
            ( updateLoggedIn model (\state -> { state | reservations = [] }), Cmd.none )

        ApproveReservationClicked reservationId ->
            withSession model (\state -> reservationChangeCommand model state reservationId "approve")

        DeclineReservationClicked reservationId ->
            withSession model (\state -> reservationChangeCommand model state reservationId "decline")

        CancelReservationClicked reservationId ->
            withSession model (\state -> reservationChangeCommand model state reservationId "cancel")

        ReservationChangeReceived (Ok reservation) ->
            ( updateLoggedIn model (\state -> { state | reservationMessage = Just (View.reservationSuccessLabel reservation) })
            , refreshDetailReservations model
            )

        ReservationChangeReceived (Err error) ->
            ( updateLoggedIn model (\state -> { state | reservationMessage = Just (httpErrorLabel error) }), Cmd.none )

        SubmissionsReceived (Ok response) ->
            ( updateLoggedIn model (\state -> { state | submissions = response.submissions }), Cmd.none )

        SubmissionsReceived (Err error) ->
            ( updateLoggedIn model (\state -> { state | submissions = [], submitMessage = Just (httpErrorLabel error) }), Cmd.none )

        SubmitInputChanged value ->
            ( updateLoggedIn model (\state -> { state | submitInput = value }), Cmd.none )

        SubmitClicked ->
            withSession model (\state -> submitCommand model state)

        SubmitReceived (Ok created) ->
            ( updateLoggedIn model (\state -> { state | submitMessage = Just (View.submitSuccessLabel created) })
            , refreshDetailSubmissions model
            )

        SubmitReceived (Err error) ->
            ( updateLoggedIn model (\state -> { state | submitMessage = Just (httpErrorLabel error) }), Cmd.none )

        ReviewNoteChanged value ->
            ( updateLoggedIn model (\state -> { state | reviewNote = value }), Cmd.none )

        ReviewPartialCreditChanged value ->
            ( updateLoggedIn model (\state -> { state | reviewPartialCredit = value }), Cmd.none )

        ReviewTipChanged value ->
            ( updateLoggedIn model (\state -> { state | reviewTip = value }), Cmd.none )

        ReviewBanChanged value ->
            ( updateLoggedIn model (\state -> { state | reviewBan = value }), Cmd.none )

        AcceptClicked submissionId ->
            withSession model (\state -> acceptCommand model state submissionId)

        RequestChangesClicked submissionId ->
            withSession model (\state -> requestChangesCommand model state submissionId)

        RejectClicked submissionId ->
            withSession model (\state -> rejectCommand model state submissionId)

        ReviewActionReceived (Ok _) ->
            ( updateLoggedIn model (\state -> { state | reviewMessage = Just "Review saved." }), refreshAfterAccept model )

        ReviewActionReceived (Err error) ->
            ( updateLoggedIn model (\state -> { state | reviewMessage = Just (httpErrorLabel error) }), Cmd.none )

        CollectibleNameChanged value ->
            ( updateLoggedIn model (\state -> { state | collectibleName = value }), Cmd.none )

        CollectibleKindChosen kind ->
            ( updateLoggedIn model (\state -> { state | collectibleKind = kind }), Cmd.none )

        CollectiblePolicyChosen policy ->
            ( updateLoggedIn model (\state -> { state | collectiblePolicy = policy }), Cmd.none )

        MintClicked ->
            withSession model (\state -> mintCommand model state)

        MintReceived (Ok collectible) ->
            ( updateLoggedIn model
                (\state ->
                    { state
                        | collectibleName = ""
                        , collectibleMessage = Just (View.mintSuccessLabel collectible)
                    }
                )
            , refreshCollectibles model
            )

        MintReceived (Err error) ->
            ( updateLoggedIn model (\state -> { state | collectibleMessage = Just (httpErrorLabel error) }), Cmd.none )

        CollectiblesReceived result ->
            ( updateLoggedIn model (\state -> { state | collectibles = collectiblesFromResult result }), Cmd.none )

        AwardTaskIdChanged value ->
            ( updateLoggedIn model (\state -> { state | awardTaskId = value }), Cmd.none )

        AwardClicked collectibleId ->
            withSession model (\state -> awardCommand model state collectibleId)

        AwardReceived (Ok collectible) ->
            let
                updated =
                    updateLoggedIn model (\state -> { state | awardMessage = Just (View.awardSuccessLabel collectible) })
            in
            withSession updated (\state -> ( updated, Cmd.batch [ fetchCollectibles state.accessToken, fetchTasks state.accessToken state.taskStateFilter ] ))

        AwardReceived (Err error) ->
            ( updateLoggedIn model (\state -> { state | awardMessage = Just (httpErrorLabel error) }), Cmd.none )

        OrganizationsReceived result ->
            ( updateLoggedIn model (\state -> { state | organizations = organizationsFromResult result }), Cmd.none )

        CreateOrgNameChanged value ->
            ( updateLoggedIn model (\state -> { state | createOrgName = value }), Cmd.none )

        CreateOrgClicked ->
            withSession model (\state -> createOrgCommand model state)

        CreateOrgReceived (Ok organization) ->
            ( updateLoggedIn model (\state -> { state | createOrgName = "", orgMessage = Just ("Created organization " ++ organization.name) }), refreshOrganizations model )

        CreateOrgReceived (Err error) ->
            ( updateLoggedIn model (\state -> { state | orgMessage = Just (httpErrorLabel error) }), Cmd.none )

        OrgBalanceReceived result ->
            ( updateLoggedIn model (\state -> { state | orgBalance = balanceFromResult result }), Cmd.none )

        OrgTeamsReceived result ->
            ( updateLoggedIn model (\state -> { state | orgTeams = teamsFromResult result }), Cmd.none )

        OrgMembersReceived result ->
            ( updateLoggedIn model (\state -> { state | orgMembers = membersFromResult result }), Cmd.none )

        UserProfileReceived result ->
            ( updateLoggedIn model (\state -> { state | userProfile = Result.toMaybe result }), Cmd.none )

        UserWorkReceived result ->
            ( updateLoggedIn model (\state -> { state | userWork = tasksFromResult result }), Cmd.none )

        UserSubmissionsReceived result ->
            ( updateLoggedIn model (\state -> { state | userSubmissions = submissionsFromResult result }), Cmd.none )

        SeriesDetailReceived result ->
            ( updateLoggedIn model (\state -> { state | seriesDetail = Result.toMaybe result }), Cmd.none )

        TeamDetailReceived result ->
            ( updateLoggedIn model (\state -> { state | teamDetail = Result.toMaybe result }), Cmd.none )

        TeamMemberEmailChanged value ->
            ( updateLoggedIn model (\state -> { state | teamMemberEmail = value }), Cmd.none )

        AddTeamMemberClicked teamId ->
            withSession model (\state -> ( model, postAddTeamMember state.accessToken teamId state.teamMemberEmail ))

        AddTeamMemberReceived (Ok detail) ->
            ( updateLoggedIn model (\state -> { state | teamDetail = Just detail, teamMemberEmail = "", teamMemberMessage = Just "Member added." }), Cmd.none )

        AddTeamMemberReceived (Err error) ->
            ( updateLoggedIn model (\state -> { state | teamMemberMessage = Just (httpErrorLabel error) }), Cmd.none )

        OrgTasksReceived result ->
            ( updateLoggedIn model (\state -> { state | orgTasks = tasksFromResult result }), Cmd.none )

        CreateOrgTeamNameChanged value ->
            ( updateLoggedIn model (\state -> { state | createOrgTeamName = value }), Cmd.none )

        CreateOrgTeamClicked ->
            withSession model (\state -> createOrgTeamCommand model state)

        CreateOrgTeamReceived (Ok team) ->
            let
                updated =
                    updateLoggedIn model (\state -> { state | createOrgTeamName = "", orgTeamMessage = Just ("Created team " ++ team.name) })
            in
            withSession updated (\state -> ( updated, fetchOrgTeams state.accessToken state.activeOrgId ))

        CreateOrgTeamReceived (Err error) ->
            ( updateLoggedIn model (\state -> { state | orgTeamMessage = Just (httpErrorLabel error) }), Cmd.none )

        ProvisionMemberEmailChanged value ->
            ( updateLoggedIn model (\state -> { state | provisionMemberEmail = value }), Cmd.none )

        ProvisionMemberClicked ->
            withSession model (\state -> provisionMemberCommand model state)

        ProvisionMemberReceived (Ok ()) ->
            let
                updated =
                    updateLoggedIn model (\state -> { state | provisionMemberEmail = "", provisionMemberMessage = Just "Member provisioned." })
            in
            withSession updated (\state -> ( updated, authorizedRequest "GET" state.accessToken ("/api/organizations/" ++ state.activeOrgId ++ "/members") Http.emptyBody (Http.expectJson OrgMembersReceived Organization.organizationMembersResponseDecoder) ))

        ProvisionMemberReceived (Err error) ->
            ( updateLoggedIn model (\state -> { state | provisionMemberMessage = Just (httpErrorLabel error) }), Cmd.none )

        CreateTaskOwnerChanged value ->
            ( updateLoggedIn model (\state -> { state | createTaskOwner = value }), Cmd.none )

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
                    , routeLoadCmd state.accessToken page
                    )

                LoggedOut ->
                    ( { model | route = page }, Cmd.none )


withSession : Model -> (LoggedInModel -> ( Model, Cmd Msg )) -> ( Model, Cmd Msg )
withSession model run =
    case model.session of
        LoggedIn state ->
            run state

        LoggedOut ->
            ( model, Cmd.none )


updateLoggedIn : Model -> (LoggedInModel -> LoggedInModel) -> Model
updateLoggedIn model change =
    case model.session of
        LoggedIn state ->
            { model | session = LoggedIn (change state) }

        LoggedOut ->
            model


balanceFromResult : Result Http.Error Ledger.BalanceResponse -> Maybe Int
balanceFromResult result =
    case result of
        Ok response ->
            Just response.amount

        Err _ ->
            Nothing


entriesFromResult : Result Http.Error Ledger.LedgerResponse -> List Ledger.LedgerEntryResponse
entriesFromResult result =
    case result of
        Ok response ->
            response.entries

        Err _ ->
            []


tasksFromResult : Result Http.Error Task.TasksResponse -> List Task.TaskListItemResponse
tasksFromResult result =
    case result of
        Ok response ->
            response.tasks

        Err _ ->
            []


credentialsFromResult : Result Http.Error Agent.AgentCredentialsResponse -> List Agent.AgentCredentialResponse
credentialsFromResult result =
    case result of
        Ok response ->
            response.credentials

        Err _ ->
            []


collectiblesFromResult : Result Http.Error Collectible.CollectiblesResponse -> List Collectible.CollectibleResponse
collectiblesFromResult result =
    case result of
        Ok response ->
            response.collectibles

        Err _ ->
            []


boolQuery : Bool -> String
boolQuery value =
    if value then
        "true"

    else
        "false"


toggleScope : Agent.AgentScope -> List Agent.AgentScope -> List Agent.AgentScope
toggleScope scope scopes =
    if List.member scope scopes then
        List.filter (\existing -> existing /= scope) scopes

    else
        scope :: scopes


fundTaskCommand : Model -> LoggedInModel -> ( Model, Cmd Msg )
fundTaskCommand model state =
    case String.toInt state.fundAmount of
        Just amount ->
            ( updateLoggedIn model (\current -> { current | fundMessage = Nothing }), postFunding state.accessToken state.fundTaskId amount state.fundOrganizationId )

        Nothing ->
            ( updateLoggedIn model (\current -> { current | fundMessage = Just "Amount must be a whole number of credits." }), Cmd.none )


createAgentCommand : Model -> LoggedInModel -> ( Model, Cmd Msg )
createAgentCommand model state =
    if List.isEmpty state.agentScopes then
        ( updateLoggedIn model (\current -> { current | agentMessage = Just "Select at least one scope." }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | agentMessage = Nothing, newCredential = Nothing }), postAgent state.accessToken state.agentLabel state.agentScopes )


createTaskCommand : Model -> LoggedInModel -> ( Model, Cmd Msg )
createTaskCommand model state =
    if String.isEmpty (String.trim state.createTitle) || String.isEmpty (String.trim state.createDescription) then
        ( updateLoggedIn model (\current -> { current | createMessage = Just "Title and description are required." }), Cmd.none )

    else if reservationHoursValue state.createReservationHours < 1 || reservationHoursValue state.createReservationHours > 720 then
        ( updateLoggedIn model (\current -> { current | createMessage = Just "Reservation expiry must be between 1 and 720 hours." }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | createMessage = Nothing })
        , postCreateTask state
        )


submitCommand : Model -> LoggedInModel -> ( Model, Cmd Msg )
submitCommand model state =
    case state.page of
        TaskDetailPage taskId ->
            ( updateLoggedIn model (\current -> { current | submitMessage = Nothing })
            , postSubmission state.accessToken taskId state.submitInput
            )

        _ ->
            ( model, Cmd.none )


acceptCommand : Model -> LoggedInModel -> String -> ( Model, Cmd Msg )
acceptCommand model state submissionId =
    case state.page of
        TaskDetailPage taskId ->
            ( updateLoggedIn model (\current -> { current | reviewMessage = Nothing }), postAccept state.accessToken taskId submissionId state.reviewPartialCredit state.reviewTip )

        _ ->
            ( model, Cmd.none )


requestChangesCommand : Model -> LoggedInModel -> String -> ( Model, Cmd Msg )
requestChangesCommand model state submissionId =
    case state.page of
        TaskDetailPage taskId ->
            ( updateLoggedIn model (\current -> { current | reviewMessage = Nothing }), postRequestChanges state.accessToken taskId submissionId state.reviewNote )

        _ ->
            ( model, Cmd.none )


rejectCommand : Model -> LoggedInModel -> String -> ( Model, Cmd Msg )
rejectCommand model state submissionId =
    case state.page of
        TaskDetailPage taskId ->
            ( updateLoggedIn model (\current -> { current | reviewMessage = Nothing }), postReject state.accessToken taskId submissionId state.reviewNote state.reviewPartialCredit state.reviewTip state.reviewBan )

        _ ->
            ( model, Cmd.none )


reservationChangeCommand : Model -> LoggedInModel -> String -> String -> ( Model, Cmd Msg )
reservationChangeCommand model state reservationId action =
    case state.page of
        TaskDetailPage taskId ->
            ( updateLoggedIn model (\current -> { current | reservationMessage = Nothing })
            , postReservationChange state.accessToken taskId reservationId action
            )

        _ ->
            ( model, Cmd.none )


mintCommand : Model -> LoggedInModel -> ( Model, Cmd Msg )
mintCommand model state =
    if String.isEmpty (String.trim state.collectibleName) then
        ( updateLoggedIn model (\current -> { current | collectibleMessage = Just "Name is required." }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | collectibleMessage = Nothing })
        , postCollectible state.accessToken state.collectibleName state.collectibleKind state.collectiblePolicy
        )


awardCommand : Model -> LoggedInModel -> String -> ( Model, Cmd Msg )
awardCommand model state collectibleId =
    if String.isEmpty (String.trim state.awardTaskId) then
        ( updateLoggedIn model (\current -> { current | awardMessage = Just "Task ID is required." }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | awardMessage = Nothing })
        , postCollectibleReward state.accessToken state.awardTaskId collectibleId
        )


loadAfterAuth : String -> Cmd Msg
loadAfterAuth token =
    Cmd.batch [ fetchBalance token, fetchLedger token, fetchTasks token "", fetchCredentials token, fetchCollectibles token, fetchOrganizations token ]


refreshCollectibles : Model -> Cmd Msg
refreshCollectibles model =
    case model.session of
        LoggedIn state ->
            fetchCollectibles state.accessToken

        LoggedOut ->
            Cmd.none


refreshLedger : Model -> Cmd Msg
refreshLedger model =
    case model.session of
        LoggedIn state ->
            Cmd.batch [ fetchBalance state.accessToken, fetchLedger state.accessToken ]

        LoggedOut ->
            Cmd.none


refreshTasksAndLedger : Model -> Cmd Msg
refreshTasksAndLedger model =
    case model.session of
        LoggedIn state ->
            Cmd.batch [ fetchTasks state.accessToken state.taskStateFilter, fetchBalance state.accessToken, fetchLedger state.accessToken ]

        LoggedOut ->
            Cmd.none


refreshTasksAndDiscovery : Model -> Cmd Msg
refreshTasksAndDiscovery model =
    case model.session of
        LoggedIn state ->
            Cmd.batch [ fetchTasks state.accessToken state.taskStateFilter, fetchDiscovery state.accessToken state.discoveryIncludeReserved ]

        LoggedOut ->
            Cmd.none


routeLoadCmd : String -> Page -> Cmd Msg
routeLoadCmd token page =
    case page of
        OverviewPage ->
            Cmd.batch [ fetchBalance token, fetchLedger token ]

        TasksPage ->
            fetchTasks token ""

        CreateTaskPage ->
            fetchOrganizations token

        TaskDetailPage taskId ->
            fetchDetailCommands token taskId

        DiscoveryPage ->
            fetchDiscovery token False

        FundingPage ->
            fetchTasks token ""

        AgentsPage ->
            fetchCredentials token

        CollectiblesPage ->
            Cmd.batch [ fetchCollectibles token, fetchTasks token "" ]

        OrganizationsPage ->
            fetchOrganizations token

        OrganizationDetailPage organizationId ->
            Cmd.batch [ fetchOrganizations token, loadOrganization token organizationId ]

        UserDetailPage userId ->
            fetchUserProfile token userId

        UserWorkPage userId ->
            authorizedRequest "GET" token ("/api/users/" ++ userId ++ "/work") Http.emptyBody (Http.expectJson UserWorkReceived Task.tasksResponseDecoder)

        UserSubmissionsPage userId ->
            authorizedRequest "GET" token ("/api/users/" ++ userId ++ "/submissions") Http.emptyBody (Http.expectJson UserSubmissionsReceived Submission.submissionsResponseDecoder)

        CollectibleDetailPage _ ->
            fetchCollectibles token

        SeriesDetailPage seriesId ->
            authorizedRequest "GET" token ("/api/task-series/" ++ seriesId) Http.emptyBody (Http.expectJson SeriesDetailReceived TaskSeries.taskSeriesResponseDecoder)

        TeamDetailPage teamId ->
            authorizedRequest "GET" token ("/api/teams/" ++ teamId) Http.emptyBody (Http.expectJson TeamDetailReceived Team.teamDetailResponseDecoder)


fetchUserProfile : String -> String -> Cmd Msg
fetchUserProfile token userId =
    authorizedRequest "GET" token ("/api/users/" ++ userId) Http.emptyBody (Http.expectJson UserProfileReceived Task.userProfileResponseDecoder)


postAddTeamMember : String -> String -> String -> Cmd Msg
postAddTeamMember token teamId email =
    authorizedRequest "POST"
        token
        ("/api/teams/" ++ teamId ++ "/members")
        (Http.jsonBody (Encode.object [ ( "email", Encode.string email ) ]))
        (Http.expectJson AddTeamMemberReceived Team.teamDetailResponseDecoder)


refreshCredentials : Model -> Cmd Msg
refreshCredentials model =
    case model.session of
        LoggedIn state ->
            fetchCredentials state.accessToken

        LoggedOut ->
            Cmd.none


refreshDetailSubmissions : Model -> Cmd Msg
refreshDetailSubmissions model =
    case model.session of
        LoggedIn state ->
            case state.page of
                TaskDetailPage taskId ->
                    fetchSubmissions state.accessToken taskId

                _ ->
                    Cmd.none

        LoggedOut ->
            Cmd.none


refreshDetailReservations : Model -> Cmd Msg
refreshDetailReservations model =
    case model.session of
        LoggedIn state ->
            case state.page of
                TaskDetailPage taskId ->
                    Cmd.batch [ fetchPublicTaskDetail state.accessToken taskId, fetchReservations state.accessToken taskId ]

                _ ->
                    Cmd.none

        LoggedOut ->
            Cmd.none


fetchDetailCommands : String -> String -> Cmd Msg
fetchDetailCommands token taskId =
    Cmd.batch [ fetchPublicTaskDetail token taskId, fetchSubmissions token taskId, fetchReservations token taskId ]


refreshAfterAccept : Model -> Cmd Msg
refreshAfterAccept model =
    case model.session of
        LoggedIn state ->
            case state.page of
                TaskDetailPage taskId ->
                    Cmd.batch
                        [ fetchSubmissions state.accessToken taskId
                        , fetchBalance state.accessToken
                        ]

                _ ->
                    Cmd.none

        LoggedOut ->
            Cmd.none


postAuth : String -> Model -> Cmd Msg
postAuth url model =
    Http.post
        { url = url
        , body = Http.jsonBody (authRequestBody model)
        , expect = Http.expectJson AuthReceived Auth.authResponseDecoder
        }


postRefresh : Cmd Msg
postRefresh =
    Http.post
        { url = "/api/auth/refresh"
        , body = Http.emptyBody
        , expect = Http.expectJson RefreshReceived Auth.authResponseDecoder
        }


postLogout : Cmd Msg
postLogout =
    Http.post
        { url = "/api/auth/logout"
        , body = Http.emptyBody
        , expect = Http.expectWhatever LogoutReceived
        }


authRequestBody : Model -> Encode.Value
authRequestBody model =
    Encode.object
        [ ( "email", Encode.string model.email )
        , ( "password", Encode.string model.password )
        ]


fetchBalance : String -> Cmd Msg
fetchBalance token =
    authorizedRequest "GET" token "/api/credits/balance" Http.emptyBody (Http.expectJson BalanceReceived Ledger.balanceResponseDecoder)


fetchLedger : String -> Cmd Msg
fetchLedger token =
    authorizedRequest "GET" token "/api/credits/ledger" Http.emptyBody (Http.expectJson LedgerReceived Ledger.ledgerResponseDecoder)


fetchTasks : String -> String -> Cmd Msg
fetchTasks token stateFilter =
    let
        query =
            if stateFilter == "" then
                "/api/tasks?scope=user"

            else
                "/api/tasks?scope=user&state=" ++ stateFilter
    in
    authorizedRequest "GET" token query Http.emptyBody (Http.expectJson TasksReceived Task.tasksResponseDecoder)


fetchCredentials : String -> Cmd Msg
fetchCredentials token =
    authorizedRequest "GET" token "/api/agent-credentials" Http.emptyBody (Http.expectJson CredentialsReceived Agent.agentCredentialsResponseDecoder)


postCreateTask : LoggedInModel -> Cmd Msg
postCreateTask state =
    authorizedRequest "POST"
        state.accessToken
        "/api/tasks"
        (Http.jsonBody (createTaskRequestBody state))
        (Http.expectJson CreateTaskReceived taskDetailDecoder)


fetchDiscovery : String -> Bool -> Cmd Msg
fetchDiscovery token includeReserved =
    authorizedRequest "GET" token ("/api/tasks?scope=public&include_reserved=" ++ boolQuery includeReserved) Http.emptyBody (Http.expectJson DiscoveryReceived Task.tasksResponseDecoder)


fetchPublicTaskDetail : String -> String -> Cmd Msg
fetchPublicTaskDetail token taskId =
    authorizedRequest "GET" token ("/api/tasks/" ++ taskId) Http.emptyBody (Http.expectJson DetailReceived publicTaskDetailDecoder)


fetchSubmissions : String -> String -> Cmd Msg
fetchSubmissions token taskId =
    authorizedRequest "GET" token ("/api/tasks/" ++ taskId ++ "/submissions") Http.emptyBody (Http.expectJson SubmissionsReceived Submission.submissionsResponseDecoder)


fetchReservations : String -> String -> Cmd Msg
fetchReservations token taskId =
    authorizedRequest "GET" token ("/api/tasks/" ++ taskId ++ "/reservations") Http.emptyBody (Http.expectJson ReservationsReceived Task.taskReservationsResponseDecoder)


postFunding : String -> String -> Int -> String -> Cmd Msg
postFunding token taskId amount organizationId =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/funding")
        (Http.jsonBody (fundingRequestBody taskId amount organizationId))
        (Http.expectJson FundReceived Ledger.taskEscrowResponseDecoder)


postOpenTask : String -> String -> Cmd Msg
postOpenTask token taskId =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/open")
        (Http.jsonBody (Encode.object []))
        (Http.expectJson OpenTaskReceived taskDetailDecoder)


postRefundTask : String -> String -> Cmd Msg
postRefundTask token taskId =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/refund")
        (Http.jsonBody (Encode.object [ ( "idempotency_key", Encode.string ("ui-refund:" ++ taskId) ) ]))
        (Http.expectJson RefundTaskReceived Ledger.taskEscrowResponseDecoder)


postReservation : String -> String -> Cmd Msg
postReservation token taskId =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/reservations")
        (Http.jsonBody (Encode.object []))
        (Http.expectJson ReservationReceived Task.taskReservationResponseDecoder)


postReservationChange : String -> String -> String -> String -> Cmd Msg
postReservationChange token taskId reservationId action =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/reservations/" ++ reservationId ++ "/" ++ action)
        (Http.jsonBody (Encode.object []))
        (Http.expectJson ReservationChangeReceived Task.taskReservationResponseDecoder)


postAgent : String -> String -> List Agent.AgentScope -> Cmd Msg
postAgent token agentLabel scopes =
    authorizedRequest "POST"
        token
        "/api/agent-credentials"
        (Http.jsonBody (agentRequestBody agentLabel scopes))
        (Http.expectJson AgentCreated Agent.agentCredentialCreatedResponseDecoder)


postSubmission : String -> String -> String -> Cmd Msg
postSubmission token taskId responseJson =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/submissions")
        (Http.jsonBody (submissionRequestBody responseJson))
        (Http.expectJson SubmitReceived Submission.submissionCreatedResponseDecoder)


postAccept : String -> String -> String -> String -> String -> Cmd Msg
postAccept token taskId submissionId payoutAmount tipAmount =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/submissions/" ++ submissionId ++ "/accept")
        (Http.jsonBody (acceptRequestBody submissionId payoutAmount tipAmount))
        (Http.expectWhatever ReviewActionReceived)


postRequestChanges : String -> String -> String -> String -> Cmd Msg
postRequestChanges token taskId submissionId reviewNote =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/submissions/" ++ submissionId ++ "/request-changes")
        (Http.jsonBody (requestChangesBody reviewNote))
        (Http.expectWhatever ReviewActionReceived)


postReject : String -> String -> String -> String -> String -> String -> Bool -> Cmd Msg
postReject token taskId submissionId reviewNote partialCredit tipAmount banImplementor =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/submissions/" ++ submissionId ++ "/reject")
        (Http.jsonBody (rejectRequestBody submissionId reviewNote partialCredit tipAmount banImplementor))
        (Http.expectWhatever ReviewActionReceived)


fetchCollectibles : String -> Cmd Msg
fetchCollectibles token =
    authorizedRequest "GET" token "/api/collectibles" Http.emptyBody (Http.expectJson CollectiblesReceived Collectible.collectiblesResponseDecoder)


fetchOrganizations : String -> Cmd Msg
fetchOrganizations token =
    authorizedRequest "GET" token "/api/organizations" Http.emptyBody (Http.expectJson OrganizationsReceived Organization.organizationsResponseDecoder)


refreshOrganizations : Model -> Cmd Msg
refreshOrganizations model =
    case model.session of
        LoggedIn state ->
            fetchOrganizations state.accessToken

        LoggedOut ->
            Cmd.none


loadOrganization : String -> String -> Cmd Msg
loadOrganization token organizationId =
    if organizationId == "" then
        Cmd.none

    else
        Cmd.batch
            [ authorizedRequest "GET" token ("/api/organizations/" ++ organizationId ++ "/credits/balance") Http.emptyBody (Http.expectJson OrgBalanceReceived Ledger.balanceResponseDecoder)
            , fetchOrgTeams token organizationId
            , authorizedRequest "GET" token ("/api/organizations/" ++ organizationId ++ "/members") Http.emptyBody (Http.expectJson OrgMembersReceived Organization.organizationMembersResponseDecoder)
            , authorizedRequest "GET" token ("/api/tasks?scope=organization&organization_id=" ++ organizationId) Http.emptyBody (Http.expectJson OrgTasksReceived Task.tasksResponseDecoder)
            ]


fetchOrgTeams : String -> String -> Cmd Msg
fetchOrgTeams token organizationId =
    authorizedRequest "GET" token ("/api/organizations/" ++ organizationId ++ "/teams") Http.emptyBody (Http.expectJson OrgTeamsReceived Team.teamsResponseDecoder)


createOrgTeamCommand : Model -> LoggedInModel -> ( Model, Cmd Msg )
createOrgTeamCommand model state =
    if String.isEmpty (String.trim state.createOrgTeamName) || state.activeOrgId == "" then
        ( updateLoggedIn model (\current -> { current | orgTeamMessage = Just "A team name is required." }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | orgTeamMessage = Nothing })
        , authorizedRequest "POST"
            state.accessToken
            ("/api/organizations/" ++ state.activeOrgId ++ "/teams")
            (Http.jsonBody (Encode.object [ ( "name", Encode.string (String.trim state.createOrgTeamName) ) ]))
            (Http.expectJson CreateOrgTeamReceived Team.teamResponseDecoder)
        )


provisionMemberCommand : Model -> LoggedInModel -> ( Model, Cmd Msg )
provisionMemberCommand model state =
    if String.isEmpty (String.trim state.provisionMemberEmail) || state.activeOrgId == "" then
        ( updateLoggedIn model (\current -> { current | provisionMemberMessage = Just "A member email is required." }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | provisionMemberMessage = Nothing })
        , authorizedRequest "POST"
            state.accessToken
            ("/api/organizations/" ++ state.activeOrgId ++ "/members")
            (Http.jsonBody (Encode.object [ ( "email", Encode.string (String.trim state.provisionMemberEmail) ), ( "roles", Encode.list Encode.string [ "member" ] ) ]))
            (Http.expectWhatever ProvisionMemberReceived)
        )


teamsFromResult : Result Http.Error Team.TeamsResponse -> List Team.TeamResponse
teamsFromResult result =
    case result of
        Ok response ->
            response.teams

        Err _ ->
            []


membersFromResult : Result Http.Error Organization.OrganizationMembersResponse -> List Organization.OrganizationMemberResponse
membersFromResult result =
    case result of
        Ok response ->
            response.members

        Err _ ->
            []


submissionsFromResult : Result Http.Error Submission.SubmissionsResponse -> List Submission.SubmissionResponse
submissionsFromResult result =
    case result of
        Ok response ->
            response.submissions

        Err _ ->
            []


createOrgCommand : Model -> LoggedInModel -> ( Model, Cmd Msg )
createOrgCommand model state =
    if String.isEmpty (String.trim state.createOrgName) then
        ( updateLoggedIn model (\current -> { current | orgMessage = Just "Organization name is required." }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | orgMessage = Nothing })
        , authorizedRequest "POST"
            state.accessToken
            "/api/organizations"
            (Http.jsonBody (Encode.object [ ( "name", Encode.string (String.trim state.createOrgName) ) ]))
            (Http.expectJson CreateOrgReceived Organization.organizationResponseDecoder)
        )


organizationsFromResult : Result Http.Error Organization.OrganizationsResponse -> List Organization.OrganizationResponse
organizationsFromResult result =
    case result of
        Ok response ->
            response.organizations

        Err _ ->
            []


postCollectible : String -> String -> Collectible.CollectibleKind -> Collectible.CollectibleTransferPolicy -> Cmd Msg
postCollectible token name kind policy =
    authorizedRequest "POST"
        token
        "/api/collectibles"
        (Http.jsonBody (collectibleRequestBody name kind policy))
        (Http.expectJson MintReceived Collectible.collectibleResponseDecoder)


postCollectibleReward : String -> String -> String -> Cmd Msg
postCollectibleReward token taskId collectibleId =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/collectible-reward")
        (Http.jsonBody (collectibleRewardRequestBody collectibleId))
        (Http.expectJson AwardReceived Collectible.collectibleResponseDecoder)


revokeAgent : String -> String -> Cmd Msg
revokeAgent token credentialId =
    authorizedRequest "POST"
        token
        ("/api/agent-credentials/" ++ credentialId ++ "/revoke")
        (Http.jsonBody (Encode.object []))
        (Http.expectJson AgentRevoked Agent.agentCredentialResponseDecoder)


fundingRequestBody : String -> Int -> String -> Encode.Value
fundingRequestBody taskId amount organizationId =
    Encode.object
        [ ( "amount", Encode.int amount )
        , ( "idempotency_key", Encode.string ("fund:" ++ taskId) )
        , ( "organization_id", Encode.string organizationId )
        ]


createTaskRequestBody : LoggedInModel -> Encode.Value
createTaskRequestBody state =
    Encode.object
        [ ( "owner", createOwnerBody state )
        , ( "title", Encode.string state.createTitle )
        , ( "description", Encode.string state.createDescription )
        , ( "reward", createRewardBody state.createRewardAmount )
        , ( "participation", createParticipationBody state )
        , ( "visibility", createVisibilityBody state )
        , ( "placement", Encode.object [ ( "kind", Encode.string "standalone" ), ( "series_id", Encode.string "" ), ( "series_title", Encode.string "" ), ( "series_position", Encode.int 0 ) ] )
        , ( "response_schema_json", Encode.string "{\"kind\":\"freeform\"}" )
        , ( "payload", Encode.object [ ( "kind", Encode.string "none" ), ( "json", Encode.string "" ) ] )
        ]


createRewardBody : String -> Encode.Value
createRewardBody rawAmount =
    case String.toInt rawAmount of
        Just amount ->
            if amount > 0 then
                Encode.object [ ( "kind", Encode.string "credit" ), ( "credit_amount", Encode.int amount ) ]

            else
                Encode.object [ ( "kind", Encode.string "none" ), ( "credit_amount", Encode.int 0 ) ]

        Nothing ->
            Encode.object [ ( "kind", Encode.string "none" ), ( "credit_amount", Encode.int 0 ) ]


createParticipationBody : LoggedInModel -> Encode.Value
createParticipationBody state =
    Encode.object
        [ ( "policy", Encode.string state.createParticipationPolicy )
        , ( "assignee_scope", Encode.string (assigneeScopeTag state.createAssigneeScope) )
        , ( "reservation_expiry_hours", Encode.int (reservationHoursValue state.createReservationHours) )
        ]


createOwnerBody : LoggedInModel -> Encode.Value
createOwnerBody state =
    if state.createTaskOwner == "" then
        Encode.object [ ( "kind", Encode.string "user" ), ( "user_id", Encode.string state.subjectId ), ( "team_id", Encode.string "" ), ( "organization_id", Encode.string "" ) ]

    else
        Encode.object [ ( "kind", Encode.string "organization" ), ( "user_id", Encode.string "" ), ( "team_id", Encode.string "" ), ( "organization_id", Encode.string state.createTaskOwner ) ]


createVisibilityBody : LoggedInModel -> Encode.Value
createVisibilityBody state =
    Encode.object
        [ ( "kind", Encode.string state.createVisibility )
        , ( "user_id"
          , Encode.string
                (if state.createVisibility == visibilityUserTag then
                    state.createScopeUserId

                 else
                    ""
                )
          )
        , ( "team_id"
          , Encode.string
                (if state.createVisibility == visibilityTeamTag then
                    state.createScopeTeamId

                 else
                    ""
                )
          )
        , ( "organization_id"
          , Encode.string
                (if state.createVisibility == visibilityOrganizationTag then
                    state.createScopeOrganizationId

                 else
                    ""
                )
          )
        ]


reservationHoursValue : String -> Int
reservationHoursValue raw =
    case String.toInt raw of
        Just hours ->
            hours

        Nothing ->
            48


agentRequestBody : String -> List Agent.AgentScope -> Encode.Value
agentRequestBody agentLabel scopes =
    Encode.object
        [ ( "label", Encode.string agentLabel )
        , ( "scopes", Encode.list Agent.agentScopeEncoder scopes )
        ]


submissionRequestBody : String -> Encode.Value
submissionRequestBody responseJson =
    Encode.object
        [ ( "response_json", Encode.string responseJson )
        ]


acceptRequestBody : String -> String -> String -> Encode.Value
acceptRequestBody submissionId payoutAmount tipAmount =
    Encode.object
        [ ( "idempotency_key", Encode.string ("ui-accept:" ++ submissionId) )
        , ( "payout_amount", Encode.int (intInputOrZero payoutAmount) )
        , ( "tip_amount", Encode.int (intInputOrZero tipAmount) )
        ]


requestChangesBody : String -> Encode.Value
requestChangesBody reviewNote =
    Encode.object
        [ ( "review_note", Encode.string reviewNote )
        ]


rejectRequestBody : String -> String -> String -> String -> Bool -> Encode.Value
rejectRequestBody submissionId reviewNote partialCredit tipAmount banImplementor =
    Encode.object
        [ ( "idempotency_key", Encode.string ("ui-reject:" ++ submissionId) )
        , ( "review_note", Encode.string reviewNote )
        , ( "partial_credit_amount", Encode.int (intInputOrZero partialCredit) )
        , ( "tip_amount", Encode.int (intInputOrZero tipAmount) )
        , ( "ban_implementor", Encode.bool banImplementor )
        ]


intInputOrZero : String -> Int
intInputOrZero raw =
    raw
        |> String.trim
        |> String.toInt
        |> Maybe.withDefault 0


collectibleRequestBody : String -> Collectible.CollectibleKind -> Collectible.CollectibleTransferPolicy -> Encode.Value
collectibleRequestBody name kind policy =
    Encode.object
        [ ( "name", Encode.string name )
        , ( "kind", Collectible.collectibleKindEncoder kind )
        , ( "transfer_policy", Collectible.collectibleTransferPolicyEncoder policy )
        ]


collectibleRewardRequestBody : String -> Encode.Value
collectibleRewardRequestBody collectibleId =
    Encode.object
        [ ( "collectible_id", Encode.string collectibleId )
        ]


taskDetailDecoder : Decode.Decoder TaskDetail
taskDetailDecoder =
    Decode.map taskDetailFromResponse Task.taskResponseDecoder


publicTaskDetailDecoder : Decode.Decoder PublicTaskDetail
publicTaskDetailDecoder =
    Decode.map publicTaskDetailFromResponse Task.taskResponseDecoder


taskDetailFromResponse : Task.TaskResponse -> TaskDetail
taskDetailFromResponse response =
    { id = response.id
    , title = response.title
    , description = response.description
    , state = response.state
    , rewardKind = response.rewardKind
    , rewardCreditAmount = response.rewardCreditAmount
    , rewardCollectibleCount = response.rewardCollectibleCount
    , participationPolicy = response.participationPolicy
    , assigneeScope = response.assigneeScope
    , reservationExpiryHours = response.reservationExpiryHours
    , availabilityKind = response.availabilityKind
    , viewerAction = response.viewerAction
    , responseSchemaJson = response.responseSchemaJSON
    , createdBy = response.createdBy
    }


publicTaskDetailFromResponse : Task.TaskResponse -> PublicTaskDetail
publicTaskDetailFromResponse response =
    taskDetailFromResponse response



authorizedRequest : String -> String -> String -> Http.Body -> Http.Expect Msg -> Cmd Msg
authorizedRequest method token url body expect =
    Http.request
        { method = method
        , headers = [ Http.header "Authorization" ("Bearer " ++ token) ]
        , url = url
        , body = body
        , expect = expect
        , timeout = Nothing
        , tracker = Nothing
        }

