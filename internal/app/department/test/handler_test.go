package test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/66gu1/easygodocs/internal/app/department"
	"github.com/66gu1/easygodocs/internal/app/department/mock"
	"github.com/go-chi/chi/v5"
	"github.com/gojuno/minimock/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler_Create(t *testing.T) {
	t.Parallel()
	mc := minimock.NewController(t)

	id := uuid.New()
	parentID := uuid.New()
	ctx := context.WithValue(context.Background(), "test", true)
	wantedReq := department.CreateDepartmentReq{
		Name:     "HR",
		ParentID: &parentID,
	}
	expErr := errors.New("error")

	tests := []struct {
		name      string
		inputBody interface{}
		setup     func(svc *mock.ServiceMock)
		wantCode  int
	}{
		{
			name:      "valid request",
			inputBody: wantedReq,
			setup: func(svc *mock.ServiceMock) {
				svc.CreateMock.Times(1).Expect(ctx, wantedReq).Return(id, nil)
			},
			wantCode: http.StatusCreated,
		},
		{
			name:      "invalid JSON",
			inputBody: "invalid-json", // строка, не сериализуемая в структуру
			wantCode:  http.StatusInternalServerError,
		},
		{
			name:      "missing name",
			inputBody: department.CreateDepartmentReq{},
			wantCode:  http.StatusBadRequest,
		},
		{
			name:      "service error",
			inputBody: wantedReq,
			setup: func(svc *mock.ServiceMock) {
				svc.CreateMock.Times(1).Expect(ctx, wantedReq).Return(uuid.Nil, expErr)
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := mock.NewServiceMock(mc)
			if tc.setup != nil {
				tc.setup(svc)
			}

			h := department.NewHandler(svc)

			// prepare request
			var body bytes.Buffer
			switch v := tc.inputBody.(type) {
			case string:
				body.WriteString(v)
			default:
				_ = json.NewEncoder(&body).Encode(v)
			}

			req := httptest.NewRequest(http.MethodPost, "/departments", &body)
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()
			h.Create(rr, req)

			require.Equal(t, tc.wantCode, rr.Code)

			if tc.wantCode != http.StatusCreated {
				var errResp struct {
					Error string `json:"error"`
				}
				require.NoError(t, json.NewDecoder(rr.Body).Decode(&errResp))
				assert.NotEmpty(t, errResp.Error)

				return
			}

			var resp struct {
				ID string `json:"id"`
			}
			err := json.NewDecoder(rr.Body).Decode(&resp)
			require.NoError(t, err, "response body must be valid JSON")

			parsedID, err := uuid.Parse(resp.ID)
			require.NoError(t, err, "id must be a valid UUID")
			require.Equal(t, id, parsedID, "id must be equal")

		})
	}
}

func TestHandler_Update(t *testing.T) {
	t.Parallel()
	mc := minimock.NewController(t)

	id := uuid.New()
	parentID := uuid.New()
	ctx := context.WithValue(context.Background(), "test", true)
	wantedReq := department.UpdateDepartmentReq{
		ID:       id,
		Name:     "HR Updated",
		ParentID: &parentID,
	}
	expErr := errors.New("error")

	tests := []struct {
		name      string
		inputBody interface{}
		setup     func(svc *mock.ServiceMock)
		wantCode  int
	}{
		{
			name:      "valid request",
			inputBody: wantedReq,
			setup: func(svc *mock.ServiceMock) {
				svc.UpdateMock.Times(1).Expect(ctx, wantedReq).Return(nil)
			},
			wantCode: http.StatusNoContent,
		},
		{
			name:      "invalid JSON",
			inputBody: "invalid-json", // строка, не сериализуемая в структуру
			wantCode:  http.StatusInternalServerError,
		},
		{
			name:      "missing ID",
			inputBody: department.UpdateDepartmentReq{Name: "Finance Updated"},
			wantCode:  http.StatusBadRequest,
		},
		{
			name:      "empty name",
			inputBody: department.UpdateDepartmentReq{ID: id, Name: ""},
			wantCode:  http.StatusBadRequest,
		},
		{
			name:      "parent id is uuid.Nil",
			inputBody: department.UpdateDepartmentReq{ID: id, Name: "Finance", ParentID: &uuid.Nil},
			wantCode:  http.StatusBadRequest,
		},
		{
			name:      "parent id equals id",
			inputBody: department.UpdateDepartmentReq{ID: id, Name: "Finance", ParentID: &id},
			wantCode:  http.StatusBadRequest,
		},
		{
			name:      "service error",
			inputBody: wantedReq,
			setup: func(svc *mock.ServiceMock) {
				svc.UpdateMock.Times(1).Expect(ctx, wantedReq).Return(expErr)
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := mock.NewServiceMock(mc)
			if tc.setup != nil {
				tc.setup(svc)
			}

			h := department.NewHandler(svc)

			var body bytes.Buffer
			switch v := tc.inputBody.(type) {
			case string:
				body.WriteString(v)
			default:
				err := json.NewEncoder(&body).Encode(v)
				require.NoError(t, err, "input body must be valid JSON")
			}

			req := httptest.NewRequest(http.MethodPut, "/departments/"+id.String(), &body)
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()
			h.Update(rr, req)
			require.Equal(t, tc.wantCode, rr.Code)
			if tc.wantCode != http.StatusNoContent {
				var errResp struct {
					Error string `json:"error"`
				}
				require.NoError(t, json.NewDecoder(rr.Body).Decode(&errResp), "response body must be valid JSON")
				assert.NotEmpty(t, errResp.Error)
				return
			}
			assert.Empty(t, rr.Body.String(), "response body must be empty for 204 No Content")
		})
	}
}

