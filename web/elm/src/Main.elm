port module Main exposing (main)

import Browser
import Browser.Events
import Browser.Navigation as Nav
import Dict
import File
import File.Select as FileSelect
import Http
import Sharecrop.Api as Api
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
import Sharecrop.Labels exposing (httpErrorLabel, participationPolicyTag)
import Sharecrop.Types exposing (..)
import Sharecrop.View as View
import Task as ElmTask
import Time
import Url exposing (Url)




main : Program Flags Model Msg
main =
    Browser.application
        { init = \flags url key -> ( initialModel flags key url, Api.postRefresh )
        , update = update
        , subscriptions =
            -- Access tokens expire after 15 minutes; rotate every 10 so a
            -- tab left open does not silently start failing every request.
            -- Only while logged in - the refresh endpoint would just 401
            -- on the auth screen. The 15-second poll keeps the unread badge
            -- (and the current page's live lists) fresh, and returning to a
            -- hidden tab refetches immediately instead of waiting a tick.
            \model ->
                case model.session of
                    LoggedIn _ ->
                        Sub.batch
                            [ Time.every (10 * 60 * 1000) SessionRefreshTick
                            , Time.every (15 * 1000) PollTick
                            , Browser.Events.onVisibilityChange VisibilityChanged
                            ]

                    LoggedOut ->
                        Sub.none
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
    , shauth = flags.shauth
    , key = key
    , route = pageFromUrl url
    , email = ""
    , password = ""
    , registerName = ""
    , resetEmail = ""
    , resetToken = ""
    , resetPassword = ""
    , authError = Nothing
    , authNotice = Nothing
    , sessionNotice = Nothing
    , session = LoggedOut
    }


emptyLoggedIn : Auth.AuthResponse -> LoggedInModel
emptyLoggedIn response =
    { accessToken = response.accessToken
    , subjectId = response.subjectID
    , username = response.username
    , displayName = response.displayName
    , now = Time.millisToPosix 0
    , isAdmin = response.role == "admin"
    , page = OverviewPage
    , openNavMenu = Nothing
    , balance = Nothing
    , entries = loadedNone
    , ledgerOffset = 0
    , ledgerNextOffset = 0
    , ledgerTotal = 0
    , unreadCount = 0
    , inboxUnreadOnly = False
    , activityEvents = []
    , activityCursor = ""
    , webhookSubscriptions = []
    , webhookURL = ""
    , webhookKinds = []
    , webhookAudience = Events.WebhookAudienceRecipient
    , webhookFilterTaskType = ""
    , webhookFilterMinReward = ""
    , webhookMessage = Nothing
    , newWebhookSecret = Nothing
    , activeWebhookDeliveriesID = Nothing
    , webhookDeliveries = []
    , createTitle = ""
    , createTitleInvalid = False
    , createDescription = ""
    , createDescriptionInvalid = False
    , createResponseSchema = "{\"kind\":\"freeform\"}"
    , createSchemaFields = []
    , createPayloadJson = ""
    , createRewardKind = "none"
    , createRewardAmount = ""
    , createRewardAmountInvalid = False
    , createRewardCollectibleIds = []
    , createVisibility = visibilityDefaultTag
    , createScopeUserId = ""
    , createScopeTeamId = ""
    , createScopeOrganizationId = ""
    , createAssigneeScope = Task.TaskAssigneeScopeUser
    , createParticipationPolicy = participationPolicyTag Task.TaskParticipationPolicyOpen
    , createReservationHours = "48"
    , createExpiresAt = ""
    , createAttachments = []
    , createMessage = Nothing
    , fundTaskId = ""
    , fundAmount = ""
    , fundOrganizationId = ""
    , fundMessage = Nothing
    , fundNonce = 0
    , sendRecipientKind = "user"
    , sendRecipientId = ""
    , sendAmount = ""
    , sendNote = ""
    , sendKey = ""
    , sendMessage = Nothing
    , tasks = loadedNone
    , taskStateFilter = []
    , taskListOffset = 0
    , taskListNextOffset = 0
    , taskListTotal = 0
    , taskListQuery = ""
    , taskListTypeFilter = ""
    , taskListSort = "newest"
    , agentLabel = ""
    , agentScopes = [ Agent.AgentScopeTasksRead, Agent.AgentScopeSubmissionsWrite ]
    , agentExpiresHours = ""
    , credentials = loadedNone
    , newCredential = Nothing
    , agentMessage = Nothing
    , discoveryTasks = loadedNone
    , discoveryIncludeReserved = False
    , discoveryFundedOnly = False
    , discoveryOffset = 0
    , discoveryNextOffset = 0
    , discoveryTotal = 0
    , discoveryQuery = ""
    , detail = Nothing
    , detailError = Nothing
    , reservations = []
    , reservationOrganizationId = ""
    , reservationTeamId = ""
    , reservationMessage = Nothing
    , reservationSecret = Nothing
    , submissions = []
    , submitInput = ""
    , submitFieldValues = Dict.empty
    , submitRawMode = False
    , submitAttachments = []
    , submitMessage = Nothing
    , moderationReason = Moderation.ModerationReasonPolicy
    , moderationSubject = ReportAboutTask
    , moderationDetails = ""
    , moderationMessage = Nothing
    , reviewNote = ""
    , reviewPartialCredit = ""
    , reviewTip = ""
    , reviewTipCollectibleId = ""
    , reviewBan = Ledger.BanSelectionNone
    , reviewMessage = Nothing
    , collectibles = loadedNone
    , collectibleName = ""
    , collectibleKind = Collectible.CollectibleKindBadge
    , collectiblePolicy = Collectible.CollectibleTransferPolicyNonTransferableExceptPayout
    , collectibleMessage = Nothing
    , awardTaskId = ""
    , awardMessage = Nothing
    , awardDefaultMessage = Nothing
    , collectibleCatalog = loadedNone
    , awardRecipientKind = "user"
    , awardRecipientId = ""
    , transferRecipientId = ""
    , transferMessage = Nothing
    , openSendCollectibleID = Nothing
    , sendCollectibleTargetKind = "user"
    , withdrawMessage = Nothing
    , catalogSlug = ""
    , catalogName = ""
    , catalogKind = Collectible.CollectibleKindBadge
    , catalogPolicy = Collectible.CollectibleTransferPolicyNonTransferableExceptPayout
    , catalogArt = ""
    , catalogMaxEditions = ""
    , catalogMessage = Nothing
    , organizations = loadedNone
    , createOrgName = ""
    , orgMessage = Nothing
    , activeOrgId = ""
    , orgBalance = Nothing
    , orgLedger = loadedNone
    , orgLedgerOffset = 0
    , orgLedgerNextOffset = 0
    , orgLedgerTotal = 0
    , orgAuditEvents = []
    , orgAuditMessage = Nothing
    , orgTeams = loadedNone
    , standaloneTeams = loadedNone
    , createTeamName = ""
    , createTeamMessage = Nothing
    , orgMembers = loadedNone
    , orgTasks = []
    , orgTaskQuery = ""
    , orgTaskFilter = ""
    , orgTaskTypeFilter = ""
    , orgTaskSort = "newest"
    , orgTaskOffset = 0
    , orgTaskNextOffset = 0
    , orgTaskTotal = 0
    , orgTaskMessage = Nothing
    , orgTaskSavedViewName = ""
    , orgTaskSavedViews = []
    , orgCollectibles = []
    , orgCollectiblesMessage = Nothing
    , awardOrgCollectibleRecipientId = ""
    , awardOrgCollectibleMessage = Nothing
    , orgSendCollectibleRecipientId = ""
    , orgSendCollectibleMessage = Nothing
    , orgCredentials = loadedNone
    , orgCredentialLabel = ""
    , orgCredentialScopes = [ Agent.AgentScopeOrgRead ]
    , orgCredentialExpiresHours = ""
    , newOrgCredential = Nothing
    , orgCredentialMessage = Nothing
    , teamCollectibles = []
    , teamCollectiblesMessage = Nothing
    , userProfile = Nothing
    , userProfileError = Nothing
    , userWork = loadedNone
    , userSubmissions = loadedNone
    , userSubmissionsOffset = 0
    , userSubmissionsNextOffset = 0
    , userSubmissionsTotal = 0
    , pendingRevisionTaskID = Nothing
    , pendingRevisionResponse = ""
    , seriesDetail = Nothing
    , seriesDetailError = Nothing
    , seriesList = loadedNone
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
    , teamWorkNextOffset = 0
    , teamWorkTotal = 0
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
    , taskActionMessage = Nothing
    , userAgentToken = Nothing
    , accountProfile = Nothing
    , displayNameDraft = ""
    , accountEmail = ""
    , currentPassword = ""
    , newPassword = ""
    , emailVerificationToken = ""
    , emailVerificationInput = ""
    , accountMessage = Nothing
    , deactivateConfirming = False
    , myPrivacyRequests = []
    , userDirectory = []
    , userDirectoryQuery = ""
    , userDirectoryOffset = 0
    , userDirectoryNextOffset = 0
    , organizationQuery = ""
    , organizationOffset = 0
    , organizationNextOffset = 0
    , standaloneTeamQuery = ""
    , standaloneTeamOffset = 0
    , standaloneTeamNextOffset = 0
    , orgTeamQuery = ""
    , orgTeamOffset = 0
    , orgTeamNextOffset = 0
    , operations = Nothing
    , auditEvents = []
    , auditEventsOffset = 0
    , auditEventsNextOffset = 0
    , platformAdmins = []
    , platformAdminsOffset = 0
    , platformAdminsNextOffset = 0
    , adminSelectedUserId = ""
    , adminModerationReports = []
    , adminModerationStateFilter = "open"
    , adminModerationOffset = 0
    , adminModerationNextOffset = 0
    , adminModerationTotal = 0
    , adminModerationResolutionNote = ""
    , adminPrivacyRequests = []
    , adminPrivacyOffset = 0
    , adminPrivacyNextOffset = 0
    , adminPrivacyResolutionNote = ""
    , adminRetentionRedactedFieldCount = Nothing
    , grantTargetKind = "user"
    , grantTargetId = ""
    , grantAmount = ""
    , grantNote = ""
    , grantKey = ""
    , grantMessage = Nothing
    , auditActionFilter = ""
    , auditSubjectKindFilter = ""
    , auditSubjectIDFilter = ""
    , adminMessage = Nothing
    , notifications = []
    , notificationsOffset = 0
    , notificationsNextOffset = 0
    , notificationsTotal = 0
    , inboxMessage = Nothing
    }


issuedCredentialSecret : String -> Maybe String -> Maybe String
issuedCredentialSecret rawSecret previous =
    if rawSecret == "" then
        previous

    else
        Just rawSecret


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


replaceModerationReport : Moderation.ModerationReportResponse -> List Moderation.ModerationReportResponse -> List Moderation.ModerationReportResponse
replaceModerationReport replacement reports =
    List.map
        (\report ->
            if report.id == replacement.id then
                replacement

            else
                report
        )
        reports


replacePlatformAdmin : Admin.PlatformAdminResponse -> List Admin.PlatformAdminResponse -> List Admin.PlatformAdminResponse
replacePlatformAdmin replacement admins =
    if List.any (\admin -> admin.userID == replacement.userID) admins then
        List.map
            (\admin ->
                if admin.userID == replacement.userID then
                    replacement

                else
                    admin
            )
            admins

    else
        replacement :: admins


removePlatformAdmin : String -> List Admin.PlatformAdminResponse -> List Admin.PlatformAdminResponse
removePlatformAdmin userID admins =
    List.filter (\admin -> admin.userID /= userID) admins


attachmentMaxBytes : Int
attachmentMaxBytes =
    500 * 1024


attachmentMaxCount : Int
attachmentMaxCount =
    5


allowedAttachmentTypes : List String
allowedAttachmentTypes =
    [ "image/png", "image/jpeg", "image/gif", "image/webp", "text/plain", "application/json", "application/pdf" ]


selectAttachment : (File.File -> Msg) -> Cmd Msg
selectAttachment toMsg =
    FileSelect.file allowedAttachmentTypes toMsg


readCreateAttachment : File.File -> Cmd Msg
readCreateAttachment file =
    readAttachment file CreateAttachmentSelected CreateAttachmentRejected


readSubmitAttachment : File.File -> Cmd Msg
readSubmitAttachment file =
    readAttachment file SubmitAttachmentSelected SubmitAttachmentRejected


readAttachment : File.File -> (String -> String -> Int -> String -> Msg) -> (String -> Msg) -> Cmd Msg
readAttachment file success rejected =
    let
        contentType =
            File.mime file

        sizeBytes =
            File.size file
    in
    if sizeBytes > attachmentMaxBytes then
        Cmd.batch [ ElmTask.perform identity (ElmTask.succeed (rejected "Attachment must be under 500 KiB.")) ]

    else if not (List.member contentType allowedAttachmentTypes) then
        Cmd.batch [ ElmTask.perform identity (ElmTask.succeed (rejected "Attachment type is not allowed.")) ]

    else
        File.toUrl file
            |> ElmTask.perform (success (File.name file) contentType sizeBytes)


removeAt : Int -> List a -> List a
removeAt index values =
    values
        |> List.indexedMap Tuple.pair
        |> List.filter (\( currentIndex, _ ) -> currentIndex /= index)
        |> List.map Tuple.second


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
    enterPage page { state | page = page }


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
            -- Discovery's content lives on the Tasks hub now; redirect anyone
            -- with an old bookmark or link rather than 404ing them.
            TasksPage

        [ "funding" ] ->
            FundingPage

        [ "agents" ] ->
            AgentsPage

        [ "collectibles" ] ->
            CollectiblesPage

        [ "collectibles", collectibleId ] ->
            CollectibleDetailPage collectibleId

        [ "series" ] ->
            -- Series' content lives on the Tasks hub now; redirect anyone
            -- with an old bookmark or link rather than 404ing them.
            TasksPage

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
-- a deep link or back/forward leaves the model consistent with the URL. Also
-- closes any open nav-bar dropdown: its floating panel has no page-content
-- reason to still be visible once the route it linked to has loaded, and
-- left open it would sit over the new page and intercept clicks.
enterPage : Page -> LoggedInModel -> LoggedInModel
enterPage page state =
    let
        nextState =
            enterPageFields page state
    in
    { nextState | openNavMenu = Nothing }


enterPageFields : Page -> LoggedInModel -> LoggedInModel
enterPageFields page state =
    case page of
        TasksPage ->
            -- The Tasks hub embeds My tasks, Discover public tasks, My
            -- submissions, and Series all on one page, so entering it resets
            -- every one of those sections' filters/offsets, not just the
            -- My-tasks ones.
            { state
                | page = page
                , taskStateFilter = []
                , taskListOffset = 0
                , taskListQuery = ""
                , taskListTypeFilter = ""
                , taskListSort = "newest"
                , discoveryIncludeReserved = False
                , discoveryFundedOnly = False
                , discoveryOffset = 0
                , discoveryQuery = ""
                , userSubmissions = loadedNone
                , userSubmissionsOffset = 0
                , seriesMessage = Nothing
            }

        OrganizationDetailPage organizationId ->
            { state | page = page, activeOrgId = organizationId, orgBalance = Nothing, orgLedger = loadedNone, orgLedgerOffset = 0, orgAuditEvents = [], orgAuditMessage = Nothing, orgTeams = loadedNone, orgMembers = loadedNone, orgTasks = [], orgTaskQuery = "", orgTaskFilter = "", orgTaskTypeFilter = "", orgTaskSort = "newest", orgTaskOffset = 0, orgTaskMessage = Nothing, orgCollectibles = [], orgCollectiblesMessage = Nothing, awardOrgCollectibleRecipientId = "", awardOrgCollectibleMessage = Nothing, orgSendCollectibleRecipientId = "", orgSendCollectibleMessage = Nothing, sendRecipientKind = "user", sendRecipientId = "", sendAmount = "", sendNote = "", sendKey = "", sendMessage = Nothing, orgTeamMessage = Nothing, provisionMemberRoles = [ "member" ], provisionMemberMessage = Nothing }

        UserDetailPage _ ->
            { state | page = page, userProfile = Nothing, userProfileError = Nothing, accountProfile = Nothing, displayNameDraft = "" }

        UserWorkPage _ ->
            { state | page = page, userWork = loadedNone }

        UserSubmissionsPage _ ->
            { state | page = page, userSubmissions = loadedNone, userSubmissionsOffset = 0 }

        SeriesDetailPage _ ->
            { state | page = page, seriesDetail = Nothing, seriesDetailError = Nothing, seriesMessage = Nothing, addSeriesTaskId = "", seriesCommentBody = "", seriesRenameTitle = "", seriesRenameDescription = "" }

        TeamDetailPage _ ->
            { state | page = page, teamDetail = Nothing, teamDetailError = Nothing, teamWork = [], teamWorkQuery = "", teamWorkFilter = "", teamWorkTypeFilter = "", teamWorkSort = "newest", teamWorkOffset = 0, teamWorkMessage = Nothing, teamCollectibles = [], teamCollectiblesMessage = Nothing, teamMemberEmail = "", teamMemberMessage = Nothing }

        AdminPage ->
            { state | page = page, operations = Nothing, auditEvents = [], auditEventsOffset = 0, platformAdmins = [], platformAdminsOffset = 0, adminSelectedUserId = "", adminModerationReports = [], adminModerationStateFilter = "open", adminModerationOffset = 0, adminModerationResolutionNote = "", adminPrivacyRequests = [], adminPrivacyOffset = 0, adminPrivacyResolutionNote = "", adminRetentionRedactedFieldCount = Nothing, grantTargetKind = "user", grantTargetId = "", grantAmount = "", grantNote = "", grantKey = "", grantMessage = Nothing, auditActionFilter = "", auditSubjectKindFilter = "", auditSubjectIDFilter = "", adminMessage = Nothing }

        OverviewPage ->
            -- The feed cursor and rows survive route changes: a revisit
            -- resumes after the last event already shown (routeLoadCmd passes
            -- the kept cursor) instead of re-reading the stream from its
            -- start. appendActivityEvents dedupes and caps the render list.
            -- The send-credits panel resets so a stale confirmation or
            -- half-typed draft does not reappear on return.
            { state | page = page, sendRecipientKind = "user", sendRecipientId = "", sendAmount = "", sendNote = "", sendKey = "", sendMessage = Nothing }

        AgentsPage ->
            { state | page = page, webhookURL = "", webhookKinds = [], webhookAudience = Events.WebhookAudienceRecipient, webhookFilterTaskType = "", webhookFilterMinReward = "", webhookMessage = Nothing, newWebhookSecret = Nothing, activeWebhookDeliveriesID = Nothing, webhookDeliveries = [] }

        InboxPage ->
            { state | page = page, notifications = [], notificationsOffset = 0, notificationsNextOffset = 0, inboxUnreadOnly = False, inboxMessage = Nothing }

        CollectibleDetailPage _ ->
            { state | page = page, transferMessage = Nothing, transferRecipientId = "" }

        TaskDetailPage taskId ->
            -- Clear the previous task's detail substate so a task->task link does
            -- not briefly show the prior task's badges, submissions, or comments.
            -- Review form fields are reset here too so the prior submission's
            -- note / partial credit / tip / ban does not carry over to the next.
            -- fundTaskId/awardTaskId are synced to the task being viewed so the
            -- inline "Fund this task" and "Award a collectible" panels (see
            -- ownerControlsCard) always target *this* task when submitted,
            -- regardless of whatever task was last selected on the standalone
            -- Funding/Collectibles pages.
            { state | page = page, detail = Nothing, detailError = Nothing, reservations = [], reservationOrganizationId = "", reservationTeamId = "", reservationMessage = Nothing, reservationSecret = Nothing, submissions = [], submitInput = revisionDraftFor taskId state, submitFieldValues = Dict.empty, submitRawMode = revisionDraftFor taskId state /= "", submitAttachments = [], submitMessage = Nothing, moderationReason = Moderation.ModerationReasonPolicy, moderationSubject = ReportAboutTask, moderationDetails = "", moderationMessage = Nothing, reviewNote = "", reviewPartialCredit = "", reviewTip = "", reviewTipCollectibleId = "", reviewBan = Ledger.BanSelectionNone, reviewMessage = Nothing, taskComments = [], taskCommentBody = "", taskCommentMessage = Nothing, submissionComments = [], activeSubmissionCommentsID = Nothing, submissionCommentBody = "", submissionCommentMessage = Nothing, taskAgentToken = Nothing, taskActionMessage = Nothing, pendingRevisionTaskID = Nothing, pendingRevisionResponse = "", fundTaskId = taskId, fundAmount = "", fundMessage = Nothing, fundNonce = state.fundNonce + 1, awardTaskId = taskId, awardMessage = Nothing }

        CollectiblesPage ->
            -- Reset the award / mint / transfer messages and drafts so a stale
            -- "Awarded" note or prefilled recipient does not reappear on return.
            { state | page = page, awardMessage = Nothing, awardDefaultMessage = Nothing, collectibleMessage = Nothing, transferMessage = Nothing, collectibleName = "", awardRecipientId = "", awardTaskId = "", openSendCollectibleID = Nothing, sendCollectibleTargetKind = "user", transferRecipientId = "", withdrawMessage = Nothing, catalogSlug = "", catalogName = "", catalogArt = "", catalogMaxEditions = "", catalogMessage = Nothing }

        CreateTaskPage ->
            -- Clear a half-finished draft and any stale create message on entry.
            { state | page = page, createTitle = "", createTitleInvalid = False, createDescription = "", createDescriptionInvalid = False, createResponseSchema = "{\"kind\":\"freeform\"}", createSchemaFields = [], createPayloadJson = "", createRewardKind = "none", createRewardAmount = "", createRewardAmountInvalid = False, createRewardCollectibleIds = [], createAttachments = [], createMessage = Nothing, createTaskType = "general", createReferenceURL = "", createParticipationPolicy = participationPolicyTag Task.TaskParticipationPolicyOpen, createReservationHours = "48", createExpiresAt = "" }

        FundingPage ->
            { state | page = page, fundMessage = Nothing }

        _ ->
            { state | page = page }


submissionOutcomeNote : Submission.SubmissionCreatedResponse -> Note
submissionOutcomeNote created =
    case created.submission.state of
        Submission.SubmissionStateInvalid ->
            FailureNote (View.submitSuccessLabel created)

        _ ->
            SuccessNote (View.submitSuccessLabel created)


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

        RegisterNameChanged value ->
            ( { model | registerName = value }, Cmd.none )

        NowReceived posix ->
            ( Api.updateLoggedIn model (\state -> { state | now = posix }), Cmd.none )

        RegisterClicked ->
            ( { model | authError = Nothing, authNotice = Nothing, sessionNotice = Nothing }, Api.postAuth "/api/auth/register" model )

        LoginClicked ->
            ( { model | authError = Nothing, authNotice = Nothing, sessionNotice = Nothing }, Api.postAuth "/api/auth/login" model )

        GuestClicked ->
            ( { model | authError = Nothing, authNotice = Nothing, sessionNotice = Nothing }, Api.postGuest )

        AuthReceived (Ok response) ->
            let
                state =
                    loggedInForPage response model.route
            in
            -- Load the current route's page data too, exactly like the
            -- refresh path below: someone who logs in while sitting on a
            -- deep link (a task, org, or profile URL) would otherwise stay
            -- on "Loading..." forever, since loadAfterAuth only fetches the
            -- shared dashboard data.
            ( { model | password = "", registerName = "", authError = Nothing, sessionNotice = Nothing, session = LoggedIn { state | accountEmail = model.email } }
            , Cmd.batch [ Api.loadAfterAuth response.accessToken, Api.routeLoadCmd response.accessToken response.subjectID "" model.route, ElmTask.perform NowReceived Time.now ]
            )

        AuthReceived (Err error) ->
            ( { model | authError = Just (httpErrorLabel error) }, Cmd.none )

        RefreshReceived (Ok response) ->
            ( { model | sessionNotice = Nothing, session = LoggedIn (loggedInForPage response model.route) }
            , Cmd.batch [ Api.loadAfterAuth response.accessToken, Api.routeLoadCmd response.accessToken response.subjectID "" model.route, ElmTask.perform NowReceived Time.now ]
            )

        RefreshReceived (Err _) ->
            if model.shauth then
                ( model, Nav.load "/api/auth/shauth" )

            else
                ( model, Cmd.none )

        SessionRefreshTick _ ->
            ( model, Api.postSessionRefresh )

        SessionEnded ->
            -- Some signed-in request came back `unauthenticated`: the token
            -- is expired, revoked, or tampered with. Clear the session and
            -- land on the auth screen with a factual notice instead of
            -- letting every later click fail into fake empty states. The
            -- notice is skipped when there is no session to end (a stray
            -- late response after an ordinary logout).
            case model.session of
                LoggedIn _ ->
                    if model.shauth then
                        ( { model | session = LoggedOut, authError = Nothing }
                        , Nav.load "/api/auth/shauth"
                        )

                    else
                        ( { model | session = LoggedOut, authError = Nothing, sessionNotice = Just Api.sessionEndedNotice }
                        , Nav.pushUrl model.key "#/"
                        )

                LoggedOut ->
                    ( model, Cmd.none )

        SessionRefreshed (Ok response) ->
            -- Swap only the rotated token (and any changed role) into the
            -- existing state: rebuilding the page here would wipe whatever
            -- the user is in the middle of typing.
            ( Api.updateLoggedIn model (\state -> { state | accessToken = response.accessToken, subjectId = response.subjectID, username = response.username, displayName = response.displayName, isAdmin = response.role == "admin" }), Cmd.none )

        SessionRefreshed (Err _) ->
            -- The refresh cookie is gone or revoked: the session is over.
            -- Land on the auth screen with an explanation instead of letting
            -- every later click fail into fake empty states.
            if model.shauth then
                ( { model | session = LoggedOut, authError = Nothing }
                , Nav.load "/api/auth/shauth"
                )

            else
                ( { model | session = LoggedOut, authError = Just "Your session expired. Log in again." }
                , Nav.pushUrl model.key "#/"
                )

        PasswordResetEmailChanged value ->
            ( { model | resetEmail = value }, Cmd.none )

        PasswordResetTokenChanged value ->
            ( { model | resetToken = value }, Cmd.none )

        PasswordResetPasswordChanged value ->
            ( { model | resetPassword = value }, Cmd.none )

        RequestPasswordResetClicked ->
            ( { model | authError = Nothing, authNotice = Nothing }, Api.requestPasswordReset model )

        ConfirmPasswordResetClicked ->
            ( { model | authError = Nothing, authNotice = Nothing }, Api.confirmPasswordReset model )

        PasswordResetRequested (Ok token) ->
            if token == "" then
                -- This deployment does not send email: the token went to the
                -- server operator's log. "Instructions sent" would leave the
                -- user waiting for an email that never comes.
                ( { model | authNotice = Just "Password reset requested. This deployment does not send email - ask an operator for the reset token from the server log, then enter it below." }, Cmd.none )

            else
                ( { model | resetToken = token, authNotice = Just "Password reset token created and filled in below." }, Cmd.none )

        PasswordResetRequested (Err error) ->
            ( { model | authError = Just (httpErrorLabel error) }, Cmd.none )

        PasswordResetConfirmed (Ok ()) ->
            ( { model | resetPassword = "", resetToken = "", authNotice = Just "Password reset. Log in with the new password." }, Cmd.none )

        PasswordResetConfirmed (Err error) ->
            ( { model | authError = Just (httpErrorLabel error) }, Cmd.none )

        BalanceReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | balance = Api.balanceFromResult result }), Cmd.none )

        LedgerReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | entries = Api.loadedFromResult .entries result, ledgerNextOffset = nextOffsetFromResult .nextOffset result, ledgerTotal = totalFromResult result }), Cmd.none )

        SendRecipientKindChanged value ->
            -- Changing any send input starts a new send intent, so the
            -- idempotency key is discarded and reminted on the next submit.
            ( Api.updateLoggedIn model (\state -> { state | sendRecipientKind = value, sendRecipientId = "", sendKey = "" }), Cmd.none )

        SendRecipientIdChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | sendRecipientId = value, sendKey = "" }), Cmd.none )

        SendAmountChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | sendAmount = value, sendKey = "" }), Cmd.none )

        SendNoteChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | sendNote = value, sendKey = "" }), Cmd.none )

        SendCreditsClicked ->
            -- The key is minted on the first submit of this send intent and
            -- reused by retries (network timeouts), so the server dedupes
            -- instead of double-sending; editing any field clears it.
            Api.withSession model (\state -> Api.sendCreditsCommand model state (mintSendKey state))

        OrgSendCreditsClicked ->
            Api.withSession model (\state -> Api.orgSendCreditsCommand model state (mintSendKey state))

        CreditsSentReceived recipientLabel (Ok sent) ->
            Api.withSession model
                (\state ->
                    ( Api.updateLoggedIn model
                        (\current ->
                            { current
                                | sendMessage = Just (SuccessNote ("Sent " ++ String.fromInt sent.amount ++ " credits to " ++ recipientLabel ++ "."))
                                , sendAmount = ""
                                , sendNote = ""
                                , sendKey = ""
                            }
                        )
                    , Cmd.batch
                        (Api.refreshBalanceAndLedger model
                            :: (case state.page of
                                    OrganizationDetailPage organizationId ->
                                        [ Api.loadOrganization state.accessToken organizationId ]

                                    _ ->
                                        []
                               )
                        )
                    )
                )

        CreditsSentReceived _ (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | sendMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        PreviousLedgerPageClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            max 0 (state.ledgerOffset - Api.selectorPageSize)
                    in
                    ( Api.updateLoggedIn model (\current -> { current | ledgerOffset = offset }), Api.fetchLedger state.accessToken offset )
                )

        NextLedgerPageClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            state.ledgerOffset + Api.selectorPageSize
                    in
                    ( Api.updateLoggedIn model (\current -> { current | ledgerOffset = offset }), Api.fetchLedger state.accessToken offset )
                )

        TasksReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | tasks = Api.loadedFromResult .tasks result, taskListNextOffset = nextOffsetFromResult .nextOffset result, taskListTotal = totalFromResult result }), Cmd.none )

        TaskStateFilterToggled value ->
            let
                toggle current =
                    if List.member value current then
                        List.filter ((/=) value) current

                    else
                        value :: current

                updated =
                    Api.updateLoggedIn model (\state -> { state | taskStateFilter = toggle state.taskStateFilter, taskListOffset = 0 })
            in
            Api.withSession updated (\state -> ( updated, Api.fetchTasks state.accessToken state.taskStateFilter state.taskListTypeFilter state.taskListSort 0 ))

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
            ( Api.updateLoggedIn model (\state -> { state | createTitle = value, createTitleInvalid = state.createTitleInvalid && String.isEmpty (String.trim value) }), Cmd.none )

        CreateDescriptionChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | createDescription = value, createDescriptionInvalid = state.createDescriptionInvalid && String.isEmpty (String.trim value) }), Cmd.none )

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
            ( Api.updateLoggedIn model (\state -> { state | createRewardKind = value, createRewardCollectibleIds = [], createRewardAmountInvalid = state.createRewardAmountInvalid && Api.rewardAmountMissing value state.createRewardAmount }), Cmd.none )

        CreateRewardAmountChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | createRewardAmount = value, createRewardAmountInvalid = state.createRewardAmountInvalid && Api.rewardAmountMissing state.createRewardKind value }), Cmd.none )

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

        CreateExpiresAtChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | createExpiresAt = value }), Cmd.none )

        PickCreateAttachmentClicked ->
            Api.withSession model
                (\state ->
                    if List.length state.createAttachments >= attachmentMaxCount then
                        ( Api.updateLoggedIn model (\current -> { current | createMessage = Just (FailureNote "Attach up to 5 files.") }), Cmd.none )

                    else
                        ( model, selectAttachment CreateAttachmentFileChosen )
                )

        CreateAttachmentFileChosen file ->
            ( model, readCreateAttachment file )

        CreateAttachmentSelected name contentType sizeBytes dataURL ->
            ( Api.updateLoggedIn model (\state -> { state | createAttachments = state.createAttachments ++ [ { name = name, contentType = contentType, sizeBytes = sizeBytes, dataURL = dataURL } ], createMessage = Nothing }), Cmd.none )

        CreateAttachmentRejected message ->
            ( Api.updateLoggedIn model (\state -> { state | createMessage = Just (FailureNote message) }), Cmd.none )

        RemoveCreateAttachmentClicked index ->
            ( Api.updateLoggedIn model (\state -> { state | createAttachments = removeAt index state.createAttachments }), Cmd.none )

        CreateTaskClicked ->
            Api.withSession model (\state -> Api.createTaskCommand model state)

        CreateTaskReceived (Ok created) ->
            ( Api.updateLoggedIn model
                (\state ->
                    enterPage (TaskDetailPage created.id)
                        { state
                            | createTitle = ""
                            , createTitleInvalid = False
                            , createDescription = ""
                            , createDescriptionInvalid = False
                            , createResponseSchema = "{\"kind\":\"freeform\"}"
                            , createSchemaFields = []
                            , createPayloadJson = ""
                            , createTaskType = "general"
                            , createReferenceURL = ""
                            , createRewardCollectibleIds = []
                            , createAttachments = []
                            , createParticipationPolicy = participationPolicyTag Task.TaskParticipationPolicyOpen
                            , createReservationHours = "48"
                            , createExpiresAt = ""
                            , createMessage = Just (SuccessNote ("Created task " ++ created.id))
                        }
                )
            , Cmd.batch [ Api.refreshTasksAndLedger model, Nav.pushUrl model.key ("#/tasks/" ++ created.id) ]
            )

        CreateTaskReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | createMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        CredentialsReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | credentials = Api.loadedFromResult .credentials result }), Cmd.none )

        FundTaskIdChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | fundTaskId = value }), Cmd.none )

        FundAmountChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | fundAmount = value }), Cmd.none )

        FundOrganizationIdChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | fundOrganizationId = value }), Cmd.none )

        FundClicked ->
            -- fundNonce is deliberately NOT bumped here: it identifies one
            -- funding *intent*, not one click. Reusing the same idempotency
            -- key across retries (e.g. the user clicking again after a
            -- network timeout that may have actually reached the server)
            -- lets the server dedupe instead of double-charging. It only
            -- advances when a new task is opened (see TaskDetailPage) or a
            -- fund succeeds, both of which start a genuinely new intent.
            Api.withSession model (\state -> Api.fundTaskCommand model state)

        FundReceived (Ok fund) ->
            ( Api.updateLoggedIn model (\state -> { state | fundMessage = Just (SuccessNote (View.fundSuccessLabel fund)), fundNonce = state.fundNonce + 1 }), Api.refreshLedgerAndTaskDetail model )

        FundReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | fundMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        OpenTaskClicked taskId ->
            Api.withSession model (\state -> ( model, Api.postOpenTask state.accessToken taskId ))

        OpenTaskReceived (Ok detail) ->
            ( Api.updateLoggedIn model (\state -> { state | detail = Just detail, taskActionMessage = Just (SuccessNote "Task opened.") })
            , Api.refreshTasksAndDiscovery model
            )

        OpenTaskReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | taskActionMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        UnpublishTaskClicked taskId ->
            Api.withSession model (\state -> ( model, Api.postUnpublishTask state.accessToken taskId ))

        UnpublishTaskReceived (Ok detail) ->
            ( Api.updateLoggedIn model (\state -> { state | detail = Just detail, taskActionMessage = Just (SuccessNote "Task moved back to draft.") })
            , Api.refreshTasksAndDiscovery model
            )

        UnpublishTaskReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | taskActionMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        RefundTaskClicked taskId ->
            Api.withSession model (\state -> ( model, Api.postRefundTask state.accessToken taskId ))

        RefundTaskReceived (Ok _) ->
            -- Neutral wording: the same endpoint backs the owner's Reclaim and
            -- the active implementor's Refund, so the confirmation reads
            -- correctly for both ("returned" to whoever funded it - always the
            -- owner).
            ( Api.updateLoggedIn model (\state -> { state | taskActionMessage = Just (SuccessNote "Reward returned and the task was cancelled.") })
            , Cmd.batch [ Api.refreshTasksAndLedger model, Api.refreshAfterAccept model ]
            )

        RefundTaskReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | taskActionMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        CancelTaskClicked taskId ->
            Api.withSession model (\state -> ( model, Api.postCancelTask state.accessToken taskId ))

        CancelTaskReceived (Ok detail) ->
            -- Cancelling a funded task returns its allocated credits to the
            -- owner's spendable balance, so refresh the wallet + ledger too,
            -- not just the task lists (matches Fund/Refund/Accept).
            ( Api.updateLoggedIn model (\state -> { state | detail = Just detail, taskActionMessage = Just (SuccessNote "Task cancelled.") })
            , Cmd.batch [ Api.refreshTasksAndDiscovery model, Api.refreshBalanceAndLedger model ]
            )

        CancelTaskReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | taskActionMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        RefundCollectibleRewardClicked taskId ->
            Api.withSession model (\state -> ( model, Api.postRefundCollectibleReward state.accessToken taskId ))

        RefundCollectibleRewardReceived (Ok _) ->
            ( Api.updateLoggedIn model (\state -> { state | taskActionMessage = Just (SuccessNote "Collectible reward returned and the task was cancelled.") })
            , Cmd.batch [ Api.refreshAfterAccept model, Api.refreshCollectibles model ]
            )

        RefundCollectibleRewardReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | taskActionMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        AgentLabelChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | agentLabel = value }), Cmd.none )

        ToggleScope scope ->
            ( Api.updateLoggedIn model (\state -> { state | agentScopes = Api.toggleScope scope state.agentScopes }), Cmd.none )

        AgentScopePresetClicked scopes ->
            -- A preset replaces the current selection with the recommended
            -- set; the checkboxes stay editable afterwards.
            ( Api.updateLoggedIn model (\state -> { state | agentScopes = scopes }), Cmd.none )

        AgentExpiresHoursChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | agentExpiresHours = value }), Cmd.none )

        CreateAgentClicked ->
            Api.withSession model (\state -> Api.createAgentCommand model state)

        AgentExpiresAtResolved now ->
            Api.withSession model
                (\state ->
                    ( model, Api.postAgent state.accessToken state.agentLabel state.agentScopes (Api.expiresAtFromHours now state.agentExpiresHours) )
                )

        AgentCreated (Ok created) ->
            -- Reset the scope checkboxes to the same defaults a fresh session
            -- starts with, not to none - an all-unchecked form right after a
            -- successful create otherwise fails "Select at least one scope."
            -- on the next attempt.
            ( Api.updateLoggedIn model (\state -> { state | newCredential = Just created, agentMessage = Nothing, agentLabel = "", agentScopes = [ Agent.AgentScopeTasksRead, Agent.AgentScopeSubmissionsWrite ], agentExpiresHours = "" }), Api.refreshCredentials model )

        AgentCreated (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | agentMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        MintTaskTokenClicked ->
            Api.withSession model (\state -> ( model, Api.mintTaskToken state.accessToken ))

        TaskTokenMinted (Ok created) ->
            ( Api.updateLoggedIn model (\state -> { state | taskAgentToken = Just created.secret }), Cmd.none )

        TaskTokenMinted (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | taskActionMessage = Just (FailureNote ("Could not create agent token: " ++ httpErrorLabel error)) }), Cmd.none )

        MintUserTokenClicked ->
            Api.withSession model (\state -> ( model, Api.mintUserToken state.accessToken ))

        UserTokenMinted (Ok created) ->
            ( Api.updateLoggedIn model (\state -> { state | userAgentToken = Just created.secret }), Cmd.none )

        UserTokenMinted (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | taskActionMessage = Just (FailureNote ("Could not create agent token: " ++ httpErrorLabel error)) }), Cmd.none )

        CopyClicked clipboardText ->
            ( model, copyToClipboard clipboardText )

        RevokeClicked credentialId ->
            Api.withSession model (\state -> ( model, Api.revokeAgent state.accessToken credentialId ))

        AgentRevoked (Ok _) ->
            ( Api.updateLoggedIn model (\state -> { state | agentMessage = Nothing }), Api.refreshCredentials model )

        AgentRevoked (Err error) ->
            -- A failed revoke must not look like a success: without an error
            -- note the list refresh shows the credential still active with no
            -- explanation.
            ( Api.updateLoggedIn model (\state -> { state | agentMessage = Just (FailureNote ("Could not revoke the credential: " ++ httpErrorLabel error)) }), Cmd.none )

        OrgCredentialsReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | orgCredentials = Api.loadedFromResult .credentials result }), Cmd.none )

        OrgCredentialLabelChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | orgCredentialLabel = value }), Cmd.none )

        ToggleOrgCredentialScope scope ->
            ( Api.updateLoggedIn model (\state -> { state | orgCredentialScopes = Api.toggleScope scope state.orgCredentialScopes }), Cmd.none )

        OrgCredentialExpiresHoursChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | orgCredentialExpiresHours = value }), Cmd.none )

        CreateOrgCredentialClicked ->
            Api.withSession model (\state -> Api.createOrgCredentialCommand model state)

        OrgCredentialExpiresAtResolved now ->
            Api.withSession model
                (\state ->
                    ( model, Api.postOrgCredential state.accessToken state.activeOrgId state.orgCredentialLabel state.orgCredentialScopes (Api.expiresAtFromHours now state.orgCredentialExpiresHours) )
                )

        OrgCredentialCreated (Ok created) ->
            ( Api.updateLoggedIn model (\state -> { state | newOrgCredential = Just created, orgCredentialMessage = Nothing, orgCredentialLabel = "", orgCredentialScopes = [], orgCredentialExpiresHours = "" }), Api.refreshOrgCredentials model )

        OrgCredentialCreated (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | orgCredentialMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        RevokeOrgCredentialClicked credentialId ->
            Api.withSession model (\state -> ( model, Api.postRevokeOrgCredential state.accessToken state.activeOrgId credentialId ))

        OrgCredentialRevoked (Ok _) ->
            ( model, Api.refreshOrgCredentials model )

        OrgCredentialRevoked (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | orgCredentialMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        LogoutClicked ->
            ( { model | authError = Nothing }
            , Api.postLogout
            )

        LogoutReceived (Ok logoutURL) ->
            if logoutURL == "" then
                ( { model | session = LoggedOut, email = "", password = "", authError = Nothing }
                , Nav.pushUrl model.key "#/"
                )

            else
                ( { model | session = LoggedOut, email = "", password = "", authError = Nothing }
                , Nav.load logoutURL
                )

        LogoutReceived (Err error) ->
            ( { model | session = LoggedOut, authError = Just ("Sharecrop signed out, but global sign out failed: " ++ httpErrorLabel error) }
            , Nav.pushUrl model.key "#/"
            )

        DiscoveryIncludeReservedChanged value ->
            Api.withSession model
                (\state ->
                    let
                        nextState =
                            { state | discoveryIncludeReserved = value, discoveryOffset = 0 }
                    in
                    ( Api.updateLoggedIn model (\_ -> nextState), Api.fetchDiscovery state.accessToken value state.discoveryFundedOnly 0 )
                )

        DiscoveryFundedOnlyChanged value ->
            Api.withSession model
                (\state ->
                    ( Api.updateLoggedIn model (\current -> { current | discoveryFundedOnly = value, discoveryOffset = 0 })
                    , Api.fetchDiscovery state.accessToken state.discoveryIncludeReserved value 0
                    )
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
                    ( Api.updateLoggedIn model (\current -> { current | discoveryOffset = offset }), Api.fetchDiscovery state.accessToken state.discoveryIncludeReserved state.discoveryFundedOnly offset )
                )

        NextDiscoveryPageClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            state.discoveryOffset + Api.selectorPageSize
                    in
                    ( Api.updateLoggedIn model (\current -> { current | discoveryOffset = offset }), Api.fetchDiscovery state.accessToken state.discoveryIncludeReserved state.discoveryFundedOnly offset )
                )

        DiscoveryReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | discoveryTasks = Api.loadedFromResult .tasks result, discoveryNextOffset = nextOffsetFromResult .nextOffset result, discoveryTotal = totalFromResult result }), Cmd.none )

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
                        , submitFieldValues = Dict.empty
                        , submitRawMode = False
                        , submitMessage = Nothing
                        , reviewNote = ""
                        , reviewPartialCredit = ""
                        , reviewTip = ""
                        , reviewTipCollectibleId = ""
                        , reviewBan = Ledger.BanSelectionNone
                        , reviewMessage = Nothing
                        , taskActionMessage = Nothing
                        , taskComments = []
                        , taskCommentBody = ""
                        , submissionCommentBody = ""
                        , activeSubmissionCommentsID = Nothing
                        , taskAgentToken = Nothing
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
                    ( Api.updateLoggedIn model (\current -> { current | reservationOrganizationId = value, reservationTeamId = "", orgTeams = loadedNone, orgTeamQuery = "", orgTeamOffset = 0 })
                    , if value == "" then
                        Cmd.none

                      else
                        Api.fetchOrgTeams state.accessToken value
                    )
                )

        ReservationTeamIdChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | reservationTeamId = value }), Cmd.none )

        ReservationReceived (Ok reservation) ->
            ( Api.updateLoggedIn model
                (\state ->
                    { state
                        | reservationMessage = Just (SuccessNote (View.reservationSuccessLabel reservation))
                        , reservationSecret = issuedCredentialSecret reservation.issuedWorkerCredential state.reservationSecret
                    }
                )
            , Api.refreshDetailReservations model
            )

        ReservationReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | reservationMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        ReservationsReceived (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | reservations = response.reservations }), Cmd.none )

        ReservationsReceived (Err _) ->
            ( Api.updateLoggedIn model (\state -> { state | reservations = [] }), Cmd.none )

        CancelReservationClicked reservationId ->
            Api.withSession model (\state -> Api.reservationChangeCommand model state reservationId "cancel")

        ReservationChangeReceived (Ok reservation) ->
            ( Api.updateLoggedIn model
                (\state ->
                    { state
                        | reservationMessage = Just (SuccessNote (View.reservationSuccessLabel reservation))
                        , reservationSecret = issuedCredentialSecret reservation.issuedWorkerCredential state.reservationSecret
                    }
                )
            , Api.refreshDetailReservations model
            )

        ReservationChangeReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | reservationMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        SubmissionsReceived (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | submissions = response.submissions }), Cmd.none )

        SubmissionsReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | submissions = [], reviewMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        SubmitInputChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | submitInput = value }), Cmd.none )

        SubmitFieldChanged name value ->
            ( Api.updateLoggedIn model (\state -> { state | submitFieldValues = Dict.insert name value state.submitFieldValues }), Cmd.none )

        SubmitRawModeToggled enabled ->
            -- Switching to the raw JSON editor seeds it from whatever the
            -- worker already typed in the structured fields (best effort), so
            -- the toggle doesn't read as data loss. Only seeds when the raw
            -- box is still empty, to avoid clobbering hand-edited JSON on a
            -- re-toggle.
            ( Api.updateLoggedIn model
                (\state ->
                    { state
                        | submitRawMode = enabled
                        , submitInput =
                            if enabled && String.trim state.submitInput == "" then
                                Api.seedRawSubmitInput state

                            else
                                state.submitInput
                    }
                )
            , Cmd.none
            )

        PickSubmitAttachmentClicked ->
            Api.withSession model
                (\state ->
                    if List.length state.submitAttachments >= attachmentMaxCount then
                        ( Api.updateLoggedIn model (\current -> { current | submitMessage = Just (FailureNote "Attach up to 5 files.") }), Cmd.none )

                    else
                        ( model, selectAttachment SubmitAttachmentFileChosen )
                )

        SubmitAttachmentFileChosen file ->
            ( model, readSubmitAttachment file )

        SubmitAttachmentSelected name contentType sizeBytes dataURL ->
            ( Api.updateLoggedIn model (\state -> { state | submitAttachments = state.submitAttachments ++ [ { name = name, contentType = contentType, sizeBytes = sizeBytes, dataURL = dataURL } ], submitMessage = Nothing }), Cmd.none )

        SubmitAttachmentRejected message ->
            ( Api.updateLoggedIn model (\state -> { state | submitMessage = Just (FailureNote message) }), Cmd.none )

        RemoveSubmitAttachmentClicked index ->
            ( Api.updateLoggedIn model (\state -> { state | submitAttachments = removeAt index state.submitAttachments }), Cmd.none )

        SubmitClicked ->
            Api.withSession model (\state -> Api.submitCommand model state)

        SubmitReceived (Ok created) ->
            Api.withSession model
                (\state ->
                    -- A created-but-invalid submission is a failure from the
                    -- worker's point of view (the validation errors are in
                    -- the message); anything else reads as success.
                    ( Api.updateLoggedIn model (\current -> { current | submitInput = "", submitFieldValues = Dict.empty, submitAttachments = [], submitMessage = Just (submissionOutcomeNote created), activeSubmissionCommentsID = Just created.submission.id, submissionComments = [], submissionCommentMessage = Nothing })
                    , Cmd.batch
                        [ Api.refreshDetailSubmissions model
                        , Api.fetchSubmissionComments state.accessToken created.submission.id
                        ]
                    )
                )

        SubmitReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | submitMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        ModerationReasonChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | moderationReason = value }), Cmd.none )

        ModerationDetailsChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | moderationDetails = value }), Cmd.none )

        FileDisputeClicked submissionId ->
            -- Preset the report form to a dispute about this submission; the
            -- form opens via its subject-derived open flag (see
            -- moderationReportCard) and submits through the same
            -- ReportTaskClicked path.
            ( Api.updateLoggedIn model
                (\state ->
                    { state
                        | moderationReason = Moderation.ModerationReasonDispute
                        , moderationSubject = ReportAboutSubmission submissionId
                        , moderationMessage = Nothing
                    }
                )
            , Cmd.none
            )

        ReportSubjectClearedClicked ->
            ( Api.updateLoggedIn model (\state -> { state | moderationSubject = ReportAboutTask }), Cmd.none )

        ReportTaskClicked taskId ->
            Api.withSession model
                (\state ->
                    ( Api.updateLoggedIn model (\current -> { current | moderationMessage = Nothing })
                    , Api.reportTask state.accessToken taskId state.moderationSubject state.moderationReason state.moderationDetails
                    )
                )

        ModerationReportReceived (Ok report) ->
            ( Api.updateLoggedIn model (\state -> { state | moderationDetails = "", moderationMessage = Just (SuccessNote ("Report submitted: " ++ report.reason)) }), Cmd.none )

        ModerationReportReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | moderationMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        ReviewNoteChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | reviewNote = value }), Cmd.none )

        ReviewPartialCreditChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | reviewPartialCredit = value }), Cmd.none )

        ReviewTipChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | reviewTip = value }), Cmd.none )

        ReviewTipCollectibleChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | reviewTipCollectibleId = value }), Cmd.none )

        ReviewBanChanged value ->
            ( Api.updateLoggedIn model
                (\state ->
                    { state
                        | reviewBan =
                            if value then
                                Ledger.BanSelectionBanImplementor

                            else
                                Ledger.BanSelectionNone
                    }
                )
            , Cmd.none
            )

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
                    ( Api.updateLoggedIn model (\current -> { current | reviewMessage = Just (SuccessNote "Review saved."), reviewNote = "", reviewPartialCredit = "", reviewTip = "", reviewTipCollectibleId = "", reviewBan = Ledger.BanSelectionNone, activeSubmissionCommentsID = Just submissionId, submissionComments = [], submissionCommentMessage = Nothing })
                    , Cmd.batch
                        [ Api.refreshAfterAccept model
                        , Api.fetchSubmissionComments state.accessToken submissionId
                        ]
                    )
                )

        ReviewActionReceived _ (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | reviewMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

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
                        , collectibleMessage = Just (SuccessNote (View.mintSuccessLabel collectible))
                    }
                )
            , Api.refreshCollectibles model
            )

        MintReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | collectibleMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        CollectiblesReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | collectibles = Api.loadedFromResult .collectibles result }), Cmd.none )

        AwardTaskIdChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | awardTaskId = value }), Cmd.none )

        AwardClicked collectibleId ->
            Api.withSession model (\state -> Api.awardCommand model state collectibleId)

        AwardReceived (Ok collectible) ->
            let
                updated =
                    Api.updateLoggedIn model (\state -> { state | awardMessage = Just (SuccessNote (View.awardSuccessLabel collectible)) })
            in
            Api.withSession updated
                (\state ->
                    ( updated
                    , Cmd.batch
                        [ Api.fetchCollectibles state.accessToken
                        , Api.fetchTasks state.accessToken state.taskStateFilter state.taskListTypeFilter state.taskListSort state.taskListOffset
                        , Api.fetchPublicTaskDetail state.accessToken state.awardTaskId
                        ]
                    )
                )

        AwardReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | awardMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        AwardOrgCollectibleRecipientIdChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | awardOrgCollectibleRecipientId = value }), Cmd.none )

        AwardOrgCollectibleClicked collectibleId ->
            Api.withSession model
                (\state ->
                    if String.isEmpty (String.trim state.awardOrgCollectibleRecipientId) then
                        ( Api.updateLoggedIn model (\current -> { current | awardOrgCollectibleMessage = Just (FailureNote "Choose a member first.") }), Cmd.none )

                    else
                        ( Api.updateLoggedIn model (\current -> { current | awardOrgCollectibleMessage = Nothing })
                        , Api.postAwardOrganizationCollectible state.accessToken state.activeOrgId collectibleId state.awardOrgCollectibleRecipientId
                        )
                )

        AwardOrgCollectibleReceived (Ok collectible) ->
            let
                updated =
                    Api.updateLoggedIn model (\state -> { state | awardOrgCollectibleMessage = Just (SuccessNote (View.awardSuccessLabel collectible)) })
            in
            Api.withSession updated (\state -> ( updated, Api.fetchOrganizationCollectibles state.accessToken state.activeOrgId ))

        AwardOrgCollectibleReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | awardOrgCollectibleMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        CollectibleCatalogReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | collectibleCatalog = Api.loadedFromResult .entries result }), Cmd.none )

        AwardRecipientKindChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | awardRecipientKind = value, awardRecipientId = "" }), Cmd.none )

        AwardRecipientIdChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | awardRecipientId = value }), Cmd.none )

        AwardDefaultClicked slug ->
            Api.withSession model
                (\state ->
                    if String.trim state.awardRecipientId == "" then
                        ( Api.updateLoggedIn model (\current -> { current | awardDefaultMessage = Just (FailureNote "Enter a recipient id first.") }), Cmd.none )

                    else
                        ( model, Api.awardDefaultCollectible state.accessToken slug state.awardRecipientKind state.awardRecipientId )
                )

        AwardDefaultReceived (Ok _) ->
            let
                updated =
                    Api.updateLoggedIn model (\state -> { state | awardDefaultMessage = Just (SuccessNote "Awarded the collectible.") })
            in
            ( updated, Api.refreshCollectibles updated )

        AwardDefaultReceived (Err error) ->
            -- Show the actual failure: the award button is already gated to
            -- platform admins, so hardcoding a permissions explanation here
            -- told an admin whose request failed for another reason (bad
            -- recipient, expired session, network) something false.
            ( Api.updateLoggedIn model (\state -> { state | awardDefaultMessage = Just (FailureNote ("Could not award the collectible: " ++ httpErrorLabel error)) }), Cmd.none )

        TransferRecipientIdChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | transferRecipientId = value }), Cmd.none )

        TransferCollectibleClicked collectibleId ->
            Api.withSession model
                (\state ->
                    if String.trim state.transferRecipientId == "" then
                        ( Api.updateLoggedIn model (\current -> { current | transferMessage = Just (FailureNote "Enter a recipient id first.") }), Cmd.none )

                    else
                        ( model, Api.transferCollectible state.accessToken collectibleId "user" state.transferRecipientId )
                )

        TransferCollectibleReceived (Ok collectible) ->
            let
                updated =
                    Api.updateLoggedIn model (\state -> { state | transferMessage = Just (SuccessNote ("Sent " ++ collectible.name ++ ".")), openSendCollectibleID = Nothing })
            in
            ( updated, Api.refreshCollectibles updated )

        TransferCollectibleReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | transferMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        ToggleSendCollectible collectibleId ->
            ( Api.updateLoggedIn model
                (\state ->
                    if state.openSendCollectibleID == Just collectibleId then
                        { state | openSendCollectibleID = Nothing }

                    else
                        { state | openSendCollectibleID = Just collectibleId, sendCollectibleTargetKind = "user", transferRecipientId = "", transferMessage = Nothing }
                )
            , Cmd.none
            )

        SendCollectibleTargetKindChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | sendCollectibleTargetKind = value, transferRecipientId = "" }), Cmd.none )

        SendCollectibleClicked collectibleId ->
            Api.withSession model
                (\state ->
                    if String.trim state.transferRecipientId == "" then
                        ( Api.updateLoggedIn model (\current -> { current | transferMessage = Just (FailureNote "Choose a recipient first.") }), Cmd.none )

                    else
                        ( model, Api.transferCollectible state.accessToken collectibleId state.sendCollectibleTargetKind state.transferRecipientId )
                )

        OrgSendCollectibleRecipientIdChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | orgSendCollectibleRecipientId = value }), Cmd.none )

        OrgSendCollectibleClicked collectibleId ->
            Api.withSession model
                (\state ->
                    if String.trim state.orgSendCollectibleRecipientId == "" then
                        ( Api.updateLoggedIn model (\current -> { current | orgSendCollectibleMessage = Just (FailureNote "Choose a user first.") }), Cmd.none )

                    else
                        ( Api.updateLoggedIn model (\current -> { current | orgSendCollectibleMessage = Nothing })
                        , Api.postOrgSendCollectible state.accessToken state.activeOrgId collectibleId state.orgSendCollectibleRecipientId
                        )
                )

        OrgSendCollectibleReceived (Ok collectible) ->
            let
                updated =
                    Api.updateLoggedIn model (\state -> { state | orgSendCollectibleMessage = Just (SuccessNote ("Sent " ++ collectible.name ++ ".")) })
            in
            Api.withSession updated (\state -> ( updated, Api.fetchOrganizationCollectibles state.accessToken state.activeOrgId ))

        OrgSendCollectibleReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | orgSendCollectibleMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        CatalogSlugChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | catalogSlug = value }), Cmd.none )

        CatalogNameChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | catalogName = value }), Cmd.none )

        CatalogKindChosen kind ->
            ( Api.updateLoggedIn model (\state -> { state | catalogKind = kind }), Cmd.none )

        CatalogPolicyChosen policy ->
            ( Api.updateLoggedIn model (\state -> { state | catalogPolicy = policy }), Cmd.none )

        CatalogArtChosen slug ->
            ( Api.updateLoggedIn model (\state -> { state | catalogArt = slug }), Cmd.none )

        CatalogMaxEditionsChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | catalogMaxEditions = value }), Cmd.none )

        AddCatalogEntryClicked ->
            Api.withSession model (\state -> Api.addCatalogEntryCommand model state)

        CatalogEntryMutated (Ok entry) ->
            Api.withSession model
                (\state ->
                    ( Api.updateLoggedIn model
                        (\current ->
                            { current
                                | catalogMessage = Just (SuccessNote (catalogMutationLabel entry))
                                , catalogSlug = ""
                                , catalogName = ""
                                , catalogMaxEditions = ""
                                , catalogArt = ""
                            }
                        )
                    , Api.fetchCollectibleCatalog state.accessToken
                    )
                )

        CatalogEntryMutated (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | catalogMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        WithdrawCatalogEntryClicked slug ->
            Api.withSession model (\state -> ( Api.updateLoggedIn model (\current -> { current | catalogMessage = Nothing }), Api.postWithdrawCatalogEntry state.accessToken slug ))

        DeleteCatalogEntryClicked slug ->
            Api.withSession model (\state -> ( Api.updateLoggedIn model (\current -> { current | catalogMessage = Nothing }), Api.deleteCatalogEntryCmd state.accessToken slug ))

        CatalogEntryDeleted (Ok ()) ->
            Api.withSession model
                (\state ->
                    ( Api.updateLoggedIn model (\current -> { current | catalogMessage = Just (SuccessNote "Deleted the catalog entry.") })
                    , Api.fetchCollectibleCatalog state.accessToken
                    )
                )

        CatalogEntryDeleted (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | catalogMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        WithdrawCollectibleClicked collectibleId ->
            Api.withSession model (\state -> ( Api.updateLoggedIn model (\current -> { current | withdrawMessage = Nothing }), Api.postWithdrawCollectible state.accessToken collectibleId ))

        CollectibleWithdrawnReceived (Ok collectible) ->
            Api.withSession model
                (\state ->
                    ( Api.updateLoggedIn model (\current -> { current | withdrawMessage = Just (SuccessNote ("Withdrew '" ++ collectible.name ++ "'.")) })
                    , Cmd.batch [ Api.fetchCollectibles state.accessToken, Api.fetchCollectibleCatalog state.accessToken ]
                    )
                )

        CollectibleWithdrawnReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | withdrawMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        DeleteCollectibleClicked collectibleId ->
            Api.withSession model (\state -> ( Api.updateLoggedIn model (\current -> { current | withdrawMessage = Nothing }), Api.deleteCollectibleCmd state.accessToken collectibleId ))

        CollectibleDeleted (Ok ()) ->
            Api.withSession model
                (\state ->
                    ( Api.updateLoggedIn model (\current -> { current | withdrawMessage = Just (SuccessNote "Deleted the collectible.") })
                    , Cmd.batch [ Api.fetchCollectibles state.accessToken, Api.fetchCollectibleCatalog state.accessToken ]
                    )
                )

        CollectibleDeleted (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | withdrawMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        OrganizationsReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | organizations = Api.loadedFromResult .organizations result, organizationNextOffset = nextOffsetFromResult .nextOffset result }), Cmd.none )

        CreateOrgNameChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | createOrgName = value }), Cmd.none )

        CreateOrgClicked ->
            Api.withSession model (\state -> Api.createOrgCommand model state)

        CreateOrgReceived (Ok organization) ->
            ( Api.updateLoggedIn model (\state -> { state | createOrgName = "", orgMessage = Just (SuccessNote ("Created organization " ++ organization.name)) }), Api.refreshOrganizations model )

        CreateOrgReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | orgMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        OrgBalanceReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | orgBalance = Api.balanceFromResult result }), Cmd.none )

        OrgLedgerReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | orgLedger = Api.loadedFromResult .entries result, orgLedgerNextOffset = nextOffsetFromResult .nextOffset result, orgLedgerTotal = totalFromResult result }), Cmd.none )

        PreviousOrgLedgerPageClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            max 0 (state.orgLedgerOffset - Api.selectorPageSize)
                    in
                    ( Api.updateLoggedIn model (\current -> { current | orgLedgerOffset = offset }), Api.fetchOrganizationLedgerPage state.accessToken state.activeOrgId offset )
                )

        NextOrgLedgerPageClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            state.orgLedgerOffset + Api.selectorPageSize
                    in
                    ( Api.updateLoggedIn model (\current -> { current | orgLedgerOffset = offset }), Api.fetchOrganizationLedgerPage state.accessToken state.activeOrgId offset )
                )

        OrgAuditEventsReceived (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | orgAuditEvents = response.events, orgAuditMessage = Nothing }), Cmd.none )

        OrgAuditEventsReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | orgAuditEvents = [], orgAuditMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        OrgTeamsReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | orgTeams = Api.loadedFromResult .teams result, orgTeamNextOffset = nextOffsetFromResult .nextOffset result }), Cmd.none )

        StandaloneTeamsReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | standaloneTeams = Api.loadedFromResult .teams result, standaloneTeamNextOffset = nextOffsetFromResult .nextOffset result }), Cmd.none )

        CreateTeamNameChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | createTeamName = value }), Cmd.none )

        CreateTeamClicked ->
            Api.withSession model
                (\state ->
                    if String.trim state.createTeamName == "" then
                        ( Api.updateLoggedIn model (\current -> { current | createTeamMessage = Just (FailureNote "Enter a team name first.") }), Cmd.none )

                    else
                        ( Api.updateLoggedIn model (\current -> { current | createTeamMessage = Nothing }), Api.createStandaloneTeam state.accessToken state.createTeamName )
                )

        TeamCreated (Ok team) ->
            Api.withSession model
                (\state ->
                    ( Api.updateLoggedIn model (\current -> { current | createTeamName = "", createTeamMessage = Just (SuccessNote ("Created team " ++ team.name ++ ".")) })
                    , Api.fetchStandaloneTeams state.accessToken
                    )
                )

        TeamCreated (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | createTeamMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        UserDirectoryReceived result ->
            ( Api.updateLoggedIn model
                (\state ->
                    case result of
                        Ok page ->
                            { state | userDirectory = page.users, userDirectoryNextOffset = page.nextOffset }

                        Err _ ->
                            { state | userDirectory = [], userDirectoryNextOffset = 0 }
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
                    -- fetchOrgTeamsPage is a no-op without an organization id
                    -- (there is nothing to query); say so instead of letting
                    -- the Search button silently do nothing.
                    if orgTeamSearchOrganizationID state == "" then
                        ( Api.updateLoggedIn model (\current -> { current | reservationMessage = Just (FailureNote "Choose an organization first, then search its teams.") }), Cmd.none )

                    else
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
            ( Api.updateLoggedIn model (\state -> { state | orgMembers = Api.loadedFromResult .members result }), Cmd.none )

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
            ( Api.updateLoggedIn model (\state -> { state | userWork = Api.loadedFromResult .tasks result }), Cmd.none )

        UserSubmissionsReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | userSubmissions = Api.loadedFromResult .submissions result, userSubmissionsNextOffset = nextOffsetFromResult .nextOffset result, userSubmissionsTotal = totalFromResult result }), Cmd.none )

        PreviousUserSubmissionsPageClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            max 0 (state.userSubmissionsOffset - Api.selectorPageSize)
                    in
                    ( Api.updateLoggedIn model (\current -> { current | userSubmissionsOffset = offset }), Api.fetchUserSubmissionsPage state.accessToken state.subjectId offset )
                )

        NextUserSubmissionsPageClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            state.userSubmissionsOffset + Api.selectorPageSize
                    in
                    ( Api.updateLoggedIn model (\current -> { current | userSubmissionsOffset = offset }), Api.fetchUserSubmissionsPage state.accessToken state.subjectId offset )
                )

        StartRevisionClicked taskId responseJson ->
            ( Api.updateLoggedIn model (\state -> { state | pendingRevisionTaskID = Just taskId, pendingRevisionResponse = responseJson })
            , Nav.pushUrl model.key ("#/tasks/" ++ taskId)
            )

        SeriesListReceived result ->
            ( Api.updateLoggedIn model (\state -> { state | seriesList = Api.loadedFromResult .series result }), Cmd.none )

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
                        , seriesMessage = Just (SuccessNote "Series saved.")
                    }
                )
            , seriesListRefresh model
            )

        SeriesMutationReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | seriesMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

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
            ( Api.updateLoggedIn model (\state -> { state | seriesMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

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
                            { state | teamWork = response.tasks, teamWorkNextOffset = response.nextOffset, teamWorkTotal = response.total, teamWorkMessage = Nothing }

                        Err error ->
                            { state | teamWork = [], teamWorkNextOffset = 0, teamWorkTotal = 0, teamWorkMessage = Just (FailureNote (httpErrorLabel error)) }
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
                        ( Api.updateLoggedIn model (\current -> { current | teamWorkMessage = Just (FailureNote "A saved view name is required.") }), Cmd.none )

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
                                        , teamWorkMessage = Just (SuccessNote ("Applied view: " ++ view.name))
                                    }
                                )
                            , Api.fetchTeamWork state.accessToken detail.team.id view.query view.typeFilter view.sort 0
                            )

                        ( _, Nothing ) ->
                            ( Api.updateLoggedIn model (\current -> { current | teamWorkMessage = Just (FailureNote "Saved view was not found.") }), Cmd.none )

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
            Api.withSession model
                (\state ->
                    if String.trim state.teamMemberEmail == "" then
                        ( Api.updateLoggedIn model (\current -> { current | teamMemberMessage = Just (FailureNote "A member email is required.") }), Cmd.none )

                    else
                        ( model, Api.postAddTeamMember state.accessToken teamId state.teamMemberEmail )
                )

        AddTeamMemberReceived (Ok detail) ->
            ( Api.updateLoggedIn model (\state -> { state | teamDetail = Just detail, teamMemberEmail = "", teamMemberMessage = Just (SuccessNote "Member added.") }), Cmd.none )

        AddTeamMemberReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | teamMemberMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        OrgTasksReceived (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | orgTasks = response.tasks, orgTaskNextOffset = response.nextOffset, orgTaskTotal = response.total, orgTaskMessage = Nothing }), Cmd.none )

        OrgTasksReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | orgTasks = [], orgTaskNextOffset = 0, orgTaskTotal = 0, orgTaskMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

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
                        ( Api.updateLoggedIn model (\current -> { current | orgTaskMessage = Just (FailureNote "A saved view name is required.") }), Cmd.none )

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
                                        , orgTaskMessage = Just (SuccessNote ("Applied view: " ++ view.name))
                                    }
                                )
                            , Api.fetchOrgTasksPage state.accessToken state.activeOrgId view.query view.stateFilter view.typeFilter view.sort 0
                            )

                        Nothing ->
                            ( Api.updateLoggedIn model (\current -> { current | orgTaskMessage = Just (FailureNote "Saved view was not found.") }), Cmd.none )
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
            ( Api.updateLoggedIn model (\state -> { state | orgCollectibles = [], orgCollectiblesMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        TeamCollectiblesReceived (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | teamCollectibles = response.collectibles, teamCollectiblesMessage = Nothing }), Cmd.none )

        TeamCollectiblesReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | teamCollectibles = [], teamCollectiblesMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        CreateOrgTeamNameChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | createOrgTeamName = value }), Cmd.none )

        CreateOrgTeamClicked ->
            Api.withSession model (\state -> Api.createOrgTeamCommand model state)

        CreateOrgTeamReceived (Ok team) ->
            let
                updated =
                    Api.updateLoggedIn model (\state -> { state | createOrgTeamName = "", orgTeamMessage = Just (SuccessNote ("Created team " ++ team.name)) })
            in
            Api.withSession updated (\state -> ( updated, Api.fetchOrgTeams state.accessToken state.activeOrgId ))

        CreateOrgTeamReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | orgTeamMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        ProvisionMemberEmailChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | provisionMemberEmail = value }), Cmd.none )

        ToggleProvisionMemberRole role ->
            ( Api.updateLoggedIn model (\state -> { state | provisionMemberRoles = toggleString role state.provisionMemberRoles }), Cmd.none )

        ProvisionMemberClicked ->
            Api.withSession model (\state -> Api.provisionMemberCommand model state)

        ProvisionMemberReceived (Ok ()) ->
            let
                updated =
                    Api.updateLoggedIn model (\state -> { state | provisionMemberEmail = "", provisionMemberMessage = Just (SuccessNote "Member provisioned.") })
            in
            Api.withSession updated (\state -> ( updated, Api.authorizedRequest "GET" state.accessToken ("/api/organizations/" ++ state.activeOrgId ++ "/members") Http.emptyBody (Http.expectJson OrgMembersReceived Organization.organizationMembersResponseDecoder) ))

        ProvisionMemberReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | provisionMemberMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        UpdateMemberRolesClicked userId roles ->
            Api.withSession model (\state -> Api.updateMemberRolesCommand model state userId roles)

        UpdateMemberRolesReceived (Ok _) ->
            let
                updated =
                    Api.updateLoggedIn model (\state -> { state | provisionMemberMessage = Just (SuccessNote "Member roles updated.") })
            in
            Api.withSession updated (\state -> ( updated, Api.authorizedRequest "GET" state.accessToken ("/api/organizations/" ++ state.activeOrgId ++ "/members") Http.emptyBody (Http.expectJson OrgMembersReceived Organization.organizationMembersResponseDecoder) ))

        UpdateMemberRolesReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | provisionMemberMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        DeactivateMemberClicked userId ->
            Api.withSession model (\state -> Api.deactivateMemberCommand model state userId)

        DeactivateMemberReceived (Ok _) ->
            let
                updated =
                    Api.updateLoggedIn model (\state -> { state | provisionMemberMessage = Just (SuccessNote "Member deactivated.") })
            in
            Api.withSession updated (\state -> ( updated, Api.authorizedRequest "GET" state.accessToken ("/api/organizations/" ++ state.activeOrgId ++ "/members") Http.emptyBody (Http.expectJson OrgMembersReceived Organization.organizationMembersResponseDecoder) ))

        DeactivateMemberReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | provisionMemberMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

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
                        ( Api.updateLoggedIn model (\current -> { current | taskCommentMessage = Just (FailureNote "Write a comment first.") }), Cmd.none )

                    else
                        ( Api.updateLoggedIn model (\current -> { current | taskCommentMessage = Nothing })
                        , Api.postTaskComment state.accessToken taskId (String.trim state.taskCommentBody)
                        )
                )

        TaskCommentReceived (Ok comment) ->
            ( Api.updateLoggedIn model (\state -> { state | taskComments = state.taskComments ++ [ comment ], taskCommentBody = "", taskCommentMessage = Nothing }), Cmd.none )

        TaskCommentReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | taskCommentMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

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
            ( Api.updateLoggedIn model (\state -> { state | submissionCommentMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        SubmissionCommentBodyChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | submissionCommentBody = value }), Cmd.none )

        AddSubmissionCommentClicked submissionId ->
            Api.withSession model
                (\state ->
                    if String.trim state.submissionCommentBody == "" then
                        ( Api.updateLoggedIn model (\current -> { current | submissionCommentMessage = Just (FailureNote "Write a comment first.") }), Cmd.none )

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
            ( Api.updateLoggedIn model (\state -> { state | submissionCommentMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        AccountEmailChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | accountEmail = value }), Cmd.none )

        AccountProfileReceived (Ok profile) ->
            -- Prefill the email form from the profile only when the field is
            -- still blank, so a half-typed new address is not overwritten.
            ( Api.updateLoggedIn model
                (\state ->
                    { state
                        | accountProfile = Just profile
                        , displayNameDraft = profile.displayName
                        , accountEmail =
                            if String.trim state.accountEmail == "" then
                                profile.email

                            else
                                state.accountEmail
                    }
                )
            , Cmd.none
            )

        AccountProfileReceived (Err _) ->
            -- The identity card simply stays in its loading state; the rest
            -- of the profile page still works.
            ( model, Cmd.none )

        DisplayNameDraftChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | displayNameDraft = value }), Cmd.none )

        SaveDisplayNameClicked ->
            Api.withSession model
                (\state ->
                    if String.trim state.displayNameDraft == "" then
                        ( Api.updateLoggedIn model (\current -> { current | accountMessage = Just (FailureNote "Enter a display name first.") }), Cmd.none )

                    else
                        ( Api.updateLoggedIn model (\current -> { current | accountMessage = Nothing }), Api.patchDisplayName state.accessToken (String.trim state.displayNameDraft) )
                )

        DisplayNameSaved (Ok ()) ->
            -- The header shows the display name, so it updates immediately
            -- rather than waiting for the next session refresh.
            Api.withSession model
                (\state ->
                    ( Api.updateLoggedIn model (\current -> { current | displayName = String.trim current.displayNameDraft, accountMessage = Just (SuccessNote "Display name updated.") })
                    , Api.fetchAccountProfile state.accessToken
                    )
                )

        DisplayNameSaved (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | accountMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

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
            Api.withSession model
                (\state ->
                    -- The email field starts blank after a reload (the client
                    -- has no endpoint that returns the current email), so a
                    -- blank save would send "" for a guaranteed rejection.
                    if String.trim state.accountEmail == "" then
                        ( Api.updateLoggedIn model (\current -> { current | accountMessage = Just (FailureNote "Enter the new email address first.") }), Cmd.none )

                    else
                        ( Api.updateLoggedIn model (\current -> { current | accountMessage = Nothing }), Api.updateProfile state.accessToken state.accountEmail )
                )

        ChangePasswordClicked ->
            Api.withSession model (\state -> ( Api.updateLoggedIn model (\current -> { current | accountMessage = Nothing }), Api.changePassword state.accessToken state.currentPassword state.newPassword ))

        DeactivateAccountClicked ->
            -- Deactivation is irreversible (credential removal, token
            -- revocation, email anonymization), so a single stray click must
            -- not trigger it: the first click arms an explicit confirm step.
            ( Api.updateLoggedIn model (\current -> { current | accountMessage = Nothing, deactivateConfirming = True }), Cmd.none )

        ConfirmDeactivateAccountClicked ->
            Api.withSession model (\state -> ( Api.updateLoggedIn model (\current -> { current | accountMessage = Nothing, deactivateConfirming = False }), Api.deactivateAccount state.accessToken ))

        CancelDeactivateAccountClicked ->
            ( Api.updateLoggedIn model (\current -> { current | deactivateConfirming = False }), Cmd.none )

        PrivacyRequestClicked kind ->
            Api.withSession model (\state -> ( Api.updateLoggedIn model (\current -> { current | accountMessage = Nothing }), Api.requestPrivacy state.accessToken kind ))

        EmailVerificationRequested (Ok token) ->
            if token == "" then
                -- No email delivery exists in this deployment; the token went
                -- to the server operator's log (see the matching password-
                -- reset copy on the auth screen).
                ( Api.updateLoggedIn model (\state -> { state | accountMessage = Just (SuccessNote "Verification requested. This deployment does not send email - ask an operator for the token from the server log, then paste it below.") }), Cmd.none )

            else
                ( Api.updateLoggedIn model (\state -> { state | emailVerificationToken = token, emailVerificationInput = token, accountMessage = Just (SuccessNote "Verification token created.") }), Cmd.none )

        EmailVerificationRequested (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | accountMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        AccountActionReceived (Ok ()) ->
            ( Api.updateLoggedIn model (\state -> { state | currentPassword = "", newPassword = "", emailVerificationInput = "", accountMessage = Just (SuccessNote "Account updated.") }), Cmd.none )

        AccountActionReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | accountMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        DeactivateAccountReceived (Ok ()) ->
            ( { model | session = LoggedOut, email = "", password = "" }
            , Nav.pushUrl model.key "#/"
            )

        DeactivateAccountReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | accountMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        PrivacyRequestReceived (Ok response) ->
            Api.withSession model
                (\state ->
                    ( Api.updateLoggedIn model (\current -> { current | accountMessage = Just (SuccessNote ("Privacy request queued: " ++ response.kind)) })
                    , Api.fetchMyPrivacyRequests state.accessToken
                    )
                )

        PrivacyRequestReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | accountMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        MyPrivacyRequestsReceived (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | myPrivacyRequests = response.requests }), Cmd.none )

        MyPrivacyRequestsReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | accountMessage = Just (FailureNote ("Could not load your privacy requests: " ++ httpErrorLabel error)) }), Cmd.none )

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
                ( Api.updateLoggedIn model (\state -> { state | teamWorkSavedViews = saveQueueView view state.teamWorkSavedViews, teamWorkSavedViewName = "", teamWorkMessage = Just (SuccessNote ("Saved view: " ++ view.name)) }), Cmd.none )

            else if response.scope == orgTaskSavedViewScope then
                ( Api.updateLoggedIn model (\state -> { state | orgTaskSavedViews = saveQueueView view state.orgTaskSavedViews, orgTaskSavedViewName = "", orgTaskMessage = Just (SuccessNote ("Saved view: " ++ view.name)) }), Cmd.none )

            else
                ( model, Cmd.none )

        SavedQueueViewSaved (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | teamWorkMessage = Just (FailureNote (httpErrorLabel error)), orgTaskMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        OperationsReceived (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | operations = Just response, adminMessage = Nothing }), Cmd.none )

        OperationsReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | operations = Nothing, adminMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        AuditEventsReceived (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | auditEvents = response.events, auditEventsNextOffset = response.nextOffset, adminMessage = Nothing }), Cmd.none )

        AuditEventsReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | auditEvents = [], auditEventsNextOffset = 0, adminMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        PlatformAdminsReceived (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | platformAdmins = response.admins, platformAdminsNextOffset = response.nextOffset }), Cmd.none )

        PlatformAdminsReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | platformAdmins = [], platformAdminsNextOffset = 0, adminMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        AdminSelectedUserChanged userId ->
            ( Api.updateLoggedIn model (\state -> { state | adminSelectedUserId = userId }), Cmd.none )

        GrantPlatformAdminClicked ->
            Api.withSession model (\state -> ( Api.updateLoggedIn model (\current -> { current | adminMessage = Nothing }), Api.grantPlatformAdmin state.accessToken state.adminSelectedUserId ))

        PlatformAdminGranted (Ok response) ->
            Api.withSession model (\state -> ( Api.updateLoggedIn model (\current -> { current | adminSelectedUserId = "", adminMessage = Just (SuccessNote "Platform admin granted."), platformAdminsOffset = 0 }), Api.fetchPlatformAdmins state.accessToken 0 ))

        PlatformAdminGranted (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | adminMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        RevokePlatformAdminClicked userID ->
            Api.withSession model (\state -> ( Api.updateLoggedIn model (\current -> { current | adminMessage = Nothing }), Api.revokePlatformAdmin state.accessToken userID ))

        PlatformAdminRevoked (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | platformAdmins = removePlatformAdmin response.userID state.platformAdmins, adminMessage = Just (SuccessNote "Platform admin revoked.") }), Cmd.none )

        PlatformAdminRevoked (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | adminMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        AdminModerationReportsReceived (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | adminModerationReports = response.reports, adminModerationNextOffset = response.nextOffset, adminModerationTotal = response.total, adminMessage = Nothing }), Cmd.none )

        AdminModerationReportsReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | adminModerationReports = [], adminModerationNextOffset = 0, adminModerationTotal = 0, adminMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        AdminModerationStateFilterChanged value ->
            Api.withSession model (\state -> ( Api.updateLoggedIn model (\current -> { current | adminModerationStateFilter = value, adminModerationOffset = 0 }), Api.fetchAdminModerationReports state.accessToken value 0 ))

        PreviousAdminModerationPageClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            max 0 (state.adminModerationOffset - Api.selectorPageSize)
                    in
                    ( Api.updateLoggedIn model (\current -> { current | adminModerationOffset = offset }), Api.fetchAdminModerationReports state.accessToken state.adminModerationStateFilter offset )
                )

        NextAdminModerationPageClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            state.adminModerationOffset + Api.selectorPageSize
                    in
                    ( Api.updateLoggedIn model (\current -> { current | adminModerationOffset = offset }), Api.fetchAdminModerationReports state.accessToken state.adminModerationStateFilter offset )
                )

        AdminModerationResolutionNoteChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | adminModerationResolutionNote = value }), Cmd.none )

        TriageModerationReportClicked reportID stateValue ->
            Api.withSession model (\state -> ( Api.updateLoggedIn model (\current -> { current | adminMessage = Nothing }), Api.triageModerationReport state.accessToken reportID stateValue state.adminModerationResolutionNote ))

        AdminModerationReportTriaged (Ok response) ->
            -- Replace the report in place (keeping it visible with its new
            -- state badge and resolution note) rather than refetching, so the
            -- admin sees immediate confirmation of the resolution instead of
            -- the row vanishing from the current filter.
            ( Api.updateLoggedIn model (\state -> { state | adminModerationReports = replaceModerationReport response state.adminModerationReports, adminModerationResolutionNote = "", adminMessage = Just (SuccessNote "Moderation report updated.") }), Cmd.none )

        AdminModerationReportTriaged (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | adminMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        AdminPrivacyRequestsReceived (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | adminPrivacyRequests = response.requests, adminPrivacyNextOffset = response.nextOffset, adminMessage = Nothing }), Cmd.none )

        AdminPrivacyRequestsReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | adminPrivacyRequests = [], adminPrivacyNextOffset = 0, adminMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        PreviousAdminPrivacyPageClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            max 0 (state.adminPrivacyOffset - Api.selectorPageSize)
                    in
                    ( Api.updateLoggedIn model (\current -> { current | adminPrivacyOffset = offset }), Api.fetchAdminPrivacyRequests state.accessToken offset )
                )

        NextAdminPrivacyPageClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            state.adminPrivacyOffset + Api.selectorPageSize
                    in
                    ( Api.updateLoggedIn model (\current -> { current | adminPrivacyOffset = offset }), Api.fetchAdminPrivacyRequests state.accessToken offset )
                )

        AdminPrivacyResolutionNoteChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | adminPrivacyResolutionNote = value }), Cmd.none )

        GrantTargetKindChanged value ->
            -- Changing any grant input starts a new grant intent, so the
            -- idempotency key is discarded and reminted on the next submit.
            ( Api.updateLoggedIn model (\state -> { state | grantTargetKind = value, grantTargetId = "", grantKey = "" }), Cmd.none )

        GrantTargetIdChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | grantTargetId = value, grantKey = "" }), Cmd.none )

        GrantAmountChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | grantAmount = value, grantKey = "" }), Cmd.none )

        GrantNoteChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | grantNote = value, grantKey = "" }), Cmd.none )

        GrantCreditsClicked ->
            -- The key is minted on the first submit of this form intent and
            -- reused by retries of the same intent (network timeouts), so
            -- the server can dedupe instead of double-crediting. Editing any
            -- field clears it (see the *Changed handlers above).
            Api.withSession model
                (\state ->
                    Api.grantCreditsCommand model
                        state
                        (if state.grantKey == "" then
                            "ui-grant:" ++ String.fromInt (Time.posixToMillis state.now) ++ ":" ++ state.subjectId

                         else
                            state.grantKey
                        )
                )

        CreditsGranted (Ok granted) ->
            ( Api.updateLoggedIn model
                (\state ->
                    { state
                        | grantMessage = Just (SuccessNote ("Granted " ++ String.fromInt granted.amount ++ " credits."))
                        , grantTargetId = ""
                        , grantAmount = ""
                        , grantNote = ""
                        , grantKey = ""
                    }
                )
            , Api.refreshBalanceAndLedger model
            )

        CreditsGranted (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | grantMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        RunPrivacyRetentionClicked ->
            Api.withSession model (\state -> ( Api.updateLoggedIn model (\current -> { current | adminMessage = Nothing }), Api.runPrivacyRetention state.accessToken ))

        PrivacyRetentionRunReceived (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | adminRetentionRedactedFieldCount = Just response.redactedFieldCount, adminMessage = Just (SuccessNote "Privacy retention run finished.") }), Cmd.none )

        PrivacyRetentionRunReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | adminMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        ResolveAdminPrivacyRequestClicked requestId ->
            Api.withSession model (\state -> ( Api.updateLoggedIn model (\current -> { current | adminMessage = Nothing }), Api.resolveAdminPrivacyRequest state.accessToken requestId state.adminPrivacyResolutionNote ))

        AdminPrivacyRequestResolved (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | adminPrivacyRequests = replacePrivacyRequest response state.adminPrivacyRequests, adminPrivacyResolutionNote = "", adminMessage = Just (SuccessNote "Privacy request resolved.") }), Cmd.none )

        AdminPrivacyRequestResolved (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | adminMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        AuditActionFilterChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | auditActionFilter = value }), Cmd.none )

        AuditSubjectKindFilterChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | auditSubjectKindFilter = value }), Cmd.none )

        AuditSubjectIDFilterChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | auditSubjectIDFilter = value }), Cmd.none )

        SearchAuditEventsClicked ->
            Api.withSession model (\state -> ( Api.updateLoggedIn model (\current -> { current | auditEventsOffset = 0 }), Api.fetchAuditEvents state.accessToken state.auditActionFilter state.auditSubjectKindFilter state.auditSubjectIDFilter 0 ))

        PreviousAuditEventsPageClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            max 0 (state.auditEventsOffset - Api.selectorPageSize)
                    in
                    ( Api.updateLoggedIn model (\current -> { current | auditEventsOffset = offset }), Api.fetchAuditEvents state.accessToken state.auditActionFilter state.auditSubjectKindFilter state.auditSubjectIDFilter offset )
                )

        NextAuditEventsPageClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            state.auditEventsOffset + Api.selectorPageSize
                    in
                    ( Api.updateLoggedIn model (\current -> { current | auditEventsOffset = offset }), Api.fetchAuditEvents state.accessToken state.auditActionFilter state.auditSubjectKindFilter state.auditSubjectIDFilter offset )
                )

        PreviousPlatformAdminsPageClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            max 0 (state.platformAdminsOffset - Api.selectorPageSize)
                    in
                    ( Api.updateLoggedIn model (\current -> { current | platformAdminsOffset = offset }), Api.fetchPlatformAdmins state.accessToken offset )
                )

        NextPlatformAdminsPageClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            state.platformAdminsOffset + Api.selectorPageSize
                    in
                    ( Api.updateLoggedIn model (\current -> { current | platformAdminsOffset = offset }), Api.fetchPlatformAdmins state.accessToken offset )
                )

        NotificationsReceived (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | notifications = response.notifications, notificationsNextOffset = response.nextOffset, notificationsTotal = response.total, inboxMessage = Nothing }), Cmd.none )

        NotificationsReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | notifications = [], notificationsNextOffset = 0, notificationsTotal = 0, inboxMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        PreviousNotificationsPageClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            max 0 (state.notificationsOffset - Api.selectorPageSize)
                    in
                    ( Api.updateLoggedIn model (\current -> { current | notificationsOffset = offset }), Api.fetchNotifications state.accessToken state.inboxUnreadOnly offset )
                )

        NextNotificationsPageClicked ->
            Api.withSession model
                (\state ->
                    let
                        offset =
                            state.notificationsOffset + Api.selectorPageSize
                    in
                    ( Api.updateLoggedIn model (\current -> { current | notificationsOffset = offset }), Api.fetchNotifications state.accessToken state.inboxUnreadOnly offset )
                )

        MarkNotificationReadClicked notificationId ->
            Api.withSession model (\state -> ( model, Api.markNotificationRead state.accessToken notificationId ))

        NotificationReadReceived (Ok notification) ->
            -- The nav badge counts unread rows, so it must drop by one right
            -- after a successful mark-read rather than waiting for a poll.
            Api.withSession model
                (\state ->
                    ( Api.updateLoggedIn model (\current -> { current | notifications = replaceNotification notification current.notifications, inboxMessage = Nothing })
                    , Api.fetchUnreadCount state.accessToken
                    )
                )

        NotificationReadReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | inboxMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        UnreadCountReceived (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | unreadCount = response.unreadCount }), Cmd.none )

        UnreadCountReceived (Err _) ->
            -- Keep the last known count: flashing the badge away on a
            -- transient fetch failure would misreport "nothing unread".
            ( model, Cmd.none )

        InboxUnreadOnlyToggled unreadOnly ->
            Api.withSession model
                (\state ->
                    ( Api.updateLoggedIn model (\current -> { current | inboxUnreadOnly = unreadOnly, notificationsOffset = 0 })
                    , Api.fetchNotifications state.accessToken unreadOnly 0
                    )
                )

        PollTick posix ->
            -- The tick doubles as the clock for relative timestamps ("2
            -- hours ago"), so no separate Time subscription is needed.
            Api.withSession model (\state -> ( Api.updateLoggedIn model (\current -> { current | now = posix }), pollCmd state ))

        VisibilityChanged visibility ->
            case visibility of
                Browser.Events.Visible ->
                    Api.withSession model (\state -> ( model, pollCmd state ))

                Browser.Events.Hidden ->
                    ( model, Cmd.none )

        ActivityEventsReceived (Ok response) ->
            ( Api.updateLoggedIn model
                (\state ->
                    { state
                        | activityEvents = appendActivityEvents state.activityEvents response.events
                        , activityCursor =
                            if response.nextCursor == "" then
                                state.activityCursor

                            else
                                response.nextCursor
                    }
                )
            , Cmd.none
            )

        ActivityEventsReceived (Err _) ->
            -- The feed is a live convenience; keep what is already shown
            -- rather than replacing it with an error state on one bad poll.
            ( model, Cmd.none )

        WebhooksReceived (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | webhookSubscriptions = response.subscriptions }), Cmd.none )

        WebhooksReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | webhookSubscriptions = [], webhookMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        WebhookURLChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | webhookURL = value }), Cmd.none )

        ToggleWebhookKind kind ->
            ( Api.updateLoggedIn model (\state -> { state | webhookKinds = Api.toggleWebhookKind kind state.webhookKinds }), Cmd.none )

        WebhookAudienceChosen audience ->
            -- Marketplace subscriptions may listen only for task_opened (the
            -- server enforces this), so choosing marketplace pins the kinds;
            -- switching back to recipient reopens the kind checkboxes empty.
            ( Api.updateLoggedIn model
                (\state ->
                    case audience of
                        Events.WebhookAudienceMarketplace ->
                            { state | webhookAudience = audience, webhookKinds = [ Events.DomainEventKindTaskOpened ] }

                        Events.WebhookAudienceRecipient ->
                            { state | webhookAudience = audience, webhookKinds = [], webhookFilterTaskType = "", webhookFilterMinReward = "" }
                )
            , Cmd.none
            )

        WebhookFilterTaskTypeChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | webhookFilterTaskType = value }), Cmd.none )

        WebhookFilterMinRewardChanged value ->
            ( Api.updateLoggedIn model (\state -> { state | webhookFilterMinReward = value }), Cmd.none )

        CreateWebhookClicked ->
            Api.withSession model (\state -> Api.createWebhookCommand model state)

        WebhookCreated (Ok created) ->
            Api.withSession model
                (\state ->
                    ( Api.updateLoggedIn model
                        (\current ->
                            { current
                                | newWebhookSecret = Just created
                                , webhookURL = ""
                                , webhookKinds = []
                                , webhookAudience = Events.WebhookAudienceRecipient
                                , webhookFilterTaskType = ""
                                , webhookFilterMinReward = ""
                                , webhookMessage = Nothing
                            }
                        )
                    , Api.fetchWebhookSubscriptions state.accessToken
                    )
                )

        WebhookCreated (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | webhookMessage = Just (FailureNote (httpErrorLabel error)) }), Cmd.none )

        RevokeWebhookClicked subscriptionId ->
            Api.withSession model (\state -> ( model, Api.revokeWebhookSubscription state.accessToken subscriptionId ))

        WebhookRevoked (Ok _) ->
            Api.withSession model
                (\state ->
                    ( Api.updateLoggedIn model (\current -> { current | webhookMessage = Just (SuccessNote "Webhook revoked. Deliveries to it have stopped.") })
                    , Api.fetchWebhookSubscriptions state.accessToken
                    )
                )

        WebhookRevoked (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | webhookMessage = Just (FailureNote ("Could not revoke the webhook: " ++ httpErrorLabel error)) }), Cmd.none )

        OpenWebhookDeliveries subscriptionId ->
            Api.withSession model
                (\state ->
                    if state.activeWebhookDeliveriesID == Just subscriptionId then
                        ( Api.updateLoggedIn model (\current -> { current | activeWebhookDeliveriesID = Nothing, webhookDeliveries = [] }), Cmd.none )

                    else
                        ( Api.updateLoggedIn model (\current -> { current | activeWebhookDeliveriesID = Just subscriptionId, webhookDeliveries = [] })
                        , Api.fetchWebhookDeliveries state.accessToken subscriptionId
                        )
                )

        WebhookDeliveriesReceived (Ok response) ->
            ( Api.updateLoggedIn model (\state -> { state | webhookDeliveries = response.deliveries }), Cmd.none )

        WebhookDeliveriesReceived (Err error) ->
            ( Api.updateLoggedIn model (\state -> { state | webhookMessage = Just (FailureNote ("Could not load deliveries: " ++ httpErrorLabel error)) }), Cmd.none )

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
                    , Api.routeLoadCmd state.accessToken state.subjectId state.activityCursor page
                    )

                LoggedOut ->
                    ( { model | route = page }, Cmd.none )

        ResetDemoClicked ->
            ( model, reloadDemo () )

        ToggleNavMenu identifier ->
            ( Api.updateLoggedIn model
                (\state ->
                    { state
                        | openNavMenu =
                            if state.openNavMenu == Just identifier then
                                Nothing

                            else
                                Just identifier
                    }
                )
            , Cmd.none
            )


