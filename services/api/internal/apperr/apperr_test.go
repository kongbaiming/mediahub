package apperr

import (
	"errors"
	"net/http"
	"testing"
)

func TestAppError_HTTPStatus(t *testing.T) {
	tests := []struct {
		code Code
		want int
	}{
		{CodeBadRequest, http.StatusBadRequest},
		{CodeValidation, http.StatusBadRequest},
		{CodeUnsupported, http.StatusBadRequest},
		{CodeUnauthorized, http.StatusUnauthorized},
		{CodeForbidden, http.StatusForbidden},
		{CodeNotFound, http.StatusNotFound},
		{CodeFileNotFound, http.StatusNotFound},
		{CodeConflict, http.StatusConflict},
		{CodeExternalAPI, http.StatusBadGateway},
		{CodeInternal, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			e := New(tt.code, "test")
			if got := e.HTTPStatus(); got != tt.want {
				t.Errorf("HTTPStatus(%q) = %d, want %d", tt.code, got, tt.want)
			}
		})
	}
}

func TestAppError_Error(t *testing.T) {
	e := New(CodeNotFound, "user not found")
	got := e.Error()
	want := "[not_found] user not found"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestAppError_Wrap(t *testing.T) {
	base := errors.New("connection refused")
	e := Wrap(base, CodeExternalAPI, "TMDB 调用失败")
	if e.Err != base {
		t.Errorf("Err = %v, want %v", e.Err, base)
	}
	if e.Code != CodeExternalAPI {
		t.Errorf("Code = %q, want %q", e.Code, CodeExternalAPI)
	}
	// Error() should include wrapped err
	if got := e.Error(); got == "" {
		t.Error("Error() returned empty string")
	}
}

func TestAppError_As(t *testing.T) {
	e := New(CodeBadRequest, "invalid input")
	var err error = e

	got, ok := As(err)
	if !ok {
		t.Fatal("As() should return true for AppError")
	}
	if got.Code != CodeBadRequest {
		t.Errorf("As() code = %q, want %q", got.Code, CodeBadRequest)
	}
}

func TestAppError_As_NotAppError(t *testing.T) {
	err := errors.New("plain error")
	got, ok := As(err)
	if ok {
		t.Errorf("As() should return false for plain error, got %v", got)
	}
}

func TestConstructors(t *testing.T) {
	tests := []struct {
		name string
		got  *AppError
		code Code
	}{
		{"BadRequest", BadRequest("x"), CodeBadRequest},
		{"Unauthorized", Unauthorized("x"), CodeUnauthorized},
		{"Forbidden", Forbidden("x"), CodeForbidden},
		{"NotFound", NotFound("x"), CodeNotFound},
		{"Internal", Internal("x"), CodeInternal},
		{"Validation", Validation("x"), CodeValidation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got.Code != tt.code {
				t.Errorf("%s code = %q, want %q", tt.name, tt.got.Code, tt.code)
			}
		})
	}
}
