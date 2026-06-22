module Main exposing (main)

import Browser
import Browser.Navigation as Nav
import Html exposing (Html, div, form, label, main_, option, p, select, span, table, tbody, td, text, th, thead, tr)
import Html.Attributes exposing (checked, disabled, placeholder, selected, type_, value)
import Html.Events exposing (onCheck, onClick, onInput, onSubmit)
import Http
import Json.Decode as Decode
import Json.Encode as Encode
import Sharecrop.Generated.Agent as Agent
import Sharecrop.Generated.Auth as Auth
import Sharecrop.Generated.Collectible as Collectible
import Sharecrop.Generated.Ledger as Ledger
import Sharecrop.Generated.Submission as Submission
import Sharecrop.Generated.Task as Task
import Sharecrop.Ui as Ui exposing (testId)
import Url exposing (Url)


type alias Flags =
    { origin : String }


type Session
    = LoggedOut
    | LoggedIn LoggedInModel


type Page
    = DashboardPage
    | DiscoveryPage
    | TaskDetailPage String


type alias LoggedInModel =
    { accessToken : String
    , subjectId : String
    , page : Page
    , balance : Maybe Int
    , entries : List Ledger.LedgerEntryResponse
    , createTitle : String
    , createDescription : String
    , createRewardAmount : String
    , createPublic : Bool
    , createParticipationPolicy : String
    , createReservationHours : String
    , createMessage : Maybe String
    , fundTaskId : String
    , fundAmount : String
    , fundMessage : Maybe String
    , tasks : List Task.TaskListItemResponse
    , selectedTask : Maybe TaskDetail
    , agentLabel : String
    , agentScopes : List Agent.AgentScope
    , credentials : List Agent.AgentCredentialResponse
    , newCredential : Maybe Agent.AgentCredentialCreatedResponse
    , agentMessage : Maybe String
    , discoveryTasks : List Task.TaskListItemResponse
    , discoveryIncludeReserved : Bool
    , detail : Maybe PublicTaskDetail
    , reservations : List Task.TaskReservationResponse
    , reservationMessage : Maybe String
    , submissions : List Submission.SubmissionResponse
    , submitInput : String
    , submitMessage : Maybe String
    , reviewNote : String
    , reviewPartialCredit : String
    , reviewTip : String
    , reviewBan : Bool
    , reviewMessage : Maybe String
    , collectibles : List Collectible.CollectibleResponse
    , collectibleName : String
    , collectibleKind : Collectible.CollectibleKind
    , collectiblePolicy : Collectible.CollectibleTransferPolicy
    , collectibleMessage : Maybe String
    , awardTaskId : String
    , awardMessage : Maybe String
    }


type alias TaskDetail =
    { id : String
    , title : String
    , description : String
    , state : Task.TaskState
    , rewardKind : String
    , rewardCreditAmount : Int
    , rewardCollectibleCount : Int
    , participationPolicy : Task.TaskParticipationPolicy
    , assigneeScope : Task.TaskAssigneeScope
    , reservationExpiryHours : Int
    , availabilityKind : Task.TaskAvailabilityKind
    , viewerAction : Task.TaskViewerAction
    , responseSchemaJson : String
    , createdBy : String
    }


type alias PublicTaskDetail =
    TaskDetail


type alias Model =
    { origin : String
    , key : Nav.Key
    , route : Page
    , email : String
    , password : String
    , authError : Maybe String
    , session : Session
    }


