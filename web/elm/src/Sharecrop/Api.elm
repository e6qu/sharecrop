module Sharecrop.Api exposing (..)

import Http
import Json.Decode as Decode
import Json.Encode as Encode
import Sharecrop.Generated.Admin as Admin
import Sharecrop.Generated.Agent as Agent
import Sharecrop.Generated.Auth as Auth
import Sharecrop.Generated.Collectible as Collectible
import Sharecrop.Generated.Events as Events
import Sharecrop.Generated.Ledger as Ledger
import Sharecrop.Generated.Moderation as Moderation
import Sharecrop.Generated.Notification as Notification
import Sharecrop.Generated.Organization as Organization
import Sharecrop.Generated.Privacy as Privacy
import Sharecrop.Generated.SavedQueueViews as SavedQueueViews
import Sharecrop.Generated.Submission as Submission
import Sharecrop.Generated.Task as Task
import Sharecrop.Generated.TaskSeries as TaskSeries
import Sharecrop.Generated.Team as Team
import Sharecrop.Labels exposing (assigneeScopeTag, collectibleKindTag, collectiblePolicyTag, domainEventKindTag, httpErrorLabel, participationUsesReservation)
import Sharecrop.ResponseSchema as ResponseSchema
import Sharecrop.Types exposing (..)
import Task as ElmTask
import Time
import Url


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


balanceFromResult : Result Http.Error Ledger.BalanceResponse -> Maybe Wallet
balanceFromResult result =
    case result of
        Ok response ->
            Just { spendable = response.spendableCredits, allocated = response.allocatedCredits }

        Err _ ->
            Nothing


{-| The one place a fetched list's outcome is mapped into model state: a
success carries its rows, a failure carries the user-facing error text so
views can render a visible load-error state instead of a fake "nothing
here yet" empty state (`unauthenticated` never reaches this — it is routed
to SessionEnded by the expect helpers below).
-}
loadedFromResult : (response -> List a) -> Result Http.Error response -> Loaded a
loadedFromResult toItems result =
    case result of
        Ok response ->
            { items = toItems response, failure = Nothing }

        Err error ->
            { items = [], failure = Just (httpErrorLabel error) }


boolQuery : Bool -> String
boolQuery value =
    if value then
        "true"

    else
        "false"


selectorPageSize : Int
selectorPageSize =
    pageSize


selectorQuery : String -> Int -> String -> String
selectorQuery queryText offset base =
    let
        clean =
            String.trim queryText

        queryPart =
            if clean == "" then
                ""

            else
                "&query=" ++ Url.percentEncode clean
    in
    base ++ "?limit=" ++ String.fromInt selectorPageSize ++ "&offset=" ++ String.fromInt offset ++ queryPart


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
            if amount <= 0 then
                ( updateLoggedIn model (\current -> { current | fundMessage = Just (FailureNote "Amount must be a positive number of credits.") }), Cmd.none )

            else
                ( updateLoggedIn model (\current -> { current | fundMessage = Nothing }), postFunding state.accessToken state.fundTaskId amount state.fundOrganizationId state.fundNonce )

        Nothing ->
            ( updateLoggedIn model (\current -> { current | fundMessage = Just (FailureNote "Amount must be a whole number of credits.") }), Cmd.none )


createAgentCommand : Model -> LoggedInModel -> ( Model, Cmd Msg )
createAgentCommand model state =
    if List.isEmpty state.agentScopes then
        ( updateLoggedIn model (\current -> { current | agentMessage = Just (FailureNote "Select at least one scope.") }), Cmd.none )

    else if not (expiresHoursIsValid state.agentExpiresHours) then
        ( updateLoggedIn model (\current -> { current | agentMessage = Just (FailureNote "Expires in (hours) must be a positive whole number, or blank for never.") }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | agentMessage = Nothing, newCredential = Nothing }), ElmTask.perform AgentExpiresAtResolved Time.now )


createOrgCredentialCommand : Model -> LoggedInModel -> ( Model, Cmd Msg )
createOrgCredentialCommand model state =
    if state.activeOrgId == "" then
        ( updateLoggedIn model (\current -> { current | orgCredentialMessage = Just (FailureNote "Open an organization first.") }), Cmd.none )

    else if List.isEmpty state.orgCredentialScopes then
        ( updateLoggedIn model (\current -> { current | orgCredentialMessage = Just (FailureNote "Select at least one scope.") }), Cmd.none )

    else if not (expiresHoursIsValid state.orgCredentialExpiresHours) then
        ( updateLoggedIn model (\current -> { current | orgCredentialMessage = Just (FailureNote "Expires in (hours) must be a positive whole number, or blank for never.") }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | orgCredentialMessage = Nothing, newOrgCredential = Nothing }), ElmTask.perform OrgCredentialExpiresAtResolved Time.now )


{-| True for a blank draft (never expires) or a positive whole number of
hours. Kept separate from `expiresAtFromHours` so a typo like "0", "-1", or
"12h" is rejected with a message rather than silently minting a
never-expiring credential — the worst-case outcome for a least-privilege
input error.
-}
expiresHoursIsValid : String -> Bool
expiresHoursIsValid raw =
    case String.toInt (String.trim raw) of
        Just hours ->
            hours > 0

        Nothing ->
            String.trim raw == ""


createTaskCommand : Model -> LoggedInModel -> ( Model, Cmd Msg )
createTaskCommand model state =
    let
        titleMissing =
            String.isEmpty (String.trim state.createTitle)

        descriptionMissing =
            String.isEmpty (String.trim state.createDescription)

        amountMissing =
            rewardAmountMissing state.createRewardKind state.createRewardAmount

        collectibleMissing =
            rewardCollectibleMissing state.createRewardKind state.createRewardCollectibleIds
    in
    if titleMissing || descriptionMissing || amountMissing then
        ( updateLoggedIn model
            (\current ->
                { current
                    | createTitleInvalid = titleMissing
                    , createDescriptionInvalid = descriptionMissing
                    , createRewardAmountInvalid = amountMissing
                    , createMessage = Just (FailureNote "Fill in the required fields below.")
                }
            )
        , Cmd.none
        )

    else if collectibleMissing then
        ( updateLoggedIn model (\current -> { current | createMessage = Just (FailureNote "Select at least one collectible for this reward.") }), Cmd.none )

    else if participationUsesReservation state.createParticipationPolicy && (reservationHoursValue state.createReservationHours < 1 || reservationHoursValue state.createReservationHours > 720) then
        ( updateLoggedIn model (\current -> { current | createMessage = Just (FailureNote "Reservation expiry must be between 1 and 720 hours.") }), Cmd.none )

    else if not (expiryDraftIsValid state.createExpiresAt) then
        ( updateLoggedIn model (\current -> { current | createMessage = Just (FailureNote "Pick a complete expiry date and time, or clear the field for no expiration.") }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | createTitleInvalid = False, createDescriptionInvalid = False, createRewardAmountInvalid = False, createMessage = Nothing })
        , postCreateTask state
        )


{-| A credit or bundle reward needs a positive whole credit amount; without
one the request would either be rejected by the server or - worse - silently
turn into a different reward kind than the user picked, so block the submit
instead.
-}
rewardAmountMissing : String -> String -> Bool
rewardAmountMissing kind rawAmount =
    (kind == "credit" || kind == "bundle")
        && (Maybe.withDefault 0 (String.toInt (String.trim rawAmount)) < 1)


{-| A collectible or bundle reward needs at least one collectible selected;
the server rejects an empty list, so block the submit with a clear message
instead of letting the create round-trip fail.
-}
rewardCollectibleMissing : String -> List String -> Bool
rewardCollectibleMissing kind collectibleIds =
    (kind == "collectible" || kind == "bundle")
        && List.isEmpty collectibleIds


submitCommand : Model -> LoggedInModel -> ( Model, Cmd Msg )
submitCommand model state =
    case state.page of
        TaskDetailPage taskId ->
            case schemaFormFields state of
                Just fields ->
                    -- Schema-driven form: build the response JSON from the
                    -- typed per-field inputs and validate before sending.
                    case ResponseSchema.buildSubmission fields state.submitFieldValues of
                        Ok encoded ->
                            ( updateLoggedIn model (\current -> { current | submitMessage = Nothing })
                            , postSubmission state.accessToken taskId encoded state.submitAttachments
                            )

                        Err message ->
                            ( updateLoggedIn model (\current -> { current | submitMessage = Just (FailureNote message) }), Cmd.none )

                Nothing ->
                    submitRawCommand model state taskId

        _ ->
            ( model, Cmd.none )


-- schemaFormFields returns the typed form fields when the current task's
-- response schema is a top-level object and the worker has not switched to
-- the raw JSON editor.
schemaFormFields : LoggedInModel -> Maybe (List ResponseSchema.FormField)
schemaFormFields state =
    if state.submitRawMode then
        Nothing

    else
        state.detail
            |> Maybe.andThen (\detail -> ResponseSchema.parse detail.responseSchemaJson)
            |> Maybe.andThen ResponseSchema.formFields


{-| Best-effort JSON for the raw editor, built from the structured fields the
worker has filled in so far. Empty when there is no object schema or nothing
has been filled.
-}
seedRawSubmitInput : LoggedInModel -> String
seedRawSubmitInput state =
    case schemaFormFields state of
        Just fields ->
            ResponseSchema.buildPartial fields state.submitFieldValues

        Nothing ->
            ""


submitRawCommand : Model -> LoggedInModel -> String -> ( Model, Cmd Msg )
submitRawCommand model state taskId =
    let
        trimmed =
            String.trim state.submitInput
    in
    -- Guard obviously-invalid input before hitting the server: the
    -- response payload must be non-empty and parse as JSON.
    if trimmed == "" then
        ( updateLoggedIn model (\current -> { current | submitMessage = Just (FailureNote "Enter a response first.") }), Cmd.none )

    else
        case Decode.decodeString Decode.value trimmed of
            Ok _ ->
                ( updateLoggedIn model (\current -> { current | submitMessage = Nothing })
                , postSubmission state.accessToken taskId trimmed state.submitAttachments
                )

            Err _ ->
                ( updateLoggedIn model (\current -> { current | submitMessage = Just (FailureNote "Response must be valid JSON.") }), Cmd.none )


acceptCommand : Model -> LoggedInModel -> String -> ( Model, Cmd Msg )
acceptCommand model state submissionId =
    case state.page of
        TaskDetailPage taskId ->
            ( updateLoggedIn model (\current -> { current | reviewMessage = Nothing }), postAccept state.accessToken taskId submissionId state.reviewPartialCredit state.reviewTip state.reviewTipCollectibleId )

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
        ( updateLoggedIn model (\current -> { current | collectibleMessage = Just (FailureNote "Name is required.") }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | collectibleMessage = Nothing })
        , postCollectible state.accessToken state.collectibleName state.collectibleKind state.collectiblePolicy
        )


awardCommand : Model -> LoggedInModel -> String -> ( Model, Cmd Msg )
awardCommand model state collectibleId =
    if String.isEmpty (String.trim state.awardTaskId) then
        ( updateLoggedIn model (\current -> { current | awardMessage = Just (FailureNote "Task ID is required.") }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | awardMessage = Nothing })
        , postCollectibleReward state.accessToken state.awardTaskId collectibleId
        )


loadAfterAuth : String -> Cmd Msg
loadAfterAuth token =
    Cmd.batch [ fetchBalance token, fetchLedger token 0, fetchTasks token [] "" "newest" 0, fetchCredentials token, fetchCollectibles token, fetchOrganizations token, fetchUserDirectory token, fetchStandaloneTeams token, fetchSavedQueueViews token, fetchUnreadCount token ]


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
            Cmd.batch [ fetchBalance state.accessToken, fetchLedger state.accessToken state.ledgerOffset ]

        LoggedOut ->
            Cmd.none


