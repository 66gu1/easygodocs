package http_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/66gu1/easygodocs/internal/app/entity"
	entity_http "github.com/66gu1/easygodocs/internal/app/entity/transport/http"
	"github.com/66gu1/easygodocs/internal/app/entity/transport/http/mocks"
	entity_usecase "github.com/66gu1/easygodocs/internal/app/entity/usecase"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

//go:generate minimock -o ./mocks -s _mock.go

func TestHandler_GetTree(t *testing.T) {
	t.Parallel()

	tree := entity.Tree{
		&entity.Node{
			ListItem: entity.ListItem{
				ID:   uuid.New(),
				Name: "Root",
			},
			Children: []*entity.Node{
				{
					ListItem: entity.ListItem{
						ID:       uuid.New(),
						Name:     "A1",
						ParentID: &[]uuid.UUID{uuid.New()}[0],
					},
					Children: []*entity.Node{
						{
							ListItem: entity.ListItem{
								ID:       uuid.New(),
								Name:     "B1",
								ParentID: &[]uuid.UUID{uuid.New()}[0],
							},
						},
						{
							ListItem: entity.ListItem{
								ID:       uuid.New(),
								Name:     "B2",
								ParentID: &[]uuid.UUID{uuid.New()}[0],
							},
						},
					},
				},
				{
					ListItem: entity.ListItem{
						ID:       uuid.New(),
						Name:     "A2",
						ParentID: &[]uuid.UUID{uuid.New()}[0],
					},
					Children: []*entity.Node{
						{
							ListItem: entity.ListItem{
								ID:       uuid.New(),
								Name:     "C1",
								ParentID: &[]uuid.UUID{uuid.New()}[0],
							},
						},
					},
				},
			},
		},
	}
	tests := []struct {
		name       string
		wantStatus int
		setup      func(s *mocks.ServiceMock)
		wantLen    int
	}{
		{
			name:       "service error -> 500",
			wantStatus: http.StatusInternalServerError,
			setup: func(s *mocks.ServiceMock) {
				s.GetTreeMock.Expect(minimock.AnyContext).Return(nil, fmt.Errorf("service error"))
			},
		},
		{
			name:       "ok -> 200 with tree JSON",
			wantStatus: http.StatusOK,
			setup: func(s *mocks.ServiceMock) {
				s.GetTreeMock.Expect(minimock.AnyContext).Return(tree, nil)
			},
			wantLen: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mock := mocks.NewServiceMock(t)
			if tc.setup != nil {
				tc.setup(mock)
			}
			h := entity_http.NewHandler(mock)
			r := chi.NewRouter()

			r.Get("/tree", h.GetTree)

			req := httptest.NewRequest(http.MethodGet, "/tree", nil)
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			require.Equal(t, tc.wantStatus, rr.Code)
			if tc.wantStatus == http.StatusOK {
				if ct := rr.Header().Get("Content-Type"); ct == "" || ct[:16] != "application/json" {
					t.Fatalf("content-type = %q; want application/json", ct)
				}
				var got entity.Tree
				err := json.Unmarshal(rr.Body.Bytes(), &got)
				require.NoError(t, err)
				require.Equal(t, tree, got)
			} else if rr.Body.Len() == 0 {
				t.Fatalf("error response body is empty; want some payload")
			}
		})
	}
}

func TestHandler_Get(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	ent := entity.Entity{
		ID:      id,
		Type:    "type",
		Name:    "Doc 1",
		Content: "Content 1",
	}
	tests := []struct {
		name       string
		entityID   string
		wantStatus int
		setup      func(s *mocks.ServiceMock)
	}{
		{
			name:       "invalid UUID -> 400",
			entityID:   "invalid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "handler error -> 500",
			entityID:   id.String(),
			wantStatus: http.StatusInternalServerError,
			setup: func(s *mocks.ServiceMock) {
				s.GetMock.Expect(minimock.AnyContext, id).Return(entity.Entity{}, fmt.Errorf("handler error"))
			},
		},
		{
			name:       "ok -> 200 with entity JSON",
			entityID:   id.String(),
			wantStatus: http.StatusOK,
			setup: func(s *mocks.ServiceMock) {
				s.GetMock.Expect(minimock.AnyContext, id).Return(ent, nil)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mock := mocks.NewServiceMock(t)
			if tc.setup != nil {
				tc.setup(mock)
			}
			h := entity_http.NewHandler(mock)
			r := chi.NewRouter()

			r.Get("/entity/{"+entity_http.URLParamEntityID+"}", h.Get)

			req := httptest.NewRequest(http.MethodGet, "/entity/"+tc.entityID, nil)
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			require.Equal(t, tc.wantStatus, rr.Code)
			if tc.wantStatus == http.StatusOK {
				if ct := rr.Header().Get("Content-Type"); ct == "" || ct[:16] != "application/json" {
					t.Fatalf("content-type = %q; want application/json", ct)
				}
				var got entity.Entity
				err := json.Unmarshal(rr.Body.Bytes(), &got)
				require.NoError(t, err)
				require.Equal(t, ent, got)
			} else if rr.Body.Len() == 0 {
				t.Fatalf("error response body is empty; want some payload")
			}
		})
	}
}