type Msg
    = EmailChanged String
    | PasswordChanged String
    | RegisterClicked
    | LoginClicked
    | AuthReceived (Result Http.Error Auth.AuthResponse)
    | RefreshReceived (Result Http.Error Auth.AuthResponse)
    | BalanceReceived (Result Http.Error Ledger.BalanceResponse)
    | LedgerReceived (Result Http.Error Ledger.LedgerResponse)
    | TasksReceived (Result Http.Error Task.TasksResponse)
    | CreateTitleChanged String
    | CreateDescriptionChanged String
    | CreateRewardAmountChanged String
    | CreatePublicChanged Bool
    | CreateParticipationChanged String
    | CreateReservationHoursChanged String
    | CreateTaskClicked
    | CreateTaskReceived (Result Http.Error TaskDetail)
    | CredentialsReceived (Result Http.Error Agent.AgentCredentialsResponse)
    | FundTaskIdChanged String
    | FundAmountChanged String
    | FundClicked
    | FundReceived (Result Http.Error Ledger.TaskEscrowResponse)
    | SelectTask String
    | TaskDetailReceived (Result Http.Error TaskDetail)
    | OpenTaskClicked String
    | OpenTaskReceived (Result Http.Error TaskDetail)
    | RefundTaskClicked String
    | RefundTaskReceived (Result Http.Error Ledger.TaskEscrowResponse)
    | AgentLabelChanged String
    | ToggleScope Agent.AgentScope
    | CreateAgentClicked
    | AgentCreated (Result Http.Error Agent.AgentCredentialCreatedResponse)
    | RevokeClicked String
    | AgentRevoked (Result Http.Error Agent.AgentCredentialResponse)
    | LogoutClicked
    | LogoutReceived (Result Http.Error ())
    | NavDashboard
    | NavDiscovery
    | DiscoveryIncludeReservedChanged Bool
    | DiscoveryReceived (Result Http.Error Task.TasksResponse)
    | DiscoveryViewClicked String
    | DetailBackClicked
    | DetailReceived (Result Http.Error PublicTaskDetail)
    | ReserveClicked String
    | ReservationReceived (Result Http.Error Task.TaskReservationResponse)
    | ReservationsReceived (Result Http.Error Task.TaskReservationsResponse)
    | ApproveReservationClicked String
    | DeclineReservationClicked String
    | CancelReservationClicked String
    | ReservationChangeReceived (Result Http.Error Task.TaskReservationResponse)
    | SubmissionsReceived (Result Http.Error Submission.SubmissionsResponse)
    | SubmitInputChanged String
    | SubmitClicked
    | SubmitReceived (Result Http.Error Submission.SubmissionCreatedResponse)
    | ReviewNoteChanged String
    | ReviewPartialCreditChanged String
    | ReviewTipChanged String
    | ReviewBanChanged Bool
    | AcceptClicked String
    | RequestChangesClicked String
    | RejectClicked String
    | ReviewActionReceived (Result Http.Error ())
    | CollectibleNameChanged String
    | CollectibleKindChosen Collectible.CollectibleKind
    | CollectiblePolicyChosen Collectible.CollectibleTransferPolicy
    | MintClicked
    | MintReceived (Result Http.Error Collectible.CollectibleResponse)
    | CollectiblesReceived (Result Http.Error Collectible.CollectiblesResponse)
    | AwardTaskIdChanged String
    | AwardClicked String
    | AwardReceived (Result Http.Error Collectible.CollectibleResponse)
    | LinkClicked Browser.UrlRequest
    | UrlChanged Url