{-| Like refreshLedger, but also re-fetches the currently-viewed task's own
detail record. Funding/awarding a collectible to a task from its own detail
page changes fields (reward kind, owner-controls buttons) that live on that
record, not just the ledger - without this, those buttons stay stale until
the user navigates away and back. Harmless no-op refetch if fundTaskId
happens not to be the task actually being viewed (e.g. picked on the
standalone Funding page instead).
-}
refreshLedgerAndTaskDetail : Model -> Cmd Msg
refreshLedgerAndTaskDetail model =
    case model.session of
        LoggedIn state ->
            Cmd.batch [ fetchBalance state.accessToken, fetchLedger state.accessToken state.ledgerOffset, fetchPublicTaskDetail state.accessToken state.fundTaskId ]

        LoggedOut ->
            Cmd.none


refreshTasksAndLedger : Model -> Cmd Msg
refreshTasksAndLedger model =
    case model.session of
        LoggedIn state ->
            Cmd.batch [ fetchTasks state.accessToken state.taskStateFilter state.taskListTypeFilter state.taskListSort state.taskListOffset, fetchBalance state.accessToken, fetchLedger state.accessToken state.ledgerOffset ]

        LoggedOut ->
            Cmd.none


refreshBalanceAndLedger : Model -> Cmd Msg
refreshBalanceAndLedger model =
    case model.session of
        LoggedIn state ->
            Cmd.batch [ fetchBalance state.accessToken, fetchLedger state.accessToken state.ledgerOffset ]

        LoggedOut ->
            Cmd.none


refreshTasksAndDiscovery : Model -> Cmd Msg
refreshTasksAndDiscovery model =
    case model.session of
        LoggedIn state ->
            Cmd.batch [ fetchTasks state.accessToken state.taskStateFilter state.taskListTypeFilter state.taskListSort state.taskListOffset, fetchDiscovery state.accessToken state.discoveryIncludeReserved state.discoveryFundedOnly state.discoveryOffset ]

        LoggedOut ->
            Cmd.none


routeLoadCmd : String -> String -> String -> Page -> Cmd Msg
routeLoadCmd token subjectId activityCursor page =
    case page of
        OverviewPage ->
            -- Tasks are fetched too: the Needs-review card is derived from
            -- the My-tasks data and must reflect reviews done since the list
            -- was last loaded. The activity feed resumes from the cursor the
            -- session already holds, so a revisit continues where it left
            -- off instead of re-reading the stream from its start.
            Cmd.batch [ fetchBalance token, fetchLedger token 0, fetchEvents token activityCursor, fetchTasks token [] "" "newest" 0 ]

        TasksPage ->
            -- The Tasks hub embeds My tasks, Discover public tasks, My
            -- submissions, and Series all on one page, so entering it loads
            -- data for all four sections at once.
            Cmd.batch
                [ fetchTasks token [] "" "newest" 0
                , fetchDiscovery token False False 0
                , fetchUserSubmissionsPage token subjectId 0
                , fetchSeriesList token
                ]

        CreateTaskPage ->
            Cmd.batch [ fetchOrganizations token, fetchCollectibles token, fetchUserDirectory token, fetchStandaloneTeams token ]

        TaskDetailPage taskId ->
            fetchDetailCommands token subjectId taskId

        FundingPage ->
            Cmd.batch [ fetchTasks token [] "" "newest" 0, fetchOrganizations token ]

        AgentsPage ->
            Cmd.batch [ fetchCredentials token, fetchWebhookSubscriptions token ]

        CollectiblesPage ->
            Cmd.batch [ fetchCollectibles token, fetchCollectibleCatalog token, fetchTasks token [] "" "newest" 0, fetchOrganizations token ]

        OrganizationsPage ->
            fetchOrganizations token

        OrganizationDetailPage organizationId ->
            Cmd.batch [ fetchOrganizations token, loadOrganization token organizationId, fetchOrganizationCollectibles token organizationId ]

        UserDetailPage userId ->
            if userId == subjectId then
                -- Your own profile page carries the account-settings card,
                -- whose Privacy section lists your own privacy requests, and
                -- the identity card showing your display name and email.
                Cmd.batch [ fetchUserProfile token userId, fetchMyPrivacyRequests token, fetchAccountProfile token ]

            else
                fetchUserProfile token userId

        UserWorkPage userId ->
            authorizedRequest "GET" token ("/api/users/" ++ userId ++ "/work") Http.emptyBody (expectJsonWithServerError UserWorkReceived Task.tasksResponseDecoder)

        UserSubmissionsPage userId ->
            fetchUserSubmissionsPage token userId 0

        CollectibleDetailPage _ ->
            fetchCollectibles token

        SeriesDetailPage seriesId ->
            fetchSeriesDetail token seriesId

        TeamDetailPage teamId ->
            Cmd.batch
                [ authorizedRequest "GET" token ("/api/teams/" ++ teamId) Http.emptyBody (expectJsonWithServerError TeamDetailReceived Team.teamDetailResponseDecoder)
                , fetchTeamWork token teamId "" "" "newest" 0
                , fetchTeamCollectibles token teamId
                ]

        AdminPage ->
            Cmd.batch
                [ authorizedRequest "GET" token "/api/admin/operations" Http.emptyBody (expectJsonWithServerError OperationsReceived Admin.operationsResponseDecoder)
                , fetchAuditEvents token "" "" "" 0
                , fetchPlatformAdmins token 0
                , fetchUserDirectory token
                , fetchAdminModerationReports token "open" 0
                , fetchAdminPrivacyRequests token 0
                ]

        InboxPage ->
            fetchNotifications token False 0

        NotFoundPage ->
            Cmd.none


fetchOrganizationCollectibles : String -> String -> Cmd Msg
fetchOrganizationCollectibles token orgId =
    authorizedRequest "GET" token ("/api/organizations/" ++ orgId ++ "/collectibles") Http.emptyBody (expectJsonWithServerError OrgCollectiblesReceived Collectible.collectiblesResponseDecoder)


fetchTeamCollectibles : String -> String -> Cmd Msg
fetchTeamCollectibles token teamId =
    authorizedRequest "GET" token ("/api/teams/" ++ teamId ++ "/collectibles") Http.emptyBody (expectJsonWithServerError TeamCollectiblesReceived Collectible.collectiblesResponseDecoder)


fetchUserProfile : String -> String -> Cmd Msg
fetchUserProfile token userId =
    authorizedRequest "GET" token ("/api/users/" ++ userId) Http.emptyBody (expectJsonWithServerError UserProfileReceived Task.userProfileResponseDecoder)


postAddTeamMember : String -> String -> String -> Cmd Msg
postAddTeamMember token teamId email =
    authorizedRequest "POST"
        token
        ("/api/teams/" ++ teamId ++ "/members")
        (Http.jsonBody (Encode.object [ ( "email", Encode.string email ) ]))
        (expectJsonWithServerError AddTeamMemberReceived Team.teamDetailResponseDecoder)


refreshCredentials : Model -> Cmd Msg
refreshCredentials model =
    case model.session of
        LoggedIn state ->
            fetchCredentials state.accessToken

        LoggedOut ->
            Cmd.none


refreshOrgCredentials : Model -> Cmd Msg
refreshOrgCredentials model =
    case model.session of
        LoggedIn state ->
            fetchOrgCredentials state.accessToken state.activeOrgId

        LoggedOut ->
            Cmd.none


refreshDetailSubmissions : Model -> Cmd Msg
refreshDetailSubmissions model =
    case model.session of
        LoggedIn state ->
            case state.page of
                TaskDetailPage taskId ->
                    Cmd.batch [ fetchSubmissions state.accessToken taskId, fetchUserSubmissions state.accessToken state.subjectId ]

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


fetchDetailCommands : String -> String -> String -> Cmd Msg
fetchDetailCommands token subjectId taskId =
    Cmd.batch [ fetchPublicTaskDetail token taskId, fetchSubmissions token taskId, fetchReservations token taskId, fetchTaskComments token taskId, fetchUserSubmissions token subjectId, fetchOrganizations token ]


fetchUserSubmissions : String -> String -> Cmd Msg
fetchUserSubmissions token userId =
    fetchUserSubmissionsPage token userId 0


fetchUserSubmissionsPage : String -> String -> Int -> Cmd Msg
fetchUserSubmissionsPage token userId offset =
    authorizedRequest "GET" token ("/api/users/" ++ userId ++ "/submissions?limit=" ++ String.fromInt selectorPageSize ++ "&offset=" ++ String.fromInt offset) Http.emptyBody (expectJsonWithServerError UserSubmissionsReceived Submission.submissionsResponseDecoder)


fetchTaskComments : String -> String -> Cmd Msg
fetchTaskComments token taskId =
    authorizedRequest "GET"
        token
        ("/api/tasks/" ++ taskId ++ "/comments")
        Http.emptyBody
        (expectJsonWithServerError TaskCommentsReceived (Decode.field "comments" (Decode.list Task.taskCommentResponseDecoder)))


postTaskComment : String -> String -> String -> Cmd Msg
postTaskComment token taskId body =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/comments")
        (Http.jsonBody (Encode.object [ ( "body", Encode.string body ) ]))
        (expectJsonWithServerError TaskCommentReceived Task.taskCommentResponseDecoder)


fetchSubmissionComments : String -> String -> Cmd Msg
fetchSubmissionComments token submissionId =
    authorizedRequest "GET"
        token
        ("/api/submissions/" ++ submissionId ++ "/comments")
        Http.emptyBody
        (expectJsonWithServerError SubmissionCommentsReceived Submission.submissionCommentsResponseDecoder)


addSubmissionComment : String -> String -> String -> Cmd Msg
addSubmissionComment token submissionId body =
    authorizedRequest "POST"
        token
        ("/api/submissions/" ++ submissionId ++ "/comments")
        (Http.jsonBody (Encode.object [ ( "body", Encode.string body ) ]))
        (expectJsonWithServerError SubmissionCommentAdded Submission.submissionCommentResponseDecoder)


refreshAfterAccept : Model -> Cmd Msg
refreshAfterAccept model =
    case model.session of
        LoggedIn state ->
            case state.page of
                TaskDetailPage taskId ->
                    Cmd.batch
                        [ fetchSubmissions state.accessToken taskId
                        , fetchBalance state.accessToken
                        , fetchPublicTaskDetail state.accessToken taskId
                        , fetchReservations state.accessToken taskId
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
        , expect = expectAuthJson AuthReceived Auth.authResponseDecoder
        }


postGuest : Cmd Msg
postGuest =
    Http.post
        { url = "/api/auth/guest"
        , body = Http.emptyBody
        , expect = expectAuthJson AuthReceived Auth.authResponseDecoder
        }


requestPasswordReset : Model -> Cmd Msg
requestPasswordReset model =
    Http.post
        { url = "/api/auth/password-reset/request"
        , body = Http.jsonBody (Encode.object [ ( "email", Encode.string model.resetEmail ) ])
        , expect = expectAuthJson PasswordResetRequested tokenDecoder
        }


confirmPasswordReset : Model -> Cmd Msg
confirmPasswordReset model =
    Http.post
        { url = "/api/auth/password-reset/confirm"
        , body = Http.jsonBody (Encode.object [ ( "token", Encode.string model.resetToken ), ( "password", Encode.string model.resetPassword ) ])
        , expect = expectAuthWhatever PasswordResetConfirmed
        }


tokenDecoder : Decode.Decoder String
tokenDecoder =
    Decode.oneOf
        [ Decode.field "token" Decode.string
        , Decode.field "status" Decode.string |> Decode.map (\_ -> "")
        ]


