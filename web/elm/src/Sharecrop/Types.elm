module Sharecrop.Types exposing (..)

import Browser
import Browser.Navigation as Nav
import Http
import Sharecrop.Generated.Admin as Admin
import Sharecrop.Generated.Agent as Agent
import Sharecrop.Generated.Auth as Auth
import Sharecrop.Generated.Collectible as Collectible
import Sharecrop.Generated.Ledger as Ledger
import Sharecrop.Generated.Notification as Notification
import Sharecrop.Generated.Organization as Organization
import Sharecrop.Generated.Submission as Submission
import Sharecrop.Generated.Task as Task
import Sharecrop.Generated.TaskSeries as TaskSeries
import Sharecrop.Generated.Team as Team
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
    | DiscoveryPage
    | FundingPage
    | AgentsPage
    | CollectiblesPage
    | OrganizationsPage
    | OrganizationDetailPage String
    | UserDetailPage String
    | UserWorkPage String
    | UserSubmissionsPage String
    | CollectibleDetailPage String
    | SeriesListPage
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


type alias LoggedInModel =
    { accessToken : String
    , subjectId : String
    , isAdmin : Bool
    , page : Page
    , balance : Maybe Int
    , entries : List Ledger.LedgerEntryResponse
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
    , createMessage : Maybe String
    , fundTaskId : String
    , fundAmount : String
    , fundOrganizationId : String
    , fundMessage : Maybe String
    , fundNonce : Int
    , tasks : List Task.TaskListItemResponse
    , taskStateFilter : String
    , agentLabel : String
    , agentScopes : List Agent.AgentScope
    , credentials : List Agent.AgentCredentialResponse
    , newCredential : Maybe Agent.AgentCredentialCreatedResponse
    , agentMessage : Maybe String
    , discoveryTasks : List Task.TaskListItemResponse
    , discoveryIncludeReserved : Bool
    , detail : Maybe PublicTaskDetail
    , detailError : Maybe String
    , reservations : List Task.TaskReservationResponse
    , reservationOrganizationId : String
    , reservationTeamId : String
    , reservationMessage : Maybe String
    , submissions : List Submission.SubmissionResponse
    , submitInput : String
    , submitMessage : Maybe String
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
    , orgTeams : List Team.TeamResponse
    , standaloneTeams : List Team.TeamResponse
    , orgMembers : List Organization.OrganizationMemberResponse
    , orgTasks : List Task.TaskListItemResponse
    , orgCollectibles : List Collectible.CollectibleResponse
    , teamCollectibles : List Collectible.CollectibleResponse
    , userProfile : Maybe Task.UserProfileResponse
    , userProfileError : Maybe String
    , userWork : List Task.TaskListItemResponse
    , userSubmissions : List Submission.SubmissionResponse
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
    , taskIntegrationOpen : Bool
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
    , adminMessage : Maybe String
    , notifications : List Notification.NotificationResponse
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
    | TasksReceived (Result Http.Error Task.TasksResponse)
    | TaskStateFilterChanged String
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
    | CreateAgentClicked
    | AgentCreated (Result Http.Error Agent.AgentCredentialCreatedResponse)
    | ToggleTaskIntegration
    | MintTaskTokenClicked
    | TaskTokenMinted (Result Http.Error Agent.AgentCredentialCreatedResponse)
    | MintUserTokenClicked
    | UserTokenMinted (Result Http.Error Agent.AgentCredentialCreatedResponse)
    | CopyClicked String
    | RevokeClicked String
    | AgentRevoked (Result Http.Error Agent.AgentCredentialResponse)
    | LogoutClicked
    | LogoutReceived (Result Http.Error ())
    | DiscoveryIncludeReservedChanged Bool
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
    | SubmitClicked
    | SubmitReceived (Result Http.Error Submission.SubmissionCreatedResponse)
    | ReviewNoteChanged String
    | ReviewPartialCreditChanged String
    | ReviewTipChanged String
    | ReviewTipCollectibleChanged String
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
    | TeamMemberEmailChanged String
    | AddTeamMemberClicked String
    | AddTeamMemberReceived (Result Http.Error Team.TeamDetailResponse)
    | OrgTasksReceived (Result Http.Error Task.TasksResponse)
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
    | OperationsReceived (Result Http.Error Admin.OperationsResponse)
    | AuditEventsReceived (Result Http.Error Admin.AuditEventsResponse)
    | NotificationsReceived (Result Http.Error Notification.NotificationsResponse)
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

        DiscoveryPage ->
            "/discovery"

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

        SeriesListPage ->
            "/series"

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
