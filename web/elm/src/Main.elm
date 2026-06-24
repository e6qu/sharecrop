module Main exposing (main)

import Browser
import Browser.Navigation as Nav
import Http
import Sharecrop.Api as Api
import Sharecrop.Generated.Agent as Agent
import Sharecrop.Generated.Auth as Auth
import Sharecrop.Generated.Collectible as Collectible
import Sharecrop.Generated.Organization as Organization
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
            ( { model | authError = Nothing }, Api.postAuth "/api/auth/register" model )

        LoginClicked ->
            ( { model | authError = Nothing }, Api.postAuth "/api/auth/login" model )

        AuthReceived (Ok response) ->
            ( { model | password = "", authError = Nothing, session = LoggedIn (loggedInForPage response model.route) }
            , Api.loadAfterAuth response.accessToken
            )

        AuthReceived (Err error) ->
            ( { model | authError = Just (httpErrorLabel error) }, Cmd.none )

        RefreshReceived (Ok response) ->
            ( { model | session = LoggedIn (loggedInForPage response model.route) }
            , Cmd.batch [ Api.loadAfterAuth response.accessToken, Api.routeLoadCmd response.accessToken model.route ]
            )

        RefreshReceived (Err _) ->
            ( model, Cmd.none )

        BalanceReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | balance = Api.balanceFromResult result }), Cmd.none )

        LedgerReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | entries = Api.entriesFromResult result }), Cmd.none )

        TasksReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | tasks = Api.tasksFromResult result }), Cmd.none )

        TaskStateFilterChanged value ->
            let
                updated =
                    Api.updateLoggedIn model (\state -> { state | taskStateFilter = value })
            in
            Api.withSession updated (\state -> ( updated, Api.fetchTasks state.accessToken value ))

        CreateTitleChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | createTitle = value }), Cmd.none )

        CreateDescriptionChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | createDescription = value }), Cmd.none )

        CreateRewardAmountChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | createRewardAmount = value }), Cmd.none )

        CreateVisibilityChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | createVisibility = value }), Cmd.none )

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
            Api.withSession model (\state -> Api.fundTaskCommand model state)

        FundReceived (Ok escrow) ->
            ( Api.updateLoggedIn model (\state -> { state | fundMessage = Just (View.fundSuccessLabel escrow) }), Api.refreshLedger model )

        FundReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | fundMessage = Just (httpErrorLabel error) }), Cmd.none )

        OpenTaskClicked taskId ->
            Api.withSession model (\state -> ( model, Api.postOpenTask state.accessToken taskId ))

        OpenTaskReceived (Ok detail) ->
            ( Api.updateLoggedIn model (\state -> { state | detail = Just detail, createMessage = Just "Task opened." })
            , Api.refreshTasksAndDiscovery model
            )

        OpenTaskReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | createMessage = Just (httpErrorLabel error) }), Cmd.none )

        RefundTaskClicked taskId ->
            Api.withSession model (\state -> ( model, Api.postRefundTask state.accessToken taskId ))

        RefundTaskReceived (Ok _) ->
            ( Api.updateLoggedIn model (\state -> { state | createMessage = Just "Task refunded and cancelled." }), Api.refreshTasksAndLedger model )

        RefundTaskReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | createMessage = Just (httpErrorLabel error) }), Cmd.none )

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

        RevokeClicked credentialId ->
            Api.withSession model (\state -> ( model, Api.revokeAgent state.accessToken credentialId ))

        AgentRevoked _ ->
            ( model, Api.refreshCredentials model )

        LogoutClicked ->
            ( { model | session = LoggedOut, email = "", password = "" }
            , Cmd.batch [ Api.postLogout, Nav.pushUrl model.key "/" ]
            )

        LogoutReceived _ ->
            ( model, Cmd.none )

        DiscoveryIncludeReservedChanged value ->
            Api.withSession model
                (\state ->
                    let
                        nextState =
                            { state | discoveryIncludeReserved = value }
                    in
                    ( Api.updateLoggedIn model (\_ -> nextState), Api.fetchDiscovery state.accessToken value )
                )

        DiscoveryReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | discoveryTasks = Api.tasksFromResult result }), Cmd.none )

        DiscoveryViewClicked taskId ->
            ( Api.updateLoggedIn model
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
            ( Api.updateLoggedIn model (\state -> { state | detail = Just detail }), Cmd.none )

        DetailReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | submitMessage = Just (httpErrorLabel error) }), Cmd.none )

        ReserveClicked taskId ->
            Api.withSession model (\state -> ( Api.updateLoggedIn model (\current -> { current | reservationMessage = Nothing }), Api.postReservation state.accessToken taskId ))

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
            ( Api.updateLoggedIn model (\state -> { state | submissions = [], submitMessage = Just (httpErrorLabel error) }), Cmd.none )

        SubmitInputChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | submitInput = value }), Cmd.none )

        SubmitClicked ->
            Api.withSession model (\state -> Api.submitCommand model state)

        SubmitReceived (Ok created) ->
            ( Api.updateLoggedIn model (\state -> { state | submitMessage = Just (View.submitSuccessLabel created) })
            , Api.refreshDetailSubmissions model
            )

        SubmitReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | submitMessage = Just (httpErrorLabel error) }), Cmd.none )

        ReviewNoteChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | reviewNote = value }), Cmd.none )

        ReviewPartialCreditChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | reviewPartialCredit = value }), Cmd.none )

        ReviewTipChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | reviewTip = value }), Cmd.none )

        ReviewBanChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | reviewBan = value }), Cmd.none )

        AcceptClicked submissionId ->
            Api.withSession model (\state -> Api.acceptCommand model state submissionId)

        RequestChangesClicked submissionId ->
            Api.withSession model (\state -> Api.requestChangesCommand model state submissionId)

        RejectClicked submissionId ->
            Api.withSession model (\state -> Api.rejectCommand model state submissionId)

        ReviewActionReceived (Ok _) ->
            ( Api.updateLoggedIn model (\state -> { state | reviewMessage = Just "Review saved." }), Api.refreshAfterAccept model )

        ReviewActionReceived (Err error) ->
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
            Api.withSession updated (\state -> ( updated, Cmd.batch [ Api.fetchCollectibles state.accessToken, Api.fetchTasks state.accessToken state.taskStateFilter ] ))

        AwardReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | awardMessage = Just (httpErrorLabel error) }), Cmd.none )

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

        OrgTeamsReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | orgTeams = Api.teamsFromResult result }), Cmd.none )

        OrgMembersReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | orgMembers = Api.membersFromResult result }), Cmd.none )

        UserProfileReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | userProfile = Result.toMaybe result }), Cmd.none )

        UserWorkReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | userWork = Api.tasksFromResult result }), Cmd.none )

        UserSubmissionsReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | userSubmissions = Api.submissionsFromResult result }), Cmd.none )

        SeriesDetailReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | seriesDetail = Result.toMaybe result }), Cmd.none )

        TeamDetailReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | teamDetail = Result.toMaybe result }), Cmd.none )

        TeamMemberEmailChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | teamMemberEmail = value }), Cmd.none )

        AddTeamMemberClicked teamId ->
            Api.withSession model (\state -> ( model, Api.postAddTeamMember state.accessToken teamId state.teamMemberEmail ))

        AddTeamMemberReceived (Ok detail) ->
            ( Api.updateLoggedIn model (\state -> { state | teamDetail = Just detail, teamMemberEmail = "", teamMemberMessage = Just "Member added." }), Cmd.none )

        AddTeamMemberReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | teamMemberMessage = Just (httpErrorLabel error) }), Cmd.none )

        OrgTasksReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | orgTasks = Api.tasksFromResult result }), Cmd.none )

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

        CreateTaskOwnerChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | createTaskOwner = value }), Cmd.none )

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
                    , Api.routeLoadCmd state.accessToken page
                    )

                LoggedOut ->
                    ( { model | route = page }, Cmd.none )