postRefresh : Cmd Msg
postRefresh =
    Http.post
        { url = "/api/auth/refresh"
        , body = Http.emptyBody
        , expect = expectAuthJson RefreshReceived Auth.authResponseDecoder
        }


-- postSessionRefresh rotates the token mid-session without rebuilding any
-- page state (unlike the boot-time postRefresh, whose handler resets the
-- whole logged-in model). Access tokens expire after 15 minutes; without
-- this, every request in a tab left open longer than that fails.
postSessionRefresh : Cmd Msg
postSessionRefresh =
    Http.post
        { url = "/api/auth/refresh"
        , body = Http.emptyBody
        , expect = expectAuthJson SessionRefreshed Auth.authResponseDecoder
        }


postLogout : Cmd Msg
postLogout =
    Http.post
        { url = "/api/auth/logout"
        , body = Http.emptyBody
        , expect = expectAuthJson LogoutReceived logoutURLDecoder
        }


logoutURLDecoder : Decode.Decoder String
logoutURLDecoder =
    Decode.field "logout_url" Decode.string


authRequestBody : Model -> Encode.Value
authRequestBody model =
    -- display_name matters only to /register (optional there; blank means
    -- "derive from the email"); /login ignores the extra field.
    Encode.object
        [ ( "email", Encode.string model.email )
        , ( "password", Encode.string model.password )
        , ( "display_name", Encode.string (String.trim model.registerName) )
        ]


fetchBalance : String -> Cmd Msg
fetchBalance token =
    authorizedRequest "GET" token "/api/credits/balance" Http.emptyBody (expectJsonWithServerError BalanceReceived Ledger.balanceResponseDecoder)


fetchLedger : String -> Int -> Cmd Msg
fetchLedger token offset =
    authorizedRequest "GET" token ("/api/credits/ledger?limit=" ++ String.fromInt selectorPageSize ++ "&offset=" ++ String.fromInt offset) Http.emptyBody (expectJsonWithServerError LedgerReceived Ledger.ledgerResponseDecoder)


fetchTasks : String -> List String -> String -> String -> Int -> Cmd Msg
fetchTasks token stateFilter typeFilter sortOrder offset =
    let
        pageQuery =
            "limit=" ++ String.fromInt selectorPageSize ++ "&offset=" ++ String.fromInt offset

        stateQuery =
            stateFilter
                |> List.map (\state -> "&state=" ++ Url.percentEncode state)
                |> String.concat

        typeQuery =
            if typeFilter == "" then
                ""

            else
                "&task_type=" ++ Url.percentEncode typeFilter

        sortQuery =
            "&sort=" ++ Url.percentEncode sortOrder
    in
    authorizedRequest "GET" token ("/api/tasks?scope=user&" ++ pageQuery ++ stateQuery ++ typeQuery ++ sortQuery) Http.emptyBody (expectJsonWithServerError TasksReceived Task.tasksResponseDecoder)


taskSearchParams : String -> String -> String -> Int -> String
taskSearchParams queryText typeFilter sortOrder offset =
    let
        pageQuery =
            "limit=" ++ String.fromInt selectorPageSize ++ "&offset=" ++ String.fromInt offset

        trimmed =
            String.trim queryText

        queryPart =
            if trimmed == "" then
                ""

            else
                "&query=" ++ Url.percentEncode trimmed

        typePart =
            if typeFilter == "" then
                ""

            else
                "&task_type=" ++ Url.percentEncode typeFilter
    in
    pageQuery ++ queryPart ++ typePart ++ "&sort=" ++ Url.percentEncode sortOrder


fetchTeamWork : String -> String -> String -> String -> String -> Int -> Cmd Msg
fetchTeamWork token teamId queryText typeFilter sortOrder offset =
    authorizedRequest "GET" token ("/api/teams/" ++ teamId ++ "/work?" ++ taskSearchParams queryText typeFilter sortOrder offset) Http.emptyBody (expectJsonWithServerError TeamWorkReceived Task.tasksResponseDecoder)


fetchSavedQueueViews : String -> Cmd Msg
fetchSavedQueueViews token =
    authorizedRequest "GET" token "/api/saved-queue-views" Http.emptyBody (expectJsonWithServerError SavedQueueViewsReceived SavedQueueViews.savedQueueViewsResponseDecoder)


saveSavedQueueView : String -> String -> QueueView -> Cmd Msg
saveSavedQueueView token scope view =
    authorizedRequest "POST"
        token
        "/api/saved-queue-views"
        (Http.jsonBody
            (Encode.object
                [ ( "scope", Encode.string scope )
                , ( "name", Encode.string view.name )
                , ( "query", Encode.string view.query )
                , ( "state_filter", Encode.string view.stateFilter )
                , ( "type_filter", Encode.string view.typeFilter )
                , ( "sort", Encode.string view.sort )
                ]
            )
        )
        (expectJsonWithServerError SavedQueueViewSaved SavedQueueViews.savedQueueViewResponseDecoder)


fetchCredentials : String -> Cmd Msg
fetchCredentials token =
    authorizedRequest "GET" token "/api/agent-credentials" Http.emptyBody (expectJsonWithServerError CredentialsReceived Agent.agentCredentialsResponseDecoder)


postCreateTask : LoggedInModel -> Cmd Msg
postCreateTask state =
    authorizedRequest "POST"
        state.accessToken
        "/api/tasks"
        (Http.jsonBody (createTaskRequestBody state))
        (expectJsonWithServerError CreateTaskReceived taskDetailDecoder)


fetchDiscovery : String -> Bool -> Bool -> Int -> Cmd Msg
fetchDiscovery token includeReserved fundedOnly offset =
    let
        fundedQuery =
            if fundedOnly then
                "&funded=reward_funded"

            else
                ""
    in
    authorizedRequest "GET" token ("/api/tasks?scope=public&include_reserved=" ++ boolQuery includeReserved ++ fundedQuery ++ "&limit=" ++ String.fromInt selectorPageSize ++ "&offset=" ++ String.fromInt offset) Http.emptyBody (expectJsonWithServerError DiscoveryReceived Task.tasksResponseDecoder)


fetchPublicTaskDetail : String -> String -> Cmd Msg
fetchPublicTaskDetail token taskId =
    authorizedRequest "GET" token ("/api/tasks/" ++ taskId) Http.emptyBody (expectJsonWithServerError DetailReceived publicTaskDetailDecoder)


fetchSubmissions : String -> String -> Cmd Msg
fetchSubmissions token taskId =
    authorizedRequest "GET" token ("/api/tasks/" ++ taskId ++ "/submissions") Http.emptyBody (expectJsonWithServerError SubmissionsReceived Submission.submissionsResponseDecoder)


fetchReservations : String -> String -> Cmd Msg
fetchReservations token taskId =
    authorizedRequest "GET" token ("/api/tasks/" ++ taskId ++ "/reservations") Http.emptyBody (expectJsonWithServerError ReservationsReceived Task.taskReservationsResponseDecoder)


postFunding : String -> String -> Int -> String -> Int -> Cmd Msg
postFunding token taskId amount organizationId nonce =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/funding")
        (Http.jsonBody (fundingRequestBody taskId amount organizationId nonce))
        (expectJsonWithServerError FundReceived Ledger.taskFundResponseDecoder)


postOpenTask : String -> String -> Cmd Msg
postOpenTask token taskId =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/open")
        (Http.jsonBody (Encode.object []))
        (expectJsonWithServerError OpenTaskReceived taskDetailDecoder)


postUnpublishTask : String -> String -> Cmd Msg
postUnpublishTask token taskId =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/unpublish")
        (Http.jsonBody (Encode.object []))
        (expectJsonWithServerError UnpublishTaskReceived taskDetailDecoder)


postRefundTask : String -> String -> Cmd Msg
postRefundTask token taskId =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/refund")
        (Http.jsonBody (Encode.object [ ( "idempotency_key", Encode.string ("ui-refund:" ++ taskId) ) ]))
        (expectJsonWithServerError RefundTaskReceived Ledger.taskFundResponseDecoder)


postCancelTask : String -> String -> Cmd Msg
postCancelTask token taskId =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/cancel")
        (Http.jsonBody (Encode.object []))
        (expectJsonWithServerError CancelTaskReceived taskDetailDecoder)


postRefundCollectibleReward : String -> String -> Cmd Msg
postRefundCollectibleReward token taskId =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/collectible-refund")
        (Http.jsonBody (Encode.object []))
        (expectJsonWithServerError RefundCollectibleRewardReceived Collectible.collectiblesResponseDecoder)


postReservation : LoggedInModel -> String -> Cmd Msg
postReservation state taskId =
    authorizedRequest "POST"
        state.accessToken
        ("/api/tasks/" ++ taskId ++ "/reservations")
        (Http.jsonBody (reservationRequestBody state))
        (expectJsonWithServerError ReservationReceived Task.taskReservationResponseDecoder)


reservationRequestBody : LoggedInModel -> Encode.Value
reservationRequestBody state =
    case state.detail of
        Just detail ->
            case detail.assigneeScope of
                Task.TaskAssigneeScopeOrganizationTeam ->
                    Encode.object
                        [ ( "assignee_kind", Encode.string "organization_team" )
                        , ( "organization_id", Encode.string state.reservationOrganizationId )
                        , ( "team_id", Encode.string state.reservationTeamId )
                        ]

                Task.TaskAssigneeScopeTeam ->
                    Encode.object
                        [ ( "assignee_kind", Encode.string "team" )
                        , ( "team_id", Encode.string state.reservationTeamId )
                        ]

                _ ->
                    Encode.object []

        Nothing ->
            Encode.object []


postReservationChange : String -> String -> String -> String -> Cmd Msg
postReservationChange token taskId reservationId action =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/reservations/" ++ reservationId ++ "/" ++ action)
        (Http.jsonBody (Encode.object []))
        (expectJsonWithServerError ReservationChangeReceived Task.taskReservationResponseDecoder)


postAgent : String -> String -> List Agent.AgentScope -> String -> Cmd Msg
postAgent token agentLabel scopes expiresAt =
    authorizedRequest "POST"
        token
        "/api/agent-credentials"
        (Http.jsonBody (agentRequestBody agentLabel scopes expiresAt))
        (expectJsonWithServerError AgentCreated Agent.agentCredentialCreatedResponseDecoder)


mintTaskToken : String -> Cmd Msg
mintTaskToken token =
    authorizedRequest "POST"
        token
        "/api/agent-credentials"
        (Http.jsonBody (agentRequestBody "Task worker token" [ Agent.AgentScopeTasksRead, Agent.AgentScopeSubmissionsWrite, Agent.AgentScopeSubmissionsRead ] ""))
        (expectJsonWithServerError TaskTokenMinted Agent.agentCredentialCreatedResponseDecoder)


mintUserToken : String -> Cmd Msg
mintUserToken token =
    authorizedRequest "POST"
        token
        "/api/agent-credentials"
        (Http.jsonBody (agentRequestBody "Personal agent token" [ Agent.AgentScopeTasksRead, Agent.AgentScopeTasksWrite, Agent.AgentScopeSubmissionsRead, Agent.AgentScopeSubmissionsWrite, Agent.AgentScopeSubmissionsReview ] ""))
        (expectJsonWithServerError UserTokenMinted Agent.agentCredentialCreatedResponseDecoder)


postSubmission : String -> String -> String -> List SelectedAttachment -> Cmd Msg
postSubmission token taskId responseJson attachments =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/submissions")
        (Http.jsonBody (submissionRequestBody responseJson attachments))
        (expectJsonWithServerError SubmitReceived Submission.submissionCreatedResponseDecoder)


