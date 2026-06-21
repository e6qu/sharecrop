module Main exposing (main)

import Browser
import Html exposing (Html, button, div, form, h1, h2, input, label, main_, p, pre, span, table, tbody, td, text, th, thead, tr)
import Html.Attributes exposing (attribute, checked, class, disabled, placeholder, type_, value)
import Html.Events exposing (onClick, onInput, onSubmit)
import Http
import Json.Decode as Decode
import Json.Encode as Encode
import Sharecrop.Generated.Agent as Agent
import Sharecrop.Generated.Auth as Auth
import Sharecrop.Generated.Ledger as Ledger
import Sharecrop.Generated.Task as Task


type alias Flags =
    { origin : String }


type Session
    = LoggedOut
    | LoggedIn LoggedInModel


type alias LoggedInModel =
    { accessToken : String
    , subjectId : String
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
    }


type alias TaskDetail =
    { id : String
    , state : String
    , responseSchemaJson : String
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


taskDetailDecoder : Decode.Decoder TaskDetail
taskDetailDecoder =
    Decode.map3 TaskDetail
        (Decode.field "id" Decode.string)
        (Decode.field "state" Decode.string)
        (Decode.field "response_schema_json" Decode.string)


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
    main_ [ class "min-h-screen bg-slate-50 p-8 text-slate-950" ]
        [ div [ class "mx-auto max-w-3xl space-y-6" ]
            [ h1 [ class "text-3xl font-semibold" ] [ text "Sharecrop" ]
            , sessionView model
            ]
        ]


sessionView : Model -> Html Msg
sessionView model =
    case model.session of
        LoggedOut ->
            authView model

        LoggedIn state ->
            dashboardView model.origin state


authView : Model -> Html Msg
authView model =
    form
        [ class cardClass, onSubmit LoginClicked ]
        [ p [ class "text-slate-600" ] [ text "Sign in or create an account to view your credit ledger and set up agents." ]
        , input [ type_ "email", class fieldClass, placeholder "Email", value model.email, onInput EmailChanged, testId "email" ] []
        , input [ type_ "password", class fieldClass, placeholder "Password", value model.password, onInput PasswordChanged, testId "password" ] []
        , div [ class "flex gap-3" ]
            [ button [ type_ "submit", class primaryButtonClass, testId "login" ] [ text "Log in" ]
            , button [ type_ "button", class secondaryButtonClass, onClick RegisterClicked, testId "register" ] [ text "Register" ]
            ]
        , maybeError model.authError "auth-error"
        ]


dashboardView : String -> LoggedInModel -> Html Msg
dashboardView origin state =
    div [ class "space-y-6" ]
        [ div [ class "flex items-center justify-between" ]
            [ h2 [ class "text-xl font-medium" ] [ text "Credit account" ]
            , button [ class secondaryButtonClass, onClick LogoutClicked, testId "logout" ] [ text "Log out" ]
            ]
        , balanceView state.balance
        , ledgerView state.entries
        , fundingView state
        , tasksView origin state
        , agentsView origin state
        ]


balanceView : Maybe Int -> Html Msg
balanceView balance =
    div [ class cardClass ]
        [ p [ class labelClass ] [ text "Balance" ]
        , p [ class "text-3xl font-semibold", testId "balance" ] [ text (balanceLabel balance) ]
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
    div [ class cardClass ]
        [ h2 [ class sectionTitleClass ] [ text "Ledger" ]
        , table [ class "w-full text-left text-sm" ]
            [ thead []
                [ tr [ class "text-slate-500" ]
                    [ th [ class "pb-2" ] [ text "Entry" ]
                    , th [ class "pb-2 text-right" ] [ text "Amount" ]
                    ]
                ]
            , tbody [ testId "ledger" ] (List.map ledgerRow entries)
            ]
        ]


ledgerRow : Ledger.LedgerEntryResponse -> Html Msg
ledgerRow entry =
    tr [ class "border-t border-slate-100", testId "ledger-entry" ]
        [ td [ class "py-2" ] [ text (kindLabel entry.kind) ]
        , td [ class "py-2 text-right tabular-nums" ] [ text (String.fromInt entry.amount) ]
        ]


fundingView : LoggedInModel -> Html Msg
fundingView state =
    form [ class cardClass, onSubmit FundClicked ]
        [ h2 [ class sectionTitleClass ] [ text "Fund a task" ]
        , input [ type_ "text", class fieldClass, placeholder "Task ID", value state.fundTaskId, onInput FundTaskIdChanged, testId "fund-task-id" ] []
        , input [ type_ "number", class fieldClass, placeholder "Amount in credits", value state.fundAmount, onInput FundAmountChanged, testId "fund-amount" ] []
        , button [ type_ "submit", class primaryButtonClass, disabled (state.fundTaskId == ""), testId "fund" ] [ text "Fund task" ]
        , maybeNote state.fundMessage "fund-message"
        ]


tasksView : String -> LoggedInModel -> Html Msg
tasksView origin state =
    div [ class cardClass ]
        [ h2 [ class sectionTitleClass ] [ text "My tasks" ]
        , tasksList state.tasks
        , taskDetailView origin state
        ]


tasksList : List Task.TaskListItemResponse -> Html Msg
tasksList tasks =
    if List.isEmpty tasks then
        p [ class "text-sm text-slate-500", testId "tasks-empty" ] [ text "No tasks yet." ]

    else
        div [ class "divide-y divide-slate-100", testId "tasks" ] (List.map taskRow tasks)


taskRow : Task.TaskListItemResponse -> Html Msg
taskRow item =
    div [ class "flex items-center justify-between py-2", testId "task-row" ]
        [ div []
            [ p [ class "font-medium" ] [ text item.title ]
            , p [ class "text-xs text-slate-500" ] [ text (taskStateLabel item.state) ]
            ]
        , button [ class secondaryButtonClass, onClick (SelectTask item.id), testId "view-task" ] [ text "View" ]
        ]


taskDetailView : String -> LoggedInModel -> Html Msg
taskDetailView origin state =
    case state.selectedTask of
        Just detail ->
            div [ class "mt-4 space-y-3 rounded-md bg-slate-50 p-4", testId "task-detail" ]
                [ p [ class labelClass ] [ text ("Task " ++ detail.id) ]
                , p [ class "text-sm" ] [ text ("State: " ++ detail.state) ]
                , p [ class labelClass ] [ text "Response schema" ]
                , pre [ class codeBlockClass, testId "task-schema" ] [ text detail.responseSchemaJson ]
                , p [ class labelClass ] [ text "Submit with the REST API" ]
                , pre [ class codeBlockClass ] [ text (restSubmitCurl origin detail.id) ]
                , p [ class labelClass ] [ text "Submit with an MCP agent" ]
                , pre [ class codeBlockClass, testId "task-mcp-curl" ] [ text (mcpSubmitCurl origin detail.id) ]
                ]

        Nothing ->
            text ""


agentsView : String -> LoggedInModel -> Html Msg
agentsView origin state =
    div [ class cardClass ]
        [ h2 [ class sectionTitleClass ] [ text "Agent setup" ]
        , p [ class "text-sm text-slate-600" ] [ text "Create a scoped credential for a local MCP agent." ]
        , form [ class "mt-3 space-y-3", onSubmit CreateAgentClicked ]
            [ input [ type_ "text", class fieldClass, placeholder "Agent label", value state.agentLabel, onInput AgentLabelChanged, testId "agent-label" ] []
            , div [ class "space-y-1" ] (List.map (scopeCheckbox state.agentScopes) allScopes)
            , button [ type_ "submit", class primaryButtonClass, testId "create-agent" ] [ text "Create credential" ]
            , maybeNote state.agentMessage "agent-message"
            ]
        , newCredentialView origin state.newCredential
        , credentialsList state.credentials
        ]


scopeCheckbox : List Agent.AgentScope -> Agent.AgentScope -> Html Msg
scopeCheckbox selected scope =
    label [ class "flex items-center gap-2 text-sm" ]
        [ input
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
            div [ class "mt-4 space-y-3 rounded-md bg-slate-50 p-4" ]
                [ p [ class labelClass ] [ text "New agent token (shown once)" ]
                , pre [ class codeBlockClass, testId "agent-secret" ] [ text credential.secret ]
                , p [ class labelClass ] [ text "MCP client configuration" ]
                , pre [ class codeBlockClass, testId "mcp-config" ] [ text (mcpConfig origin credential.secret) ]
                ]

        Nothing ->
            text ""


credentialsList : List Agent.AgentCredentialResponse -> Html Msg
credentialsList credentials =
    if List.isEmpty credentials then
        text ""

    else
        div [ class "mt-4 divide-y divide-slate-100", testId "credentials" ] (List.map credentialRow credentials)


credentialRow : Agent.AgentCredentialResponse -> Html Msg
credentialRow credential =
    div [ class "flex items-center justify-between py-2", testId "credential-row" ]
        [ div []
            [ p [ class "font-medium" ] [ text credential.label ]
            , p [ class "text-xs text-slate-500" ] [ text (credentialStateLabel credential.state ++ " · " ++ String.join ", " (List.map scopeTag credential.scopes)) ]
            ]
        , revokeButton credential
        ]


revokeButton : Agent.AgentCredentialResponse -> Html Msg
revokeButton credential =
    case credential.state of
        Agent.AgentCredentialStateActive ->
            button [ class secondaryButtonClass, onClick (RevokeClicked credential.id), testId "revoke-credential" ] [ text "Revoke" ]

        Agent.AgentCredentialStateRevoked ->
            span [ class "text-xs text-slate-400" ] [ text "revoked" ]


maybeError : Maybe String -> String -> Html Msg
maybeError message identifier =
    case message of
        Just value ->
            p [ class "text-sm text-red-600", testId identifier ] [ text value ]

        Nothing ->
            text ""


maybeNote : Maybe String -> String -> Html Msg
maybeNote message identifier =
    case message of
        Just value ->
            p [ class "text-sm text-slate-600", testId identifier ] [ text value ]

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


testId : String -> Html.Attribute Msg
testId value =
    attribute "data-testid" value


cardClass : String
cardClass =
    "space-y-4 rounded-lg border border-slate-200 bg-white p-6 shadow-sm"


fieldClass : String
fieldClass =
    "w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-slate-500 focus:outline-none"


labelClass : String
labelClass =
    "text-sm uppercase tracking-wide text-slate-500"


sectionTitleClass : String
sectionTitleClass =
    "text-lg font-medium"


codeBlockClass : String
codeBlockClass =
    "overflow-x-auto rounded-md bg-slate-900 p-3 text-xs text-slate-100"


primaryButtonClass : String
primaryButtonClass =
    "rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-700 disabled:opacity-50"


secondaryButtonClass : String
secondaryButtonClass =
    "rounded-md border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100"
