package http_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/66gu1/easygodocs/internal/app/user"
	user_http "github.com/66gu1/easygodocs/internal/app/user/transport/http"
	"github.com/66gu1/easygodocs/internal/app/user/transport/http/mocks"
	user_usecase "github.com/66gu1/easygodocs/internal/app/user/usecase"
	"github.com/go-chi/chi/v5"
	"github.com/gojuno/minimock/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

//go:generate minimock -o ./mocks -s _mock.go

func TestHandler_CreateUser(t *testing.T) {
	t.Parallel()

	var (
		input = user_http.CreateUserInput{
			Email:    "mail",
			Name:     "name",
			Password: "password",
		}
		cmd = user.CreateUserReq{
			Email:    input.Email,
			Name:     input.Name,
			Password: []byte(input.Password),
		}
	)
	body, err := json.Marshal(&input)
	require.NoError(t, err)

	tests := []struct {
		name       string
		body       []byte
		setup      func(mock *mocks.ServiceMock)
		wantStatus int
	}{
		{
			name:       "valid",
			body:       body,
			wantStatus: http.StatusCreated,
			setup: func(mock *mocks.ServiceMock) {
				mock.CreateUserMock.Expect(minimock.AnyContext, cmd).Return(nil)
			},
		},
		{
			name:       "invalid json -> 400",
			body:       []byte("{"),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "usecase error -> 500",
			body:       body,
			wantStatus: http.StatusInternalServerError,
			setup: func(mock *mocks.ServiceMock) {
				mock.CreateUserMock.Expect(minimock.AnyContext, cmd).Return(fmt.Errorf("error"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := minimock.NewController(t)

			svcMock := mocks.NewServiceMock(mc)
			if tt.setup != nil {
				tt.setup(svcMock)
			}

			h := user_http.NewHandler(svcMock)

			r := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(tt.body))
			r.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			h.CreateUser(w, r)

			res := w.Result()
			defer res.Body.Close()

			require.Equal(t, tt.wantStatus, res.StatusCode)
		})
	}
}

func TestHandler_GetUser(t *testing.T) {
	t.Parallel()

	var (
		id  = uuid.New()
		usr = user.User{
			ID:    id,
			Email: "mail",
			Name:  "name",
		}
	)

	tests := []struct {
		name       string
		userID     string
		setup      func(mock *mocks.ServiceMock)
		wantStatus int
	}{
		{
			name:       "valid",
			userID:     id.String(),
			wantStatus: http.StatusOK,
			setup: func(mock *mocks.ServiceMock) {
				mock.GetUserMock.Expect(minimock.AnyContext, id).Return(usr, nil)
			},
		},
		{
			name:       "invalid uuid -> 400",
			userID:     "id",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "usecase error -> 500",
			userID:     id.String(),
			wantStatus: http.StatusInternalServerError,
			setup: func(mock *mocks.ServiceMock) {
				mock.GetUserMock.Expect(minimock.AnyContext, id).Return(usr, fmt.Errorf("error"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := minimock.NewController(t)

			svcMock := mocks.NewServiceMock(mc)
			if tt.setup != nil {
				tt.setup(svcMock)
			}

			h := user_http.NewHandler(svcMock)
			r := chi.NewRouter()

			r.Put("/users/{"+user_http.URLParamUserID+"}", h.GetUser)

			req := httptest.NewRequest(http.MethodPut, "/users/"+tt.userID, http.NoBody)
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			require.Equal(t, tt.wantStatus, rr.Code)
			if tt.wantStatus == http.StatusOK {
				var got user.User
				err := json.NewDecoder(rr.Body).Decode(&got)
				require.NoError(t, err)
				require.Equal(t, usr, got)
			}
		})
	}
}

func TestHandler_GetUsers(t *testing.T) {
	t.Parallel()

	users := []user.User{
		{
			ID:    uuid.New(),
			Email: "mail",
			Name:  "name",
		},
	}

	tests := []struct {
		name       string
		setup      func(mock *mocks.ServiceMock)
		wantStatus int
	}{
		{
			name:       "valid",
			wantStatus: http.StatusOK,
			setup: func(mock *mocks.ServiceMock) {
				mock.GetAllUsersMock.Expect(minimock.AnyContext).Return(users, nil)
			},
		},
		{
			name:       "usecase error -> 500",
			wantStatus: http.StatusInternalServerError,
			setup: func(mock *mocks.ServiceMock) {
				mock.GetAllUsersMock.Expect(minimock.AnyContext).Return(nil, fmt.Errorf("error"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := minimock.NewController(t)

			svcMock := mocks.NewServiceMock(mc)
			if tt.setup != nil {
				tt.setup(svcMock)
			}

			h := user_http.NewHandler(svcMock)
			r := chi.NewRouter()

			r.Get("/users", h.GetAllUsers)

			req := httptest.NewRequest(http.MethodGet, "/users", http.NoBody)
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			require.Equal(t, tt.wantStatus, rr.Code)
			if tt.wantStatus == http.StatusOK {
				var got []user.User
				err := json.NewDecoder(rr.Body).Decode(&got)
				require.NoError(t, err)
				require.Equal(t, users, got)
			}
		})
	}
}

func TestHandler_UpdateUser(t *testing.T) {
	t.Parallel()

	var (
		id = uuid.New()

		input = user_http.UpdateUserInput{
			Email: "mail",
			Name:  "name",
		}
		cmd = user.UpdateUserReq{
			UserID: id,
			Email:  input.Email,
			Name:   input.Name,
		}
	)
	body, err := json.Marshal(&input)
	require.NoError(t, err)

	tests := []struct { //nolint:dupl
		name       string
		userID     string
		body       []byte
		setup      func(mock *mocks.ServiceMock)
		wantStatus int
	}{
		{
			name:       "valid",
			userID:     id.String(),
			body:       body,
			wantStatus: http.StatusNoContent,
			setup: func(mock *mocks.ServiceMock) {
				mock.UpdateUserMock.Expect(minimock.AnyContext, cmd).Return(nil)
			},
		},
		{
			name:       "invalid uuid -> 400",
			userID:     "id",
			body:       body,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid json -> 400",
			userID:     id.String(),
			body:       []byte("{"),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "usecase error -> 500",
			userID:     id.String(),
			body:       body,
			wantStatus: http.StatusInternalServerError,
			setup: func(mock *mocks.ServiceMock) {
				mock.UpdateUserMock.Expect(minimock.AnyContext, cmd).Return(fmt.Errorf("error"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := minimock.NewController(t)

			svcMock := mocks.NewServiceMock(mc)
			if tt.setup != nil {
				tt.setup(svcMock)
			}

			h := user_http.NewHandler(svcMock)
			r := chi.NewRouter()

			r.Put("/users/{"+user_http.URLParamUserID+"}", h.UpdateUser)

			req := httptest.NewRequest(http.MethodPut, "/users/"+tt.userID, bytes.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			require.Equal(t, tt.wantStatus, rr.Code)
		})
	}
}

func TestHandler_DeleteUser(t *testing.T) {
	t.Parallel()

	id := uuid.New()

	tests := []struct {
		name       string
		userID     string
		setup      func(mock *mocks.ServiceMock)
		wantStatus int
	}{
		{
			name:       "valid",
			userID:     id.String(),
			wantStatus: http.StatusNoContent,
			setup: func(mock *mocks.ServiceMock) {
				mock.DeleteUserMock.Expect(minimock.AnyContext, id).Return(nil)
			},
		},
		{
			name:       "invalid uuid -> 400",
			userID:     "id",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "usecase error -> 500",
			userID:     id.String(),
			wantStatus: http.StatusInternalServerError,
			setup: func(mock *mocks.ServiceMock) {
				mock.DeleteUserMock.Expect(minimock.AnyContext, id).Return(fmt.Errorf("error"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := minimock.NewController(t)

			svcMock := mocks.NewServiceMock(mc)
			if tt.setup != nil {
				tt.setup(svcMock)
			}

			h := user_http.NewHandler(svcMock)
			r := chi.NewRouter()

			r.Delete("/users/{"+user_http.URLParamUserID+"}", h.DeleteUser)

			req := httptest.NewRequest(http.MethodDelete, "/users/"+tt.userID, http.NoBody)
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			require.Equal(t, tt.wantStatus, rr.Code)
		})
	}
}

func TestHandler_ChangePassword(t *testing.T) {
	t.Parallel()

	var (
		id = uuid.New()

		input = user_http.ChangePasswordInput{
			OldPassword: "old_password",
			NewPassword: "new_password",
		}
		cmd = user_usecase.ChangePasswordCmd{
			ID:          id,
			OldPassword: []byte(input.OldPassword),
			NewPassword: []byte(input.NewPassword),
		}
	)
	body, err := json.Marshal(&input)
	require.NoError(t, err)

	tests := []struct { //nolint:dupl
		name       string
		userID     string
		body       []byte
		setup      func(mock *mocks.ServiceMock)
		wantStatus int
	}{
		{
			name:       "valid",
			userID:     id.String(),
			body:       body,
			wantStatus: http.StatusNoContent,
			setup: func(mock *mocks.ServiceMock) {
				mock.ChangePasswordMock.Expect(minimock.AnyContext, cmd).Return(nil)
			},
		},
		{
			name:       "invalid uuid -> 400",
			userID:     "id",
			body:       body,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid json -> 400",
			userID:     id.String(),
			body:       []byte("{"),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "usecase error -> 500",
			userID:     id.String(),
			body:       body,
			wantStatus: http.StatusInternalServerError,
			setup: func(mock *mocks.ServiceMock) {
				mock.ChangePasswordMock.Expect(minimock.AnyContext, cmd).Return(fmt.Errorf("error"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := minimock.NewController(t)

			svcMock := mocks.NewServiceMock(mc)
			if tt.setup != nil {
				tt.setup(svcMock)
			}

			h := user_http.NewHandler(svcMock)
			r := chi.NewRouter()

			r.Post("/users/{"+user_http.URLParamUserID+"}/change_password", h.ChangePassword)

			req := httptest.NewRequest(http.MethodPost, "/users/"+tt.userID+"/change_password", bytes.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			require.Equal(t, tt.wantStatus, rr.Code)
		})
	}
}