postAccept : String -> String -> String -> String -> String -> String -> Cmd Msg
postAccept token taskId submissionId payoutAmount tipAmount tipCollectibleId =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/submissions/" ++ submissionId ++ "/accept")
        (Http.jsonBody (acceptRequestBody submissionId payoutAmount tipAmount tipCollectibleId))
        (expectWhateverWithServerError (ReviewActionReceived submissionId))


postRequestChanges : String -> String -> String -> String -> Cmd Msg
postRequestChanges token taskId submissionId reviewNote =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/submissions/" ++ submissionId ++ "/request-changes")
        (Http.jsonBody (requestChangesBody submissionId reviewNote))
        (expectWhateverWithServerError (ReviewActionReceived submissionId))


postReject : String -> String -> String -> String -> String -> String -> Ledger.BanSelection -> Cmd Msg
postReject token taskId submissionId reviewNote partialCredit tipAmount banSelection =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/submissions/" ++ submissionId ++ "/reject")
        (Http.jsonBody (rejectRequestBody submissionId reviewNote partialCredit tipAmount banSelection))
        (expectWhateverWithServerError (ReviewActionReceived submissionId))


fetchCollectibles : String -> Cmd Msg
fetchCollectibles token =
    authorizedRequest "GET" token "/api/collectibles" Http.emptyBody (expectJsonWithServerError CollectiblesReceived Collectible.collectiblesResponseDecoder)


fetchCollectibleCatalog : String -> Cmd Msg
fetchCollectibleCatalog token =
    authorizedRequest "GET" token "/api/collectibles/catalog" Http.emptyBody (expectJsonWithServerError CollectibleCatalogReceived Collectible.collectibleCatalogResponseDecoder)


awardDefaultCollectible : String -> String -> String -> String -> Cmd Msg
awardDefaultCollectible token slug recipientKind recipientId =
    authorizedRequest "POST"
        token
        "/api/collectibles/award"
        (Http.jsonBody
            (Encode.object
                [ ( "slug", Encode.string slug )
                , ( "recipient_kind", Encode.string recipientKind )
                , ( "recipient_id", Encode.string recipientId )
                ]
            )
        )
        (expectJsonWithServerError AwardDefaultReceived Collectible.collectibleResponseDecoder)


transferCollectible : String -> String -> String -> String -> Cmd Msg
transferCollectible token collectibleId targetKind recipientId =
    authorizedRequest "POST"
        token
        ("/api/collectibles/" ++ collectibleId ++ "/transfer")
        (Http.jsonBody
            (Encode.object
                [ ( "target_kind", Encode.string targetKind )
                , ( "recipient_id", Encode.string recipientId )
                ]
            )
        )
        (expectJsonWithServerError TransferCollectibleReceived Collectible.collectibleResponseDecoder)


{-| Moves an organization-owned collectible to a user; the server checks the
acting member's manage_collectibles permission.
-}
postOrgSendCollectible : String -> String -> String -> String -> Cmd Msg
postOrgSendCollectible token organizationId collectibleId recipientId =
    authorizedRequest "POST"
        token
        ("/api/organizations/" ++ organizationId ++ "/collectibles/" ++ collectibleId ++ "/transfer")
        (Http.jsonBody (Encode.object [ ( "recipient_id", Encode.string recipientId ) ]))
        (expectJsonWithServerError OrgSendCollectibleReceived Collectible.collectibleResponseDecoder)


fetchOrganizations : String -> Cmd Msg
fetchOrganizations token =
    fetchOrganizationsPage token "" 0


fetchOrganizationsPage : String -> String -> Int -> Cmd Msg
fetchOrganizationsPage token queryText offset =
    authorizedRequest "GET" token (selectorQuery queryText offset "/api/organizations") Http.emptyBody (expectJsonWithServerError OrganizationsReceived Organization.organizationsResponseDecoder)


userDirectoryEntryDecoder : Decode.Decoder UserDirectoryEntry
userDirectoryEntryDecoder =
    Decode.map3 UserDirectoryEntry
        (Decode.field "id" Decode.string)
        (Decode.field "email" Decode.string)
        (Decode.field "status" Decode.string)


fetchUserDirectory : String -> Cmd Msg
fetchUserDirectory token =
    fetchUserDirectoryPage token "" 0


fetchUserDirectoryPage : String -> String -> Int -> Cmd Msg
fetchUserDirectoryPage token queryText offset =
    authorizedRequest "GET" token (selectorQuery queryText offset "/api/users") Http.emptyBody (expectJsonWithServerError UserDirectoryReceived userDirectoryPageDecoder)


userDirectoryPageDecoder : Decode.Decoder UserDirectoryPage
userDirectoryPageDecoder =
    Decode.map2 UserDirectoryPage
        (Decode.field "users" (Decode.list userDirectoryEntryDecoder))
        (Decode.field "next_offset" Decode.int)


fetchStandaloneTeams : String -> Cmd Msg
fetchStandaloneTeams token =
    fetchStandaloneTeamsPage token "" 0


createStandaloneTeam : String -> String -> Cmd Msg
createStandaloneTeam token name =
    authorizedRequest "POST"
        token
        "/api/teams"
        (Http.jsonBody (Encode.object [ ( "name", Encode.string name ) ]))
        (expectJsonWithServerError TeamCreated Team.teamResponseDecoder)


fetchStandaloneTeamsPage : String -> String -> Int -> Cmd Msg
fetchStandaloneTeamsPage token queryText offset =
    authorizedRequest "GET" token (selectorQuery queryText offset "/api/teams") Http.emptyBody (expectJsonWithServerError StandaloneTeamsReceived Team.teamsResponseDecoder)


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
            [ authorizedRequest "GET" token ("/api/organizations/" ++ organizationId ++ "/credits/balance") Http.emptyBody (expectJsonWithServerError OrgBalanceReceived Ledger.balanceResponseDecoder)
            , fetchOrganizationLedgerPage token organizationId 0
            , authorizedRequest "GET" token ("/api/organizations/" ++ organizationId ++ "/audit-events?limit=" ++ String.fromInt selectorPageSize ++ "&offset=0") Http.emptyBody (expectJsonWithServerError OrgAuditEventsReceived Admin.auditEventsResponseDecoder)
            , fetchOrgTeams token organizationId
            , authorizedRequest "GET" token ("/api/organizations/" ++ organizationId ++ "/members") Http.emptyBody (expectJsonWithServerError OrgMembersReceived Organization.organizationMembersResponseDecoder)
            , fetchOrgTasksPage token organizationId "" "" "" "newest" 0
            , fetchOrgCredentials token organizationId
            ]


fetchOrgCredentials : String -> String -> Cmd Msg
fetchOrgCredentials token organizationId =
    authorizedRequest "GET" token ("/api/organizations/" ++ organizationId ++ "/credentials") Http.emptyBody (expectJsonWithServerError OrgCredentialsReceived Agent.orgCredentialsResponseDecoder)


postOrgCredential : String -> String -> String -> List Agent.AgentScope -> String -> Cmd Msg
postOrgCredential token organizationId label scopes expiresAt =
    authorizedRequest "POST"
        token
        ("/api/organizations/" ++ organizationId ++ "/credentials")
        (Http.jsonBody (agentRequestBody label scopes expiresAt))
        (expectJsonWithServerError OrgCredentialCreated Agent.orgCredentialCreatedResponseDecoder)


postRevokeOrgCredential : String -> String -> String -> Cmd Msg
postRevokeOrgCredential token organizationId credentialId =
    authorizedRequest "POST"
        token
        ("/api/organizations/" ++ organizationId ++ "/credentials/" ++ credentialId ++ "/revoke")
        (Http.jsonBody (Encode.object []))
        (expectJsonWithServerError OrgCredentialRevoked Agent.orgCredentialResponseDecoder)


fetchOrgTasksPage : String -> String -> String -> String -> String -> String -> Int -> Cmd Msg
fetchOrgTasksPage token organizationId queryText stateFilter typeFilter sortOrder offset =
    let
        stateQuery =
            if stateFilter == "" then
                ""

            else
                "&state=" ++ stateFilter
    in
    authorizedRequest "GET" token ("/api/tasks?scope=organization&organization_id=" ++ organizationId ++ "&" ++ taskSearchParams queryText typeFilter sortOrder offset ++ stateQuery) Http.emptyBody (expectJsonWithServerError OrgTasksReceived Task.tasksResponseDecoder)


fetchAuditEvents : String -> String -> String -> String -> Int -> Cmd Msg
fetchAuditEvents token actionFilter subjectKindFilter subjectIDFilter offset =
    let
        actionQuery =
            if String.trim actionFilter == "" then
                ""

            else
                "&action=" ++ Url.percentEncode (String.trim actionFilter)

        subjectKindQuery =
            if String.trim subjectKindFilter == "" then
                ""

            else
                "&subject_kind=" ++ Url.percentEncode (String.trim subjectKindFilter)

        subjectIDQuery =
            if String.trim subjectIDFilter == "" then
                ""

            else
                "&subject_id=" ++ Url.percentEncode (String.trim subjectIDFilter)
    in
    authorizedRequest "GET" token ("/api/admin/audit-events?limit=" ++ String.fromInt selectorPageSize ++ "&offset=" ++ String.fromInt offset ++ actionQuery ++ subjectKindQuery ++ subjectIDQuery) Http.emptyBody (expectJsonWithServerError AuditEventsReceived Admin.auditEventsResponseDecoder)


fetchAdminPrivacyRequests : String -> Int -> Cmd Msg
fetchAdminPrivacyRequests token offset =
    authorizedRequest "GET" token ("/api/admin/privacy-requests?limit=" ++ String.fromInt selectorPageSize ++ "&offset=" ++ String.fromInt offset) Http.emptyBody (expectJsonWithServerError AdminPrivacyRequestsReceived Privacy.privacyRequestsResponseDecoder)


fetchPlatformAdmins : String -> Int -> Cmd Msg
fetchPlatformAdmins token offset =
    authorizedRequest "GET" token ("/api/admin/platform-admins?limit=" ++ String.fromInt selectorPageSize ++ "&offset=" ++ String.fromInt offset) Http.emptyBody (expectJsonWithServerError PlatformAdminsReceived Admin.platformAdminsResponseDecoder)


grantPlatformAdmin : String -> String -> Cmd Msg
grantPlatformAdmin token userID =
    authorizedRequest "POST"
        token
        "/api/admin/platform-admins"
        (Http.jsonBody (Encode.object [ ( "user_id", Encode.string userID ) ]))
        (expectJsonWithServerError PlatformAdminGranted Admin.platformAdminResponseDecoder)


revokePlatformAdmin : String -> String -> Cmd Msg
revokePlatformAdmin token userID =
    authorizedRequest "POST"
        token
        ("/api/admin/platform-admins/" ++ userID ++ "/revoke")
        Http.emptyBody
        (expectJsonWithServerError PlatformAdminRevoked Admin.platformAdminResponseDecoder)


fetchAdminModerationReports : String -> String -> Int -> Cmd Msg
fetchAdminModerationReports token stateFilter offset =
    let
        stateQuery =
            if String.trim stateFilter == "" then
                ""

            else
                "&state=" ++ Url.percentEncode (String.trim stateFilter)
    in
    authorizedRequest "GET" token ("/api/admin/moderation/reports?limit=" ++ String.fromInt selectorPageSize ++ "&offset=" ++ String.fromInt offset ++ stateQuery) Http.emptyBody (expectJsonWithServerError AdminModerationReportsReceived Moderation.moderationReportsResponseDecoder)


