//go:build functional
// +build functional

package functional

import (
	"bytes"
	"encoding/json"
	"github.com/66gu1/easygodocs/internal/app/department"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

var baseURL = getBaseURL()

func getBaseURL() string {
	if url := os.Getenv("BASE_URL"); url != "" {
		return url
	}
	return "http://localhost:8080"
}

func TestDepartmentWorkflow_CreateUpdateDeleteGetTree(t *testing.T) {
	client := &http.Client{Timeout: 2 * time.Second}

	// Setup
	dept1 := department.CreateDepartmentReq{Name: "1"}
	dept2 := department.CreateDepartmentReq{Name: "2"}
	dept3 := department.CreateDepartmentReq{Name: "3"}
	dept4 := department.CreateDepartmentReq{Name: "4"}
	dept5 := department.CreateDepartmentReq{Name: "5"}
	dept6 := department.CreateDepartmentReq{Name: "6"}

	// Create a hierarchy of departments
	id1 := createDepartment(t, client, dept1)
	dept2.ParentID = &id1
	dept3.ParentID = &id1
	id2 := createDepartment(t, client, dept2)
	updatedID := id2
	updatedParentID := dept2.ParentID
	dept4.ParentID = &id2
	id3 := createDepartment(t, client, dept3)
	id4 := createDepartment(t, client, dept4)
	id5 := createDepartment(t, client, dept5)
	dept6.ParentID = &id5
	deletedID := id5
	_ = createDepartment(t, client, dept6)

	wantTree := department.Tree{
		{ID: id1, Name: "1", Children: []*department.Node{
			{ID: id2, Name: "21", ParentID: &id1, Children: []*department.Node{
				{ID: id4, Name: "4", ParentID: &id2, Children: []*department.Node{}},
			}},
			{ID: id3, Name: "3", ParentID: &id1, Children: []*department.Node{}},
		}},
	}
	wantBody, err := json.Marshal(wantTree)
	if err != nil {
		t.Fatalf("failed to marshal expected tree: %v", err)
	}

	// 2. Update dept2
	updatedDept2 := department.UpdateDepartmentReq{ID: updatedID, Name: "21", ParentID: updatedParentID}
	body, _ := json.Marshal(updatedDept2)
	req, _ := http.NewRequest(http.MethodPut, baseURL+"/departments/"+updatedID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	// 3. Delete dept3
	req, _ = http.NewRequest(http.MethodDelete, baseURL+"/departments/"+deletedID.String(), nil)
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	// 4. Get department tree
	resp, err = client.Get(baseURL + "/departments")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.JSONEq(t, string(wantBody), string(data))
}

func createDepartment(t *testing.T, client *http.Client, department department.CreateDepartmentReq) uuid.UUID {
	type createResp struct {
		ID string `json:"id"`
	}

	body, _ := json.Marshal(department)
	resp, err := client.Post(baseURL+"/departments", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var respBody createResp
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	require.NoError(t, err)

	id, err := uuid.Parse(respBody.ID)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, id)

	return id
}
