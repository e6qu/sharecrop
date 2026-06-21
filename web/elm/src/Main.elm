module Main exposing (main)

import Browser
import Html exposing (Html, div, form, label, main_, p, span, table, tbody, td, text, th, thead, tr)
import Html.Attributes exposing (checked, disabled, placeholder, type_, value)
import Html.Events exposing (onClick, onInput, onSubmit)
import Http
import Json.Decode as Decode
import Json.Encode as Encode
import Sharecrop.Generated.Agent as Agent
import Sharecrop.Generated.Auth as Auth
import Sharecrop.Generated.Ledger as Ledger
import Sharecrop.Generated.Submission as Submission
import Sharecrop.Generated.Task as Task
import Sharecrop.Ui as Ui exposing (testId)


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
    , detail : Maybe PublicTaskDetail
    , submissions : List Submission.SubmissionResponse
    , submitInput : String
    , submitMessage : Maybe String
    }


type alias TaskDetail =
    { id : String
    , state : String
    , responseSchemaJson : String
    }


type alias PublicTaskDetail =
    { id : String
    , ownerKind : String
    , title : String
    , description : String
    , state : String
    , responseSchemaJson : String
    , createdBy : String
    }


type alias Model =
    { origin : String
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
    | BalanceReceived (Result Http.Error Ledger.BalanceResponse)
    | LedgerReceived (Result Http.Error Ledger.LedgerResponse)
    | TasksReceived (Result Http.Error Task.TasksResponse)
    | CredentialsReceived (Result Http.Error Agent.AgentCredentialsResponse)
    | FundTaskIdChanged String
    | FundAmountChanged String
    | FundClicked
    | FundReceived (Result Http.Error Ledger.TaskEscrowResponse)
    | SelectTask String
    | TaskDetailReceived (Result Http.Error TaskDetail)
    | AgentLabelChanged String
    | ToggleScope Agent.AgentScope
    | CreateAgentClicked
    | AgentCreated (Result Http.Error Agent.AgentCredentialCreatedResponse)
    | RevokeClicked String
    | AgentRevoked (Result Http.Error Agent.AgentCredentialResponse)
    | LogoutClicked
    | NavDashboard
    | NavDiscovery
    | DiscoveryReceived (Result Http.Error Task.TasksResponse)
    | DiscoveryViewClicked String
    | DetailBackClicked
    | DetailReceived (Result Http.Error PublicTaskDetail)
    | SubmissionsReceived (Result Http.Error Submission.SubmissionsResponse)
    | SubmitInputChanged String
    | SubmitClicked
    | SubmitReceived (Result Http.Error Submission.SubmissionCreatedResponse)
    | AcceptClicked String
    | AcceptReceived (Result Http.Error ())


main : Program Flags Model Msg
main =
    Browser.element
        { init = \flags -> ( initialModel flags, Cmd.none )
        , update = update
        , subscriptions = \_ -> Sub.none
        , view = view
        }


initialModel : Flags -> Model
initialModel flags =
    { origin = flags.origin
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
    , detail = Nothing
    , submissions = []
    , submitInput = ""
    , submitMessage = Nothing
    }


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
            ( { model | password = "", authError = Nothing, session = LoggedIn (emptyLoggedIn response) }
            , loadAfterAuth response.accessToken
            )

        AuthReceived (Err error) ->
            ( { model | authError = Just (httpErrorLabel error) }, Cmd.none )

        BalanceReceived result ->
            ( updateLoggedIn model (\state -> { state | balance = balanceFromResult result }), Cmd.none )

        LedgerReceived result ->
            ( updateLoggedIn model (\state -> { state | entries = entriesFromResult result }), Cmd.none )

        TasksReceived result ->
            ( updateLoggedIn model (\state -> { state | tasks = tasksFromResult result }), Cmd.none )

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
            withSession model (\state -> ( model, fetchTaskDetail state.accessToken taskId ))

        TaskDetailReceived (Ok detail) ->
            ( updateLoggedIn model (\state -> { state | selectedTask = Just detail }), Cmd.none )

        TaskDetailReceived (Err _) ->
            ( model, Cmd.none )

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
            ( { model | session = LoggedOut, email = "", password = "" }, Cmd.none )

        NavDashboard ->
            ( updateLoggedIn model (\state -> { state | page = DashboardPage }), Cmd.none )

        NavDiscovery ->
            withSession model
                (\state ->
                    ( updateLoggedIn model (\s -> { s | page = DiscoveryPage }), fetchDiscovery state.accessToken )
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
                                , submissions = []
                                , submitInput = ""
                                , submitMessage = Nothing
                            }
                        )
                    , Cmd.batch
                        [ fetchPublicTaskDetail state.accessToken taskId
                        , fetchSubmissions state.accessToken taskId
                        ]
                    )
                )

        DetailBackClicked ->
            ( updateLoggedIn model (\state -> { state | page = DiscoveryPage }), Cmd.none )

        DetailReceived (Ok detail) ->
            ( updateLoggedIn model (\state -> { state | detail = Just detail }), Cmd.none )

        DetailReceived (Err _) ->
            ( model, Cmd.none )

        SubmissionsReceived (Ok response) ->
            ( updateLoggedIn model (\state -> { state | submissions = response.submissions }), Cmd.none )

        SubmissionsReceived (Err _) ->
            ( updateLoggedIn model (\state -> { state | submissions = [] }), Cmd.none )

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

        AcceptClicked submissionId ->
            withSession model (\state -> acceptCommand model state submissionId)

        AcceptReceived _ ->
            ( model, refreshAfterAccept model )


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
            ( model, postAccept state.accessToken taskId submissionId )

        _ ->
            ( model, Cmd.none )