triageModerationReport : String -> String -> String -> String -> Cmd Msg
triageModerationReport token reportID stateValue resolutionNote =
    authorizedRequest "POST"
        token
        ("/api/admin/moderation/reports/" ++ reportID ++ "/triage")
        (Http.jsonBody (Encode.object [ ( "state", Encode.string stateValue ), ( "resolution_note", Encode.string resolutionNote ) ]))
        (expectJsonWithServerError AdminModerationReportTriaged Moderation.moderationReportResponseDecoder)


{-| Validates the admin grant form and posts it. The idempotency key was
minted when the form intent started (see Main's GrantCreditsClicked) so a
retried click after a network timeout dedupes server-side instead of
double-crediting.
-}
grantCreditsCommand : Model -> LoggedInModel -> String -> ( Model, Cmd Msg )
grantCreditsCommand model state key =
    if String.trim state.grantTargetId == "" then
        ( updateLoggedIn model (\current -> { current | grantMessage = Just (FailureNote "Choose a target first.") }), Cmd.none )

    else if Maybe.withDefault 0 (String.toInt (String.trim state.grantAmount)) < 1 then
        ( updateLoggedIn model (\current -> { current | grantMessage = Just (FailureNote "Amount must be a positive whole number of credits.") }), Cmd.none )

    else if String.trim state.grantNote == "" then
        ( updateLoggedIn model (\current -> { current | grantMessage = Just (FailureNote "A note is required - it appears in the beneficiary's ledger.") }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | grantMessage = Nothing, grantKey = key })
        , postCreditGrant state.accessToken state.grantTargetKind (String.trim state.grantTargetId) (Maybe.withDefault 0 (String.toInt (String.trim state.grantAmount))) (String.trim state.grantNote) key
        )


postCreditGrant : String -> String -> String -> Int -> String -> String -> Cmd Msg
postCreditGrant token targetKind targetId amount note key =
    authorizedRequest "POST"
        token
        "/api/admin/credits/grants"
        (Http.jsonBody
            (Encode.object
                [ ( "target_kind", Encode.string targetKind )
                , ( "target_id", Encode.string targetId )
                , ( "amount", Encode.int amount )
                , ( "note", Encode.string note )
                , ( "idempotency_key", Encode.string key )
                ]
            )
        )
        (expectJsonWithServerError CreditsGranted Ledger.creditGrantResponseDecoder)


{-| Validates and posts a peer credit send from the caller's own balance.
Mirrors the grant form's idempotency approach: the key identifies one send
intent (minted on first submit, cleared when any field changes), so a
retried click after a network timeout dedupes server-side.
-}
sendCreditsCommand : Model -> LoggedInModel -> String -> ( Model, Cmd Msg )
sendCreditsCommand model state key =
    sendCreditsFrom model state key "self" ""


{-| The organization-side send: the active organization pays. The server
checks the caller's billing permission on it.
-}
orgSendCreditsCommand : Model -> LoggedInModel -> String -> ( Model, Cmd Msg )
orgSendCreditsCommand model state key =
    if state.activeOrgId == "" then
        ( updateLoggedIn model (\current -> { current | sendMessage = Just (FailureNote "Open an organization first.") }), Cmd.none )

    else
        sendCreditsFrom model state key "organization" state.activeOrgId


sendCreditsFrom : Model -> LoggedInModel -> String -> String -> String -> ( Model, Cmd Msg )
sendCreditsFrom model state key sourceKind sourceOrganizationId =
    if String.trim state.sendRecipientId == "" then
        ( updateLoggedIn model (\current -> { current | sendMessage = Just (FailureNote "Choose a recipient first.") }), Cmd.none )

    else if Maybe.withDefault 0 (String.toInt (String.trim state.sendAmount)) < 1 then
        ( updateLoggedIn model (\current -> { current | sendMessage = Just (FailureNote "Amount must be a positive whole number of credits.") }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | sendMessage = Nothing, sendKey = key })
        , postCreditTransfer state.accessToken
            { sourceKind = sourceKind
            , sourceOrganizationId = sourceOrganizationId
            , targetKind = state.sendRecipientKind
            , targetId = String.trim state.sendRecipientId
            , amount = Maybe.withDefault 0 (String.toInt (String.trim state.sendAmount))
            , note = String.trim state.sendNote
            , key = key
            , recipientLabel = sendRecipientLabel state
            }
        )


{-| The human name of the chosen send recipient, for the confirmation
sentence ("Sent 25 credits to Ren Okafor."): the organization's name, or
the user row's email from the directory picker. Falls back to the raw id
for recipients no longer in the loaded page.
-}
sendRecipientLabel : LoggedInModel -> String
sendRecipientLabel state =
    let
        chosen =
            String.trim state.sendRecipientId
    in
    if state.sendRecipientKind == "organization" then
        state.organizations.items
            |> List.filter (\organization -> organization.id == chosen)
            |> List.head
            |> Maybe.map .name
            |> Maybe.withDefault chosen

    else
        state.userDirectory
            |> List.filter (\user -> user.id == chosen)
            |> List.head
            |> Maybe.map .email
            |> Maybe.withDefault chosen


postCreditTransfer :
    String
    ->
        { sourceKind : String
        , sourceOrganizationId : String
        , targetKind : String
        , targetId : String
        , amount : Int
        , note : String
        , key : String
        , recipientLabel : String
        }
    -> Cmd Msg
postCreditTransfer token transfer =
    authorizedRequest "POST"
        token
        "/api/credits/transfers"
        (Http.jsonBody
            (Encode.object
                [ ( "source_kind", Encode.string transfer.sourceKind )
                , ( "source_organization_id", Encode.string transfer.sourceOrganizationId )
                , ( "target_kind", Encode.string transfer.targetKind )
                , ( "target_id", Encode.string transfer.targetId )
                , ( "amount", Encode.int transfer.amount )
                , ( "note", Encode.string transfer.note )
                , ( "idempotency_key", Encode.string transfer.key )
                ]
            )
        )
        (expectJsonWithServerError (CreditsSentReceived transfer.recipientLabel) Ledger.creditTransferResponseDecoder)


{-| Validates and posts the admin catalog-entry form. Unique entries are
always a run of exactly 1 (sent implicitly); editions need an explicit
positive run size; badges are uncapped.
-}
addCatalogEntryCommand : Model -> LoggedInModel -> ( Model, Cmd Msg )
addCatalogEntryCommand model state =
    let
        editionSize =
            Maybe.withDefault 0 (String.toInt (String.trim state.catalogMaxEditions))

        maxEditions =
            case state.catalogKind of
                Collectible.CollectibleKindUnique ->
                    1

                Collectible.CollectibleKindEdition ->
                    editionSize

                Collectible.CollectibleKindBadge ->
                    0
    in
    if String.trim state.catalogSlug == "" || String.trim state.catalogName == "" then
        ( updateLoggedIn model (\current -> { current | catalogMessage = Just (FailureNote "A slug and a name are required.") }), Cmd.none )

    else if state.catalogArt == "" then
        ( updateLoggedIn model (\current -> { current | catalogMessage = Just (FailureNote "Pick the entry's art below.") }), Cmd.none )

    else if state.catalogKind == Collectible.CollectibleKindEdition && editionSize < 1 then
        ( updateLoggedIn model (\current -> { current | catalogMessage = Just (FailureNote "An edition needs a run size of at least 1.") }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | catalogMessage = Nothing })
        , postCatalogEntry state.accessToken (String.trim state.catalogSlug) (String.trim state.catalogName) state.catalogKind state.catalogPolicy state.catalogArt maxEditions
        )


postCatalogEntry : String -> String -> String -> Collectible.CollectibleKind -> Collectible.CollectibleTransferPolicy -> String -> Int -> Cmd Msg
postCatalogEntry token slug name kind policy art maxEditions =
    authorizedRequest "POST"
        token
        "/api/admin/collectible-catalog"
        (Http.jsonBody
            (Encode.object
                [ ( "slug", Encode.string slug )
                , ( "name", Encode.string name )
                , ( "kind", Encode.string (collectibleKindTag kind) )
                , ( "transfer_policy", Encode.string (collectiblePolicyTag policy) )
                , ( "art", Encode.string art )
                , ( "max_editions", Encode.int maxEditions )
                ]
            )
        )
        (expectJsonWithServerError CatalogEntryMutated Collectible.collectibleCatalogEntryDecoder)


postWithdrawCatalogEntry : String -> String -> Cmd Msg
postWithdrawCatalogEntry token slug =
    authorizedRequest "POST"
        token
        ("/api/admin/collectible-catalog/" ++ slug ++ "/withdraw")
        (Http.jsonBody (Encode.object []))
        (expectJsonWithServerError CatalogEntryMutated Collectible.collectibleCatalogEntryDecoder)


deleteCatalogEntryCmd : String -> String -> Cmd Msg
deleteCatalogEntryCmd token slug =
    authorizedRequest "DELETE"
        token
        ("/api/admin/collectible-catalog/" ++ slug)
        Http.emptyBody
        (expectWhateverWithServerError CatalogEntryDeleted)


postWithdrawCollectible : String -> String -> Cmd Msg
postWithdrawCollectible token collectibleId =
    authorizedRequest "POST"
        token
        ("/api/admin/collectibles/" ++ collectibleId ++ "/withdraw")
        (Http.jsonBody (Encode.object []))
        (expectJsonWithServerError CollectibleWithdrawnReceived Collectible.collectibleResponseDecoder)


deleteCollectibleCmd : String -> String -> Cmd Msg
deleteCollectibleCmd token collectibleId =
    authorizedRequest "DELETE"
        token
        ("/api/admin/collectibles/" ++ collectibleId)
        Http.emptyBody
        (expectWhateverWithServerError CollectibleDeleted)


runPrivacyRetention : String -> Cmd Msg
runPrivacyRetention token =
    authorizedRequest "POST"
        token
        "/api/admin/privacy-retention/run"
        Http.emptyBody
        (expectJsonWithServerError PrivacyRetentionRunReceived Privacy.privacyRetentionRunResponseDecoder)


resolveAdminPrivacyRequest : String -> String -> String -> Cmd Msg
resolveAdminPrivacyRequest token requestId resolutionNote =
    authorizedRequest "POST"
        token
        ("/api/admin/privacy-requests/" ++ requestId ++ "/resolve")
        (Http.jsonBody (Encode.object [ ( "resolution_note", Encode.string resolutionNote ) ]))
        (expectJsonWithServerError AdminPrivacyRequestResolved Privacy.privacyRequestResponseDecoder)


fetchOrgTeams : String -> String -> Cmd Msg
fetchOrgTeams token organizationId =
    fetchOrgTeamsPage token organizationId "" 0


fetchOrgTeamsPage : String -> String -> String -> Int -> Cmd Msg
fetchOrgTeamsPage token organizationId queryText offset =
    if organizationId == "" then
        Cmd.none

    else
        authorizedRequest "GET" token (selectorQuery queryText offset ("/api/organizations/" ++ organizationId ++ "/teams")) Http.emptyBody (expectJsonWithServerError OrgTeamsReceived Team.teamsResponseDecoder)


requestEmailVerification : String -> Cmd Msg
requestEmailVerification token =
    authorizedRequest "POST"
        token
        "/api/account/email-verification"
        (Http.jsonBody (Encode.object []))
        (expectJsonWithServerError EmailVerificationRequested tokenDecoder)


confirmEmailVerification : String -> String -> Cmd Msg
confirmEmailVerification token accountToken =
    authorizedRequest "POST"
        token
        "/api/auth/email-verification/confirm"
        (Http.jsonBody (Encode.object [ ( "token", Encode.string accountToken ) ]))
        (expectWhateverWithServerError AccountActionReceived)


fetchAccountProfile : String -> Cmd Msg
fetchAccountProfile token =
    authorizedRequest "GET" token "/api/account/profile" Http.emptyBody (expectJsonWithServerError AccountProfileReceived Auth.accountProfileResponseDecoder)


