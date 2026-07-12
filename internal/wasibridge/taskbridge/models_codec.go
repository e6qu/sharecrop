package taskbridge

import (
	"fmt"

	"github.com/e6qu/sharecrop/internal/task"
	"github.com/e6qu/sharecrop/internal/wasibridge/attachmentwire"
	"github.com/e6qu/sharecrop/internal/wasibridge/corewire"
)

// ---- task.Task ----

type taskWire struct {
	ID             string                `json:"id"`
	Owner          ownerWire             `json:"owner"`
	Title          string                `json:"title"`
	Description    string                `json:"description"`
	Type           string                `json:"type"`
	Reference      string                `json:"reference"`
	Reward         rewardSpecWire        `json:"reward"`
	Participation  string                `json:"participation"`
	AssigneeScope  string                `json:"assignee_scope"`
	ReservationTTL int                   `json:"reservation_ttl"`
	State          string                `json:"state"`
	Visibility     visibilityWire        `json:"visibility"`
	Placement      seriesPlacementWire   `json:"placement"`
	ResponseSchema string                `json:"response_schema"`
	Payload        dataPayloadWire       `json:"payload"`
	Attachments    []attachmentwire.Wire `json:"attachments,omitempty"`
	CreatedBy      string                `json:"created_by"`
}

func encodeTask(value task.Task) taskWire {
	return taskWire{
		ID:             corewire.EncodeTaskID(value.ID),
		Owner:          encodeOwner(value.Owner),
		Title:          encodeTitle(value.Title),
		Description:    encodeDescription(value.Description),
		Type:           encodeTaskType(value.Type),
		Reference:      encodeReferenceURL(value.Reference),
		Reward:         encodeRewardSpec(value.Reward),
		Participation:  encodeParticipationPolicy(value.Participation),
		AssigneeScope:  encodeAssigneeScope(value.AssigneeScope),
		ReservationTTL: encodeReservationTTL(value.ReservationTTL),
		State:          encodeState(value.State),
		Visibility:     encodeVisibility(value.Visibility),
		Placement:      encodeSeriesPlacement(value.Placement),
		ResponseSchema: encodeResponseSchema(value.ResponseSchema),
		Payload:        encodeDataPayload(value.Payload),
		Attachments:    attachmentwire.EncodeSlice(value.Attachments),
		CreatedBy:      corewire.EncodeUserID(value.CreatedBy),
	}
}

func decodeTask(wire taskWire) (task.Task, error) {
	id, err := corewire.DecodeTaskID(wire.ID)
	if err != nil {
		return task.Task{}, err
	}
	owner, err := decodeOwner(wire.Owner)
	if err != nil {
		return task.Task{}, err
	}
	title, err := decodeTitle(wire.Title)
	if err != nil {
		return task.Task{}, err
	}
	description, err := decodeDescription(wire.Description)
	if err != nil {
		return task.Task{}, err
	}
	taskType, err := decodeTaskType(wire.Type)
	if err != nil {
		return task.Task{}, err
	}
	reference, err := decodeReferenceURL(wire.Reference)
	if err != nil {
		return task.Task{}, err
	}
	reward, err := decodeRewardSpec(wire.Reward)
	if err != nil {
		return task.Task{}, err
	}
	participation, err := decodeParticipationPolicy(wire.Participation)
	if err != nil {
		return task.Task{}, err
	}
	assigneeScope, err := decodeAssigneeScope(wire.AssigneeScope)
	if err != nil {
		return task.Task{}, err
	}
	reservationTTL, err := decodeReservationTTL(wire.ReservationTTL)
	if err != nil {
		return task.Task{}, err
	}
	state, err := decodeState(wire.State)
	if err != nil {
		return task.Task{}, err
	}
	visibility, err := decodeVisibility(wire.Visibility)
	if err != nil {
		return task.Task{}, err
	}
	placement, err := decodeSeriesPlacement(wire.Placement)
	if err != nil {
		return task.Task{}, err
	}
	responseSchema, err := decodeResponseSchema(wire.ResponseSchema)
	if err != nil {
		return task.Task{}, err
	}
	payload, err := decodeDataPayload(wire.Payload)
	if err != nil {
		return task.Task{}, err
	}
	attachments, err := attachmentwire.DecodeSlice(wire.Attachments)
	if err != nil {
		return task.Task{}, err
	}
	createdBy, err := corewire.DecodeUserID(wire.CreatedBy)
	if err != nil {
		return task.Task{}, err
	}
	return task.Task{
		ID:             id,
		Owner:          owner,
		Title:          title,
		Description:    description,
		Type:           taskType,
		Reference:      reference,
		Reward:         reward,
		Participation:  participation,
		AssigneeScope:  assigneeScope,
		ReservationTTL: reservationTTL,
		State:          state,
		Visibility:     visibility,
		Placement:      placement,
		ResponseSchema: responseSchema,
		Payload:        payload,
		Attachments:    attachments,
		CreatedBy:      createdBy,
	}, nil
}