main : Program Flags Model Msg
main =
    Browser.application
        { init = \flags url key -> ( initialModel flags key url, postRefresh )
        , update = update
        , subscriptions = \_ -> Sub.none
        , view = view
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
    , page = DashboardPage
    , balance = Nothing
    , entries = []
    , createTitle = ""
    , createDescription = ""
    , createRewardAmount = ""
    , createPublic = False
    , createParticipationPolicy = participationPolicyTag Task.TaskParticipationPolicyOpen
    , createReservationHours = "48"
    , createMessage = Nothing
    , fundTaskId = ""
    , fundAmount = ""
    , fundMessage = Nothing
    , tasks = []
    , selectedTask = Nothing
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
        [ "discovery" ] ->
            DiscoveryPage

        [ "tasks", taskId ] ->
            TaskDetailPage taskId

        _ ->
            DashboardPage


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

        CreateTitleChanged value ->
            ( updateLoggedIn model (\state -> { state | createTitle = value }), Cmd.none )

        CreateDescriptionChanged value ->
            ( updateLoggedIn model (\state -> { state | createDescription = value }), Cmd.none )

        CreateRewardAmountChanged value ->
            ( updateLoggedIn model (\state -> { state | createRewardAmount = value }), Cmd.none )

        CreatePublicChanged value ->
            ( updateLoggedIn model (\state -> { state | createPublic = value }), Cmd.none )

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

        FundClicked ->
            withSession model (\state -> fundTaskCommand model state)

        FundReceived (Ok escrow) ->
            ( updateLoggedIn model (\state -> { state | fundMessage = Just (fundSuccessLabel escrow) }), refreshLedger model )

        FundReceived (Err error) ->
            ( updateLoggedIn model (\state -> { state | fundMessage = Just (httpErrorLabel error) }), Cmd.none )

        SelectTask taskId ->
            withSession model (\state -> ( model, Cmd.batch [ fetchTaskDetail state.accessToken taskId, fetchSubmissions state.accessToken taskId ] ))

        TaskDetailReceived (Ok detail) ->
            ( updateLoggedIn model (\state -> { state | selectedTask = Just detail }), Cmd.none )

        TaskDetailReceived (Err error) ->
            ( updateLoggedIn model (\state -> { state | createMessage = Just (httpErrorLabel error) }), Cmd.none )

        OpenTaskClicked taskId ->
            withSession model (\state -> ( model, postOpenTask state.accessToken taskId ))

        OpenTaskReceived (Ok detail) ->
            ( updateLoggedIn model (\state -> { state | selectedTask = Just detail, createMessage = Just "Task opened." })
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
            , Cmd.batch [ postLogout, Nav.pushUrl model.key "/dashboard" ]
            )

        LogoutReceived _ ->
            ( model, Cmd.none )

        NavDashboard ->
            ( model, Nav.pushUrl model.key "/dashboard" )

        NavDiscovery ->
            ( model, Nav.pushUrl model.key "/discovery" )

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
            withSession model
                (\state ->
                    ( updateLoggedIn model
                        (\s ->
                            { s
                                | page = TaskDetailPage taskId
                                , detail = Nothing
                                , reservations = []
                                , reservationMessage = Nothing
                                , submissions = []
                                , submitInput = ""
                                , submitMessage = Nothing
                            }
                        )
                    , fetchDetailCommands state.accessToken taskId
                    )
                )

        DetailBackClicked ->
            ( updateLoggedIn model (\state -> { state | page = DiscoveryPage }), Cmd.none )

        DetailReceived (Ok detail) ->
            ( updateLoggedIn model (\state -> { state | detail = Just detail }), Cmd.none )

        DetailReceived (Err error) ->
            ( updateLoggedIn model (\state -> { state | submitMessage = Just (httpErrorLabel error) }), Cmd.none )

        ReserveClicked taskId ->
            withSession model (\state -> ( updateLoggedIn model (\current -> { current | reservationMessage = Nothing }), postReservation state.accessToken taskId ))

        ReservationReceived (Ok reservation) ->
            ( updateLoggedIn model (\state -> { state | reservationMessage = Just (reservationSuccessLabel reservation) })
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
            ( updateLoggedIn model (\state -> { state | reservationMessage = Just (reservationSuccessLabel reservation) })
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
            ( updateLoggedIn model (\state -> { state | submitMessage = Just (submitSuccessLabel created) })
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
                        , collectibleMessage = Just (mintSuccessLabel collectible)
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
            ( updateLoggedIn model (\state -> { state | awardMessage = Just (awardSuccessLabel collectible) }), refreshCollectibles model )

        AwardReceived (Err error) ->
            ( updateLoggedIn model (\state -> { state | awardMessage = Just (httpErrorLabel error) }), Cmd.none )

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
                    ( { model | route = page, session = LoggedIn { state | page = page } }
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
            ( updateLoggedIn model (\current -> { current | fundMessage = Nothing }), postFunding state.accessToken state.fundTaskId amount )

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
    Cmd.batch [ fetchBalance token, fetchLedger token, fetchTasks token, fetchCredentials token, fetchCollectibles token ]


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
            Cmd.batch [ fetchTasks state.accessToken, fetchBalance state.accessToken, fetchLedger state.accessToken ]

        LoggedOut ->
            Cmd.none


refreshTasksAndDiscovery : Model -> Cmd Msg
refreshTasksAndDiscovery model =
    case model.session of
        LoggedIn state ->
            Cmd.batch [ fetchTasks state.accessToken, fetchDiscovery state.accessToken state.discoveryIncludeReserved ]

        LoggedOut ->
            Cmd.none


routeLoadCmd : String -> Page -> Cmd Msg
routeLoadCmd token page =
    case page of
        DashboardPage ->
            Cmd.none

        DiscoveryPage ->
            fetchDiscovery token False

        TaskDetailPage taskId ->
            fetchDetailCommands token taskId


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


fetchTasks : String -> Cmd Msg
fetchTasks token =
    authorizedRequest "GET" token "/api/tasks?scope=user" Http.emptyBody (Http.expectJson TasksReceived Task.tasksResponseDecoder)


fetchCredentials : String -> Cmd Msg
fetchCredentials token =
    authorizedRequest "GET" token "/api/agent-credentials" Http.emptyBody (Http.expectJson CredentialsReceived Agent.agentCredentialsResponseDecoder)


fetchTaskDetail : String -> String -> Cmd Msg
fetchTaskDetail token taskId =
    authorizedRequest "GET" token ("/api/tasks/" ++ taskId) Http.emptyBody (Http.expectJson TaskDetailReceived taskDetailDecoder)


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


postFunding : String -> String -> Int -> Cmd Msg
postFunding token taskId amount =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/funding")
        (Http.jsonBody (fundingRequestBody taskId amount))
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


fundingRequestBody : String -> Int -> Encode.Value
fundingRequestBody taskId amount =
    Encode.object
        [ ( "amount", Encode.int amount )
        , ( "idempotency_key", Encode.string ("fund:" ++ taskId) )
        ]


createTaskRequestBody : LoggedInModel -> Encode.Value
createTaskRequestBody state =
    Encode.object
        [ ( "owner", Encode.object [ ( "kind", Encode.string "user" ), ( "user_id", Encode.string state.subjectId ), ( "team_id", Encode.string "" ), ( "organization_id", Encode.string "" ) ] )
        , ( "title", Encode.string state.createTitle )
        , ( "description", Encode.string state.createDescription )
        , ( "reward", createRewardBody state.createRewardAmount )
        , ( "participation", createParticipationBody state )
        , ( "visibility", Encode.object [ ( "kind", Encode.string (if state.createPublic then "public" else "default") ), ( "user_id", Encode.string "" ), ( "team_id", Encode.string "" ), ( "organization_id", Encode.string "" ) ] )
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
        , ( "assignee_scope", Encode.string (assigneeScopeTag Task.TaskAssigneeScopeUser) )
        , ( "reservation_expiry_hours", Encode.int (reservationHoursValue state.createReservationHours) )
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


view : Model -> Browser.Document Msg
view model =
    { title = "Sharecrop"
    , body =
        [ main_ [ Html.Attributes.class "min-h-screen bg-slate-50 p-8 text-slate-950" ]
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
        [ navBar
        , pageView origin state
        ]


navBar : Html Msg
navBar =
    div [ Html.Attributes.class "flex gap-3" ]
        [ Ui.secondaryButton [ type_ "button", onClick NavDashboard, testId "nav-dashboard" ] "Dashboard"
        , Ui.secondaryButton [ type_ "button", onClick NavDiscovery, testId "nav-discovery" ] "Discovery"
        ]


pageView : String -> LoggedInModel -> Html Msg
pageView origin state =
    case state.page of
        DashboardPage ->
            dashboardView origin state

        DiscoveryPage ->
            discoveryView state

        TaskDetailPage _ ->
            taskDetailPageView origin state


dashboardView : String -> LoggedInModel -> Html Msg
dashboardView origin state =
    div [ Html.Attributes.class "space-y-6" ]
        [ div [ Html.Attributes.class "flex items-center justify-between" ]
            [ Ui.sectionTitle "Credit account"
            , Ui.secondaryButton [ onClick LogoutClicked, testId "logout" ] "Log out"
            ]
        , balanceView state.balance
        , ledgerView state.entries
        , createTaskView state
        , fundingView state
        , tasksView origin state
        , agentsView origin state
        , collectiblesView state
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
        , Ui.textInput [ type_ "text", placeholder "Title", value state.createTitle, onInput CreateTitleChanged, testId "create-title" ]
        , Ui.textarea_
            [ placeholder "Description"
            , value state.createDescription
            , onInput CreateDescriptionChanged
            , Html.Attributes.rows 3
            , testId "create-description"
            ]
        , Ui.textInput [ type_ "number", placeholder "Credit reward amount (blank for no reward)", value state.createRewardAmount, onInput CreateRewardAmountChanged, testId "create-reward" ]
        , Ui.label_ "Participation"
        , div [ Html.Attributes.class "flex flex-wrap gap-2" ] (List.map (participationButton state.createParticipationPolicy) allParticipationPolicies)
        , Ui.textInput [ type_ "number", placeholder "Reservation expiry hours", value state.createReservationHours, onInput CreateReservationHoursChanged, testId "create-reservation-hours" ]
        , label [ Html.Attributes.class "flex items-center gap-2 text-sm" ]
            [ Html.input [ type_ "checkbox", checked state.createPublic, onClick (CreatePublicChanged (not state.createPublic)), testId "create-public" ] []
            , span [] [ text "Publish publicly" ]
            ]
        , Ui.primaryButton [ type_ "submit", testId "create-task" ] "Create task"
        , maybeNote state.createMessage "create-message"
        ]


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
    tr [ Html.Attributes.class "border-t border-slate-100", testId "ledger-entry" ]
        [ td [ Html.Attributes.class "py-2" ] [ text (kindLabel entry.kind) ]
        , td [ Html.Attributes.class "py-2 text-right tabular-nums" ] [ text (String.fromInt entry.amount) ]
        ]


fundingView : LoggedInModel -> Html Msg
fundingView state =
    form [ Html.Attributes.class "space-y-4 rounded-lg border border-slate-200 bg-white p-6 shadow-sm", onSubmit FundClicked ]
        [ Ui.sectionTitle "Fund a task"
        , taskPicker "fund-task-id" state.fundTaskId FundTaskIdChanged state.tasks
        , Ui.textInput [ type_ "number", placeholder "Amount in credits", value state.fundAmount, onInput FundAmountChanged, testId "fund-amount" ]
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
        , tasksList state.tasks
        , taskDetailView origin state
        ]


tasksList : List Task.TaskListItemResponse -> Html Msg
tasksList tasks =
    if List.isEmpty tasks then
        p [ Html.Attributes.class "text-sm text-slate-500", testId "tasks-empty" ] [ text "No tasks yet." ]

    else
        div [ Html.Attributes.class "divide-y divide-slate-100", testId "tasks" ] (List.map taskRow tasks)


taskRow : Task.TaskListItemResponse -> Html Msg
taskRow item =
    div [ Html.Attributes.class "flex items-center justify-between py-2", testId "task-row" ]
        [ div []
            [ p [ Html.Attributes.class "font-medium" ] [ text item.title ]
            , p [ Html.Attributes.class "text-xs text-slate-500" ] [ text (taskStateLabel item.state ++ " · " ++ rewardLabel item.rewardKind item.rewardCreditAmount item.rewardCollectibleCount) ]
            ]
        , Ui.secondaryButton [ onClick (SelectTask item.id), testId "view-task" ] "View"
        ]


taskDetailView : String -> LoggedInModel -> Html Msg
taskDetailView origin state =
    case state.selectedTask of
        Just detail ->
            div [ Html.Attributes.class "mt-4 space-y-3 rounded-md bg-slate-50 p-4", testId "task-detail" ]
                [ Ui.label_ detail.title
                , p [ Html.Attributes.class "text-sm text-slate-700" ] [ text detail.description ]
                , Ui.label_ ("Task " ++ detail.id)
                , p [ Html.Attributes.class "text-sm" ] [ text ("State: " ++ taskStateLabel detail.state) ]
                , p [ Html.Attributes.class "text-sm" ] [ text ("Reward: " ++ rewardLabel detail.rewardKind detail.rewardCreditAmount detail.rewardCollectibleCount) ]
                , p [ Html.Attributes.class "text-sm" ] [ text ("Participation: " ++ participationPolicyLabel detail.participationPolicy) ]
                , p [ Html.Attributes.class "text-sm" ] [ text ("Reservation expiry: " ++ String.fromInt detail.reservationExpiryHours ++ " hours") ]
                , div [ Html.Attributes.class "flex gap-2" ]
                    [ Ui.secondaryButton [ type_ "button", onClick (OpenTaskClicked detail.id), testId "open-task" ] "Open"
                    , Ui.secondaryButton [ type_ "button", onClick (RefundTaskClicked detail.id), testId "refund-task" ] "Refund"
                    ]
                , Ui.label_ "Response schema"
                , Ui.codeBlock [ testId "task-schema" ] detail.responseSchemaJson
                , submissionsList state
                , taskInstructions origin detail.id
                ]

        Nothing ->
            text ""


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
    label [ Html.Attributes.class "flex items-center gap-2 text-sm" ]
        [ Html.input
            [ type_ "checkbox"
            , checked (List.member scope selected)
            , onClick (ToggleScope scope)
            , testId ("scope-" ++ scopeTag scope)
            ]
            []
        , span [] [ text (scopeTag scope) ]
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
            , p [ Html.Attributes.class "text-xs text-slate-500" ] [ text (credentialStateLabel credential.state ++ " · " ++ String.join ", " (List.map scopeTag credential.scopes)) ]
            ]
        , revokeButton credential
        ]


revokeButton : Agent.AgentCredentialResponse -> Html Msg
revokeButton credential =
    case credential.state of
        Agent.AgentCredentialStateActive ->
            Ui.secondaryButton [ onClick (RevokeClicked credential.id), testId "revoke-credential" ] "Revoke"

        Agent.AgentCredentialStateRevoked ->
            span [ Html.Attributes.class "text-xs text-slate-400" ] [ text "revoked" ]



-- Collectibles panel


collectiblesView : LoggedInModel -> Html Msg
collectiblesView state =
    Ui.card
        [ Ui.sectionTitle "Collectibles"
        , p [ Html.Attributes.class "text-sm text-slate-600" ] [ text "Mint collectibles and award them to tasks." ]
        , mintForm state
        , awardForm state
        , collectiblesList state
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
        Ui.primaryButton [ type_ "button", onClick msg, testId identifier ] labelText

    else
        Ui.secondaryButton [ type_ "button", onClick msg, testId identifier ] labelText


awardForm : LoggedInModel -> Html Msg
awardForm state =
    div [ Html.Attributes.class "mt-4 space-y-3" ]
        [ Ui.label_ "Award to a task"
        , taskPicker "award-task-id" state.awardTaskId AwardTaskIdChanged state.tasks
        , maybeNote state.awardMessage "award-message"
        ]


collectiblesList : LoggedInModel -> Html Msg
collectiblesList state =
    if List.isEmpty state.collectibles then
        p [ Html.Attributes.class "mt-4 text-sm text-slate-500", testId "collectibles-empty" ] [ text "No collectibles yet." ]

    else
        div [ Html.Attributes.class "mt-4 divide-y divide-slate-100", testId "collectibles" ] (List.map collectibleRow state.collectibles)


collectibleRow : Collectible.CollectibleResponse -> Html Msg
collectibleRow collectible =
    div [ Html.Attributes.class "flex items-center justify-between py-2", testId "collectible-row" ]
        [ div [ Html.Attributes.class "flex items-center gap-2" ]
            [ p [ Html.Attributes.class "font-medium" ] [ text collectible.name ]
            , Ui.badge (collectibleStateLabel collectible.state)
            , span [ Html.Attributes.class "text-xs text-slate-500" ] [ text (collectibleKindLabel collectible.kind) ]
            ]
        , awardCollectibleButton collectible
        ]


awardCollectibleButton : Collectible.CollectibleResponse -> Html Msg
awardCollectibleButton collectible =
    case collectible.state of
        Collectible.CollectibleStateMinted ->
            Ui.secondaryButton [ type_ "button", onClick (AwardClicked collectible.id), testId "award-collectible" ] "Award"

        _ ->
            text ""



-- Discovery page


discoveryView : LoggedInModel -> Html Msg
discoveryView state =
    Ui.card
        [ Ui.sectionTitle "Discover public tasks"
        , label [ Html.Attributes.class "flex items-center gap-2 text-sm" ]
            [ Html.input [ type_ "checkbox", checked state.discoveryIncludeReserved, onClick (DiscoveryIncludeReservedChanged (not state.discoveryIncludeReserved)), testId "include-reserved" ] []
            , span [] [ text "Include reserved" ]
            ]
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
    div [ Html.Attributes.class "flex items-center justify-between py-2", testId "discovery-task-row" ]
        [ div []
            [ p [ Html.Attributes.class "font-medium" ] [ text item.title ]
            , p [ Html.Attributes.class "text-xs text-slate-500" ] [ text (taskStateLabel item.state ++ " · " ++ rewardLabel item.rewardKind item.rewardCreditAmount item.rewardCollectibleCount ++ " · " ++ participationPolicyLabel item.participationPolicy) ]
            ]
        , Ui.secondaryButton [ onClick (DiscoveryViewClicked item.id), testId "discovery-view" ] "View"
        ]



-- Task detail page


taskDetailPageView : String -> LoggedInModel -> Html Msg
taskDetailPageView origin state =
    div [ Html.Attributes.class "space-y-6" ]
        [ Ui.secondaryButton [ onClick DetailBackClicked, testId "detail-back" ] "Back to discovery"
        , detailCard origin state
        , reservationCard state
        , submitCard state
        , submissionsCard state
        ]


detailCard : String -> LoggedInModel -> Html Msg
detailCard origin state =
    case state.detail of
        Just detail ->
            Ui.card
                [ p [ Html.Attributes.class "text-2xl font-semibold", testId "detail-title" ] [ text detail.title ]
                , div [ Html.Attributes.class "flex flex-wrap items-center gap-2" ]
                    [ Ui.badge (taskStateLabel detail.state)
                    , Ui.badge (availabilityKindLabel detail.availabilityKind)
                    , Ui.badge (participationPolicyLabel detail.participationPolicy)
                    ]
                , p [ Html.Attributes.class "text-sm font-medium" ] [ text ("Reward: " ++ rewardLabel detail.rewardKind detail.rewardCreditAmount detail.rewardCollectibleCount) ]
                , p [ Html.Attributes.class "text-sm text-slate-700" ] [ text detail.description ]
                , Ui.label_ "Response schema"
                , Ui.codeBlock [ testId "detail-schema" ] detail.responseSchemaJson
                , taskInstructions origin detail.id
                ]

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
            , label [ Html.Attributes.class "flex items-center gap-2 pt-6 text-sm text-slate-700" ]
                [ Html.input [ type_ "checkbox", checked state.reviewBan, onCheck ReviewBanChanged, testId "review-ban" ] []
                , text "Ban implementor"
                ]
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
        ]


reviewButtons : LoggedInModel -> Submission.SubmissionResponse -> Html Msg
reviewButtons state submission =
    case submission.state of
        Submission.SubmissionStateSubmitted ->
            div [ Html.Attributes.class "flex flex-wrap justify-end gap-2" ]
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


taskInstructions : String -> String -> Html Msg
taskInstructions origin taskId =
    div [ Html.Attributes.class "space-y-3", testId "task-instructions" ]
        [ Ui.label_ "REST API"
        , Ui.codeBlock [ testId "task-rest-submit" ] (restSubmitCurl origin taskId)
        , Ui.codeBlock [ testId "task-rest-reserve" ] (restReserveCurl origin taskId)
        , Ui.label_ "MCP"
        , Ui.codeBlock [ testId "task-mcp-submit" ] (mcpSubmitCurl origin taskId)
        , Ui.codeBlock [ testId "task-mcp-schema" ] (mcpSchemaCurl origin taskId)
        ]


restSubmitCurl : String -> String -> String
restSubmitCurl origin taskId =
    "curl -X POST "
        ++ origin
        ++ "/api/tasks/"
        ++ taskId
        ++ "/submissions \\\n  -H \"Authorization: Bearer <ACCESS_TOKEN>\" \\\n  -H \"Content-Type: application/json\" \\\n  -d '{\"response_json\":\"{}\"}'"


restReserveCurl : String -> String -> String
restReserveCurl origin taskId =
    "curl -X POST "
        ++ origin
        ++ "/api/tasks/"
        ++ taskId
        ++ "/reservations \\\n  -H \"Authorization: Bearer <ACCESS_TOKEN>\""


mcpSubmitCurl : String -> String -> String
mcpSubmitCurl origin taskId =
    "curl -X POST "
        ++ origin
        ++ "/mcp \\\n  -H \"Authorization: Bearer <AGENT_TOKEN>\" \\\n  -H \"Content-Type: application/json\" \\\n  -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"sharecrop.submit_response\",\"arguments\":{\"task_id\":\""
        ++ taskId
        ++ "\",\"response_json\":\"{}\"}}}'"


mcpSchemaCurl : String -> String -> String
mcpSchemaCurl origin taskId =
    "curl -X POST "
        ++ origin
        ++ "/mcp \\\n  -H \"Authorization: Bearer <AGENT_TOKEN>\" \\\n  -H \"Content-Type: application/json\" \\\n  -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"sharecrop.get_task_schema\",\"arguments\":{\"task_id\":\""
        ++ taskId
        ++ "\"}}}'"


fundSuccessLabel : Ledger.TaskEscrowResponse -> String
fundSuccessLabel escrow =
    "Escrowed " ++ String.fromInt escrow.amount ++ " credits (" ++ escrowStateLabel escrow.state ++ ")."


submitSuccessLabel : Submission.SubmissionCreatedResponse -> String
submitSuccessLabel created =
    "Submission " ++ created.submission.id ++ " (" ++ submissionStateLabel created.submission.state ++ ")."


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
collectibleKindLabel =
    collectibleKindTag


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
collectiblePolicyLabel =
    collectiblePolicyTag


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


assigneeScopeLabel : Task.TaskAssigneeScope -> String
assigneeScopeLabel scope =
    case scope of
        Task.TaskAssigneeScopeUser ->
            "user"

        Task.TaskAssigneeScopeOrganizationTeam ->
            "organization team"


availabilityKindLabel : Task.TaskAvailabilityKind -> String
availabilityKindLabel kind =
    case kind of
        Task.TaskAvailabilityKindAvailable ->
            "available"

        Task.TaskAvailabilityKindReserved ->
            "reserved"

        Task.TaskAvailabilityKindAwaitingApproval ->
            "awaiting approval"

        Task.TaskAvailabilityKindClosed ->
            "closed"


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


credentialStateLabel : Agent.AgentCredentialState -> String
credentialStateLabel state =
    case state of
        Agent.AgentCredentialStateActive ->
            "active"

        Agent.AgentCredentialStateRevoked ->
            "revoked"


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
            "signup_grant"

        Ledger.LedgerEntryKindTaskEscrow ->
            "task_escrow"

        Ledger.LedgerEntryKindTaskRefund ->
            "task_refund"

        Ledger.LedgerEntryKindTaskPayout ->
            "task_payout"

        Ledger.LedgerEntryKindTaskTip ->
            "task_tip"

        Ledger.LedgerEntryKindManualAdjustment ->
            "manual_adjustment"


escrowStateLabel : Ledger.EscrowState -> String
escrowStateLabel state =
    case state of
        Ledger.EscrowStateHeld ->
            "held"

        Ledger.EscrowStateReleased ->
            "released"

        Ledger.EscrowStateRefunded ->
            "refunded"


rewardLabel : String -> Int -> Int -> String
rewardLabel kind amount collectibleCount =
    case kind of
        "credit" ->
            String.fromInt amount ++ " credits"

        "collectible" ->
            String.fromInt collectibleCount ++ " collectible"

        "bundle" ->
            String.fromInt amount ++ " credits + " ++ String.fromInt collectibleCount ++ " collectible"

        _ ->
            "no reward"


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
            "The response was unexpected: " ++ message