patchDisplayName : String -> String -> Cmd Msg
patchDisplayName token name =
    authorizedRequest "PATCH"
        token
        "/api/account/display-name"
        (Http.jsonBody (Encode.object [ ( "display_name", Encode.string name ) ]))
        (expectWhateverWithServerError DisplayNameSaved)


updateProfile : String -> String -> Cmd Msg
updateProfile token email =
    authorizedRequest "PATCH"
        token
        "/api/account/profile"
        (Http.jsonBody (Encode.object [ ( "email", Encode.string email ) ]))
        (expectWhateverWithServerError AccountActionReceived)


changePassword : String -> String -> String -> Cmd Msg
changePassword token current next =
    authorizedRequest "PATCH"
        token
        "/api/account/password"
        (Http.jsonBody (Encode.object [ ( "current_password", Encode.string current ), ( "new_password", Encode.string next ) ]))
        (expectWhateverWithServerError AccountActionReceived)


deactivateAccount : String -> Cmd Msg
deactivateAccount token =
    authorizedRequest "DELETE"
        token
        "/api/account"
        Http.emptyBody
        (expectWhateverWithServerError DeactivateAccountReceived)


requestPrivacy : String -> Privacy.PrivacyRequestKind -> Cmd Msg
requestPrivacy token kind =
    authorizedRequest "POST"
        token
        "/api/privacy-requests"
        (Http.jsonBody (Encode.object [ ( "kind", Privacy.privacyRequestKindEncoder kind ) ]))
        (expectJsonWithServerError PrivacyRequestReceived Privacy.privacyRequestResponseDecoder)


fetchMyPrivacyRequests : String -> Cmd Msg
fetchMyPrivacyRequests token =
    authorizedRequest "GET"
        token
        ("/api/privacy-requests?limit=" ++ String.fromInt selectorPageSize ++ "&offset=0")
        Http.emptyBody
        (expectJsonWithServerError MyPrivacyRequestsReceived Privacy.privacyRequestsResponseDecoder)


{-| Files a moderation report about the task itself or, for a dispute,
about one of the viewer's own submissions on it (see ModerationSubject).
-}
reportTask : String -> String -> ModerationSubject -> Moderation.ModerationReason -> String -> Cmd Msg
reportTask token taskId subject reason details =
    let
        ( subjectKind, subjectId ) =
            case subject of
                ReportAboutTask ->
                    ( "task", taskId )

                ReportAboutSubmission submissionId ->
                    ( "submission", submissionId )
    in
    authorizedRequest "POST"
        token
        "/api/moderation/reports"
        (Http.jsonBody
            (Encode.object
                [ ( "subject_kind", Encode.string subjectKind )
                , ( "subject_id", Encode.string subjectId )
                , ( "reason", Moderation.moderationReasonEncoder reason )
                , ( "details", Encode.string details )
                ]
            )
        )
        (expectJsonWithServerError ModerationReportReceived Moderation.moderationReportResponseDecoder)


createOrgTeamCommand : Model -> LoggedInModel -> ( Model, Cmd Msg )
createOrgTeamCommand model state =
    if String.isEmpty (String.trim state.createOrgTeamName) || state.activeOrgId == "" then
        ( updateLoggedIn model (\current -> { current | orgTeamMessage = Just (FailureNote "A team name is required.") }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | orgTeamMessage = Nothing })
        , authorizedRequest "POST"
            state.accessToken
            ("/api/organizations/" ++ state.activeOrgId ++ "/teams")
            (Http.jsonBody (Encode.object [ ( "name", Encode.string (String.trim state.createOrgTeamName) ) ]))
            (expectJsonWithServerError CreateOrgTeamReceived Team.teamResponseDecoder)
        )


provisionMemberCommand : Model -> LoggedInModel -> ( Model, Cmd Msg )
provisionMemberCommand model state =
    if String.isEmpty (String.trim state.provisionMemberEmail) || state.activeOrgId == "" then
        ( updateLoggedIn model (\current -> { current | provisionMemberMessage = Just (FailureNote "A member email is required.") }), Cmd.none )

    else if List.isEmpty state.provisionMemberRoles then
        ( updateLoggedIn model (\current -> { current | provisionMemberMessage = Just (FailureNote "Select at least one role.") }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | provisionMemberMessage = Nothing })
        , authorizedRequest "POST"
            state.accessToken
            ("/api/organizations/" ++ state.activeOrgId ++ "/members")
            (Http.jsonBody (Encode.object [ ( "email", Encode.string (String.trim state.provisionMemberEmail) ), ( "roles", Encode.list Encode.string state.provisionMemberRoles ) ]))
            (expectWhateverWithServerError ProvisionMemberReceived)
        )


updateMemberRolesCommand : Model -> LoggedInModel -> String -> List String -> ( Model, Cmd Msg )
updateMemberRolesCommand model state userId roles =
    if state.activeOrgId == "" || List.isEmpty roles then
        ( updateLoggedIn model (\current -> { current | provisionMemberMessage = Just (FailureNote "Select at least one role.") }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | provisionMemberMessage = Nothing })
        , authorizedRequest "PATCH"
            state.accessToken
            ("/api/organizations/" ++ state.activeOrgId ++ "/members/" ++ userId ++ "/roles")
            (Http.jsonBody (Encode.object [ ( "roles", Encode.list Encode.string roles ) ]))
            (expectJsonWithServerError UpdateMemberRolesReceived Organization.organizationMemberResponseDecoder)
        )


deactivateMemberCommand : Model -> LoggedInModel -> String -> ( Model, Cmd Msg )
deactivateMemberCommand model state userId =
    if state.activeOrgId == "" then
        ( updateLoggedIn model (\current -> { current | provisionMemberMessage = Just (FailureNote "Open an organization first.") }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | provisionMemberMessage = Nothing })
        , authorizedRequest "PATCH"
            state.accessToken
            ("/api/organizations/" ++ state.activeOrgId ++ "/members/" ++ userId ++ "/deactivate")
            (Http.jsonBody (Encode.object []))
            (expectWhateverWithServerError DeactivateMemberReceived)
        )


createOrgCommand : Model -> LoggedInModel -> ( Model, Cmd Msg )
createOrgCommand model state =
    if String.isEmpty (String.trim state.createOrgName) then
        ( updateLoggedIn model (\current -> { current | orgMessage = Just (FailureNote "Organization name is required.") }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | orgMessage = Nothing })
        , authorizedRequest "POST"
            state.accessToken
            "/api/organizations"
            (Http.jsonBody (Encode.object [ ( "name", Encode.string (String.trim state.createOrgName) ) ]))
            (expectJsonWithServerError CreateOrgReceived Organization.organizationResponseDecoder)
        )


postCollectible : String -> String -> Collectible.CollectibleKind -> Collectible.CollectibleTransferPolicy -> Cmd Msg
postCollectible token name kind policy =
    authorizedRequest "POST"
        token
        "/api/collectibles"
        (Http.jsonBody (collectibleRequestBody name kind policy))
        (expectJsonWithServerError MintReceived Collectible.collectibleResponseDecoder)


postCollectibleReward : String -> String -> String -> Cmd Msg
postCollectibleReward token taskId collectibleId =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/collectible-reward")
        (Http.jsonBody (collectibleRewardRequestBody collectibleId))
        (expectJsonWithServerError AwardReceived Collectible.collectibleResponseDecoder)


postAwardOrganizationCollectible : String -> String -> String -> String -> Cmd Msg
postAwardOrganizationCollectible token organizationId collectibleId recipientId =
    authorizedRequest "POST"
        token
        ("/api/organizations/" ++ organizationId ++ "/collectibles/" ++ collectibleId ++ "/award")
        (Http.jsonBody (Encode.object [ ( "recipient_id", Encode.string recipientId ) ]))
        (expectJsonWithServerError AwardOrgCollectibleReceived Collectible.collectibleResponseDecoder)


revokeAgent : String -> String -> Cmd Msg
revokeAgent token credentialId =
    authorizedRequest "POST"
        token
        ("/api/agent-credentials/" ++ credentialId ++ "/revoke")
        (Http.jsonBody (Encode.object []))
        (expectJsonWithServerError AgentRevoked Agent.agentCredentialResponseDecoder)


fundingRequestBody : String -> Int -> String -> Int -> Encode.Value
fundingRequestBody taskId amount organizationId nonce =
    Encode.object
        [ ( "amount", Encode.int amount )
        , ( "idempotency_key", Encode.string ("fund:" ++ taskId ++ ":" ++ String.fromInt nonce) )
        , ( "organization_id", Encode.string organizationId )
        ]


createTaskRequestBody : LoggedInModel -> Encode.Value
createTaskRequestBody state =
    Encode.object
        [ ( "owner", createOwnerBody state )
        , ( "title", Encode.string state.createTitle )
        , ( "description", Encode.string state.createDescription )
        , ( "reward", createRewardBody state.createRewardKind state.createRewardAmount state.createRewardCollectibleIds )
        , ( "participation", createParticipationBody state )
        , ( "visibility", createVisibilityBody state )
        , ( "placement", Encode.object [ ( "kind", Encode.string "standalone" ), ( "series_id", Encode.string "" ), ( "series_title", Encode.string "" ), ( "series_position", Encode.int 0 ) ] )
        , ( "response_schema_json", Encode.string (createSchemaString state) )
        , ( "payload", createPayloadBody state )
        , ( "task_type", Encode.string state.createTaskType )
        , ( "reference_url", Encode.string state.createReferenceURL )
        , ( "attachments", Encode.list attachmentRequestBody state.createAttachments )
        , ( "expires_at", Encode.string (expiryRFC3339 state.createExpiresAt) )
        ]


{-| Converts the expiry field's native datetime-local value
("YYYY-MM-DDTHH:MM", seconds optional) into the RFC3339 UTC instant the API
expects. The field is labeled UTC, so no timezone conversion happens here.
Blank stays blank (no expiration).
-}
expiryRFC3339 : String -> String
expiryRFC3339 raw =
    let
        trimmed =
            String.trim raw
    in
    if trimmed == "" then
        ""

    else if String.length trimmed == 16 then
        trimmed ++ ":00Z"

    else if String.length trimmed == 19 then
        trimmed ++ "Z"

    else
        trimmed


{-| Whether the expiry draft is blank or a complete datetime-local value.
The native picker cannot produce free text, so this only guards a partially
filled control (some browsers report "" there, but not all).
-}
expiryDraftIsValid : String -> Bool
expiryDraftIsValid raw =
    let
        trimmed =
            String.trim raw
    in
    trimmed == "" || String.length trimmed == 16 || String.length trimmed == 19


createSchemaString : LoggedInModel -> String
createSchemaString state =
    if String.trim state.createResponseSchema == "" then
        "{\"kind\":\"freeform\"}"

    else
        state.createResponseSchema


createPayloadBody : LoggedInModel -> Encode.Value
createPayloadBody state =
    if String.trim state.createPayloadJson == "" then
        Encode.object [ ( "kind", Encode.string "none" ), ( "json", Encode.string "" ) ]

    else
        Encode.object [ ( "kind", Encode.string "json" ), ( "json", Encode.string state.createPayloadJson ) ]


