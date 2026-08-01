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
	Users      []userDirectoryEntrySummary `json:"users"`
	NextOffset int                         `json:"next_offset"`
}

type userProfilePayload struct {
	ID         string        `json:"id"`
	Tasks      []taskListRow `json:"tasks"`
	NextOffset int           `json:"next_offset"`
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
		return core.UserID{}, invalidIDArgument("user_id")
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
	page, pageProblem := parseMCPPageArguments(arguments)
	if pageProblem != nil {
		return pageProblem
	}
	result := server.services.ListUsers(ctx, args.Query, page.Probe())
	listed, matched := result.(auth.UsersListed)
	if !matched {
		return toolFailed{code: result.(auth.UserDirectoryRejected).Reason.Code(), message: result.(auth.UserDirectoryRejected).Reason.Description()}
	}
	visible, nextOffset := core.ProbeListWindow(len(listed.Values), page)
	entries := make([]userDirectoryEntrySummary, 0, visible)
	for index := range listed.Values[:visible] {
		entries = append(entries, userDirectoryEntrySummary{ID: listed.Values[index].ID.String(), Email: listed.Values[index].Email.String(), Status: listed.Values[index].Status})
	}
	return marshalPayload(userDirectoryPayload{Users: entries, NextOffset: nextOffset})
}

func (server Server) callGetUserProfile(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	userID, problem := parseUserID(arguments)
	if problem != nil {
		return problem
	}
	page, pageProblem := parseMCPPageArguments(arguments)
	if pageProblem != nil {
		return pageProblem
	}
	result := server.services.GetUserProfile(ctx, subject, userID, page.Probe())
	listed, matched := result.(task.TasksListed)
	if !matched {
		return toolFailed{code: result.(task.ListRejected).Reason.Code(), message: result.(task.ListRejected).Reason.Description()}
	}
	visible, nextOffset := core.ProbeListWindow(len(listed.Values), page)
	tasks := make([]taskListRow, 0, visible)
	for index := range listed.Values[:visible] {
		tasks = append(tasks, listItemToRow(listed.Values[index]))
	}
	return marshalPayload(userProfilePayload{ID: userID.String(), Tasks: tasks, NextOffset: nextOffset})
}

func (server Server) callGetUserWork(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	userID, problem := parseUserID(arguments)
	if problem != nil {
		return problem
	}
	page, pageProblem := parseMCPPageArguments(arguments)
	if pageProblem != nil {
		return pageProblem
	}
	result := server.services.GetUserWork(ctx, subject, userID, page.Probe())
	listed, matched := result.(task.TasksListed)
	if !matched {
		return toolFailed{code: result.(task.ListRejected).Reason.Code(), message: result.(task.ListRejected).Reason.Description()}
	}
	visible, nextOffset := core.ProbeListWindow(len(listed.Values), page)
	rows := make([]taskListRow, 0, visible)
	for index := range listed.Values[:visible] {
		rows = append(rows, listItemToRow(listed.Values[index]))
	}
	return marshalPayload(tasksPayload{Tasks: rows, NextOffset: nextOffset, Total: listed.Total})
}

func (server Server) callGetUserSubmissions(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	userID, problem := parseUserID(arguments)
	if problem != nil {
		return problem
	}
	page, pageProblem := parseMCPPageArguments(arguments)
	if pageProblem != nil {
		return pageProblem
	}
	result := server.services.GetUserSubmissions(ctx, subject, userID, page.Probe())
	listed, matched := result.(submission.SubmissionsListed)
	if !matched {
		return toolFailed{code: result.(submission.ListRejected).Reason.Code(), message: result.(submission.ListRejected).Reason.Description()}
	}
	visible, nextOffset := core.ProbeListWindow(len(listed.Values), page)
	summaries := make([]submissionSummary, 0, visible)
	for index := range listed.Values[:visible] {
		summaries = append(summaries, submissionToSummary(listed.Values[index]))
	}
	return marshalPayload(submissionsPayload{Submissions: summaries, NextOffset: nextOffset, Total: listed.Total})
}
