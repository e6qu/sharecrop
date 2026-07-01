module Sharecrop.Api exposing (..)

import Http
import Json.Decode as Decode
import Json.Encode as Encode
import Sharecrop.Generated.Admin as Admin
import Sharecrop.Generated.Agent as Agent
import Sharecrop.Generated.Auth as Auth
import Sharecrop.Generated.Collectible as Collectible
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
import Sharecrop.Labels exposing (assigneeScopeTag, participationUsesReservation)
import Sharecrop.Types exposing (..)
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


selectorPageSize : Int
selectorPageSize =
    20


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
                ( updateLoggedIn model (\current -> { current | fundMessage = Just "Amount must be a positive number of credits." }), Cmd.none )

            else
                ( updateLoggedIn model (\current -> { current | fundMessage = Nothing }), postFunding state.accessToken state.fundTaskId amount state.fundOrganizationId state.fundNonce )

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

    else if participationUsesReservation state.createParticipationPolicy && (reservationHoursValue state.createReservationHours < 1 || reservationHoursValue state.createReservationHours > 720) then
        ( updateLoggedIn model (\current -> { current | createMessage = Just "Reservation expiry must be between 1 and 720 hours." }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | createMessage = Nothing })
        , postCreateTask state
        )


submitCommand : Model -> LoggedInModel -> ( Model, Cmd Msg )
submitCommand model state =
    case state.page of
        TaskDetailPage taskId ->
            let
                trimmed =
                    String.trim state.submitInput
            in
            -- Guard obviously-invalid input before hitting the server: the
            -- response payload must be non-empty and parse as JSON.
            if trimmed == "" then
                ( updateLoggedIn model (\current -> { current | submitMessage = Just "Enter a response first." }), Cmd.none )

            else
                case Decode.decodeString Decode.value trimmed of
                    Ok _ ->
                        ( updateLoggedIn model (\current -> { current | submitMessage = Nothing })
                        , postSubmission state.accessToken taskId trimmed state.submitAttachments
                        )

                    Err _ ->
                        ( updateLoggedIn model (\current -> { current | submitMessage = Just "Response must be valid JSON." }), Cmd.none )

        _ ->
            ( model, Cmd.none )


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
    Cmd.batch [ fetchBalance token, fetchLedger token 0, fetchTasks token "" "" "newest" 0, fetchCredentials token, fetchCollectibles token, fetchOrganizations token, fetchUserDirectory token, fetchStandaloneTeams token, fetchSavedQueueViews token ]


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


refreshTasksAndLedger : Model -> Cmd Msg
refreshTasksAndLedger model =
    case model.session of
        LoggedIn state ->
            Cmd.batch [ fetchTasks state.accessToken state.taskStateFilter state.taskListTypeFilter state.taskListSort state.taskListOffset, fetchBalance state.accessToken, fetchLedger state.accessToken state.ledgerOffset ]

        LoggedOut ->
            Cmd.none


refreshTasksAndDiscovery : Model -> Cmd Msg
refreshTasksAndDiscovery model =
    case model.session of
        LoggedIn state ->
            Cmd.batch [ fetchTasks state.accessToken state.taskStateFilter state.taskListTypeFilter state.taskListSort state.taskListOffset, fetchDiscovery state.accessToken state.discoveryIncludeReserved state.discoveryOffset ]

        LoggedOut ->
            Cmd.none


