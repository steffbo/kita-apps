package handler_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/testutil"
)

// Group Handler Tests

func TestGroupHandler_List_Success(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin@example.com").
		AsAdmin().
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	_, err = testutil.NewGroupBuilder().WithName("Group A").Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)
	_, err = testutil.NewGroupBuilder().WithName("Group B").Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "GET", "/api/groups", nil, admin)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusOK)

	type GroupResponse struct {
		ID    int64  `json:"id"`
		Name  string `json:"name"`
		Color string `json:"color"`
	}

	var response []GroupResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Len(t, response, 2)
}

func TestGroupHandler_Create_Success(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin@example.com").
		AsAdmin().
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "POST", "/api/groups", map[string]interface{}{
		"name":        "New Group",
		"description": "Test group description",
		"color":       "#FF5733",
	}, admin)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusCreated)
}

func TestGroupHandler_Create_Forbidden_NonAdmin(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	employee, err := testutil.NewEmployeeBuilder().
		WithEmail("employee@example.com").
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "POST", "/api/groups", map[string]interface{}{
		"name": "New Group",
	}, employee)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusForbidden)
}

func TestGroupHandler_Get_Success(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin@example.com").
		AsAdmin().
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	group, err := testutil.NewGroupBuilder().
		WithName("Test Group").
		WithDescription("A description").
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "GET", "/api/groups/"+strconv.FormatInt(group.ID, 10), nil, admin)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusOK)
}

func TestGroupHandler_Update_Success(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin@example.com").
		AsAdmin().
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	group, err := testutil.NewGroupBuilder().
		WithName("Original Name").
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "PUT", "/api/groups/"+strconv.FormatInt(group.ID, 10), map[string]interface{}{
		"name":  "Updated Name",
		"color": "#00FF00",
	}, admin)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusOK)
}

func TestGroupHandler_Delete_Success(t *testing.T) {
	server := setupHandlerTest(t)
	defer server.Close()

	admin, err := testutil.NewEmployeeBuilder().
		WithEmail("admin@example.com").
		AsAdmin().
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	group, err := testutil.NewGroupBuilder().
		WithName("To Delete").
		Create(suite.Ctx, suite.Container.DB)
	require.NoError(t, err)

	req := server.AuthenticatedRequest(t, "DELETE", "/api/groups/"+strconv.FormatInt(group.ID, 10), nil, admin)

	resp, err := server.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	testutil.AssertStatus(t, resp, http.StatusNoContent)
}