{-| One freshness pass: always the unread badge count, plus whatever live
list the current page shows (Inbox rows, Overview activity). Runs on the
15-second poll tick and when a hidden tab becomes visible again.
-}
pollCmd : LoggedInModel -> Cmd Msg
pollCmd state =
    Cmd.batch
        (Api.fetchUnreadCount state.accessToken
            :: (case state.page of
                    InboxPage ->
                        [ Api.fetchNotifications state.accessToken state.inboxUnreadOnly state.notificationsOffset ]

                    OverviewPage ->
                        [ Api.fetchEvents state.accessToken state.activityCursor ]

                    _ ->
                        []
               )
        )


{-| The next_offset carried by a paged list response, or 0 (no next page)
when the request failed - a failed page must not offer paging deeper into
nothing.
-}
nextOffsetFromResult : (response -> Int) -> Result Http.Error response -> Int
nextOffsetFromResult toNextOffset result =
    case result of
        Ok response ->
            toNextOffset response

        Err _ ->
            0


{-| The total row count carried by a counted list response, or 0 when the
request failed (the pager then reads "Page 1 of 1" over an empty list).
-}
totalFromResult : Result Http.Error { response | total : Int } -> Int
totalFromResult result =
    case result of
        Ok response ->
            response.total

        Err _ ->
            0