routeLoadCmd : String -> String -> Page -> Cmd Msg
routeLoadCmd token subjectId page =
    case page of
        OverviewPage ->
            Cmd.batch [ fetchBalance token, fetchLedger token 0 ]

        TasksPage ->
            fetchTasks token "" "" "newest" 0

        CreateTaskPage ->
            Cmd.batch [ fetchOrganizations token, fetchCollectibles token, fetchUserDirectory token, fetchStandaloneTeams token ]

        TaskDetailPage taskId ->
            fetchDetailCommands token subjectId taskId

        DiscoveryPage ->
            fetchDiscovery token False 0

        FundingPage ->
            Cmd.batch [ fetchTasks token "" "" "newest" 0, fetchOrganizations token ]

        AgentsPage ->
            fetchCredentials token

        CollectiblesPage ->
            Cmd.batch [ fetchCollectibles token, fetchCollectibleCatalog token, fetchTasks token "" "" "newest" 0, fetchOrganizations token ]

        OrganizationsPage ->
            fetchOrganizations token

        OrganizationDetailPage organizationId ->
            Cmd.batch [ fetchOrganizations token, loadOrganization token organizationId, fetchOrganizationCollectibles token organizationId ]

        UserDetailPage userId ->
            fetchUserProfile token userId

        UserWorkPage userId ->
            authorizedRequest "GET" token ("/api/users/" ++ userId ++ "/work") Http.emptyBody (Http.expectJson UserWorkReceived Task.tasksResponseDecoder)

        UserSubmissionsPage userId ->
            fetchUserSubmissionsPage token userId 0

        CollectibleDetailPage _ ->
            fetchCollectibles token

        SeriesListPage ->
            fetchSeriesList token

        SeriesDetailPage seriesId ->
            fetchSeriesDetail token seriesId

        TeamDetailPage teamId ->
            Cmd.batch
                [ authorizedRequest "GET" token ("/api/teams/" ++ teamId) Http.emptyBody (Http.expectJson TeamDetailReceived Team.teamDetailResponseDecoder)
                , fetchTeamWork token teamId "" "" "newest" 0
                , fetchTeamCollectibles token teamId
                ]

        AdminPage ->
            Cmd.batch
                [ authorizedRequest "GET" token "/api/admin/operations" Http.emptyBody (Http.expectJson OperationsReceived Admin.operationsResponseDecoder)
                , fetchAuditEvents token "" "" "" 0
                , fetchPlatformAdmins token 0
                , fetchUserDirectory token
                , fetchAdminModerationReports token "open" 0
                , fetchAdminPrivacyRequests token 0
                ]

        InboxPage ->
            fetchNotifications token 0

        NotFoundPage ->
            Cmd.none


fetchOrganizationCollectibles : String -> String -> Cmd Msg
fetchOrganizationCollectibles token orgId =
    authorizedRequest "GET" token ("/api/organizations/" ++ orgId ++ "/collectibles") Http.emptyBody (Http.expectJson OrgCollectiblesReceived Collectible.collectiblesResponseDecoder)


fetchTeamCollectibles : String -> String -> Cmd Msg
fetchTeamCollectibles token teamId =
    authorizedRequest "GET" token ("/api/teams/" ++ teamId ++ "/collectibles") Http.emptyBody (Http.expectJson TeamCollectiblesReceived Collectible.collectiblesResponseDecoder)


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
    authorizedRequest "GET" token ("/api/users/" ++ userId ++ "/submissions?limit=" ++ String.fromInt selectorPageSize ++ "&offset=" ++ String.fromInt offset) Http.emptyBody (Http.expectJson UserSubmissionsReceived Submission.submissionsResponseDecoder)


fetchTaskComments : String -> String -> Cmd Msg
fetchTaskComments token taskId =
    authorizedRequest "GET"
        token
        ("/api/tasks/" ++ taskId ++ "/comments")
        Http.emptyBody
        (Http.expectJson TaskCommentsReceived (Decode.field "comments" (Decode.list Task.taskCommentResponseDecoder)))


postTaskComment : String -> String -> String -> Cmd Msg
postTaskComment token taskId body =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/comments")
        (Http.jsonBody (Encode.object [ ( "body", Encode.string body ) ]))
        (Http.expectJson TaskCommentReceived Task.taskCommentResponseDecoder)


fetchSubmissionComments : String -> String -> Cmd Msg
fetchSubmissionComments token submissionId =
    authorizedRequest "GET"
        token
        ("/api/submissions/" ++ submissionId ++ "/comments")
        Http.emptyBody
        (Http.expectJson SubmissionCommentsReceived Submission.submissionCommentsResponseDecoder)


addSubmissionComment : String -> String -> String -> Cmd Msg
addSubmissionComment token submissionId body =
    authorizedRequest "POST"
        token
        ("/api/submissions/" ++ submissionId ++ "/comments")
        (Http.jsonBody (Encode.object [ ( "body", Encode.string body ) ]))
        (Http.expectJson SubmissionCommentAdded Submission.submissionCommentResponseDecoder)


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
        , expect = Http.expectJson AuthReceived Auth.authResponseDecoder
        }


postGuest : Cmd Msg
postGuest =
    Http.post
        { url = "/api/auth/guest"
        , body = Http.emptyBody
        , expect = Http.expectJson AuthReceived Auth.authResponseDecoder
        }


requestPasswordReset : Model -> Cmd Msg
requestPasswordReset model =
    Http.post
        { url = "/api/auth/password-reset/request"
        , body = Http.jsonBody (Encode.object [ ( "email", Encode.string model.resetEmail ) ])
        , expect = Http.expectJson PasswordResetRequested tokenDecoder
        }


