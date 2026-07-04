package mcp

import (
	"context"
	"encoding/json"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
)

type userDirectoryEntrySummary struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	Status string `json:"status"`
}

type userDirectoryPayload struct {
	Users []userDirectoryEntrySummary `json:"users"`
}

type userProfilePayload struct {
	ID    string        `json:"id"`
	Tasks []taskSummary `json:"tasks"`
}

func (userDirectoryPayload) payloadValue() {}

func (userProfilePayload) payloadValue() {}

func parseUserID(arguments json.RawMessage) (core.UserID, toolResult) {
	var args struct {
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return core.UserID{}, invalidArguments()
	}
	result := core.ParseUserID(args.UserID)
	userID, matched := result.(core.UserIDCreated)
	if !matched {
		return core.UserID{}, toolProtocolError{code: codeInvalidParams, message: result.(core.UserIDRejected).Reason.Description()}
	}
	return userID.Value, nil
}

func (server Server) callListUsers(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	result := server.services.ListUsers(ctx, args.Query, core.DefaultPage())
	listed, matched := result.(auth.UsersListed)
	if !matched {
		return toolFailed{message: result.(auth.UserDirectoryRejected).Reason.Description()}
	}
	entries := make([]userDirectoryEntrySummary, 0, len(listed.Values))
	for index := range listed.Values {
		entries = append(entries, userDirectoryEntrySummary{ID: listed.Values[index].ID.String(), Email: listed.Values[index].Email.String(), Status: listed.Values[index].Status})
	}
	return marshalPayload(userDirectoryPayload{Users: entries})
}

func (server Server) callGetUserProfile(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	userID, problem := parseUserID(arguments)
	if problem != nil {
		return problem
	}
	result := server.services.GetUserProfile(ctx, subject, userID, core.DefaultPage())
	listed, matched := result.(task.TasksListed)
	if !matched {
		return toolFailed{message: result.(task.ListRejected).Reason.Description()}
	}
	tasks := make([]taskSummary, 0, len(listed.Values))
	for index := range listed.Values {
		tasks = append(tasks, taskToSummary(listed.Values[index].Task))
	}
	return marshalPayload(userProfilePayload{ID: userID.String(), Tasks: tasks})
}

func (server Server) callGetUserWork(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	userID, problem := parseUserID(arguments)
	if problem != nil {
		return problem
	}
	result := server.services.GetUserWork(ctx, subject, userID, core.DefaultPage())
	listed, matched := result.(task.TasksListed)
	if !matched {
		return toolFailed{message: result.(task.ListRejected).Reason.Description()}
	}
	summaries := make([]taskSummary, 0, len(listed.Values))
	for index := range listed.Values {
		summaries = append(summaries, taskToSummary(listed.Values[index].Task))
	}
	return marshalPayload(tasksPayload{Tasks: summaries})
}

func (server Server) callGetUserSubmissions(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	userID, problem := parseUserID(arguments)
	if problem != nil {
		return problem
	}
	result := server.services.GetUserSubmissions(ctx, subject, userID, core.DefaultPage())
	listed, matched := result.(submission.SubmissionsListed)
	if !matched {
		return toolFailed{message: result.(submission.ListRejected).Reason.Description()}
	}
	summaries := make([]submissionSummary, 0, len(listed.Values))
	for index := range listed.Values {
		summaries = append(summaries, submissionToSummary(listed.Values[index]))
	}
	return marshalPayload(submissionsPayload{Submissions: summaries})
}