{-| Encode exactly the reward kind the user picked. createTaskCommand rejects
a credit/bundle reward without a positive amount before this runs, so a zero
amount can only reach the server through a bypassed client check, where the
server rejects it loudly - never silently create a different reward kind
than the one selected.
-}
createRewardBody : String -> String -> List String -> Encode.Value
createRewardBody kind rawAmount collectibleIds =
    let
        amount =
            Maybe.withDefault 0 (String.toInt (String.trim rawAmount))
    in
    case kind of
        "credit" ->
            Encode.object [ ( "kind", Encode.string "credit" ), ( "credit_amount", Encode.int amount ), ( "collectible_ids", Encode.list Encode.string [] ) ]

        "collectible" ->
            Encode.object [ ( "kind", Encode.string "collectible" ), ( "credit_amount", Encode.int 0 ), ( "collectible_ids", Encode.list Encode.string collectibleIds ) ]

        "bundle" ->
            Encode.object [ ( "kind", Encode.string "bundle" ), ( "credit_amount", Encode.int amount ), ( "collectible_ids", Encode.list Encode.string collectibleIds ) ]

        _ ->
            Encode.object [ ( "kind", Encode.string "none" ), ( "credit_amount", Encode.int 0 ), ( "collectible_ids", Encode.list Encode.string [] ) ]


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


agentRequestBody : String -> List Agent.AgentScope -> String -> Encode.Value
agentRequestBody agentLabel scopes expiresAt =
    Encode.object
        [ ( "label", Encode.string agentLabel )
        , ( "scopes", Encode.list Agent.agentScopeEncoder scopes )
        , ( "expires_at", Encode.string expiresAt )
        ]


{-| Converts an "expires in N hours" draft field into the absolute RFC3339
timestamp the REST API expects, or "" for "never expires" (blank/non-positive).
-}
expiresAtFromHours : Time.Posix -> String -> String
expiresAtFromHours now rawHours =
    case String.toInt (String.trim rawHours) of
        Just hours ->
            if hours > 0 then
                formatRFC3339 (Time.millisToPosix (Time.posixToMillis now + hours * 3600000))

            else
                ""

        Nothing ->
            ""


formatRFC3339 : Time.Posix -> String
formatRFC3339 posix =
    String.padLeft 4 '0' (String.fromInt (Time.toYear Time.utc posix))
        ++ "-"
        ++ String.padLeft 2 '0' (String.fromInt (monthNumber (Time.toMonth Time.utc posix)))
        ++ "-"
        ++ String.padLeft 2 '0' (String.fromInt (Time.toDay Time.utc posix))
        ++ "T"
        ++ String.padLeft 2 '0' (String.fromInt (Time.toHour Time.utc posix))
        ++ ":"
        ++ String.padLeft 2 '0' (String.fromInt (Time.toMinute Time.utc posix))
        ++ ":"
        ++ String.padLeft 2 '0' (String.fromInt (Time.toSecond Time.utc posix))
        ++ "Z"


monthNumber : Time.Month -> Int
monthNumber month =
    case month of
        Time.Jan ->
            1

        Time.Feb ->
            2

        Time.Mar ->
            3

        Time.Apr ->
            4

        Time.May ->
            5

        Time.Jun ->
            6

        Time.Jul ->
            7

        Time.Aug ->
            8

        Time.Sep ->
            9

        Time.Oct ->
            10

        Time.Nov ->
            11

        Time.Dec ->
            12


submissionRequestBody : String -> List SelectedAttachment -> Encode.Value
submissionRequestBody responseJson attachments =
    Encode.object
        [ ( "response_json", Encode.string responseJson )
        , ( "attachments", Encode.list attachmentRequestBody attachments )
        ]


attachmentRequestBody : SelectedAttachment -> Encode.Value
attachmentRequestBody attachment =
    Encode.object
        [ ( "name", Encode.string attachment.name )
        , ( "content_type", Encode.string attachment.contentType )
        , ( "data_url", Encode.string attachment.dataURL )
        ]


acceptRequestBody : String -> String -> String -> String -> Encode.Value
acceptRequestBody submissionId payoutAmount tipAmount tipCollectibleId =
    Encode.object
        [ ( "idempotency_key", Encode.string ("ui-accept:" ++ submissionId) )
        , ( "payout_amount", Encode.int (intInputOrZero payoutAmount) )
        , ( "tip_amount", Encode.int (intInputOrZero tipAmount) )
        , ( "tip_collectible_id", Encode.string tipCollectibleId )
        ]


-- The request-changes endpoint requires an idempotency key like accept and
-- reject do; the key is minted the same way (a stable per-submission prefix)
-- so a retried click after a network timeout dedupes server-side instead of
-- failing.


requestChangesBody : String -> String -> Encode.Value
requestChangesBody submissionId reviewNote =
    Encode.object
        [ ( "idempotency_key", Encode.string ("ui-request-changes:" ++ submissionId) )
        , ( "review_note", Encode.string reviewNote )
        ]


rejectRequestBody : String -> String -> String -> String -> Ledger.BanSelection -> Encode.Value
rejectRequestBody submissionId reviewNote partialCredit tipAmount banSelection =
    Encode.object
        [ ( "idempotency_key", Encode.string ("ui-reject:" ++ submissionId) )
        , ( "review_note", Encode.string reviewNote )
        , ( "partial_credit_amount", Encode.int (intInputOrZero partialCredit) )
        , ( "tip_amount", Encode.int (intInputOrZero tipAmount) )
        , ( "ban_selection", Ledger.banSelectionEncoder banSelection )
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
    , allocatedCredits = response.allocatedCredits
    , allocatedCollectibleIDs = response.allocatedCollectibleIDs
    , participationPolicy = response.participationPolicy
    , assigneeScope = response.assigneeScope
    , reservationExpiryHours = response.reservationExpiryHours
    , availabilityKind = response.availabilityKind
    , viewerAction = response.viewerAction
    , reviewerAction = response.reviewerAction
    , responseSchemaJson = response.responseSchemaJSON
    , payloadKind = response.payloadKind
    , payloadJson = response.payloadJSON
    , attachments = response.attachments
    , createdBy = response.createdBy
    , creatorDisplayName = response.creatorDisplayName
    , seriesID = response.seriesID
    , taskType = response.taskType
    , referenceURL = response.referenceURL
    , expiresAt = response.expiresAt
    }


publicTaskDetailFromResponse : Task.TaskResponse -> PublicTaskDetail
publicTaskDetailFromResponse response =
    taskDetailFromResponse response


seriesTaskEntryDecoder : Decode.Decoder SeriesTaskEntry
seriesTaskEntryDecoder =
    Decode.map3 SeriesTaskEntry
        (Decode.field "id" Decode.string)
        (Decode.field "title" Decode.string)
        (Decode.field "state" Decode.string)


seriesDetailDecoder : Decode.Decoder SeriesDetailData
seriesDetailDecoder =
    Decode.map3 SeriesDetailData
        (Decode.field "series" TaskSeries.taskSeriesResponseDecoder)
        (Decode.field "tasks" (Decode.list seriesTaskEntryDecoder))
        (Decode.field "comments" (Decode.list TaskSeries.seriesCommentResponseDecoder))


fetchSeriesList : String -> Cmd Msg
fetchSeriesList token =
    authorizedRequest "GET" token "/api/task-series" Http.emptyBody (expectJsonWithServerError SeriesListReceived TaskSeries.taskSeriesListResponseDecoder)


fetchSeriesDetail : String -> String -> Cmd Msg
fetchSeriesDetail token seriesId =
    authorizedRequest "GET" token ("/api/task-series/" ++ seriesId) Http.emptyBody (expectJsonWithServerError SeriesDetailReceived seriesDetailDecoder)


createSeriesCommand : Model -> LoggedInModel -> ( Model, Cmd Msg )
createSeriesCommand model state =
    if String.isEmpty (String.trim state.createSeriesTitle) then
        ( updateLoggedIn model (\current -> { current | seriesMessage = Just (FailureNote "A series title is required.") }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | seriesMessage = Nothing })
        , authorizedRequest "POST"
            state.accessToken
            "/api/task-series"
            (Http.jsonBody (seriesBody state.createSeriesTitle state.createSeriesDescription))
            (expectJsonWithServerError SeriesMutationReceived seriesDetailDecoder)
        )


updateSeriesCommand : Model -> LoggedInModel -> String -> ( Model, Cmd Msg )
updateSeriesCommand model state seriesId =
    if String.isEmpty (String.trim state.seriesRenameTitle) then
        ( updateLoggedIn model (\current -> { current | seriesMessage = Just (FailureNote "A series title is required.") }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | seriesMessage = Nothing })
        , authorizedRequest "PATCH"
            state.accessToken
            ("/api/task-series/" ++ seriesId)
            (Http.jsonBody (seriesBody state.seriesRenameTitle state.seriesRenameDescription))
            (expectJsonWithServerError SeriesMutationReceived seriesDetailDecoder)
        )


seriesStateCommand : String -> String -> String -> Cmd Msg
seriesStateCommand token seriesId action =
    authorizedRequest "POST"
        token
        ("/api/task-series/" ++ seriesId ++ "/" ++ action)
        (Http.jsonBody (Encode.object []))
        (expectJsonWithServerError SeriesMutationReceived seriesDetailDecoder)


addSeriesTaskCommand : Model -> LoggedInModel -> String -> ( Model, Cmd Msg )
addSeriesTaskCommand model state seriesId =
    if String.isEmpty (String.trim state.addSeriesTaskId) then
        ( updateLoggedIn model (\current -> { current | seriesMessage = Just (FailureNote "A task ID is required.") }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | seriesMessage = Nothing })
        , authorizedRequest "POST"
            state.accessToken
            ("/api/task-series/" ++ seriesId ++ "/tasks")
            (Http.jsonBody (Encode.object [ ( "task_id", Encode.string (String.trim state.addSeriesTaskId) ) ]))
            (expectJsonWithServerError SeriesMutationReceived seriesDetailDecoder)
        )


removeSeriesTaskCommand : String -> String -> String -> Cmd Msg
removeSeriesTaskCommand token seriesId taskId =
    authorizedRequest "DELETE"
        token
        ("/api/task-series/" ++ seriesId ++ "/tasks/" ++ taskId)
        Http.emptyBody
        (expectJsonWithServerError SeriesMutationReceived seriesDetailDecoder)


reorderSeriesCommand : String -> String -> List String -> Cmd Msg
reorderSeriesCommand token seriesId taskIds =
    authorizedRequest "POST"
        token
        ("/api/task-series/" ++ seriesId ++ "/reorder")
        (Http.jsonBody (Encode.object [ ( "task_ids", Encode.list Encode.string taskIds ) ]))
        (expectJsonWithServerError SeriesMutationReceived seriesDetailDecoder)


addSeriesCommentCommand : Model -> LoggedInModel -> String -> ( Model, Cmd Msg )
addSeriesCommentCommand model state seriesId =
    if String.isEmpty (String.trim state.seriesCommentBody) then
        ( updateLoggedIn model (\current -> { current | seriesMessage = Just (FailureNote "A comment is required.") }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | seriesMessage = Nothing })
        , authorizedRequest "POST"
            state.accessToken
            ("/api/task-series/" ++ seriesId ++ "/comments")
            (Http.jsonBody (Encode.object [ ( "body", Encode.string (String.trim state.seriesCommentBody) ) ]))
            (expectJsonWithServerError SeriesCommentReceived TaskSeries.seriesCommentResponseDecoder)
        )


fetchOrganizationLedgerPage : String -> String -> Int -> Cmd Msg
fetchOrganizationLedgerPage token organizationId offset =
    authorizedRequest "GET" token ("/api/organizations/" ++ organizationId ++ "/credits/ledger?limit=" ++ String.fromInt selectorPageSize ++ "&offset=" ++ String.fromInt offset) Http.emptyBody (expectJsonWithServerError OrgLedgerReceived Ledger.ledgerResponseDecoder)