confirmPasswordReset : Model -> Cmd Msg
confirmPasswordReset model =
    Http.post
        { url = "/api/auth/password-reset/confirm"
        , body = Http.jsonBody (Encode.object [ ( "token", Encode.string model.resetToken ), ( "password", Encode.string model.resetPassword ) ])
        , expect = Http.expectWhatever PasswordResetConfirmed
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


fetchLedger : String -> Int -> Cmd Msg
fetchLedger token offset =
    authorizedRequest "GET" token ("/api/credits/ledger?limit=" ++ String.fromInt selectorPageSize ++ "&offset=" ++ String.fromInt offset) Http.emptyBody (Http.expectJson LedgerReceived Ledger.ledgerResponseDecoder)


fetchTasks : String -> String -> String -> String -> Int -> Cmd Msg
fetchTasks token stateFilter typeFilter sortOrder offset =
    let
        pageQuery =
            "limit=" ++ String.fromInt selectorPageSize ++ "&offset=" ++ String.fromInt offset

        stateQuery =
            if stateFilter == "" then
                ""

            else
                "&state=" ++ Url.percentEncode stateFilter

        typeQuery =
            if typeFilter == "" then
                ""

            else
                "&task_type=" ++ Url.percentEncode typeFilter

        sortQuery =
            "&sort=" ++ Url.percentEncode sortOrder
    in
    authorizedRequest "GET" token ("/api/tasks?scope=user&" ++ pageQuery ++ stateQuery ++ typeQuery ++ sortQuery) Http.emptyBody (Http.expectJson TasksReceived Task.tasksResponseDecoder)


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
    authorizedRequest "GET" token ("/api/teams/" ++ teamId ++ "/work?" ++ taskSearchParams queryText typeFilter sortOrder offset) Http.emptyBody (Http.expectJson TeamWorkReceived Task.tasksResponseDecoder)


fetchSavedQueueViews : String -> Cmd Msg
fetchSavedQueueViews token =
    authorizedRequest "GET" token "/api/saved-queue-views" Http.emptyBody (Http.expectJson SavedQueueViewsReceived SavedQueueViews.savedQueueViewsResponseDecoder)


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
        (Http.expectJson SavedQueueViewSaved SavedQueueViews.savedQueueViewResponseDecoder)


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


fetchDiscovery : String -> Bool -> Int -> Cmd Msg
fetchDiscovery token includeReserved offset =
    authorizedRequest "GET" token ("/api/tasks?scope=public&include_reserved=" ++ boolQuery includeReserved ++ "&limit=" ++ String.fromInt selectorPageSize ++ "&offset=" ++ String.fromInt offset) Http.emptyBody (Http.expectJson DiscoveryReceived Task.tasksResponseDecoder)


fetchPublicTaskDetail : String -> String -> Cmd Msg
fetchPublicTaskDetail token taskId =
    authorizedRequest "GET" token ("/api/tasks/" ++ taskId) Http.emptyBody (Http.expectJson DetailReceived publicTaskDetailDecoder)


fetchSubmissions : String -> String -> Cmd Msg
fetchSubmissions token taskId =
    authorizedRequest "GET" token ("/api/tasks/" ++ taskId ++ "/submissions") Http.emptyBody (Http.expectJson SubmissionsReceived Submission.submissionsResponseDecoder)


fetchReservations : String -> String -> Cmd Msg
fetchReservations token taskId =
    authorizedRequest "GET" token ("/api/tasks/" ++ taskId ++ "/reservations") Http.emptyBody (Http.expectJson ReservationsReceived Task.taskReservationsResponseDecoder)


postFunding : String -> String -> Int -> String -> Int -> Cmd Msg
postFunding token taskId amount organizationId nonce =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/funding")
        (Http.jsonBody (fundingRequestBody taskId amount organizationId nonce))
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


postCancelTask : String -> String -> Cmd Msg
postCancelTask token taskId =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/cancel")
        (Http.jsonBody (Encode.object []))
        (Http.expectJson CancelTaskReceived taskDetailDecoder)


postRefundCollectibleReward : String -> String -> Cmd Msg
postRefundCollectibleReward token taskId =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/collectible-refund")
        (Http.jsonBody (Encode.object []))
        (Http.expectJson RefundCollectibleRewardReceived Collectible.collectiblesResponseDecoder)


postReservation : LoggedInModel -> String -> Cmd Msg
postReservation state taskId =
    authorizedRequest "POST"
        state.accessToken
        ("/api/tasks/" ++ taskId ++ "/reservations")
        (Http.jsonBody (reservationRequestBody state))
        (Http.expectJson ReservationReceived Task.taskReservationResponseDecoder)


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
        (Http.expectJson ReservationChangeReceived Task.taskReservationResponseDecoder)


postAgent : String -> String -> List Agent.AgentScope -> Cmd Msg
postAgent token agentLabel scopes =
    authorizedRequest "POST"
        token
        "/api/agent-credentials"
        (Http.jsonBody (agentRequestBody agentLabel scopes))
        (Http.expectJson AgentCreated Agent.agentCredentialCreatedResponseDecoder)


mintTaskToken : String -> Cmd Msg
mintTaskToken token =
    authorizedRequest "POST"
        token
        "/api/agent-credentials"
        (Http.jsonBody (agentRequestBody "Task worker token" [ Agent.AgentScopeTasksRead, Agent.AgentScopeSubmissionsWrite, Agent.AgentScopeSubmissionsRead ]))
        (Http.expectJson TaskTokenMinted Agent.agentCredentialCreatedResponseDecoder)


mintUserToken : String -> Cmd Msg
mintUserToken token =
    authorizedRequest "POST"
        token
        "/api/agent-credentials"
        (Http.jsonBody (agentRequestBody "Personal agent token" [ Agent.AgentScopeTasksRead, Agent.AgentScopeTasksWrite, Agent.AgentScopeSubmissionsRead, Agent.AgentScopeSubmissionsWrite, Agent.AgentScopeSubmissionsReview ]))
        (Http.expectJson UserTokenMinted Agent.agentCredentialCreatedResponseDecoder)


postSubmission : String -> String -> String -> List SelectedAttachment -> Cmd Msg
postSubmission token taskId responseJson attachments =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/submissions")
        (Http.jsonBody (submissionRequestBody responseJson attachments))
        (Http.expectJson SubmitReceived Submission.submissionCreatedResponseDecoder)


postAccept : String -> String -> String -> String -> String -> String -> Cmd Msg
postAccept token taskId submissionId payoutAmount tipAmount tipCollectibleId =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/submissions/" ++ submissionId ++ "/accept")
        (Http.jsonBody (acceptRequestBody submissionId payoutAmount tipAmount tipCollectibleId))
        (Http.expectWhatever (ReviewActionReceived submissionId))


postRequestChanges : String -> String -> String -> String -> Cmd Msg
postRequestChanges token taskId submissionId reviewNote =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/submissions/" ++ submissionId ++ "/request-changes")
        (Http.jsonBody (requestChangesBody reviewNote))
        (Http.expectWhatever (ReviewActionReceived submissionId))


postReject : String -> String -> String -> String -> String -> String -> Bool -> Cmd Msg
postReject token taskId submissionId reviewNote partialCredit tipAmount banImplementor =
    authorizedRequest "POST"
        token
        ("/api/tasks/" ++ taskId ++ "/submissions/" ++ submissionId ++ "/reject")
        (Http.jsonBody (rejectRequestBody submissionId reviewNote partialCredit tipAmount banImplementor))
        (Http.expectWhatever (ReviewActionReceived submissionId))


fetchCollectibles : String -> Cmd Msg
fetchCollectibles token =
    authorizedRequest "GET" token "/api/collectibles" Http.emptyBody (Http.expectJson CollectiblesReceived Collectible.collectiblesResponseDecoder)


fetchCollectibleCatalog : String -> Cmd Msg
fetchCollectibleCatalog token =
    authorizedRequest "GET" token "/api/collectibles/catalog" Http.emptyBody (Http.expectJson CollectibleCatalogReceived Collectible.collectibleCatalogResponseDecoder)


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
        (Http.expectJson AwardDefaultReceived Collectible.collectibleResponseDecoder)


transferCollectible : String -> String -> String -> Cmd Msg
transferCollectible token collectibleId recipientId =
    authorizedRequest "POST"
        token
        ("/api/collectibles/" ++ collectibleId ++ "/transfer")
        (Http.jsonBody (Encode.object [ ( "recipient_id", Encode.string recipientId ) ]))
        (Http.expectJson TransferCollectibleReceived Collectible.collectibleResponseDecoder)


fetchOrganizations : String -> Cmd Msg
fetchOrganizations token =
    fetchOrganizationsPage token "" 0


fetchOrganizationsPage : String -> String -> Int -> Cmd Msg
fetchOrganizationsPage token queryText offset =
    authorizedRequest "GET" token (selectorQuery queryText offset "/api/organizations") Http.emptyBody (Http.expectJson OrganizationsReceived Organization.organizationsResponseDecoder)


userDirectoryEntryDecoder : Decode.Decoder UserDirectoryEntry
userDirectoryEntryDecoder =
    Decode.map3 UserDirectoryEntry
        (Decode.field "id" Decode.string)
        (Decode.field "email" Decode.string)
        (Decode.field "status" Decode.string)


fetchUserDirectory : String -> Cmd Msg
fetchUserDirectory token =
    fetchUserDirectoryPage token "" 0


fetchUserDirectoryQuery : String -> String -> Cmd Msg
fetchUserDirectoryQuery token queryText =
    fetchUserDirectoryPage token queryText 0


fetchUserDirectoryPage : String -> String -> Int -> Cmd Msg
fetchUserDirectoryPage token queryText offset =
    authorizedRequest "GET" token (selectorQuery queryText offset "/api/users") Http.emptyBody (Http.expectJson UserDirectoryReceived (Decode.field "users" (Decode.list userDirectoryEntryDecoder)))


fetchStandaloneTeams : String -> Cmd Msg
fetchStandaloneTeams token =
    fetchStandaloneTeamsPage token "" 0


fetchStandaloneTeamsPage : String -> String -> Int -> Cmd Msg
fetchStandaloneTeamsPage token queryText offset =
    authorizedRequest "GET" token (selectorQuery queryText offset "/api/teams") Http.emptyBody (Http.expectJson StandaloneTeamsReceived Team.teamsResponseDecoder)


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
            , fetchOrganizationLedgerPage token organizationId 0
            , authorizedRequest "GET" token ("/api/organizations/" ++ organizationId ++ "/audit-events?limit=" ++ String.fromInt selectorPageSize ++ "&offset=0") Http.emptyBody (Http.expectJson OrgAuditEventsReceived Admin.auditEventsResponseDecoder)
            , fetchOrgTeams token organizationId
            , authorizedRequest "GET" token ("/api/organizations/" ++ organizationId ++ "/members") Http.emptyBody (Http.expectJson OrgMembersReceived Organization.organizationMembersResponseDecoder)
            , fetchOrgTasksPage token organizationId "" "" "" "newest" 0
            ]