func TestHandler_GetVersion(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	version := 2
	ent := entity.Entity{
		ID:      id,
		Type:    "type",
		Name:    "Doc 1",
		Content: "Content 1 v2",
	}
	tests := []struct {
		name       string
		entityID   string
		version    string
		wantStatus int
		setup      func(s *mocks.ServiceMock)
	}{
		{
			name:       "invalid UUID -> 400",
			entityID:   "invalid",
			version:    "1",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid version -> 400",
			entityID:   id.String(),
			version:    "invalid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "handler error -> 500",
			entityID:   id.String(),
			version:    fmt.Sprintf("%d", version),
			wantStatus: http.StatusInternalServerError,
			setup: func(s *mocks.ServiceMock) {
				s.GetVersionMock.Expect(minimock.AnyContext, id, version).Return(entity.Entity{}, fmt.Errorf("handler error"))
			},
		},
		{
			name:       "ok -> 200 with entity JSON",
			entityID:   id.String(),
			version:    fmt.Sprintf("%d", version),
			wantStatus: http.StatusOK,
			setup: func(s *mocks.ServiceMock) {
				s.GetVersionMock.Expect(minimock.AnyContext, id, version).Return(ent, nil)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mock := mocks.NewServiceMock(t)
			if tc.setup != nil {
				tc.setup(mock)
			}
			h := entity_http.NewHandler(mock)
			r := chi.NewRouter()

			r.Get("/entity/{"+entity_http.URLParamEntityID+"}/version/{"+entity_http.URLParamVersion+"}", h.GetVersion)

			req := httptest.NewRequest(http.MethodGet, "/entity/"+tc.entityID+"/version/"+tc.version, nil)
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			require.Equal(t, tc.wantStatus, rr.Code)
			if tc.wantStatus == http.StatusOK {
				if ct := rr.Header().Get("Content-Type"); ct == "" || ct[:16] != "application/json" {
					t.Fatalf("content-type = %q; want application/json", ct)
				}
				var got entity.Entity
				err := json.Unmarshal(rr.Body.Bytes(), &got)
				require.NoError(t, err)
				require.Equal(t, ent, got)
			} else if rr.Body.Len() == 0 {
				t.Fatalf("error response body is empty; want some payload")
			}
		})
	}
}

func TestHandler_GetVersionsList(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	versions := []entity.Entity{
		{
			ID:      id,
			Type:    "type",
			Name:    "Doc 1",
			Content: "Content 1 v1",
		},
		{
			ID:      id,
			Type:    "type",
			Name:    "Doc 1",
			Content: "Content 1 v2",
		},
	}
	tests := []struct {
		name       string
		entityID   string
		wantStatus int
		setup      func(s *mocks.ServiceMock)
	}{
		{
			name:       "invalid UUID -> 400",
			entityID:   "invalid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "handler error -> 500",
			entityID:   id.String(),
			wantStatus: http.StatusInternalServerError,
			setup: func(s *mocks.ServiceMock) {
				s.GetVersionsListMock.Expect(minimock.AnyContext, id).Return(nil, fmt.Errorf("handler error"))
			},
		},
		{
			name:       "ok -> 200 with versions list JSON",
			entityID:   id.String(),
			wantStatus: http.StatusOK,
			setup: func(s *mocks.ServiceMock) {
				s.GetVersionsListMock.Expect(minimock.AnyContext, id).Return(versions, nil)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mock := mocks.NewServiceMock(t)
			if tc.setup != nil {
				tc.setup(mock)
			}
			h := entity_http.NewHandler(mock)
			r := chi.NewRouter()

			r.Get("/entity/{"+entity_http.URLParamEntityID+"}/versions", h.GetVersionsList)
			req := httptest.NewRequest(http.MethodGet, "/entity/"+tc.entityID+"/versions", nil)
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			require.Equal(t, tc.wantStatus, rr.Code)
			if tc.wantStatus == http.StatusOK {
				if ct := rr.Header().Get("Content-Type"); ct == "" || ct[:16] != "application/json" {
					t.Fatalf("content-type = %q; want application/json", ct)
				}
				var got []entity.Entity
				err := json.Unmarshal(rr.Body.Bytes(), &got)
				require.NoError(t, err)
				require.Equal(t, versions, got)
			} else if rr.Body.Len() == 0 {
				t.Fatalf("error response body is empty; want some payload")
			}
		})
	}
}

