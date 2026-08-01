module Sharecrop.Types exposing (..)

import Browser
import Browser.Events
import Dict
import Browser.Navigation as Nav
import File
import Http
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
import Time
import Url exposing (Url)


type alias Flags =
    { origin : String, demo : Bool, shauth : Bool }


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


type alias UserDirectoryPage =
    { users : List UserDirectoryEntry
    , nextOffset : Int
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


{-| A credit account's two sections: spendable credits (what can be spent or
used to fund tasks) and credits currently allocated to (locked in) tasks,
which cannot be spent until the task finishes or is refunded.
-}
type alias Wallet =
    { spendable : Int
    , allocated : Int
    }


{-| A fetched list that remembers whether its last load failed. An API
outage must not render as "No tasks yet": views branch on `failure` and
show a visible load-error state instead of the genuine-emptiness state.
`failure` carries the user-facing error text; the items are whatever the
last successful load returned (cleared on failure so stale rows are not
presented as current).
-}
type alias Loaded a =
    { items : List a
    , failure : Maybe String
    }


{-| The state every list starts in: nothing loaded, nothing failed.
-}
loadedNone : Loaded a
loadedNone =
    { items = [], failure = Nothing }


type alias LoggedInModel =
    { accessToken : String
    , subjectId : String
    , username : String
    , displayName : String
    , now : Time.Posix
    , isAdmin : Bool
    , page : Page
    , openNavMenu : Maybe String
    , balance : Maybe Wallet
    , entries : Loaded Ledger.LedgerEntryResponse
    , ledgerOffset : Int
    , ledgerNextOffset : Int
    , ledgerTotal : Int
    , unreadCount : Int
    , inboxUnreadOnly : Bool
    , activityEvents : List Events.EventResponse
    , activityCursor : String
    , webhookSubscriptions : List Events.WebhookSubscriptionResponse
    , webhookURL : String
    , webhookKinds : List Events.DomainEventKind
    , webhookAudience : Events.WebhookAudience
    , webhookFilterTaskType : String
    , webhookFilterMinReward : String
    , webhookMessage : Maybe Note
    , newWebhookSecret : Maybe Events.WebhookSubscriptionCreatedResponse
    , activeWebhookDeliveriesID : Maybe String
    , webhookDeliveries : List Events.WebhookDeliveryResponse
    , createTitle : String
    , createTitleInvalid : Bool
    , createDescription : String
    , createDescriptionInvalid : Bool
    , createResponseSchema : String
    , createSchemaFields : List SchemaFieldDraft
    , createPayloadJson : String
    , createRewardKind : String
    , createRewardAmount : String
    , createRewardAmountInvalid : Bool
    , createRewardCollectibleIds : List String
    , createVisibility : String
    , createScopeUserId : String
    , createScopeTeamId : String
    , createScopeOrganizationId : String
    , createAssigneeScope : Task.TaskAssigneeScope
    , createParticipationPolicy : String
    , createReservationHours : String
    , createExpiresAt : String
    , createAttachments : List SelectedAttachment
    , createMessage : Maybe Note
    , fundTaskId : String
    , fundAmount : String
    , fundOrganizationId : String
    , fundMessage : Maybe Note
    , fundNonce : Int
    , sendRecipientKind : String
    , sendRecipientId : String
    , sendAmount : String
    , sendNote : String
    , sendKey : String
    , sendMessage : Maybe Note
    , tasks : Loaded Task.TaskListItemResponse
    , taskStateFilter : List String
    , taskListOffset : Int
    , taskListNextOffset : Int
    , taskListTotal : Int
    , taskListQuery : String
    , taskListTypeFilter : String
    , taskListSort : String
    , agentLabel : String
    , agentScopes : List Agent.AgentScope
    , agentExpiresHours : String
    , credentials : Loaded Agent.AgentCredentialResponse
    , newCredential : Maybe Agent.AgentCredentialCreatedResponse
    , agentMessage : Maybe Note
    , discoveryTasks : Loaded Task.TaskListItemResponse
    , discoveryIncludeReserved : Bool
    , discoveryFundedOnly : Bool
    , discoveryOffset : Int
    , discoveryNextOffset : Int
    , discoveryTotal : Int
    , discoveryQuery : String
    , detail : Maybe PublicTaskDetail
    , detailError : Maybe String
    , reservations : List Task.TaskReservationResponse
    , reservationOrganizationId : String
    , reservationTeamId : String
    , reservationMessage : Maybe Note
    , reservationSecret : Maybe String
    , submissions : List Submission.SubmissionResponse
    , submitInput : String
    , submitFieldValues : Dict.Dict String String
    , submitRawMode : Bool
    , submitAttachments : List SelectedAttachment
    , submitMessage : Maybe Note
    , moderationReason : Moderation.ModerationReason
    , moderationSubject : ModerationSubject
    , moderationDetails : String
    , moderationMessage : Maybe Note
    , reviewNote : String
    , reviewPartialCredit : String
    , reviewTip : String
    , reviewTipCollectibleId : String
    , reviewBan : Ledger.BanSelection
    , reviewMessage : Maybe Note
    , collectibles : Loaded Collectible.CollectibleResponse
    , collectibleName : String
    , collectibleKind : Collectible.CollectibleKind
    , collectiblePolicy : Collectible.CollectibleTransferPolicy
    , collectibleMessage : Maybe Note
    , awardTaskId : String
    , awardMessage : Maybe Note
    , awardDefaultMessage : Maybe Note
    , collectibleCatalog : Loaded Collectible.CollectibleCatalogEntry
    , awardRecipientKind : String
    , awardRecipientId : String
    , transferRecipientId : String
    , transferMessage : Maybe Note
    , openSendCollectibleID : Maybe String
    , sendCollectibleTargetKind : String
    , withdrawMessage : Maybe Note
    , catalogSlug : String
    , catalogName : String
    , catalogKind : Collectible.CollectibleKind
    , catalogPolicy : Collectible.CollectibleTransferPolicy
    , catalogArt : String
    , catalogMaxEditions : String
    , catalogMessage : Maybe Note
    , organizations : Loaded Organization.OrganizationResponse
    , createOrgName : String
    , orgMessage : Maybe Note
    , activeOrgId : String
    , orgBalance : Maybe Wallet
    , orgLedger : Loaded Ledger.LedgerEntryResponse
    , orgLedgerOffset : Int
    , orgLedgerNextOffset : Int
    , orgLedgerTotal : Int
    , orgAuditEvents : List Admin.AuditEventResponse
    , orgAuditMessage : Maybe Note
    , orgTeams : Loaded Team.TeamResponse
    , standaloneTeams : Loaded Team.TeamResponse
    , createTeamName : String
    , createTeamMessage : Maybe Note
    , orgMembers : Loaded Organization.OrganizationMemberResponse
    , orgTasks : List Task.TaskListItemResponse
    , orgTaskQuery : String
    , orgTaskFilter : String
    , orgTaskTypeFilter : String
    , orgTaskSort : String
    , orgTaskOffset : Int
    , orgTaskNextOffset : Int
    , orgTaskTotal : Int
    , orgTaskMessage : Maybe Note
    , orgTaskSavedViewName : String
    , orgTaskSavedViews : List QueueView
    , orgCollectibles : List Collectible.CollectibleResponse
    , orgCollectiblesMessage : Maybe Note
    , awardOrgCollectibleRecipientId : String
    , awardOrgCollectibleMessage : Maybe Note
    , orgSendCollectibleRecipientId : String
    , orgSendCollectibleMessage : Maybe Note
    , orgCredentials : Loaded Agent.OrgCredentialResponse
    , orgCredentialLabel : String
    , orgCredentialScopes : List Agent.AgentScope
    , orgCredentialExpiresHours : String
    , newOrgCredential : Maybe Agent.OrgCredentialCreatedResponse
    , orgCredentialMessage : Maybe Note
    , teamCollectibles : List Collectible.CollectibleResponse
    , teamCollectiblesMessage : Maybe Note
    , userProfile : Maybe Task.UserProfileResponse
    , userProfileError : Maybe String
    , userWork : Loaded Task.TaskListItemResponse
    , userSubmissions : Loaded Submission.SubmissionResponse
    , userSubmissionsOffset : Int
    , userSubmissionsNextOffset : Int
    , userSubmissionsTotal : Int
    , pendingRevisionTaskID : Maybe String
    , pendingRevisionResponse : String
    , seriesDetail : Maybe SeriesDetailData
    , seriesDetailError : Maybe String
    , seriesList : Loaded TaskSeries.TaskSeriesResponse
    , createSeriesTitle : String
    , createSeriesDescription : String
    , seriesMessage : Maybe Note
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
    , teamWorkNextOffset : Int
    , teamWorkTotal : Int
    , teamWorkMessage : Maybe Note
    , teamWorkSavedViewName : String
    , teamWorkSavedViews : List QueueView
    , teamMemberEmail : String
    , teamMemberMessage : Maybe Note
    , createOrgTeamName : String
    , orgTeamMessage : Maybe Note
    , provisionMemberEmail : String
    , provisionMemberRoles : List String
    , provisionMemberMessage : Maybe Note
    , createTaskOwner : String
    , createTaskType : String
    , createReferenceURL : String
    , taskComments : List Task.TaskCommentResponse
    , taskCommentBody : String
    , taskCommentMessage : Maybe Note
    , submissionComments : List Submission.SubmissionCommentResponse
    , activeSubmissionCommentsID : Maybe String
    , submissionCommentBody : String
    , submissionCommentMessage : Maybe Note
    , taskAgentToken : Maybe String
    , taskActionMessage : Maybe Note
    , userAgentToken : Maybe String
    , accountProfile : Maybe Auth.AccountProfileResponse
    , displayNameDraft : String
    , accountEmail : String
    , currentPassword : String
    , newPassword : String
    , emailVerificationToken : String
    , emailVerificationInput : String
    , accountMessage : Maybe Note
    , deactivateConfirming : Bool
    , myPrivacyRequests : List Privacy.PrivacyRequestResponse
    , userDirectory : List UserDirectoryEntry
    , userDirectoryQuery : String
    , userDirectoryOffset : Int
    , userDirectoryNextOffset : Int
    , organizationQuery : String
    , organizationOffset : Int
    , organizationNextOffset : Int
    , standaloneTeamQuery : String
    , standaloneTeamOffset : Int
    , standaloneTeamNextOffset : Int
    , orgTeamQuery : String
    , orgTeamOffset : Int
    , orgTeamNextOffset : Int
    , operations : Maybe Admin.OperationsResponse
    , auditEvents : List Admin.AuditEventResponse
    , auditEventsOffset : Int
    , auditEventsNextOffset : Int
    , platformAdmins : List Admin.PlatformAdminResponse
    , platformAdminsOffset : Int
    , platformAdminsNextOffset : Int
    , adminSelectedUserId : String
    , adminModerationReports : List Moderation.ModerationReportResponse
    , adminModerationStateFilter : String
    , adminModerationOffset : Int
    , adminModerationNextOffset : Int
    , adminModerationTotal : Int
    , adminModerationResolutionNote : String
    , adminPrivacyRequests : List Privacy.PrivacyRequestResponse
    , adminPrivacyOffset : Int
    , adminPrivacyNextOffset : Int
    , adminPrivacyResolutionNote : String
    , adminRetentionRedactedFieldCount : Maybe Int
    , grantTargetKind : String
    , grantTargetId : String
    , grantAmount : String
    , grantNote : String
    , grantKey : String
    , grantMessage : Maybe Note
    , auditActionFilter : String
    , auditSubjectKindFilter : String
    , auditSubjectIDFilter : String
    , adminMessage : Maybe Note
    , notifications : List Notification.NotificationResponse
    , notificationsOffset : Int
    , notificationsNextOffset : Int
    , notificationsTotal : Int
    , inboxMessage : Maybe Note
    }