fetchOrgTasksPage : String -> String -> String -> String -> String -> String -> Int -> Cmd Msg
fetchOrgTasksPage token organizationId queryText stateFilter typeFilter sortOrder offset =
    let
        stateQuery =
            if stateFilter == "" then
                ""

            else
                "&state=" ++ stateFilter
    in
    authorizedRequest "GET" token ("/api/tasks?scope=organization&organization_id=" ++ organizationId ++ "&" ++ taskSearchParams queryText typeFilter sortOrder offset ++ stateQuery) Http.emptyBody (Http.expectJson OrgTasksReceived Task.tasksResponseDecoder)


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
    authorizedRequest "GET" token ("/api/admin/audit-events?limit=" ++ String.fromInt selectorPageSize ++ "&offset=" ++ String.fromInt offset ++ actionQuery ++ subjectKindQuery ++ subjectIDQuery) Http.emptyBody (Http.expectJson AuditEventsReceived Admin.auditEventsResponseDecoder)


fetchAdminPrivacyRequests : String -> Int -> Cmd Msg
fetchAdminPrivacyRequests token offset =
    authorizedRequest "GET" token ("/api/admin/privacy-requests?limit=" ++ String.fromInt selectorPageSize ++ "&offset=" ++ String.fromInt offset) Http.emptyBody (Http.expectJson AdminPrivacyRequestsReceived Privacy.privacyRequestsResponseDecoder)


