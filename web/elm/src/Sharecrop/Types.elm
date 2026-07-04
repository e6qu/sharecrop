module Sharecrop.Types exposing (..)

import Browser
import Browser.Navigation as Nav
import File
import Http
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
import Time
import Url exposing (Url)


type alias Flags =
    { origin : String, demo : Bool }


type Session
    = LoggedOut
    | LoggedIn LoggedInModel


type Page
    = OverviewPage
    | TasksPage
    | CreateTaskPage
    | TaskDetailPage String
    | FundingPage
    | AgentsPage
    | CollectiblesPage
    | OrganizationsPage
    | OrganizationDetailPage String
    | UserDetailPage String
    | UserWorkPage String
    | UserSubmissionsPage String
    | CollectibleDetailPage String
    | SeriesDetailPage String
    | TeamDetailPage String
    | AdminPage
    | InboxPage
    | NotFoundPage


type alias SchemaFieldDraft =
    { name : String
    , kind : String
    , required : Bool
    , itemKind : String
    , enumValues : String
    }


type alias SeriesTaskEntry =
    { id : String
    , title : String
    , state : String
    }


type alias SeriesDetailData =
    { series : TaskSeries.TaskSeriesResponse
    , tasks : List SeriesTaskEntry
    , comments : List TaskSeries.SeriesCommentResponse
    }


type alias UserDirectoryEntry =
    { id : String
    , email : String
    , status : String
    }


type alias QueueView =
    { name : String
    , query : String
    , stateFilter : String
    , typeFilter : String
    , sort : String
    }


type alias SelectedAttachment =
    { name : String
    , contentType : String
    , sizeBytes : Int
    , dataURL : String
    }