func encodeTasks(values []task.Task) []taskWire {
	encoded := make([]taskWire, 0, len(values))
	for index := range values {
		encoded = append(encoded, encodeTask(values[index]))
	}
	return encoded
}

func decodeTasks(wires []taskWire) ([]task.Task, error) {
	values := make([]task.Task, 0, len(wires))
	for index := range wires {
		value, err := decodeTask(wires[index])
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// ---- task.ListItem ----

type listItemWire struct {
	Task           taskWire           `json:"task"`
	ActiveAssignee activeAssigneeWire `json:"active_assignee"`
}

func encodeListItem(item task.ListItem) listItemWire {
	return listItemWire{Task: encodeTask(item.Task), ActiveAssignee: encodeActiveAssignee(item.ActiveAssignee)}
}

func decodeListItem(wire listItemWire) (task.ListItem, error) {
	value, err := decodeTask(wire.Task)
	if err != nil {
		return task.ListItem{}, err
	}
	assignee, err := decodeActiveAssignee(wire.ActiveAssignee)
	if err != nil {
		return task.ListItem{}, err
	}
	return task.ListItem{Task: value, ActiveAssignee: assignee}, nil
}

func encodeListItems(values []task.ListItem) []listItemWire {
	encoded := make([]listItemWire, 0, len(values))
	for index := range values {
		encoded = append(encoded, encodeListItem(values[index]))
	}
	return encoded
}

func decodeListItems(wires []listItemWire) ([]task.ListItem, error) {
	values := make([]task.ListItem, 0, len(wires))
	for index := range wires {
		value, err := decodeListItem(wires[index])
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// ---- task.Series ----

type seriesWire struct {
	ID          string    `json:"id"`
	Owner       ownerWire `json:"owner"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	State       string    `json:"state"`
	CreatedBy   string    `json:"created_by"`
}

func encodeSeries(series task.Series) seriesWire {
	return seriesWire{
		ID:          corewire.EncodeTaskSeriesID(series.ID),
		Owner:       encodeOwner(series.Owner),
		Title:       encodeSeriesTitle(series.Title),
		Description: encodeSeriesDescription(series.Description),
		State:       encodeSeriesState(series.State),
		CreatedBy:   corewire.EncodeUserID(series.CreatedBy),
	}
}

func decodeSeries(wire seriesWire) (task.Series, error) {
	id, err := corewire.DecodeTaskSeriesID(wire.ID)
	if err != nil {
		return task.Series{}, err
	}
	owner, err := decodeOwner(wire.Owner)
	if err != nil {
		return task.Series{}, err
	}
	title, err := decodeSeriesTitle(wire.Title)
	if err != nil {
		return task.Series{}, err
	}
	description, err := decodeSeriesDescription(wire.Description)
	if err != nil {
		return task.Series{}, err
	}
	state, err := decodeSeriesState(wire.State)
	if err != nil {
		return task.Series{}, err
	}
	createdBy, err := corewire.DecodeUserID(wire.CreatedBy)
	if err != nil {
		return task.Series{}, err
	}
	return task.Series{ID: id, Owner: owner, Title: title, Description: description, State: state, CreatedBy: createdBy}, nil
}

func encodeSeriesList(values []task.Series) []seriesWire {
	encoded := make([]seriesWire, 0, len(values))
	for index := range values {
		encoded = append(encoded, encodeSeries(values[index]))
	}
	return encoded
}

func decodeSeriesList(wires []seriesWire) ([]task.Series, error) {
	values := make([]task.Series, 0, len(wires))
	for index := range wires {
		value, err := decodeSeries(wires[index])
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// ---- task.SeriesDetail ----

type seriesDetailWire struct {
	Series seriesWire `json:"series"`
	Tasks  []taskWire `json:"tasks,omitempty"`
}

func encodeSeriesDetail(detail task.SeriesDetail) seriesDetailWire {
	return seriesDetailWire{Series: encodeSeries(detail.Series), Tasks: encodeTasks(detail.Tasks)}
}

func decodeSeriesDetail(wire *seriesDetailWire) (task.SeriesDetail, error) {
	if wire == nil {
		return task.SeriesDetail{}, fmt.Errorf("result is missing its series detail")
	}
	series, err := decodeSeries(wire.Series)
	if err != nil {
		return task.SeriesDetail{}, err
	}
	tasks, err := decodeTasks(wire.Tasks)
	if err != nil {
		return task.SeriesDetail{}, err
	}
	return task.SeriesDetail{Series: series, Tasks: tasks}, nil
}

// ---- task.Reservation ----

type reservationWire struct {
	ID          string       `json:"id"`
	TaskID      string       `json:"task_id"`
	Assignee    assigneeWire `json:"assignee"`
	State       string       `json:"state"`
	RequestedBy string       `json:"requested_by"`
}

func encodeReservation(reservation task.Reservation) reservationWire {
	return reservationWire{
		ID:          corewire.EncodeTaskReservationID(reservation.ID),
		TaskID:      corewire.EncodeTaskID(reservation.TaskID),
		Assignee:    encodeAssignee(reservation.Assignee),
		State:       encodeReservationState(reservation.State),
		RequestedBy: corewire.EncodeUserID(reservation.RequestedBy),
	}
}

func decodeReservation(wire reservationWire) (task.Reservation, error) {
	id, err := corewire.DecodeTaskReservationID(wire.ID)
	if err != nil {
		return task.Reservation{}, err
	}
	taskID, err := corewire.DecodeTaskID(wire.TaskID)
	if err != nil {
		return task.Reservation{}, err
	}
	assignee, err := decodeAssignee(wire.Assignee)
	if err != nil {
		return task.Reservation{}, err
	}
	state, err := decodeReservationState(wire.State)
	if err != nil {
		return task.Reservation{}, err
	}
	requestedBy, err := corewire.DecodeUserID(wire.RequestedBy)
	if err != nil {
		return task.Reservation{}, err
	}
	return task.Reservation{ID: id, TaskID: taskID, Assignee: assignee, State: state, RequestedBy: requestedBy}, nil
}

func encodeReservations(values []task.Reservation) []reservationWire {
	encoded := make([]reservationWire, 0, len(values))
	for index := range values {
		encoded = append(encoded, encodeReservation(values[index]))
	}
	return encoded
}

func decodeReservations(wires []reservationWire) ([]task.Reservation, error) {
	values := make([]task.Reservation, 0, len(wires))
	for index := range wires {
		value, err := decodeReservation(wires[index])
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// ---- comments ----

type seriesCommentWire struct {
	ID        string `json:"id"`
	SeriesID  string `json:"series_id"`
	AuthorID  string `json:"author_id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

func encodeSeriesComment(comment task.SeriesComment) seriesCommentWire {
	return seriesCommentWire{
		ID:        corewire.EncodeSeriesCommentID(comment.ID),
		SeriesID:  corewire.EncodeTaskSeriesID(comment.SeriesID),
		AuthorID:  corewire.EncodeUserID(comment.AuthorID),
		Body:      encodeCommentBody(comment.Body),
		CreatedAt: corewire.EncodeTime(comment.CreatedAt),
	}
}

func decodeSeriesComment(wire seriesCommentWire) (task.SeriesComment, error) {
	id, err := corewire.DecodeSeriesCommentID(wire.ID)
	if err != nil {
		return task.SeriesComment{}, err
	}
	seriesID, err := corewire.DecodeTaskSeriesID(wire.SeriesID)
	if err != nil {
		return task.SeriesComment{}, err
	}
	authorID, err := corewire.DecodeUserID(wire.AuthorID)
	if err != nil {
		return task.SeriesComment{}, err
	}
	body, err := decodeCommentBody(wire.Body)
	if err != nil {
		return task.SeriesComment{}, err
	}
	createdAt, err := corewire.DecodeTime(wire.CreatedAt)
	if err != nil {
		return task.SeriesComment{}, err
	}
	return task.SeriesComment{ID: id, SeriesID: seriesID, AuthorID: authorID, Body: body, CreatedAt: createdAt}, nil
}

func encodeSeriesComments(values []task.SeriesComment) []seriesCommentWire {
	encoded := make([]seriesCommentWire, 0, len(values))
	for index := range values {
		encoded = append(encoded, encodeSeriesComment(values[index]))
	}
	return encoded
}

func decodeSeriesComments(wires []seriesCommentWire) ([]task.SeriesComment, error) {
	values := make([]task.SeriesComment, 0, len(wires))
	for index := range wires {
		value, err := decodeSeriesComment(wires[index])
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

type taskCommentWire struct {
	ID        string `json:"id"`
	TaskID    string `json:"task_id"`
	AuthorID  string `json:"author_id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

func encodeTaskComment(comment task.TaskComment) taskCommentWire {
	return taskCommentWire{
		ID:        corewire.EncodeTaskCommentID(comment.ID),
		TaskID:    corewire.EncodeTaskID(comment.TaskID),
		AuthorID:  corewire.EncodeUserID(comment.AuthorID),
		Body:      encodeCommentBody(comment.Body),
		CreatedAt: corewire.EncodeTime(comment.CreatedAt),
	}
}

func decodeTaskComment(wire taskCommentWire) (task.TaskComment, error) {
	id, err := corewire.DecodeTaskCommentID(wire.ID)
	if err != nil {
		return task.TaskComment{}, err
	}
	taskID, err := corewire.DecodeTaskID(wire.TaskID)
	if err != nil {
		return task.TaskComment{}, err
	}
	authorID, err := corewire.DecodeUserID(wire.AuthorID)
	if err != nil {
		return task.TaskComment{}, err
	}
	body, err := decodeCommentBody(wire.Body)
	if err != nil {
		return task.TaskComment{}, err
	}
	createdAt, err := corewire.DecodeTime(wire.CreatedAt)
	if err != nil {
		return task.TaskComment{}, err
	}
	return task.TaskComment{ID: id, TaskID: taskID, AuthorID: authorID, Body: body, CreatedAt: createdAt}, nil
}

func encodeTaskComments(values []task.TaskComment) []taskCommentWire {
	encoded := make([]taskCommentWire, 0, len(values))
	for index := range values {
		encoded = append(encoded, encodeTaskComment(values[index]))
	}
	return encoded
}

func decodeTaskComments(wires []taskCommentWire) ([]task.TaskComment, error) {
	values := make([]task.TaskComment, 0, len(wires))
	for index := range wires {
		value, err := decodeTaskComment(wires[index])
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