fetchPlatformAdmins : String -> Int -> Cmd Msg
fetchPlatformAdmins token offset =
    authorizedRequest "GET" token ("/api/admin/platform-admins?limit=" ++ String.fromInt selectorPageSize ++ "&offset=" ++ String.fromInt offset) Http.emptyBody (Http.expectJson PlatformAdminsReceived Admin.platformAdminsResponseDecoder)


grantPlatformAdmin : String -> String -> Cmd Msg
grantPlatformAdmin token userID =
    authorizedRequest "POST"
        token
        "/api/admin/platform-admins"
        (Http.jsonBody (Encode.object [ ( "user_id", Encode.string userID ) ]))
        (Http.expectJson PlatformAdminGranted Admin.platformAdminResponseDecoder)


revokePlatformAdmin : String -> String -> Cmd Msg
revokePlatformAdmin token userID =
    authorizedRequest "POST"
        token
        ("/api/admin/platform-admins/" ++ userID ++ "/revoke")
        Http.emptyBody
        (Http.expectJson PlatformAdminRevoked Admin.platformAdminResponseDecoder)


fetchAdminModerationReports : String -> String -> Int -> Cmd Msg
fetchAdminModerationReports token stateFilter offset =
    let
        stateQuery =
            if String.trim stateFilter == "" then
                ""

            else
                "&state=" ++ Url.percentEncode (String.trim stateFilter)
    in
    authorizedRequest "GET" token ("/api/admin/moderation/reports?limit=" ++ String.fromInt selectorPageSize ++ "&offset=" ++ String.fromInt offset ++ stateQuery) Http.emptyBody (Http.expectJson AdminModerationReportsReceived Moderation.moderationReportsResponseDecoder)