loadAfterAuth : String -> Cmd Msg
loadAfterAuth token =
    Cmd.batch [ fetchBalance token, fetchLedger token, fetchTasks token, fetchCredentials token ]


refreshLedger : Model -> Cmd Msg
refreshLedger model =
    case model.session of
        LoggedIn state ->
            Cmd.batch [ fetchBalance state.accessToken, fetchLedger state.accessToken ]

        LoggedOut ->
            Cmd.none


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


fetchDiscovery : String -> Cmd Msg
fetchDiscovery token =
    authorizedRequest "GET" token "/api/tasks?scope=public" Http.emptyBody (Http.expectJson DiscoveryReceived Task.tasksResponseDecoder)


fetchPublicTaskDetail : String -> String -> Cmd Msg
fetchPublicTaskDetail token taskId =
    authorizedRequest "GET" token ("/api/tasks/" ++ taskId) Http.emptyBody (Http.expectJson DetailReceived publicTaskDetailDecoder)


fetchSubmissions : String -> String -> Cmd Msg
fetchSubmissions token taskId =
    authorizedRequest "GET" token ("/api/tasks/" ++ taskId ++ "/submissions") Http.emptyBody (Http.expectJson SubmissionsReceived Submission.submissionsResponseDecoder)


postFunding : String -> String -> Int -> Cmd Msg
postFunding token taskId amount =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/funding")
        (Http.jsonBody (fundingRequestBody taskId amount))
        (Http.expectJson FundReceived Ledger.taskEscrowResponseDecoder)


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


postAccept : String -> String -> String -> Cmd Msg
postAccept token taskId submissionId =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/submissions/" ++ submissionId ++ "/accept")
        (Http.jsonBody (acceptRequestBody submissionId))
        (Http.expectWhatever AcceptReceived)


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
        , ( "wallet_address", Encode.string "" )
        ]


acceptRequestBody : String -> Encode.Value
acceptRequestBody submissionId =
    Encode.object
        [ ( "idempotency_key", Encode.string ("ui-accept:" ++ submissionId) )
        ]


taskDetailDecoder : Decode.Decoder TaskDetail
taskDetailDecoder =
    Decode.map3 TaskDetail
        (Decode.field "id" Decode.string)
        (Decode.field "state" Decode.string)
        (Decode.field "response_schema_json" Decode.string)


publicTaskDetailDecoder : Decode.Decoder PublicTaskDetail
publicTaskDetailDecoder =
    Decode.map7 PublicTaskDetail
        (Decode.field "id" Decode.string)
        (Decode.field "owner_kind" Decode.string)
        (Decode.field "title" Decode.string)
        (Decode.field "description" Decode.string)
        (Decode.field "state" Decode.string)
        (Decode.field "response_schema_json" Decode.string)
        (Decode.field "created_by" Decode.string)


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


view : Model -> Html Msg
view model =
    main_ [ Html.Attributes.class "min-h-screen bg-slate-50 p-8 text-slate-950" ]
        [ div [ Html.Attributes.class "mx-auto max-w-3xl space-y-6" ]
            [ Ui.pageTitle "Sharecrop"
            , sessionView model
            ]
        ]


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
            taskDetailPageView state


dashboardView : String -> LoggedInModel -> Html Msg
dashboardView origin state =
    div [ Html.Attributes.class "space-y-6" ]
        [ div [ Html.Attributes.class "flex items-center justify-between" ]
            [ Ui.sectionTitle "Credit account"
            , Ui.secondaryButton [ onClick LogoutClicked, testId "logout" ] "Log out"
            ]
        , balanceView state.balance
        , ledgerView state.entries
        , fundingView state
        , tasksView origin state
        , agentsView origin state
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
        , Ui.textInput [ type_ "text", placeholder "Task ID", value state.fundTaskId, onInput FundTaskIdChanged, testId "fund-task-id" ]
        , Ui.textInput [ type_ "number", placeholder "Amount in credits", value state.fundAmount, onInput FundAmountChanged, testId "fund-amount" ]
        , Ui.primaryButton [ type_ "submit", disabled (state.fundTaskId == ""), testId "fund" ] "Fund task"
        , maybeNote state.fundMessage "fund-message"
        ]


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
            , p [ Html.Attributes.class "text-xs text-slate-500" ] [ text (taskStateLabel item.state) ]
            ]
        , Ui.secondaryButton [ onClick (SelectTask item.id), testId "view-task" ] "View"
        ]