type alias TaskDetail =
    { id : String
    , title : String
    , description : String
    , state : Task.TaskState
    , rewardKind : String
    , rewardCreditAmount : Int
    , rewardCollectibleCount : Int
    , allocatedCredits : Int
    , allocatedCollectibleIDs : List String
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
    , creatorDisplayName : String
    , seriesID : String
    , taskType : String
    , referenceURL : String
    , expiresAt : String
    }


type alias PublicTaskDetail =
    TaskDetail


type alias Model =
    { origin : String
    , demo : Bool
    , shauth : Bool
    , key : Nav.Key
    , route : Page
    , email : String
    , password : String
    , registerName : String
    , resetEmail : String
    , resetToken : String
    , resetPassword : String
    , authError : Maybe String
    , authNotice : Maybe String
    , sessionNotice : Maybe String
    , session : Session
    }


type Msg
    = EmailChanged String
    | PasswordChanged String
    | RegisterNameChanged String
    | NowReceived Time.Posix
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
    | SendRecipientKindChanged String
    | SendRecipientIdChanged String
    | SendAmountChanged String
    | SendNoteChanged String
    | SendCreditsClicked
    | OrgSendCreditsClicked
    | CreditsSentReceived String (Result Http.Error Ledger.CreditTransferResponse)
    | PreviousLedgerPageClicked
    | NextLedgerPageClicked
    | TasksReceived (Result Http.Error Task.TasksResponse)
    | TaskStateFilterToggled String
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
    | CreateExpiresAtChanged String
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
    | FundReceived (Result Http.Error Ledger.TaskFundResponse)
    | OpenTaskClicked String
    | OpenTaskReceived (Result Http.Error TaskDetail)
    | UnpublishTaskClicked String
    | UnpublishTaskReceived (Result Http.Error TaskDetail)
    | RefundTaskClicked String
    | RefundTaskReceived (Result Http.Error Ledger.TaskFundResponse)
    | CancelTaskClicked String
    | CancelTaskReceived (Result Http.Error TaskDetail)
    | RefundCollectibleRewardClicked String
    | RefundCollectibleRewardReceived (Result Http.Error Collectible.CollectiblesResponse)
    | AgentLabelChanged String
    | ToggleScope Agent.AgentScope
    | AgentScopePresetClicked (List Agent.AgentScope)
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
    | LogoutReceived (Result Http.Error String)
    | DiscoveryIncludeReservedChanged Bool
    | DiscoveryFundedOnlyChanged Bool
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
    | FileDisputeClicked String
    | ReportSubjectClearedClicked
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
    | AwardOrgCollectibleRecipientIdChanged String
    | AwardOrgCollectibleClicked String
    | AwardOrgCollectibleReceived (Result Http.Error Collectible.CollectibleResponse)
    | CollectibleCatalogReceived (Result Http.Error Collectible.CollectibleCatalogResponse)
    | AwardRecipientKindChanged String
    | AwardRecipientIdChanged String
    | AwardDefaultClicked String
    | AwardDefaultReceived (Result Http.Error Collectible.CollectibleResponse)
    | TransferRecipientIdChanged String
    | TransferCollectibleClicked String
    | TransferCollectibleReceived (Result Http.Error Collectible.CollectibleResponse)
    | ToggleSendCollectible String
    | SendCollectibleTargetKindChanged String
    | SendCollectibleClicked String
    | OrgSendCollectibleRecipientIdChanged String
    | OrgSendCollectibleClicked String
    | OrgSendCollectibleReceived (Result Http.Error Collectible.CollectibleResponse)
    | CatalogSlugChanged String
    | CatalogNameChanged String
    | CatalogKindChosen Collectible.CollectibleKind
    | CatalogPolicyChosen Collectible.CollectibleTransferPolicy
    | CatalogArtChosen String
    | CatalogMaxEditionsChanged String
    | AddCatalogEntryClicked
    | CatalogEntryMutated (Result Http.Error Collectible.CollectibleCatalogEntry)
    | WithdrawCatalogEntryClicked String
    | DeleteCatalogEntryClicked String
    | CatalogEntryDeleted (Result Http.Error ())
    | WithdrawCollectibleClicked String
    | CollectibleWithdrawnReceived (Result Http.Error Collectible.CollectibleResponse)
    | DeleteCollectibleClicked String
    | CollectibleDeleted (Result Http.Error ())
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
    | UserDirectoryReceived (Result Http.Error UserDirectoryPage)
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
    | AccountProfileReceived (Result Http.Error Auth.AccountProfileResponse)
    | DisplayNameDraftChanged String
    | SaveDisplayNameClicked
    | DisplayNameSaved (Result Http.Error ())
    | CurrentPasswordChanged String
    | NewPasswordChanged String
    | EmailVerificationInputChanged String
    | RequestEmailVerificationClicked
    | ConfirmEmailVerificationClicked
    | UpdateProfileClicked
    | ChangePasswordClicked
    | DeactivateAccountClicked
    | ConfirmDeactivateAccountClicked
    | CancelDeactivateAccountClicked
    | EmailVerificationRequested (Result Http.Error String)
    | AccountActionReceived (Result Http.Error ())
    | DeactivateAccountReceived (Result Http.Error ())
    | PrivacyRequestClicked Privacy.PrivacyRequestKind
    | PrivacyRequestReceived (Result Http.Error Privacy.PrivacyRequestResponse)
    | MyPrivacyRequestsReceived (Result Http.Error Privacy.PrivacyRequestsResponse)
    | CreateTeamNameChanged String
    | CreateTeamClicked
    | TeamCreated (Result Http.Error Team.TeamResponse)
    | SubmitFieldChanged String String
    | SubmitRawModeToggled Bool
    | SessionRefreshTick Time.Posix
    | SessionRefreshed (Result Http.Error Auth.AuthResponse)
    | SessionEnded
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
    | GrantTargetKindChanged String
    | GrantTargetIdChanged String
    | GrantAmountChanged String
    | GrantNoteChanged String
    | GrantCreditsClicked
    | CreditsGranted (Result Http.Error Ledger.CreditGrantResponse)
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
    | UnreadCountReceived (Result Http.Error Notification.NotificationUnreadCountResponse)
    | InboxUnreadOnlyToggled Bool
    | PollTick Time.Posix
    | VisibilityChanged Browser.Events.Visibility
    | ActivityEventsReceived (Result Http.Error Events.EventListResponse)
    | WebhooksReceived (Result Http.Error Events.WebhookSubscriptionsResponse)
    | WebhookURLChanged String
    | ToggleWebhookKind Events.DomainEventKind
    | WebhookAudienceChosen Events.WebhookAudience
    | WebhookFilterTaskTypeChanged String
    | WebhookFilterMinRewardChanged String
    | CreateWebhookClicked
    | WebhookCreated (Result Http.Error Events.WebhookSubscriptionCreatedResponse)
    | RevokeWebhookClicked String
    | WebhookRevoked (Result Http.Error Events.WebhookSubscriptionResponse)
    | OpenWebhookDeliveries String
    | WebhookDeliveriesReceived (Result Http.Error Events.WebhookDeliveriesResponse)


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


-- Note is a user-facing outcome message. Failures and successes carry the
-- same shape of text but must not look the same, so the distinction is made
-- at the type level where the outcome is known (the update handler), not
-- guessed from wording in the view.


type Note
    = SuccessNote String
    | FailureNote String


{-| What a moderation report on the task-detail page is about: the task
itself (the default), or one specific submission — the shape a worker's
dispute takes. Modeled as a union rather than an optional id so the report
form always knows exactly which subject it will send.
-}
type ModerationSubject
    = ReportAboutTask
    | ReportAboutSubmission String


-- pageSize is the limit sent with every paginated list request. Views use it
-- to tell when a page is the last one (a short page), so the Next button can
-- be disabled instead of paging into blank pages.


pageSize : Int
pageSize =
    20


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