fetchNotifications : String -> Bool -> Int -> Cmd Msg
fetchNotifications token unreadOnly offset =
    let
        stateQuery =
            if unreadOnly then
                "&state=unread"

            else
                ""
    in
    authorizedRequest "GET" token ("/api/notifications?limit=" ++ String.fromInt selectorPageSize ++ "&offset=" ++ String.fromInt offset ++ stateQuery) Http.emptyBody (expectJsonWithServerError NotificationsReceived Notification.notificationsResponseDecoder)


markNotificationRead : String -> String -> Cmd Msg
markNotificationRead token notificationId =
    authorizedRequest "POST" token ("/api/notifications/" ++ notificationId ++ "/read") Http.emptyBody (expectJsonWithServerError NotificationReadReceived Notification.notificationResponseDecoder)


fetchUnreadCount : String -> Cmd Msg
fetchUnreadCount token =
    authorizedRequest "GET" token "/api/notifications/unread-count" Http.emptyBody (expectJsonWithServerError UnreadCountReceived Notification.notificationUnreadCountResponseDecoder)


-- The events feed pages by cursor, not offset: an empty cursor starts from
-- the beginning of the caller's visible stream, and each poll resumes after
-- the last cursor already held in the model.


fetchEvents : String -> String -> Cmd Msg
fetchEvents token afterCursor =
    let
        afterQuery =
            if afterCursor == "" then
                ""

            else
                "&after=" ++ Url.percentEncode afterCursor
    in
    authorizedRequest "GET" token ("/api/events?limit=" ++ String.fromInt selectorPageSize ++ afterQuery) Http.emptyBody (expectJsonWithServerError ActivityEventsReceived Events.eventListResponseDecoder)


fetchWebhookSubscriptions : String -> Cmd Msg
fetchWebhookSubscriptions token =
    authorizedRequest "GET" token "/api/webhook-subscriptions" Http.emptyBody (expectJsonWithServerError WebhooksReceived Events.webhookSubscriptionsResponseDecoder)


createWebhookSubscription : String -> LoggedInModel -> Cmd Msg
createWebhookSubscription token state =
    authorizedRequest "POST"
        token
        "/api/webhook-subscriptions"
        (Http.jsonBody
            (Encode.object
                ([ ( "url", Encode.string (String.trim state.webhookURL) )
                 , ( "kinds", Encode.list (\kind -> Encode.string (domainEventKindTag kind)) state.webhookKinds )
                 , ( "organization_id", Encode.string "" )
                 ]
                    ++ webhookAudienceFields state
                )
            )
        )
        (expectJsonWithServerError WebhookCreated Events.webhookSubscriptionCreatedResponseDecoder)


{-| The audience block of a create-subscription request. The filter fields
are marketplace-only: the server rejects them on recipient subscriptions, so
they are omitted entirely there.
-}
webhookAudienceFields : LoggedInModel -> List ( String, Encode.Value )
webhookAudienceFields state =
    case state.webhookAudience of
        Events.WebhookAudienceRecipient ->
            [ ( "audience", Encode.string "recipient" ) ]

        Events.WebhookAudienceMarketplace ->
            [ ( "audience", Encode.string "marketplace" )
            , ( "filter_task_type", Encode.string (String.trim state.webhookFilterTaskType) )
            , ( "filter_min_credit_reward", Encode.int (intInputOrZero state.webhookFilterMinReward) )
            ]


revokeWebhookSubscription : String -> String -> Cmd Msg
revokeWebhookSubscription token subscriptionId =
    authorizedRequest "DELETE"
        token
        ("/api/webhook-subscriptions/" ++ subscriptionId)
        Http.emptyBody
        (expectJsonWithServerError WebhookRevoked Events.webhookSubscriptionResponseDecoder)


fetchWebhookDeliveries : String -> String -> Cmd Msg
fetchWebhookDeliveries token subscriptionId =
    authorizedRequest "GET"
        token
        ("/api/webhook-subscriptions/" ++ subscriptionId ++ "/deliveries")
        Http.emptyBody
        (expectJsonWithServerError WebhookDeliveriesReceived Events.webhookDeliveriesResponseDecoder)


createWebhookCommand : Model -> LoggedInModel -> ( Model, Cmd Msg )
createWebhookCommand model state =
    if String.trim state.webhookURL == "" then
        ( updateLoggedIn model (\current -> { current | webhookMessage = Just (FailureNote "Enter the endpoint URL first.") }), Cmd.none )

    else if List.isEmpty state.webhookKinds then
        ( updateLoggedIn model (\current -> { current | webhookMessage = Just (FailureNote "Select at least one event kind.") }), Cmd.none )

    else if state.webhookAudience == Events.WebhookAudienceMarketplace && not (minRewardIsValid state.webhookFilterMinReward) then
        ( updateLoggedIn model (\current -> { current | webhookMessage = Just (FailureNote "Minimum reward must be a positive whole number of credits, or blank for no floor.") }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | webhookMessage = Nothing, newWebhookSecret = Nothing })
        , createWebhookSubscription state.accessToken state
        )


minRewardIsValid : String -> Bool
minRewardIsValid raw =
    case String.toInt (String.trim raw) of
        Just floor ->
            floor > 0

        Nothing ->
            String.trim raw == ""


toggleWebhookKind : Events.DomainEventKind -> List Events.DomainEventKind -> List Events.DomainEventKind
toggleWebhookKind kind kinds =
    if List.member kind kinds then
        List.filter (\existing -> existing /= kind) kinds

    else
        kinds ++ [ kind ]


moveSeriesTaskOrder : Bool -> String -> List SeriesTaskEntry -> List String
moveSeriesTaskOrder up taskId tasks =
    let
        ids =
            List.map .id tasks
    in
    case indexOf taskId ids of
        Just index ->
            let
                target =
                    if up then
                        index - 1

                    else
                        index + 1
            in
            if target < 0 || target >= List.length ids then
                ids

            else
                swapAt index target ids

        Nothing ->
            ids


indexOf : String -> List String -> Maybe Int
indexOf value items =
    items
        |> List.indexedMap (\index item -> ( index, item ))
        |> List.filter (\( _, item ) -> item == value)
        |> List.head
        |> Maybe.map Tuple.first


swapAt : Int -> Int -> List String -> List String
swapAt a b items =
    let
        valueAt index =
            items |> List.drop index |> List.head
    in
    case ( valueAt a, valueAt b ) of
        ( Just va, Just vb ) ->
            List.indexedMap
                (\index item ->
                    if index == a then
                        vb

                    else if index == b then
                        va

                    else
                        item
                )
                items

        _ ->
            items


seriesBody : String -> String -> Encode.Value
seriesBody title description =
    Encode.object
        [ ( "title", Encode.string (String.trim title) )
        , ( "description", Encode.string description )
        ]



-- Every error response body from internal/http is `{"error": "..."}"`
-- (see internal/http/server.go's writeError). Plain expectJsonWithServerError/
-- expectWhatever discard that body entirely on a non-2xx response, leaving
-- only the numeric status code (see Labels.httpErrorLabel) - these two
-- helpers read it back out and carry it as Http.BadBody's message instead,
-- so the UI can show "task requester cannot reserve their own task" rather
-- than "The request failed with status 409."


serverErrorMessageDecoder : Decode.Decoder String
serverErrorMessageDecoder =
    Decode.map2 userFacingServerError
        (Decode.oneOf [ Decode.field "code" Decode.string, Decode.succeed "" ])
        (Decode.field "error" Decode.string)


-- Error bodies also carry a machine-readable `code`. The server's message
-- stays the primary user-facing text; the code is used to special-case rate
-- limiting (whose raw message is not phrased for end users) and to detect a
-- dead session: any `unauthenticated` failure on a signed-in request routes
-- to the single SessionEnded message instead of the per-call error path, so
-- every call site lands on the auth screen the same way without hand-patched
-- handling (see RequestFailure below).


userFacingServerError : String -> String -> String
userFacingServerError code message =
    if code == "rate_limited" then
        "Slow down — too many requests."

    else
        message


unauthenticatedCode : String
unauthenticatedCode =
    "unauthenticated"


{-| The notice shown on the auth screen after a forced sign-out. -}
sessionEndedNotice : String
sessionEndedNotice =
    "Your session ended. Sign in again."


{-| How a request failed: the session is gone (the server said
`unauthenticated`), or an ordinary error the call site handles itself.
-}
type RequestFailure
    = SessionMissing
    | RequestError Http.Error


serverErrorCodeDecoder : Decode.Decoder String
serverErrorCodeDecoder =
    Decode.oneOf [ Decode.field "code" Decode.string, Decode.succeed "" ]


responseToServerErrorResult : (String -> Result Http.Error a) -> Http.Response String -> Result RequestFailure a
responseToServerErrorResult onGoodBody response =
    case response of
        Http.BadUrl_ url ->
            Err (RequestError (Http.BadUrl url))

        Http.Timeout_ ->
            Err (RequestError Http.Timeout)

        Http.NetworkError_ ->
            Err (RequestError Http.NetworkError)

        Http.BadStatus_ metadata body ->
            if Decode.decodeString serverErrorCodeDecoder body == Ok unauthenticatedCode then
                Err SessionMissing

            else
                case Decode.decodeString serverErrorMessageDecoder body of
                    Ok message ->
                        Err (RequestError (Http.BadBody message))

                    Err _ ->
                        Err (RequestError (Http.BadStatus metadata.statusCode))

        Http.GoodStatus_ _ body ->
            Result.mapError RequestError (onGoodBody body)


{-| Turns a request outcome into the caller's message — except a dead
session, which every authorized call funnels into SessionEnded.
-}
sessionAwareMsg : (Result Http.Error a -> Msg) -> Result RequestFailure a -> Msg
sessionAwareMsg toMsg result =
    case result of
        Ok value ->
            toMsg (Ok value)

        Err SessionMissing ->
            SessionEnded

        Err (RequestError error) ->
            toMsg (Err error)


{-| For the auth endpoints themselves (login, register, password reset,
refresh, logout): a 401 there is feedback about the attempt in progress
(wrong password, expired reset token, already-signed-out), not a session
that died mid-use, so it stays on the caller's own error path.
-}
plainErrorMsg : (Result Http.Error a -> Msg) -> Result RequestFailure a -> Msg
plainErrorMsg toMsg result =
    case result of
        Ok value ->
            toMsg (Ok value)

        Err SessionMissing ->
            toMsg (Err (Http.BadBody sessionEndedNotice))

        Err (RequestError error) ->
            toMsg (Err error)


decodeGoodBody : Decode.Decoder a -> String -> Result Http.Error a
decodeGoodBody decoder body =
    case Decode.decodeString decoder body of
        Ok value ->
            Ok value

        Err error ->
            Err (Http.BadBody (Decode.errorToString error))


expectJsonWithServerError : (Result Http.Error a -> Msg) -> Decode.Decoder a -> Http.Expect Msg
expectJsonWithServerError toMsg decoder =
    Http.expectStringResponse (sessionAwareMsg toMsg)
        (responseToServerErrorResult (decodeGoodBody decoder))


expectWhateverWithServerError : (Result Http.Error () -> Msg) -> Http.Expect Msg
expectWhateverWithServerError toMsg =
    Http.expectStringResponse (sessionAwareMsg toMsg) (responseToServerErrorResult (\_ -> Ok ()))


expectAuthJson : (Result Http.Error a -> Msg) -> Decode.Decoder a -> Http.Expect Msg
expectAuthJson toMsg decoder =
    Http.expectStringResponse (plainErrorMsg toMsg)
        (responseToServerErrorResult (decodeGoodBody decoder))


expectAuthWhatever : (Result Http.Error () -> Msg) -> Http.Expect Msg
expectAuthWhatever toMsg =
    Http.expectStringResponse (plainErrorMsg toMsg) (responseToServerErrorResult (\_ -> Ok ()))


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