type alias LoggedInModel =
    { accessToken : String
    , subjectId : String
    , isAdmin : Bool
    , page : Page
    , openNavMenu : Maybe String
    , balance : Maybe Int
    , entries : List Ledger.LedgerEntryResponse
    , ledgerOffset : Int
    , createTitle : String
    , createDescription : String
    , createResponseSchema : String
    , createSchemaFields : List SchemaFieldDraft
    , createPayloadJson : String
    , createRewardKind : String
    , createRewardAmount : String
    , createRewardCollectibleIds : List String
    , createVisibility : String
    , createScopeUserId : String
    , createScopeTeamId : String
    , createScopeOrganizationId : String
    , createAssigneeScope : Task.TaskAssigneeScope
    , createParticipationPolicy : String
    , createReservationHours : String
    , createAttachments : List SelectedAttachment
    , createMessage : Maybe String
    , fundTaskId : String
    , fundAmount : String
    , fundOrganizationId : String
    , fundMessage : Maybe String
    , fundNonce : Int
    , tasks : List Task.TaskListItemResponse
    , taskStateFilter : String
    , taskListOffset : Int
    , taskListQuery : String
    , taskListTypeFilter : String
    , taskListSort : String
    , agentLabel : String
    , agentScopes : List Agent.AgentScope
    , agentExpiresHours : String
    , credentials : List Agent.AgentCredentialResponse
    , newCredential : Maybe Agent.AgentCredentialCreatedResponse
    , agentMessage : Maybe String
    , discoveryTasks : List Task.TaskListItemResponse
    , discoveryIncludeReserved : Bool
    , discoveryOffset : Int
    , discoveryQuery : String
    , detail : Maybe PublicTaskDetail
    , detailError : Maybe String
    , reservations : List Task.TaskReservationResponse
    , reservationOrganizationId : String
    , reservationTeamId : String
    , reservationMessage : Maybe String
    , reservationSecret : Maybe String
    , submissions : List Submission.SubmissionResponse
    , submitInput : String
    , submitAttachments : List SelectedAttachment
    , submitMessage : Maybe String
    , moderationReason : Moderation.ModerationReason
    , moderationDetails : String
    , moderationMessage : Maybe String
    , reviewNote : String
    , reviewPartialCredit : String
    , reviewTip : String
    , reviewTipCollectibleId : String
    , reviewBan : Bool
    , reviewMessage : Maybe String
    , collectibles : List Collectible.CollectibleResponse
    , collectibleName : String
    , collectibleKind : Collectible.CollectibleKind
    , collectiblePolicy : Collectible.CollectibleTransferPolicy
    , collectibleMessage : Maybe String
    , awardTaskId : String
    , awardMessage : Maybe String
    , awardDefaultMessage : Maybe String
    , collectibleCatalog : List Collectible.CollectibleCatalogEntry
    , awardRecipientKind : String
    , awardRecipientId : String
    , transferRecipientId : String
    , transferMessage : Maybe String
    , organizations : List Organization.OrganizationResponse
    , createOrgName : String
    , orgMessage : Maybe String
    , activeOrgId : String
    , orgBalance : Maybe Int
    , orgLedger : List Ledger.LedgerEntryResponse
    , orgLedgerOffset : Int
    , orgAuditEvents : List Admin.AuditEventResponse
    , orgAuditMessage : Maybe String
    , orgTeams : List Team.TeamResponse
    , standaloneTeams : List Team.TeamResponse
    , orgMembers : List Organization.OrganizationMemberResponse
    , orgTasks : List Task.TaskListItemResponse
    , orgTaskQuery : String
    , orgTaskFilter : String
    , orgTaskTypeFilter : String
    , orgTaskSort : String
    , orgTaskOffset : Int
    , orgTaskMessage : Maybe String
    , orgTaskSavedViewName : String
    , orgTaskSavedViews : List QueueView
    , orgCollectibles : List Collectible.CollectibleResponse
    , orgCollectiblesMessage : Maybe String
    , orgCredentials : List Agent.OrgCredentialResponse
    , orgCredentialLabel : String
    , orgCredentialScopes : List Agent.AgentScope
    , orgCredentialExpiresHours : String
    , newOrgCredential : Maybe Agent.OrgCredentialCreatedResponse
    , orgCredentialMessage : Maybe String
    , teamCollectibles : List Collectible.CollectibleResponse
    , teamCollectiblesMessage : Maybe String
    , userProfile : Maybe Task.UserProfileResponse
    , userProfileError : Maybe String
    , userWork : List Task.TaskListItemResponse
    , userSubmissions : List Submission.SubmissionResponse
    , userSubmissionsOffset : Int
    , pendingRevisionTaskID : Maybe String
    , pendingRevisionResponse : String
    , seriesDetail : Maybe SeriesDetailData
    , seriesDetailError : Maybe String
    , seriesList : List TaskSeries.TaskSeriesResponse
    , createSeriesTitle : String
    , createSeriesDescription : String
    , seriesMessage : Maybe String
    , addSeriesTaskId : String
    , seriesCommentBody : String
    , seriesRenameTitle : String
    , seriesRenameDescription : String
    , teamDetail : Maybe Team.TeamDetailResponse
    , teamDetailError : Maybe String
    , teamWork : List Task.TaskListItemResponse
    , teamWorkQuery : String
    , teamWorkFilter : String
    , teamWorkTypeFilter : String
    , teamWorkSort : String
    , teamWorkOffset : Int
    , teamWorkMessage : Maybe String
    , teamWorkSavedViewName : String
    , teamWorkSavedViews : List QueueView
    , teamMemberEmail : String
    , teamMemberMessage : Maybe String
    , createOrgTeamName : String
    , orgTeamMessage : Maybe String
    , provisionMemberEmail : String
    , provisionMemberRoles : List String
    , provisionMemberMessage : Maybe String
    , createTaskOwner : String
    , createTaskType : String
    , createReferenceURL : String
    , taskComments : List Task.TaskCommentResponse
    , taskCommentBody : String
    , taskCommentMessage : Maybe String
    , submissionComments : List Submission.SubmissionCommentResponse
    , activeSubmissionCommentsID : Maybe String
    , submissionCommentBody : String
    , submissionCommentMessage : Maybe String
    , taskAgentToken : Maybe String
    , taskActionMessage : Maybe String
    , userAgentToken : Maybe String
    , accountEmail : String
    , currentPassword : String
    , newPassword : String
    , emailVerificationToken : String
    , emailVerificationInput : String
    , accountMessage : Maybe String
    , userDirectory : List UserDirectoryEntry
    , userDirectoryQuery : String
    , userDirectoryOffset : Int
    , organizationQuery : String
    , organizationOffset : Int
    , standaloneTeamQuery : String
    , standaloneTeamOffset : Int
    , orgTeamQuery : String
    , orgTeamOffset : Int
    , operations : Maybe Admin.OperationsResponse
    , auditEvents : List Admin.AuditEventResponse
    , auditEventsOffset : Int
    , platformAdmins : List Admin.PlatformAdminResponse
    , platformAdminsOffset : Int
    , adminSelectedUserId : String
    , adminModerationReports : List Moderation.ModerationReportResponse
    , adminModerationStateFilter : String
    , adminModerationOffset : Int
    , adminModerationResolutionNote : String
    , adminPrivacyRequests : List Privacy.PrivacyRequestResponse
    , adminPrivacyOffset : Int
    , adminPrivacyResolutionNote : String
    , adminRetentionRedactedFieldCount : Maybe Int
    , auditActionFilter : String
    , auditSubjectKindFilter : String
    , auditSubjectIDFilter : String
    , adminMessage : Maybe String
    , notifications : List Notification.NotificationResponse
    , notificationsOffset : Int
    , inboxMessage : Maybe String
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
    , reviewerAction : String
    , responseSchemaJson : String
    , payloadKind : String
    , payloadJson : String
    , attachments : List Task.TaskAttachmentResponse
    , createdBy : String
    , seriesID : String
    , taskType : String
    , referenceURL : String
    }


