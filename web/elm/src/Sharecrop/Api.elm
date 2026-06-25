module Sharecrop.Api exposing (..)

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
import Sharecrop.Labels exposing (assigneeScopeTag, participationUsesReservation)
import Sharecrop.Types exposing (..)


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
            Cmd.batch [ fetchCollectibles token, fetchCollectibleCatalog token, fetchTasks token "" ]

        OrganizationsPage ->
            fetchOrganizations token

        OrganizationDetailPage organizationId ->
            Cmd.batch [ fetchOrganizations token, loadOrganization token organizationId, fetchOrganizationCollectibles token organizationId ]

        UserDetailPage userId ->
            fetchUserProfile token userId

        UserWorkPage userId ->
            authorizedRequest "GET" token ("/api/users/" ++ userId ++ "/work") Http.emptyBody (Http.expectJson UserWorkReceived Task.tasksResponseDecoder)

        UserSubmissionsPage userId ->
            authorizedRequest "GET" token ("/api/users/" ++ userId ++ "/submissions") Http.emptyBody (Http.expectJson UserSubmissionsReceived Submission.submissionsResponseDecoder)

        CollectibleDetailPage _ ->
            fetchCollectibles token

        SeriesListPage ->
            fetchSeriesList token

        SeriesDetailPage seriesId ->
            fetchSeriesDetail token seriesId

        TeamDetailPage teamId ->
            Cmd.batch
                [ authorizedRequest "GET" token ("/api/teams/" ++ teamId) Http.emptyBody (Http.expectJson TeamDetailReceived Team.teamDetailResponseDecoder)
                , fetchTeamCollectibles token teamId
                ]


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
    Cmd.batch [ fetchPublicTaskDetail token taskId, fetchSubmissions token taskId, fetchReservations token taskId, fetchTaskComments token taskId ]


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
        , ( "response_schema_json", Encode.string (createSchemaString state) )
        , ( "payload", createPayloadBody state )
        , ( "task_type", Encode.string state.createTaskType )
        , ( "reference_url", Encode.string state.createReferenceURL )
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
    , payloadKind = response.payloadKind
    , payloadJson = response.payloadJSON
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