triageModerationReport : String -> String -> String -> String -> Cmd Msg
triageModerationReport token reportID stateValue resolutionNote =
    authorizedRequest "POST"
        token
        ("/api/admin/moderation/reports/" ++ reportID ++ "/triage")
        (Http.jsonBody (Encode.object [ ( "state", Encode.string stateValue ), ( "resolution_note", Encode.string resolutionNote ) ]))
        (Http.expectJson AdminModerationReportTriaged Moderation.moderationReportResponseDecoder)


runPrivacyRetention : String -> Cmd Msg
runPrivacyRetention token =
    authorizedRequest "POST"
        token
        "/api/admin/privacy-retention/run"
        Http.emptyBody
        (Http.expectJson PrivacyRetentionRunReceived Privacy.privacyRetentionRunResponseDecoder)


resolveAdminPrivacyRequest : String -> String -> String -> Cmd Msg
resolveAdminPrivacyRequest token requestId resolutionNote =
    authorizedRequest "POST"
        token
        ("/api/admin/privacy-requests/" ++ requestId ++ "/resolve")
        (Http.jsonBody (Encode.object [ ( "resolution_note", Encode.string resolutionNote ) ]))
        (Http.expectJson AdminPrivacyRequestResolved Privacy.privacyRequestResponseDecoder)


fetchOrgTeams : String -> String -> Cmd Msg
fetchOrgTeams token organizationId =
    fetchOrgTeamsPage token organizationId "" 0


fetchOrgTeamsPage : String -> String -> String -> Int -> Cmd Msg
fetchOrgTeamsPage token organizationId queryText offset =
    if organizationId == "" then
        Cmd.none

    else
        authorizedRequest "GET" token (selectorQuery queryText offset ("/api/organizations/" ++ organizationId ++ "/teams")) Http.emptyBody (Http.expectJson OrgTeamsReceived Team.teamsResponseDecoder)


requestEmailVerification : String -> Cmd Msg
requestEmailVerification token =
    authorizedRequest "POST"
        token
        "/api/account/email-verification"
        (Http.jsonBody (Encode.object []))
        (Http.expectJson EmailVerificationRequested tokenDecoder)


confirmEmailVerification : String -> String -> Cmd Msg
confirmEmailVerification token accountToken =
    authorizedRequest "POST"
        token
        "/api/auth/email-verification/confirm"
        (Http.jsonBody (Encode.object [ ( "token", Encode.string accountToken ) ]))
        (Http.expectWhatever AccountActionReceived)


updateProfile : String -> String -> Cmd Msg
updateProfile token email =
    authorizedRequest "PATCH"
        token
        "/api/account/profile"
        (Http.jsonBody (Encode.object [ ( "email", Encode.string email ) ]))
        (Http.expectWhatever AccountActionReceived)


changePassword : String -> String -> String -> Cmd Msg
changePassword token current next =
    authorizedRequest "PATCH"
        token
        "/api/account/password"
        (Http.jsonBody (Encode.object [ ( "current_password", Encode.string current ), ( "new_password", Encode.string next ) ]))
        (Http.expectWhatever AccountActionReceived)


deactivateAccount : String -> Cmd Msg
deactivateAccount token =
    authorizedRequest "DELETE"
        token
        "/api/account"
        Http.emptyBody
        (Http.expectWhatever DeactivateAccountReceived)


requestPrivacy : String -> Privacy.PrivacyRequestKind -> Cmd Msg
requestPrivacy token kind =
    authorizedRequest "POST"
        token
        "/api/privacy-requests"
        (Http.jsonBody (Encode.object [ ( "kind", Privacy.privacyRequestKindEncoder kind ) ]))
        (Http.expectJson PrivacyRequestReceived Privacy.privacyRequestResponseDecoder)


reportTask : String -> String -> Moderation.ModerationReason -> String -> Cmd Msg
reportTask token taskId reason details =
    authorizedRequest "POST"
        token
        "/api/moderation/reports"
        (Http.jsonBody
            (Encode.object
                [ ( "subject_kind", Encode.string "task" )
                , ( "subject_id", Encode.string taskId )
                , ( "reason", Moderation.moderationReasonEncoder reason )
                , ( "details", Encode.string details )
                ]
            )
        )
        (Http.expectJson ModerationReportReceived Moderation.moderationReportResponseDecoder)


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

    else if List.isEmpty state.provisionMemberRoles then
        ( updateLoggedIn model (\current -> { current | provisionMemberMessage = Just "Select at least one role." }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | provisionMemberMessage = Nothing })
        , authorizedRequest "POST"
            state.accessToken
            ("/api/organizations/" ++ state.activeOrgId ++ "/members")
            (Http.jsonBody (Encode.object [ ( "email", Encode.string (String.trim state.provisionMemberEmail) ), ( "roles", Encode.list Encode.string state.provisionMemberRoles ) ]))
            (Http.expectWhatever ProvisionMemberReceived)
        )


