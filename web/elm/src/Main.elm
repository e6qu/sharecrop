module Main exposing (main)

import Browser
import Html exposing (Html, button, div, form, h1, h2, input, main_, p, span, table, tbody, td, text, th, thead, tr)
import Html.Attributes exposing (attribute, class, disabled, placeholder, type_, value)
import Html.Events exposing (onClick, onInput, onSubmit)
import Http
import Json.Decode as Decode
import Json.Encode as Encode
import Sharecrop.Generated.Auth as Auth
import Sharecrop.Generated.Ledger as Ledger


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
    }


type alias Model =
    { email : String
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
    | FundTaskIdChanged String
    | FundAmountChanged String
    | FundClicked
    | FundReceived (Result Http.Error Ledger.TaskEscrowResponse)
    | LogoutClicked


main : Program () Model Msg
main =
    Browser.element
        { init = \_ -> ( initialModel, Cmd.none )
        , update = update
        , subscriptions = \_ -> Sub.none
        , view = view
        }


initialModel : Model
initialModel =
    { email = ""
    , password = ""
    , authError = Nothing
    , session = LoggedOut
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
            ( { model
                | password = ""
                , authError = Nothing
                , session =
                    LoggedIn
                        { accessToken = response.accessToken
                        , subjectId = response.subjectID
                        , balance = Nothing
                        , entries = []
                        , fundTaskId = ""
                        , fundAmount = ""
                        , fundMessage = Nothing
                        }
              }
            , Cmd.batch [ fetchBalance response.accessToken, fetchLedger response.accessToken ]
            )

        AuthReceived (Err error) ->
            ( { model | authError = Just (httpErrorLabel error) }, Cmd.none )

        BalanceReceived result ->
            ( updateLoggedIn model (\state -> { state | balance = balanceFromResult result }), Cmd.none )

        LedgerReceived result ->
            ( updateLoggedIn model (\state -> { state | entries = entriesFromResult result }), Cmd.none )

        FundTaskIdChanged value ->
            ( updateLoggedIn model (\state -> { state | fundTaskId = value }), Cmd.none )

        FundAmountChanged value ->
            ( updateLoggedIn model (\state -> { state | fundAmount = value }), Cmd.none )

        FundClicked ->
            case model.session of
                LoggedIn state ->
                    fundTaskCommand model state

                LoggedOut ->
                    ( model, Cmd.none )

        FundReceived (Ok escrow) ->
            ( updateLoggedIn model (\state -> { state | fundMessage = Just (fundSuccessLabel escrow) })
            , refreshAfterFund model
            )

        FundReceived (Err error) ->
            ( updateLoggedIn model (\state -> { state | fundMessage = Just (httpErrorLabel error) }), Cmd.none )

        LogoutClicked ->
            ( initialModel, Cmd.none )


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


fundTaskCommand : Model -> LoggedInModel -> ( Model, Cmd Msg )
fundTaskCommand model state =
    case String.toInt state.fundAmount of
        Just amount ->
            ( updateLoggedIn model (\current -> { current | fundMessage = Nothing })
            , postFunding state.accessToken state.fundTaskId amount
            )

        Nothing ->
            ( updateLoggedIn model (\current -> { current | fundMessage = Just "Amount must be a whole number of credits." })
            , Cmd.none
            )


refreshAfterFund : Model -> Cmd Msg
refreshAfterFund model =
    case model.session of
        LoggedIn state ->
            Cmd.batch [ fetchBalance state.accessToken, fetchLedger state.accessToken ]

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


postFunding : String -> String -> Int -> Cmd Msg
postFunding token taskId amount =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/funding")
        (Http.jsonBody (fundingRequestBody taskId amount))
        (Http.expectJson FundReceived Ledger.taskEscrowResponseDecoder)


fundingRequestBody : String -> Int -> Encode.Value
fundingRequestBody taskId amount =
    Encode.object
        [ ( "amount", Encode.int amount )
        , ( "idempotency_key", Encode.string ("fund:" ++ taskId) )
        ]


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
            dashboardView state


authView : Model -> Html Msg
authView model =
    form
        [ class "space-y-4 rounded-lg border border-slate-200 bg-white p-6 shadow-sm"
        , onSubmit LoginClicked
        ]
        [ p [ class "text-slate-600" ] [ text "Sign in or create an account to view your credit ledger." ]
        , input
            [ type_ "email"
            , class fieldClass
            , placeholder "Email"
            , value model.email
            , onInput EmailChanged
            , testId "email"
            ]
            []
        , input
            [ type_ "password"
            , class fieldClass
            , placeholder "Password"
            , value model.password
            , onInput PasswordChanged
            , testId "password"
            ]
            []
        , div [ class "flex gap-3" ]
            [ button
                [ type_ "submit"
                , class primaryButtonClass
                , testId "login"
                ]
                [ text "Log in" ]
            , button
                [ type_ "button"
                , class secondaryButtonClass
                , onClick RegisterClicked
                , testId "register"
                ]
                [ text "Register" ]
            ]
        , authErrorView model.authError
        ]


authErrorView : Maybe String -> Html Msg
authErrorView authError =
    case authError of
        Just message ->
            p [ class "text-sm text-red-600", testId "auth-error" ] [ text message ]

        Nothing ->
            text ""


dashboardView : LoggedInModel -> Html Msg
dashboardView state =
    div [ class "space-y-6" ]
        [ div [ class "flex items-center justify-between" ]
            [ h2 [ class "text-xl font-medium" ] [ text "Credit account" ]
            , button [ class secondaryButtonClass, onClick LogoutClicked, testId "logout" ] [ text "Log out" ]
            ]
        , balanceView state.balance
        , ledgerView state.entries
        , fundingView state
        ]


balanceView : Maybe Int -> Html Msg
balanceView balance =
    div [ class "rounded-lg border border-slate-200 bg-white p-6 shadow-sm" ]
        [ p [ class "text-sm uppercase tracking-wide text-slate-500" ] [ text "Balance" ]
        , p [ class "text-3xl font-semibold", testId "balance" ]
            [ text (balanceLabel balance) ]
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
    div [ class "rounded-lg border border-slate-200 bg-white p-6 shadow-sm" ]
        [ h2 [ class "mb-3 text-lg font-medium" ] [ text "Ledger" ]
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
    form
        [ class "space-y-4 rounded-lg border border-slate-200 bg-white p-6 shadow-sm"
        , onSubmit FundClicked
        ]
        [ h2 [ class "text-lg font-medium" ] [ text "Fund a task" ]
        , input
            [ type_ "text"
            , class fieldClass
            , placeholder "Task ID"
            , value state.fundTaskId
            , onInput FundTaskIdChanged
            , testId "fund-task-id"
            ]
            []
        , input
            [ type_ "number"
            , class fieldClass
            , placeholder "Amount in credits"
            , value state.fundAmount
            , onInput FundAmountChanged
            , testId "fund-amount"
            ]
            []
        , button
            [ type_ "submit"
            , class primaryButtonClass
            , disabled (state.fundTaskId == "")
            , testId "fund"
            ]
            [ text "Fund task" ]
        , fundMessageView state.fundMessage
        ]


fundMessageView : Maybe String -> Html Msg
fundMessageView fundMessage =
    case fundMessage of
        Just message ->
            p [ class "text-sm text-slate-600", testId "fund-message" ] [ text message ]

        Nothing ->
            text ""


fundSuccessLabel : Ledger.TaskEscrowResponse -> String
fundSuccessLabel escrow =
    "Escrowed " ++ String.fromInt escrow.amount ++ " credits (" ++ escrowStateLabel escrow.state ++ ")."


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


fieldClass : String
fieldClass =
    "w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-slate-500 focus:outline-none"


primaryButtonClass : String
primaryButtonClass =
    "rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-700 disabled:opacity-50"


secondaryButtonClass : String
secondaryButtonClass =
    "rounded-md border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100"