func TestHandler_Delete(t *testing.T) {
	t.Parallel()
	mc := minimock.NewController(t)

	id := uuid.New()
	expErr := errors.New("error")

	tests := []struct {
		name     string
		setup    func(svc *mock.ServiceMock, ctx context.Context)
		wantCode int
		urlID    string
	}{
		{
			name: "valid request",
			setup: func(svc *mock.ServiceMock, ctx context.Context) {
				svc.DeleteMock.Times(1).Expect(ctx, id).Return(nil)
			},
			wantCode: http.StatusNoContent,
			urlID:    id.String(),
		},
		{
			name: "service error",
			setup: func(svc *mock.ServiceMock, ctx context.Context) {
				svc.DeleteMock.Times(1).Expect(ctx, id).Return(expErr)
			},
			wantCode: http.StatusInternalServerError,
			urlID:    id.String(),
		},
		{
			name:     "missing ID",
			wantCode: http.StatusBadRequest,
			urlID:    "invalid",
		},
		{
			name:     "ID = uuid.Nil",
			wantCode: http.StatusBadRequest,
			urlID:    uuid.Nil.String(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := mock.NewServiceMock(mc)

			rc := chi.NewRouteContext()
			rc.URLParams.Add("id", tc.urlID)
			ctx := context.WithValue(context.Background(), chi.RouteCtxKey, rc)
			if tc.setup != nil {
				tc.setup(svc, ctx)
			}

			h := department.NewHandler(svc)

			req := httptest.NewRequest(http.MethodDelete, "/departments/"+tc.urlID, nil)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			h.Delete(rr, req)

			require.Equal(t, tc.wantCode, rr.Code)
			if tc.wantCode != http.StatusNoContent {
				var errResp struct {
					Error string `json:"error"`
				}
				require.NoError(t, json.NewDecoder(rr.Body).Decode(&errResp), "response body must be valid JSON")
				assert.NotEmpty(t, errResp.Error)
				return
			}
			assert.Empty(t, rr.Body.String(), "response body must be empty for 204 No Content")
		})
	}
}

func TestHandler_GetDepartmentTree(t *testing.T) {
	t.Parallel()
	mc := minimock.NewController(t)

	tree := department.Tree{
		{
			ID:       uuid.New(),
			Name:     "A",
			ParentID: nil,
			Children: []*department.Node{
				{
					ID:       uuid.New(),
					Name:     "B",
					ParentID: nil,
					Children: nil,
				},
			},
		},
	}

	wantBody, err := json.Marshal(tree)
	if err != nil {
		t.Fatalf("failed to marshal expected tree: %v", err)
	}

	tests := []struct {
		name     string
		setup    func(svc *mock.ServiceMock)
		wantCode int
		wantBody string
	}{
		{
			name: "valid request",
			setup: func(svc *mock.ServiceMock) {
				svc.GetDepartmentTreeMock.Times(1).Return(tree, nil)
			},
			wantCode: http.StatusOK,
			wantBody: string(wantBody),
		},
		{
			name: "service error",
			setup: func(svc *mock.ServiceMock) {
				svc.GetDepartmentTreeMock.Times(1).Return(nil, errors.New("service error"))
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := mock.NewServiceMock(mc)
			if tc.setup != nil {
				tc.setup(svc)
			}

			h := department.NewHandler(svc)

			req := httptest.NewRequest(http.MethodGet, "/departments/tree", nil)
			ctx := context.WithValue(req.Context(), "test", true)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			h.GetDepartmentTree(rr, req)

			assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
			require.Equal(t, tc.wantCode, rr.Code)

			if tc.wantCode == http.StatusOK {
				assert.JSONEq(t, tc.wantBody, rr.Body.String(), "response body must match expected tree")
			} else {
				var errResp struct {
					Error string `json:"error"`
				}
				require.NoError(t, json.NewDecoder(rr.Body).Decode(&errResp), "response body must be valid JSON")
				assert.NotEmpty(t, errResp.Error)
			}
		})
	}
}