taskDetailView : String -> LoggedInModel -> Html Msg
taskDetailView origin state =
    case state.selectedTask of
        Just detail ->
            div [ Html.Attributes.class "mt-4 space-y-3 rounded-md bg-slate-50 p-4", testId "task-detail" ]
                [ Ui.label_ ("Task " ++ detail.id)
                , p [ Html.Attributes.class "text-sm" ] [ text ("State: " ++ detail.state) ]
                , Ui.label_ "Response schema"
                , Ui.codeBlock [ testId "task-schema" ] detail.responseSchemaJson
                , Ui.label_ "Submit with the REST API"
                , Ui.codeBlock [] (restSubmitCurl origin detail.id)
                , Ui.label_ "Submit with an MCP agent"
                , Ui.codeBlock [ testId "task-mcp-curl" ] (mcpSubmitCurl origin detail.id)
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



-- Discovery page


discoveryView : LoggedInModel -> Html Msg
discoveryView state =
    Ui.card
        [ Ui.sectionTitle "Discover public tasks"
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
            , p [ Html.Attributes.class "text-xs text-slate-500" ] [ text (taskStateLabel item.state) ]
            ]
        , Ui.secondaryButton [ onClick (DiscoveryViewClicked item.id), testId "discovery-view" ] "View"
        ]



-- Task detail page


taskDetailPageView : LoggedInModel -> Html Msg
taskDetailPageView state =
    div [ Html.Attributes.class "space-y-6" ]
        [ Ui.secondaryButton [ onClick DetailBackClicked, testId "detail-back" ] "Back to discovery"
        , detailCard state
        , submitCard state
        , submissionsCard state
        ]


detailCard : LoggedInModel -> Html Msg
detailCard state =
    case state.detail of
        Just detail ->
            Ui.card
                [ p [ Html.Attributes.class "text-2xl font-semibold", testId "detail-title" ] [ text detail.title ]
                , div [ Html.Attributes.class "flex items-center gap-2" ] [ Ui.badge detail.state ]
                , p [ Html.Attributes.class "text-sm text-slate-700" ] [ text detail.description ]
                , Ui.label_ "Response schema"
                , Ui.codeBlock [ testId "detail-schema" ] detail.responseSchemaJson
                ]

        Nothing ->
            Ui.card [ p [ Html.Attributes.class "text-sm text-slate-500" ] [ text "Loading task…" ] ]


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
        , submissionsList state
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
    div [ Html.Attributes.class "flex items-center justify-between py-2", testId "submission-row" ]
        [ div [ Html.Attributes.class "flex items-center gap-2" ]
            [ Ui.badge (submissionStateLabel submission.state) ]
        , acceptButton state submission
        ]


acceptButton : LoggedInModel -> Submission.SubmissionResponse -> Html Msg
acceptButton _ submission =
    case submission.state of
        Submission.SubmissionStateSubmitted ->
            Ui.primaryButton [ onClick (AcceptClicked submission.id), testId "accept-submission" ] "Accept"

        _ ->
            text ""



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


restSubmitCurl : String -> String -> String
restSubmitCurl origin taskId =
    "curl -X POST "
        ++ origin
        ++ "/api/tasks/"
        ++ taskId
        ++ "/submissions \\\n  -H \"Authorization: Bearer <ACCESS_TOKEN>\" \\\n  -H \"Content-Type: application/json\" \\\n  -d '{\"response_json\":\"{}\"}'"


mcpSubmitCurl : String -> String -> String
mcpSubmitCurl origin taskId =
    "curl -X POST "
        ++ origin
        ++ "/mcp \\\n  -H \"Authorization: Bearer <AGENT_TOKEN>\" \\\n  -H \"Content-Type: application/json\" \\\n  -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"sharecrop.submit_response\",\"arguments\":{\"task_id\":\""
        ++ taskId
        ++ "\",\"response_json\":\"{}\"}}}'"


fundSuccessLabel : Ledger.TaskEscrowResponse -> String
fundSuccessLabel escrow =
    "Escrowed " ++ String.fromInt escrow.amount ++ " credits (" ++ escrowStateLabel escrow.state ++ ")."


submitSuccessLabel : Submission.SubmissionCreatedResponse -> String
submitSuccessLabel created =
    "Submission " ++ created.submission.id ++ " (" ++ submissionStateLabel created.submission.state ++ ")."


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