{-| Append newly fetched feed events, skipping any id already shown (the
first poll can race the page-entry fetch before a cursor exists), and keep
only the newest 50 so a long-lived tab does not grow without bound.
-}
appendActivityEvents : List Events.EventResponse -> List Events.EventResponse -> List Events.EventResponse
appendActivityEvents existing incoming =
    let
        knownIds =
            List.map .id existing

        merged =
            existing ++ List.filter (\event -> not (List.member event.id knownIds)) incoming
    in
    List.drop (List.length merged - 50) merged


seriesListRefresh : Model -> Cmd Msg
seriesListRefresh model =
    case model.session of
        LoggedIn state ->
            if state.page == TasksPage then
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


{-| The idempotency key for the current send-credits intent: minted on the
first submit, reused for retries of the same intent (see the *Changed
handlers, which clear it).
-}
mintSendKey : LoggedInModel -> String
mintSendKey state =
    if state.sendKey == "" then
        "ui-send:" ++ String.fromInt (Time.posixToMillis state.now) ++ ":" ++ state.subjectId

    else
        state.sendKey


{-| The confirmation for an admin catalog mutation, phrased by the state the
entry landed in (an add leaves it available; a withdraw marks it withdrawn).
-}
catalogMutationLabel : Collectible.CollectibleCatalogEntry -> String
catalogMutationLabel entry =
    case entry.state of
        Collectible.CollectibleCatalogEntryStateAvailable ->
            "Catalog entry '" ++ entry.name ++ "' is available to award."

        Collectible.CollectibleCatalogEntryStateWithdrawn ->
            "Withdrew '" ++ entry.name ++ "' from the catalog. Copies already minted keep circulating."