updateMemberRolesCommand : Model -> LoggedInModel -> String -> List String -> ( Model, Cmd Msg )
updateMemberRolesCommand model state userId roles =
    if state.activeOrgId == "" || List.isEmpty roles then
        ( updateLoggedIn model (\current -> { current | provisionMemberMessage = Just "Select at least one role." }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | provisionMemberMessage = Nothing })
        , authorizedRequest "PATCH"
            state.accessToken
            ("/api/organizations/" ++ state.activeOrgId ++ "/members/" ++ userId ++ "/roles")
            (Http.jsonBody (Encode.object [ ( "roles", Encode.list Encode.string roles ) ]))
            (Http.expectJson UpdateMemberRolesReceived Organization.organizationMemberResponseDecoder)
        )


deactivateMemberCommand : Model -> LoggedInModel -> String -> ( Model, Cmd Msg )
deactivateMemberCommand model state userId =
    if state.activeOrgId == "" then
        ( updateLoggedIn model (\current -> { current | provisionMemberMessage = Just "Open an organization first." }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | provisionMemberMessage = Nothing })
        , authorizedRequest "PATCH"
            state.accessToken
            ("/api/organizations/" ++ state.activeOrgId ++ "/members/" ++ userId ++ "/deactivate")
            (Http.jsonBody (Encode.object []))
            (Http.expectWhatever DeactivateMemberReceived)
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
        ]


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


createRewardBody : String -> String -> List String -> Encode.Value
createRewardBody kind rawAmount collectibleIds =
    case String.toInt rawAmount of
        Just amount ->
            if kind == "credit" && amount > 0 then
                Encode.object [ ( "kind", Encode.string "credit" ), ( "credit_amount", Encode.int amount ), ( "collectible_ids", Encode.list Encode.string [] ) ]

            else if kind == "collectible" then
                Encode.object [ ( "kind", Encode.string "collectible" ), ( "credit_amount", Encode.int 0 ), ( "collectible_ids", Encode.list Encode.string collectibleIds ) ]

            else if kind == "bundle" && amount > 0 then
                Encode.object [ ( "kind", Encode.string "bundle" ), ( "credit_amount", Encode.int amount ), ( "collectible_ids", Encode.list Encode.string collectibleIds ) ]

            else
                Encode.object [ ( "kind", Encode.string "none" ), ( "credit_amount", Encode.int 0 ), ( "collectible_ids", Encode.list Encode.string [] ) ]

        Nothing ->
            if kind == "collectible" then
                Encode.object [ ( "kind", Encode.string "collectible" ), ( "credit_amount", Encode.int 0 ), ( "collectible_ids", Encode.list Encode.string collectibleIds ) ]

            else
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


agentRequestBody : String -> List Agent.AgentScope -> Encode.Value
agentRequestBody agentLabel scopes =
    Encode.object
        [ ( "label", Encode.string agentLabel )
        , ( "scopes", Encode.list Agent.agentScopeEncoder scopes )
        ]


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
    , reviewerAction = response.reviewerAction
    , responseSchemaJson = response.responseSchemaJSON
    , payloadKind = response.payloadKind
    , payloadJson = response.payloadJSON
    , attachments = response.attachments
    , createdBy = response.createdBy
    , seriesID = response.seriesID
    , taskType = response.taskType
    , referenceURL = response.referenceURL
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


seriesFromResult : Result Http.Error TaskSeries.TaskSeriesListResponse -> List TaskSeries.TaskSeriesResponse
seriesFromResult result =
    case result of
        Ok response ->
            response.series

        Err _ ->
            []


fetchSeriesList : String -> Cmd Msg
fetchSeriesList token =
    authorizedRequest "GET" token "/api/task-series" Http.emptyBody (Http.expectJson SeriesListReceived TaskSeries.taskSeriesListResponseDecoder)


fetchSeriesDetail : String -> String -> Cmd Msg
fetchSeriesDetail token seriesId =
    authorizedRequest "GET" token ("/api/task-series/" ++ seriesId) Http.emptyBody (Http.expectJson SeriesDetailReceived seriesDetailDecoder)