type alias PublicTaskDetail =
    TaskDetail


type alias Model =
    { origin : String
    , demo : Bool
    , key : Nav.Key
    , route : Page
    , email : String
    , password : String
    , resetEmail : String
    , resetToken : String
    , resetPassword : String
    , authError : Maybe String
    , session : Session
    }


type Msg
    = EmailChanged String
    | PasswordChanged String
    | RegisterClicked
    | LoginClicked
    | GuestClicked
    | ToggleNavMenu String
    | AuthReceived (Result Http.Error Auth.AuthResponse)
    | RefreshReceived (Result Http.Error Auth.AuthResponse)
    | PasswordResetEmailChanged String
    | PasswordResetTokenChanged String
    | PasswordResetPasswordChanged String
    | RequestPasswordResetClicked
    | ConfirmPasswordResetClicked
    | PasswordResetRequested (Result Http.Error String)
    | PasswordResetConfirmed (Result Http.Error ())
    | BalanceReceived (Result Http.Error Ledger.BalanceResponse)
    | LedgerReceived (Result Http.Error Ledger.LedgerResponse)
    | PreviousLedgerPageClicked
    | NextLedgerPageClicked
    | TasksReceived (Result Http.Error Task.TasksResponse)
    | TaskStateFilterChanged String
    | TaskListQueryChanged String
    | TaskListTypeFilterChanged String
    | TaskListSortChanged String
    | PreviousTasksPageClicked
    | NextTasksPageClicked
    | CreateTitleChanged String
    | CreateDescriptionChanged String
    | CreateResponseSchemaChanged String
    | AddSchemaFieldClicked
    | RemoveSchemaFieldClicked Int
    | SchemaFieldNameChanged Int String
    | SchemaFieldKindChanged Int String
    | SchemaFieldRequiredChanged Int Bool
    | SchemaFieldItemKindChanged Int String
    | SchemaFieldEnumValuesChanged Int String
    | CreatePayloadChanged String
    | CreateRewardKindChanged String
    | CreateRewardAmountChanged String
    | ToggleCreateRewardCollectible String
    | CreateVisibilityChanged String
    | CreateScopeUserIdChanged String
    | CreateScopeTeamIdChanged String
    | CreateScopeOrganizationIdChanged String
    | CreateAssigneeScopeChosen Task.TaskAssigneeScope
    | CreateParticipationChanged String
    | CreateReservationHoursChanged String
    | PickCreateAttachmentClicked
    | CreateAttachmentFileChosen File.File
    | CreateAttachmentSelected String String Int String
    | CreateAttachmentRejected String
    | RemoveCreateAttachmentClicked Int
    | CreateTaskClicked
    | CreateTaskReceived (Result Http.Error TaskDetail)
    | CredentialsReceived (Result Http.Error Agent.AgentCredentialsResponse)
    | FundTaskIdChanged String
    | FundAmountChanged String
    | FundClicked
    | FundOrganizationIdChanged String
    | FundReceived (Result Http.Error Ledger.TaskEscrowResponse)
    | OpenTaskClicked String
    | OpenTaskReceived (Result Http.Error TaskDetail)
    | RefundTaskClicked String
    | RefundTaskReceived (Result Http.Error Ledger.TaskEscrowResponse)
    | CancelTaskClicked String
    | CancelTaskReceived (Result Http.Error TaskDetail)
    | RefundCollectibleRewardClicked String
    | RefundCollectibleRewardReceived (Result Http.Error Collectible.CollectiblesResponse)
    | AgentLabelChanged String
    | ToggleScope Agent.AgentScope
    | AgentExpiresHoursChanged String
    | CreateAgentClicked
    | AgentExpiresAtResolved Time.Posix
    | AgentCreated (Result Http.Error Agent.AgentCredentialCreatedResponse)
    | MintTaskTokenClicked
    | TaskTokenMinted (Result Http.Error Agent.AgentCredentialCreatedResponse)
    | MintUserTokenClicked
    | UserTokenMinted (Result Http.Error Agent.AgentCredentialCreatedResponse)
    | CopyClicked String
    | RevokeClicked String
    | AgentRevoked (Result Http.Error Agent.AgentCredentialResponse)
    | OrgCredentialsReceived (Result Http.Error Agent.OrgCredentialsResponse)
    | OrgCredentialLabelChanged String
    | ToggleOrgCredentialScope Agent.AgentScope
    | OrgCredentialExpiresHoursChanged String
    | CreateOrgCredentialClicked
    | OrgCredentialExpiresAtResolved Time.Posix
    | OrgCredentialCreated (Result Http.Error Agent.OrgCredentialCreatedResponse)
    | RevokeOrgCredentialClicked String
    | OrgCredentialRevoked (Result Http.Error Agent.OrgCredentialResponse)
    | LogoutClicked
    | LogoutReceived (Result Http.Error ())
    | DiscoveryIncludeReservedChanged Bool
    | DiscoveryQueryChanged String
    | PreviousDiscoveryPageClicked
    | NextDiscoveryPageClicked
    | DiscoveryReceived (Result Http.Error Task.TasksResponse)
    | DiscoveryViewClicked String
    | DetailReceived (Result Http.Error PublicTaskDetail)
    | ReserveClicked String
    | ReservationOrganizationIdChanged String
    | ReservationTeamIdChanged String
    | ReservationReceived (Result Http.Error Task.TaskReservationResponse)
    | ReservationsReceived (Result Http.Error Task.TaskReservationsResponse)
    | ApproveReservationClicked String
    | DeclineReservationClicked String
    | CancelReservationClicked String
    | ReservationChangeReceived (Result Http.Error Task.TaskReservationResponse)
    | SubmissionsReceived (Result Http.Error Submission.SubmissionsResponse)
    | SubmitInputChanged String
    | PickSubmitAttachmentClicked
    | SubmitAttachmentFileChosen File.File
    | SubmitAttachmentSelected String String Int String
    | SubmitAttachmentRejected String
    | RemoveSubmitAttachmentClicked Int
    | SubmitClicked
    | SubmitReceived (Result Http.Error Submission.SubmissionCreatedResponse)
    | ModerationReasonChanged Moderation.ModerationReason
    | ModerationDetailsChanged String
    | ReportTaskClicked String
    | ModerationReportReceived (Result Http.Error Moderation.ModerationReportResponse)
    | ReviewNoteChanged String
    | ReviewPartialCreditChanged String
    | ReviewTipChanged String
    | ReviewTipCollectibleChanged String
    | ReviewBanChanged Bool
    | AcceptClicked String
    | RequestChangesClicked String
    | RejectClicked String
    | ReviewActionReceived String (Result Http.Error ())
    | CollectibleNameChanged String
    | CollectibleKindChosen Collectible.CollectibleKind
    | CollectiblePolicyChosen Collectible.CollectibleTransferPolicy
    | MintClicked
    | MintReceived (Result Http.Error Collectible.CollectibleResponse)
    | CollectiblesReceived (Result Http.Error Collectible.CollectiblesResponse)
    | AwardTaskIdChanged String
    | AwardClicked String
    | AwardReceived (Result Http.Error Collectible.CollectibleResponse)
    | CollectibleCatalogReceived (Result Http.Error Collectible.CollectibleCatalogResponse)
    | AwardRecipientKindChanged String
    | AwardRecipientIdChanged String
    | AwardDefaultClicked String
    | AwardDefaultReceived (Result Http.Error Collectible.CollectibleResponse)
    | TransferRecipientIdChanged String
    | TransferCollectibleClicked String
    | TransferCollectibleReceived (Result Http.Error Collectible.CollectibleResponse)
    | OrganizationsReceived (Result Http.Error Organization.OrganizationsResponse)
    | CreateOrgNameChanged String
    | CreateOrgClicked
    | CreateOrgReceived (Result Http.Error Organization.OrganizationResponse)
    | OrgBalanceReceived (Result Http.Error Ledger.BalanceResponse)
    | OrgLedgerReceived (Result Http.Error Ledger.LedgerResponse)
    | PreviousOrgLedgerPageClicked
    | NextOrgLedgerPageClicked
    | OrgAuditEventsReceived (Result Http.Error Admin.AuditEventsResponse)
    | OrgTeamsReceived (Result Http.Error Team.TeamsResponse)
    | StandaloneTeamsReceived (Result Http.Error Team.TeamsResponse)
    | UserDirectoryReceived (Result Http.Error (List UserDirectoryEntry))
    | UserDirectoryQueryChanged String
    | SearchUserDirectoryClicked
    | PreviousUserDirectoryPageClicked
    | NextUserDirectoryPageClicked
    | OrganizationQueryChanged String
    | SearchOrganizationsClicked
    | PreviousOrganizationsPageClicked
    | NextOrganizationsPageClicked
    | StandaloneTeamQueryChanged String
    | SearchStandaloneTeamsClicked
    | PreviousStandaloneTeamsPageClicked
    | NextStandaloneTeamsPageClicked
    | OrgTeamQueryChanged String
    | SearchOrgTeamsClicked
    | PreviousOrgTeamsPageClicked
    | NextOrgTeamsPageClicked
    | OrgMembersReceived (Result Http.Error Organization.OrganizationMembersResponse)
    | UserProfileReceived (Result Http.Error Task.UserProfileResponse)
    | UserWorkReceived (Result Http.Error Task.TasksResponse)
    | UserSubmissionsReceived (Result Http.Error Submission.SubmissionsResponse)
    | PreviousUserSubmissionsPageClicked
    | NextUserSubmissionsPageClicked
    | StartRevisionClicked String String
    | SeriesListReceived (Result Http.Error TaskSeries.TaskSeriesListResponse)
    | CreateSeriesTitleChanged String
    | CreateSeriesDescriptionChanged String
    | CreateSeriesClicked
    | SeriesDetailReceived (Result Http.Error SeriesDetailData)
    | SeriesMutationReceived (Result Http.Error SeriesDetailData)
    | PublishSeriesClicked String
    | UnpublishSeriesClicked String
    | CloseSeriesClicked String
    | ReopenSeriesClicked String
    | AddSeriesTaskIdChanged String
    | AddSeriesTaskClicked String
    | RemoveSeriesTaskClicked String String
    | MoveSeriesTaskUpClicked String String
    | MoveSeriesTaskDownClicked String String
    | SeriesCommentBodyChanged String
    | AddSeriesCommentClicked String
    | SeriesCommentReceived (Result Http.Error TaskSeries.SeriesCommentResponse)
    | SeriesRenameTitleChanged String
    | SeriesRenameDescriptionChanged String
    | UpdateSeriesClicked String
    | TeamDetailReceived (Result Http.Error Team.TeamDetailResponse)
    | TeamWorkReceived (Result Http.Error Task.TasksResponse)
    | TeamWorkQueryChanged String
    | TeamWorkFilterChanged String
    | TeamWorkTypeFilterChanged String
    | TeamWorkSortChanged String
    | TeamWorkSavedViewNameChanged String
    | SaveTeamWorkViewClicked
    | ApplyTeamWorkViewClicked String
    | SavedQueueViewsReceived (Result Http.Error SavedQueueViews.SavedQueueViewsResponse)
    | SavedQueueViewSaved (Result Http.Error SavedQueueViews.SavedQueueViewResponse)
    | SearchTeamWorkClicked
    | PreviousTeamWorkPageClicked
    | NextTeamWorkPageClicked
    | TeamMemberEmailChanged String
    | AddTeamMemberClicked String
    | AddTeamMemberReceived (Result Http.Error Team.TeamDetailResponse)
    | OrgTasksReceived (Result Http.Error Task.TasksResponse)
    | OrgTaskQueryChanged String
    | OrgTaskFilterChanged String
    | OrgTaskTypeFilterChanged String
    | OrgTaskSortChanged String
    | OrgTaskSavedViewNameChanged String
    | SaveOrgTaskViewClicked
    | ApplyOrgTaskViewClicked String
    | SearchOrgTasksClicked
    | PreviousOrgTasksPageClicked
    | NextOrgTasksPageClicked
    | OrgCollectiblesReceived (Result Http.Error Collectible.CollectiblesResponse)
    | TeamCollectiblesReceived (Result Http.Error Collectible.CollectiblesResponse)
    | CreateOrgTeamNameChanged String
    | CreateOrgTeamClicked
    | CreateOrgTeamReceived (Result Http.Error Team.TeamResponse)
    | ProvisionMemberEmailChanged String
    | ToggleProvisionMemberRole String
    | ProvisionMemberClicked
    | ProvisionMemberReceived (Result Http.Error ())
    | UpdateMemberRolesClicked String (List String)
    | UpdateMemberRolesReceived (Result Http.Error Organization.OrganizationMemberResponse)
    | DeactivateMemberClicked String
    | DeactivateMemberReceived (Result Http.Error ())
    | CreateTaskOwnerChanged String
    | CreateTaskTypeChanged String
    | CreateReferenceURLChanged String
    | TaskCommentBodyChanged String
    | AddTaskCommentClicked String
    | TaskCommentReceived (Result Http.Error Task.TaskCommentResponse)
    | TaskCommentsReceived (Result Http.Error (List Task.TaskCommentResponse))
    | OpenSubmissionComments String
    | SubmissionCommentsReceived (Result Http.Error Submission.SubmissionCommentsResponse)
    | SubmissionCommentBodyChanged String
    | AddSubmissionCommentClicked String
    | SubmissionCommentAdded (Result Http.Error Submission.SubmissionCommentResponse)
    | LinkClicked Browser.UrlRequest
    | UrlChanged Url
    | ResetDemoClicked
    | AccountEmailChanged String
    | CurrentPasswordChanged String
    | NewPasswordChanged String
    | EmailVerificationInputChanged String
    | RequestEmailVerificationClicked
    | ConfirmEmailVerificationClicked
    | UpdateProfileClicked
    | ChangePasswordClicked
    | DeactivateAccountClicked
    | EmailVerificationRequested (Result Http.Error String)
    | AccountActionReceived (Result Http.Error ())
    | DeactivateAccountReceived (Result Http.Error ())
    | PrivacyRequestClicked Privacy.PrivacyRequestKind
    | PrivacyRequestReceived (Result Http.Error Privacy.PrivacyRequestResponse)
    | OperationsReceived (Result Http.Error Admin.OperationsResponse)
    | AuditEventsReceived (Result Http.Error Admin.AuditEventsResponse)
    | PreviousAuditEventsPageClicked
    | NextAuditEventsPageClicked
    | PlatformAdminsReceived (Result Http.Error Admin.PlatformAdminsResponse)
    | PreviousPlatformAdminsPageClicked
    | NextPlatformAdminsPageClicked
    | AdminSelectedUserChanged String
    | GrantPlatformAdminClicked
    | PlatformAdminGranted (Result Http.Error Admin.PlatformAdminResponse)
    | RevokePlatformAdminClicked String
    | PlatformAdminRevoked (Result Http.Error Admin.PlatformAdminResponse)
    | AdminModerationReportsReceived (Result Http.Error Moderation.ModerationReportsResponse)
    | AdminModerationStateFilterChanged String
    | PreviousAdminModerationPageClicked
    | NextAdminModerationPageClicked
    | AdminModerationResolutionNoteChanged String
    | TriageModerationReportClicked String String
    | AdminModerationReportTriaged (Result Http.Error Moderation.ModerationReportResponse)
    | AdminPrivacyRequestsReceived (Result Http.Error Privacy.PrivacyRequestsResponse)
    | PreviousAdminPrivacyPageClicked
    | NextAdminPrivacyPageClicked
    | AdminPrivacyResolutionNoteChanged String
    | RunPrivacyRetentionClicked
    | PrivacyRetentionRunReceived (Result Http.Error Privacy.PrivacyRetentionRunResponse)
    | ResolveAdminPrivacyRequestClicked String
    | AdminPrivacyRequestResolved (Result Http.Error Privacy.PrivacyRequestResponse)
    | AuditActionFilterChanged String
    | AuditSubjectKindFilterChanged String
    | AuditSubjectIDFilterChanged String
    | SearchAuditEventsClicked
    | NotificationsReceived (Result Http.Error Notification.NotificationsResponse)
    | PreviousNotificationsPageClicked
    | NextNotificationsPageClicked
    | MarkNotificationReadClicked String
    | NotificationReadReceived (Result Http.Error Notification.NotificationResponse)


