//go:build http_e2e

package http_e2e_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTaskSeriesEndpoints(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "series-rest")
	created := createSeriesTask(t, server, owner)

	listResponse := getWithBearer(t, server.URL+"/api/task-series?scope=user", owner.AccessToken)
	defer listResponse.Body.Close()
	assertStatus(t, listResponse, http.StatusOK)
	var listBody struct {
		Series []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"series"`
	}
	if err := json.NewDecoder(listResponse.Body).Decode(&listBody); err != nil {
		t.Fatalf("decode series list: %v", err)
	}
	found := false
	for index := range listBody.Series {
		if listBody.Series[index].ID == created.SeriesID {
			found = true
		}
	}
	if !found {
		t.Fatalf("created series not in list")
	}

	detailResponse := getWithBearer(t, server.URL+"/api/task-series/"+created.SeriesID, owner.AccessToken)
	defer detailResponse.Body.Close()
	assertStatus(t, detailResponse, http.StatusOK)
	var detailBody struct {
		Series struct {
			ID string `json:"id"`
		} `json:"series"`
		Tasks []struct {
			ID string `json:"id"`
		} `json:"tasks"`
	}
	if err := json.NewDecoder(detailResponse.Body).Decode(&detailBody); err != nil {
		t.Fatalf("decode series detail: %v", err)
	}
	if detailBody.Series.ID != created.SeriesID {
		t.Fatalf("series detail id mismatch")
	}
	if len(detailBody.Tasks) != 1 {
		t.Fatalf("series task count = %d, want 1", len(detailBody.Tasks))
	}
}

func TestMCPSeriesTools(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "mcp-series")
	created := createSeriesTask(t, server, owner)
	credential := createAgentCredential(t, server, owner.AccessToken, []string{"tasks_read"})
	sessionID := initializeMCPSession(t, server, credential)

	listSeries := toolText(t, decodeRPC(t, mcpCall(t, server, credential, sessionID, `1`, "sharecrop.list_task_series", `{}`)))
	if !strings.Contains(listSeries, created.SeriesID) {
		t.Fatalf("list_task_series missing series id: %s", listSeries)
	}
	getSeries := toolText(t, decodeRPC(t, mcpCall(t, server, credential, sessionID, `2`, "sharecrop.get_task_series", `{"series_id":"`+created.SeriesID+`"}`)))
	if !strings.Contains(getSeries, created.ID) {
		t.Fatalf("get_task_series missing task id: %s", getSeries)
	}

}

func createSeriesTask(t *testing.T, server *httptest.Server, owner authHTTPResponse) struct {
	ID       string `json:"id"`
	SeriesID string `json:"series_id"`
} {
	t.Helper()
	body := strings.Replace(`{"owner":{"kind":"user","user_id":"<USER_ID>","team_id":"","organization_id":""},"title":"Series task","description":"A task placed in a new series.","reward":{"kind":"none","credit_amount":0},"visibility":{"kind":"default","user_id":"","team_id":"","organization_id":""},"placement":{"kind":"new_series","series_id":"","series_title":"Browser series","series_position":1},"response_schema_json":"{\"kind\":\"freeform\"}","payload":{"kind":"none","json":""}}`, "<USER_ID>", owner.SubjectID, 1)
	response := postJSONWithBearer(t, server.URL+"/api/tasks", []byte(body), owner.AccessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusCreated)
	var decoded struct {
		ID       string `json:"id"`
		SeriesID string `json:"series_id"`
	}
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode series task: %v", err)
	}
	if decoded.SeriesID == "" {
		t.Fatalf("series id is empty")
	}
	return decoded
}
