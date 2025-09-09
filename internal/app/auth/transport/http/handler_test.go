package http_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/66gu1/easygodocs/internal/app/auth"
	auth_http "github.com/66gu1/easygodocs/internal/app/auth/transport/http"
	"github.com/66gu1/easygodocs/internal/app/auth/transport/http/mocks"
	"github.com/66gu1/easygodocs/internal/app/auth/usecase"
	user_http "github.com/66gu1/easygodocs/internal/app/user/transport/http"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

//go:generate minimock -o ./mocks -s _mock.go

func TestHandler_GetSessionsByUserID(t *testing.T) {
	t.Parallel()

	validID := uuid.New()
	sessions := []auth.Session{{}, {}}
	tests := []struct {
		name       string
		userIDStr  string
		wantStatus int
		setup      func(s *mocks.AuthServiceMock)
	}{
		{
			name:       "invalid UUID -> 400 and service not called",
			userIDStr:  "not-a-uuid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:      "service error -> 500",
			userIDStr: validID.String(),
			setup: func(s *mocks.AuthServiceMock) {
				s.GetSessionsByUserIDMock.Expect(minimock.AnyContext, validID).Return(nil, fmt.Errorf("service error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "ok -> 200 with sessions JSON",
			userIDStr:  validID.String(),
			wantStatus: http.StatusOK,
			setup: func(s *mocks.AuthServiceMock) {
				s.GetSessionsByUserIDMock.Expect(minimock.AnyContext, validID).Return(sessions, nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mock := mocks.NewAuthServiceMock(t)
			if tc.setup != nil {
				tc.setup(mock)
			}
			h := auth_http.NewHandler(mock)
			r := chi.NewRouter()

			r.Get("/users/{"+user_http.URLParamUserID+"}/sessions", h.GetSessionsByUserID)

			req := httptest.NewRequest(http.MethodGet, "/users/"+tc.userIDStr+"/sessions", nil)
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			require.Equal(t, tc.wantStatus, rr.Code)
			if tc.wantStatus == http.StatusOK {
				if ct := rr.Header().Get("Content-Type"); ct == "" || ct[:16] != "application/json" {
					t.Fatalf("content-type = %q; want application/json", ct)
				}
				var got []auth.Session
				err := json.Unmarshal(rr.Body.Bytes(), &got)
				require.NoError(t, err)
				require.Equal(t, sessions, got)
			} else if rr.Body.Len() == 0 {
				t.Fatalf("error response body is empty; want some payload")
			}
		})
	}
}

func TestHandler_DeleteSession(t *testing.T) {
	t.Parallel()

	var (
		sessionID = uuid.New()
		userID    = uuid.New()
	)

	tests := []struct {
		name         string
		userIDStr    string
		sessionIDStr string
		wantStatus   int
		setup        func(s *mocks.AuthServiceMock)
	}{
		{
			name:         "invalid user UUID -> 400 and service not called",
			userIDStr:    "not-a-uuid",
			sessionIDStr: sessionID.String(),
			wantStatus:   http.StatusBadRequest,
		},
		{
			name:         "invalid session UUID -> 400 and service not called",
			userIDStr:    userID.String(),
			sessionIDStr: "not-a-uuid",
			wantStatus:   http.StatusBadRequest,
		},
		{
			name:         "service error -> 500",
			userIDStr:    userID.String(),
			sessionIDStr: sessionID.String(),
			setup: func(s *mocks.AuthServiceMock) {
				s.DeleteSessionMock.Expect(minimock.AnyContext, userID, sessionID).Return(fmt.Errorf("service error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:         "ok -> 204 no content",
			userIDStr:    userID.String(),
			sessionIDStr: sessionID.String(),
			setup: func(s *mocks.AuthServiceMock) {
				s.DeleteSessionMock.Expect(minimock.AnyContext, userID, sessionID).Return(nil)
			},
			wantStatus: http.StatusNoContent,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mock := mocks.NewAuthServiceMock(t)
			if tc.setup != nil {
				tc.setup(mock)
			}
			h := auth_http.NewHandler(mock)
			r := chi.NewRouter()

			r.Delete("/users/{"+user_http.URLParamUserID+"}/sessions/{"+auth_http.URLParamSessionID+"}", h.DeleteSession)

			req := httptest.NewRequest(http.MethodDelete, "/users/"+tc.userIDStr+"/sessions/"+tc.sessionIDStr, nil)
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

func TestHandler_DeleteSessionsByUserID(t *testing.T) {
	t.Parallel()

	validID := uuid.New()
	tests := []struct {
		name       string
		userIDStr  string
		wantStatus int
		setup      func(s *mocks.AuthServiceMock)
	}{
		{
			name:       "invalid UUID -> 400 and service not called",
			userIDStr:  "not-a-uuid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:      "service error -> 500",
			userIDStr: validID.String(),
			setup: func(s *mocks.AuthServiceMock) {
				s.DeleteSessionsByUserIDMock.Expect(minimock.AnyContext, validID).Return(fmt.Errorf("service error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "ok -> 204 no content",
			userIDStr:  validID.String(),
			wantStatus: http.StatusNoContent,
			setup: func(s *mocks.AuthServiceMock) {
				s.DeleteSessionsByUserIDMock.Expect(minimock.AnyContext, validID).Return(nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mock := mocks.NewAuthServiceMock(t)
			if tc.setup != nil {
				tc.setup(mock)
			}
			h := auth_http.NewHandler(mock)
			r := chi.NewRouter()

			r.Delete("/users/{"+user_http.URLParamUserID+"}/sessions", h.DeleteSessionsByUserID)

			req := httptest.NewRequest(http.MethodDelete, "/users/"+tc.userIDStr+"/sessions", nil)
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

func TestHandler_AddUserRole(t *testing.T) {
	t.Parallel()

	entityID := uuid.New()
	userID := uuid.New()
	userRole := auth.UserRole{
		UserID:   userID,
		Role:     "role",
		EntityID: &entityID,
	}
	body, err := json.Marshal(userRole)
	require.NoError(t, err)
	tests := []struct {
		name       string
		body       []byte
		wantStatus int
		setup      func(s *mocks.AuthServiceMock)
	}{
		{
			name:       "invalid JSON -> 400 and service not called",
			body:       []byte("not-a-json"),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "service error -> 500",
			body:       body,
			wantStatus: http.StatusInternalServerError,
			setup: func(s *mocks.AuthServiceMock) {
				s.AddUserRoleMock.Expect(minimock.AnyContext, userRole).Return(fmt.Errorf("service error"))
			},
		},
		{
			name:       "ok -> 204 no content",
			body:       body,
			wantStatus: http.StatusNoContent,
			setup: func(s *mocks.AuthServiceMock) {
				s.AddUserRoleMock.Expect(minimock.AnyContext, userRole).Return(nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mock := mocks.NewAuthServiceMock(t)
			if tc.setup != nil {
				tc.setup(mock)
			}
			h := auth_http.NewHandler(mock)
			r := chi.NewRouter()

			r.Post("/roles", h.AddUserRole)

			req := httptest.NewRequest(http.MethodPost,
				"/roles",
				bytes.NewReader(tc.body),
			)
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

func TestHandler_DeleteUserRole(t *testing.T) {
	t.Parallel()

	entityID := uuid.New()
	userID := uuid.New()
	userRole := auth.UserRole{
		UserID:   userID,
		Role:     "role",
		EntityID: &entityID,
	}
	body, err := json.Marshal(userRole)
	require.NoError(t, err)
	tests := []struct {
		name       string
		body       []byte
		wantStatus int
		setup      func(s *mocks.AuthServiceMock)
	}{
		{
			name:       "invalid JSON -> 400 and service not called",
			body:       []byte("not-a-json"),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "service error -> 500",
			body:       body,
			wantStatus: http.StatusInternalServerError,
			setup: func(s *mocks.AuthServiceMock) {
				s.DeleteUserRoleMock.Expect(minimock.AnyContext, userRole).Return(fmt.Errorf("service error"))
			},
		},
		{
			name:       "ok -> 204 no content",
			body:       body,
			wantStatus: http.StatusNoContent,
			setup: func(s *mocks.AuthServiceMock) {
				s.DeleteUserRoleMock.Expect(minimock.AnyContext, userRole).Return(nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mock := mocks.NewAuthServiceMock(t)
			if tc.setup != nil {
				tc.setup(mock)
			}
			h := auth_http.NewHandler(mock)
			r := chi.NewRouter()

			r.Delete("/roles", h.DeleteUserRole)

			req := httptest.NewRequest(http.MethodDelete,
				"/roles",
				bytes.NewReader(tc.body),
			)
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

func TestHandler_ListUserRoles(t *testing.T) {
	t.Parallel()

	validID := uuid.New()
	roles := []auth.UserRole{{}, {}}
	tests := []struct {
		name       string
		userIDStr  string
		wantStatus int
		setup      func(s *mocks.AuthServiceMock)
		wantLen    int
	}{
		{
			name:       "invalid UUID -> 400 and service not called",
			userIDStr:  "not-a-uuid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:      "service error -> 500",
			userIDStr: validID.String(),
			setup: func(s *mocks.AuthServiceMock) {
				s.ListUserRolesMock.Expect(minimock.AnyContext, validID).Return(nil, fmt.Errorf("service error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "ok -> 200 with roles JSON",
			userIDStr:  validID.String(),
			wantStatus: http.StatusOK,
			setup: func(s *mocks.AuthServiceMock) {
				s.ListUserRolesMock.Expect(minimock.AnyContext, validID).Return(roles, nil)
			},
			wantLen: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mock := mocks.NewAuthServiceMock(t)
			if tc.setup != nil {
				tc.setup(mock)
			}
			h := auth_http.NewHandler(mock)
			r := chi.NewRouter()

			r.Get("/users/{"+user_http.URLParamUserID+"}/roles", h.ListUserRoles)

			req := httptest.NewRequest(http.MethodGet, "/users/"+tc.userIDStr+"/roles", nil)
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			require.Equal(t, tc.wantStatus, rr.Code)
			if tc.wantStatus == http.StatusOK {
				if ct := rr.Header().Get("Content-Type"); ct == "" || ct[:16] != "application/json" {
					t.Fatalf("content-type = %q; want application/json", ct)
				}
				var got []auth.UserRole
				err := json.Unmarshal(rr.Body.Bytes(), &got)
				require.NoError(t, err)
				require.Equal(t, roles, got)
			} else if rr.Body.Len() == 0 {
				t.Fatalf("error response body is empty; want some payload")
			}
		})
	}
}

func TestHandler_RefreshTokens(t *testing.T) {
	t.Parallel()

	req := auth.RefreshToken{
		SessionID: uuid.New(),
		Token:     "refresh",
	}
	resp := auth.Tokens{
		AccessToken: "new-access",
		RefreshToken: auth.RefreshToken{
			SessionID: uuid.New(),
			Token:     "new-refresh",
		},
	}
	body, err := json.Marshal(req)
	require.NoError(t, err)
	tests := []struct {
		name       string
		body       []byte
		wantStatus int
		setup      func(s *mocks.AuthServiceMock)
	}{
		{
			name:       "invalid JSON -> 400 and service not called",
			body:       []byte("not-a-json"),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "service error -> 500",
			body:       body,
			wantStatus: http.StatusInternalServerError,
			setup: func(s *mocks.AuthServiceMock) {
				s.RefreshTokensMock.Expect(minimock.AnyContext, req).Return(auth.Tokens{}, fmt.Errorf("service error"))
			},
		},
		{
			name:       "ok -> 200 with tokens JSON",
			body:       body,
			wantStatus: http.StatusOK,
			setup: func(s *mocks.AuthServiceMock) {
				s.RefreshTokensMock.Expect(minimock.AnyContext, req).Return(resp, nil)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mock := mocks.NewAuthServiceMock(t)
			if tc.setup != nil {
				tc.setup(mock)
			}
			h := auth_http.NewHandler(mock)
			r := chi.NewRouter()

			r.Post("/tokens/refresh", h.RefreshTokens)

			req := httptest.NewRequest(http.MethodPost,
				"/tokens/refresh",
				bytes.NewReader(tc.body),
			)
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			require.Equal(t, tc.wantStatus, rr.Code)
			if tc.wantStatus == http.StatusOK {
				if ct := rr.Header().Get("Content-Type"); ct == "" || ct[:16] != "application/json" {
					t.Fatalf("content-type = %q; want application/json", ct)
				}
				var got auth.Tokens
				err := json.Unmarshal(rr.Body.Bytes(), &got)
				require.NoError(t, err)
				require.Equal(t, resp, got)
			} else if rr.Body.Len() == 0 {
				t.Fatalf("error response body is empty; want some payload")
			}
		})
	}
}

func TestHandler_Login(t *testing.T) {
	t.Parallel()

	input := auth_http.LoginInput{
		Email:    "mail",
		Password: "pass",
	}
	req := usecase.LoginCmd{
		Email:    input.Email,
		Password: []byte(input.Password),
	}
	resp := auth.Tokens{
		AccessToken: "new-access",
		RefreshToken: auth.RefreshToken{
			SessionID: uuid.New(),
			Token:     "new-refresh",
		},
	}
	body, err := json.Marshal(input)
	require.NoError(t, err)

	tests := []struct {
		name       string
		body       []byte
		wantStatus int
		setup      func(s *mocks.AuthServiceMock)
	}{
		{
			name:       "invalid JSON -> 400 and service not called",
			body:       []byte("not-a-json"),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "service error -> 500",
			body:       body,
			wantStatus: http.StatusInternalServerError,
			setup: func(s *mocks.AuthServiceMock) {
				s.LoginMock.Expect(minimock.AnyContext, req).Return(auth.Tokens{}, fmt.Errorf("service error"))
			},
		},
		{
			name:       "ok -> 200 with tokens JSON",
			body:       body,
			wantStatus: http.StatusOK,
			setup: func(s *mocks.AuthServiceMock) {
				s.LoginMock.Expect(minimock.AnyContext, req).Return(resp, nil)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mock := mocks.NewAuthServiceMock(t)
			if tc.setup != nil {
				tc.setup(mock)
			}
			h := auth_http.NewHandler(mock)
			r := chi.NewRouter()

			r.Post("/login", h.Login)

			req := httptest.NewRequest(http.MethodPost,
				"/login",
				bytes.NewReader(tc.body),
			)
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			require.Equal(t, tc.wantStatus, rr.Code)
			if tc.wantStatus == http.StatusOK {
				if ct := rr.Header().Get("Content-Type"); ct == "" || ct[:16] != "application/json" {
					t.Fatalf("content-type = %q; want application/json", ct)
				}
				var got auth.Tokens
				err := json.Unmarshal(rr.Body.Bytes(), &got)
				require.NoError(t, err)
				require.Equal(t, resp, got)
			} else if rr.Body.Len() == 0 {
				t.Fatalf("error response body is empty; want some payload")
			}
		})
	}
}