pageToPath : Page -> String
pageToPath page =
    case page of
        OverviewPage ->
            "/"

        TasksPage ->
            "/tasks"

        CreateTaskPage ->
            "/tasks/new"

        TaskDetailPage taskId ->
            "/tasks/" ++ taskId

        FundingPage ->
            "/funding"

        AgentsPage ->
            "/agents"

        CollectiblesPage ->
            "/collectibles"

        OrganizationsPage ->
            "/organizations"

        OrganizationDetailPage organizationId ->
            "/organizations/" ++ organizationId

        UserDetailPage userId ->
            "/users/" ++ userId

        UserWorkPage userId ->
            "/users/" ++ userId ++ "/work"

        UserSubmissionsPage userId ->
            "/users/" ++ userId ++ "/submissions"

        CollectibleDetailPage collectibleId ->
            "/collectibles/" ++ collectibleId

        SeriesDetailPage seriesId ->
            "/series/" ++ seriesId

        TeamDetailPage teamId ->
            "/teams/" ++ teamId

        AdminPage ->
            "/admin"

        InboxPage ->
            "/inbox"

        NotFoundPage ->
            "/not-found"


visibilityPublicTag : String
visibilityPublicTag =
    "public"


visibilityDefaultTag : String
visibilityDefaultTag =
    "default"


visibilityUserTag : String
visibilityUserTag =
    "user"


visibilityTeamTag : String
visibilityTeamTag =
    "team"


visibilityOrganizationTag : String
visibilityOrganizationTag =
    "organization"


allVisibilityTags : List String
allVisibilityTags =
    [ visibilityPublicTag, visibilityDefaultTag, visibilityUserTag, visibilityTeamTag, visibilityOrganizationTag ]


visibilityLabel : String -> String
visibilityLabel tag =
    if tag == visibilityPublicTag then
        "Public"

    else if tag == visibilityUserTag then
        "Specific user"

    else if tag == visibilityTeamTag then
        "Team"

    else if tag == visibilityOrganizationTag then
        "Organization"

    else
        "Private (default)"