createSeriesCommand : Model -> LoggedInModel -> ( Model, Cmd Msg )
createSeriesCommand model state =
    if String.isEmpty (String.trim state.createSeriesTitle) then
        ( updateLoggedIn model (\current -> { current | seriesMessage = Just "A series title is required." }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | seriesMessage = Nothing })
        , authorizedRequest "POST"
            state.accessToken
            "/api/task-series"
            (Http.jsonBody (seriesBody state.createSeriesTitle state.createSeriesDescription))
            (Http.expectJson SeriesMutationReceived seriesDetailDecoder)
        )


updateSeriesCommand : Model -> LoggedInModel -> String -> ( Model, Cmd Msg )
updateSeriesCommand model state seriesId =
    if String.isEmpty (String.trim state.seriesRenameTitle) then
        ( updateLoggedIn model (\current -> { current | seriesMessage = Just "A series title is required." }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | seriesMessage = Nothing })
        , authorizedRequest "PATCH"
            state.accessToken
            ("/api/task-series/" ++ seriesId)
            (Http.jsonBody (seriesBody state.seriesRenameTitle state.seriesRenameDescription))
            (Http.expectJson SeriesMutationReceived seriesDetailDecoder)
        )


seriesStateCommand : String -> String -> String -> Cmd Msg
seriesStateCommand token seriesId action =
    authorizedRequest "POST"
        token
        ("/api/task-series/" ++ seriesId ++ "/" ++ action)
        (Http.jsonBody (Encode.object []))
        (Http.expectJson SeriesMutationReceived seriesDetailDecoder)


addSeriesTaskCommand : Model -> LoggedInModel -> String -> ( Model, Cmd Msg )
addSeriesTaskCommand model state seriesId =
    if String.isEmpty (String.trim state.addSeriesTaskId) then
        ( updateLoggedIn model (\current -> { current | seriesMessage = Just "A task ID is required." }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | seriesMessage = Nothing })
        , authorizedRequest "POST"
            state.accessToken
            ("/api/task-series/" ++ seriesId ++ "/tasks")
            (Http.jsonBody (Encode.object [ ( "task_id", Encode.string (String.trim state.addSeriesTaskId) ) ]))
            (Http.expectJson SeriesMutationReceived seriesDetailDecoder)
        )


removeSeriesTaskCommand : String -> String -> String -> Cmd Msg
removeSeriesTaskCommand token seriesId taskId =
    authorizedRequest "DELETE"
        token
        ("/api/task-series/" ++ seriesId ++ "/tasks/" ++ taskId)
        Http.emptyBody
        (Http.expectJson SeriesMutationReceived seriesDetailDecoder)


reorderSeriesCommand : String -> String -> List String -> Cmd Msg
reorderSeriesCommand token seriesId taskIds =
    authorizedRequest "POST"
        token
        ("/api/task-series/" ++ seriesId ++ "/reorder")
        (Http.jsonBody (Encode.object [ ( "task_ids", Encode.list Encode.string taskIds ) ]))
        (Http.expectJson SeriesMutationReceived seriesDetailDecoder)


addSeriesCommentCommand : Model -> LoggedInModel -> String -> ( Model, Cmd Msg )
addSeriesCommentCommand model state seriesId =
    if String.isEmpty (String.trim state.seriesCommentBody) then
        ( updateLoggedIn model (\current -> { current | seriesMessage = Just "A comment is required." }), Cmd.none )

    else
        ( updateLoggedIn model (\current -> { current | seriesMessage = Nothing })
        , authorizedRequest "POST"
            state.accessToken
            ("/api/task-series/" ++ seriesId ++ "/comments")
            (Http.jsonBody (Encode.object [ ( "body", Encode.string (String.trim state.seriesCommentBody) ) ]))
            (Http.expectJson SeriesCommentReceived TaskSeries.seriesCommentResponseDecoder)
        )


fetchOrganizationLedgerPage : String -> String -> Int -> Cmd Msg
fetchOrganizationLedgerPage token organizationId offset =
    authorizedRequest "GET" token ("/api/organizations/" ++ organizationId ++ "/credits/ledger?limit=" ++ String.fromInt selectorPageSize ++ "&offset=" ++ String.fromInt offset) Http.emptyBody (Http.expectJson OrgLedgerReceived Ledger.ledgerResponseDecoder)


fetchNotifications : String -> Int -> Cmd Msg
fetchNotifications token offset =
    authorizedRequest "GET" token ("/api/notifications?limit=" ++ String.fromInt selectorPageSize ++ "&offset=" ++ String.fromInt offset) Http.emptyBody (Http.expectJson NotificationsReceived Notification.notificationsResponseDecoder)


markNotificationRead : String -> String -> Cmd Msg
markNotificationRead token notificationId =
    authorizedRequest "POST" token ("/api/notifications/" ++ notificationId ++ "/read") Http.emptyBody (Http.expectJson NotificationReadReceived Notification.notificationResponseDecoder)


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