func TestHandler_Create(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	parentID := uuid.New()
	ent := entity_usecase.CreateEntityCmd{
		Type:     "type",
		Name:     "Doc 1",
		Content:  "Content 1",
		ParentID: &parentID,
		IsDraft:  true,
	}
	want := entity_http.CreateEntityResp{ID: id}
	body, err := json.Marshal(ent)
	require.NoError(t, err)
	tests := []struct {
		name       string
		body       []byte
		wantStatus int
		setup      func(s *mocks.ServiceMock)
	}{
		{
			name:       "invalid JSON -> 400",
			body:       []byte("invalid"),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "handler error -> 500",
			body:       body,
			wantStatus: http.StatusInternalServerError,
			setup: func(s *mocks.ServiceMock) {
				s.CreateMock.Expect(minimock.AnyContext, ent).Return(uuid.Nil, fmt.Errorf("handler error"))
			},
		},
		{
			name:       "ok -> 201 with Location header and entity ID JSON",
			body:       body,
			wantStatus: http.StatusCreated,
			setup: func(s *mocks.ServiceMock) {
				s.CreateMock.Expect(minimock.AnyContext, ent).Return(id, nil)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mock := mocks.NewServiceMock(t)
			if tc.setup != nil {
				tc.setup(mock)
			}
			h := entity_http.NewHandler(mock)
			r := chi.NewRouter()

			r.Post("/entity", h.Create)

			req := httptest.NewRequest(http.MethodPost, "/entity", bytes.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			require.Equal(t, tc.wantStatus, rr.Code)
			if tc.wantStatus == http.StatusCreated {
				if loc := rr.Header().Get("Location"); loc != "/entities/"+id.String() {
					t.Fatalf("Location header = %q; want /entities/%s", loc, id.String())
				}
				if ct := rr.Header().Get("Content-Type"); ct == "" || ct[:16] != "application/json" {
					t.Fatalf("content-type = %q; want application/json", ct)
				}
				var got entity_http.CreateEntityResp
				err = json.Unmarshal(rr.Body.Bytes(), &got)
				require.NoError(t, err)
				require.Equal(t, want, got)
			} else if rr.Body.Len() == 0 {
				t.Fatalf("error response body is empty; want some payload")
			}
		})
	}
}

func TestHandler_Update(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	input := entity_http.UpdateEntityInput{
		Name:    "Doc 1 Updated",
		Content: "Content 1 Updated",
	}
	cmd := entity_usecase.UpdateEntityCmd{
		ID:      id,
		Name:    input.Name,
		Content: input.Content,
	}
	body, err := json.Marshal(input)
	require.NoError(t, err)
	tests := []struct {
		name       string
		entityID   string
		body       []byte
		wantStatus int
		setup      func(s *mocks.ServiceMock)
	}{
		{
			name:       "invalid UUID -> 400",
			entityID:   "invalid",
			body:       body,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid JSON -> 400",
			entityID:   id.String(),
			body:       []byte("invalid"),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "handler error -> 500",
			entityID:   id.String(),
			body:       body,
			wantStatus: http.StatusInternalServerError,
			setup: func(s *mocks.ServiceMock) {
				s.UpdateMock.Expect(minimock.AnyContext, cmd).Return(fmt.Errorf("handler error"))
			},
		},
		{
			name:       "ok -> 204 No Content",
			entityID:   id.String(),
			body:       body,
			wantStatus: http.StatusNoContent,
			setup: func(s *mocks.ServiceMock) {
				s.UpdateMock.Expect(minimock.AnyContext, cmd).Return(nil)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mock := mocks.NewServiceMock(t)
			if tc.setup != nil {
				tc.setup(mock)
			}
			h := entity_http.NewHandler(mock)
			r := chi.NewRouter()

			r.Put("/entity/{"+entity_http.URLParamEntityID+"}", h.Update)

			req := httptest.NewRequest(http.MethodPut, "/entity/"+tc.entityID, bytes.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			require.Equal(t, tc.wantStatus, rr.Code)
			if tc.wantStatus != http.StatusNoContent {
				if rr.Body.Len() == 0 {
					t.Fatalf("error response body is empty; want some payload")
				}
			}
		})
	}
}

func TestHandler_Delete(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	tests := []struct {
		name       string
		entityID   string
		wantStatus int
		setup      func(s *mocks.ServiceMock)
	}{
		{
			name:       "invalid UUID -> 400",
			entityID:   "invalid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "handler error -> 500",
			entityID:   id.String(),
			wantStatus: http.StatusInternalServerError,
			setup: func(s *mocks.ServiceMock) {
				s.DeleteMock.Expect(minimock.AnyContext, id).Return(fmt.Errorf("handler error"))
			},
		},
		{
			name:       "ok -> 204 No Content",
			entityID:   id.String(),
			wantStatus: http.StatusNoContent,
			setup: func(s *mocks.ServiceMock) {
				s.DeleteMock.Expect(minimock.AnyContext, id).Return(nil)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mock := mocks.NewServiceMock(t)
			if tc.setup != nil {
				tc.setup(mock)
			}
			h := entity_http.NewHandler(mock)
			r := chi.NewRouter()

			r.Delete("/entity/{"+entity_http.URLParamEntityID+"}", h.Delete)

			req := httptest.NewRequest(http.MethodDelete, "/entity/"+tc.entityID, nil)

			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			require.Equal(t, tc.wantStatus, rr.Code)
			if tc.wantStatus != http.StatusNoContent {
				if rr.Body.Len() == 0 {
					t.Fatalf("error response body is empty; want some payload")
				}
			}
		})
	}
}
