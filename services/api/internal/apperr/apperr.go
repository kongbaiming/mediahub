// Package apperr 提供统一的业务错误类型
package apperr

import (
	"errors"
	"fmt"
	"net/http"
)

// Code 是错误码
type Code string

const (
	CodeBadRequest    Code = "bad_request"
	CodeUnauthorized  Code = "unauthorized"
	CodeForbidden     Code = "forbidden"
	CodeNotFound      Code = "not_found"
	CodeConflict      Code = "conflict"
	CodeValidation    Code = "validation"
	CodeInternal      Code = "internal"
	CodeExternalAPI   Code = "external_api"
	CodeScrapeFailed  Code = "scrape_failed"
	CodeFileNotFound  Code = "file_not_found"
	CodeUnsupported   Code = "unsupported"
)

// AppError 是带错误码的应用错误
type AppError struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
	Detail  any    `json:"detail,omitempty"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error { return e.Err }

// HTTPStatus 返回对应的 HTTP 状态码
func (e *AppError) HTTPStatus() int {
	switch e.Code {
	case CodeBadRequest, CodeValidation, CodeUnsupported:
		return http.StatusBadRequest
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeNotFound, CodeFileNotFound:
		return http.StatusNotFound
	case CodeConflict:
		return http.StatusConflict
	case CodeExternalAPI:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

// 构造器
func New(code Code, msg string) *AppError {
	return &AppError{Code: code, Message: msg}
}

func Wrap(err error, code Code, msg string) *AppError {
	return &AppError{Code: code, Message: msg, Err: err}
}

func BadRequest(msg string) *AppError     { return New(CodeBadRequest, msg) }
func Unauthorized(msg string) *AppError   { return New(CodeUnauthorized, msg) }
func Forbidden(msg string) *AppError      { return New(CodeForbidden, msg) }
func NotFound(msg string) *AppError       { return New(CodeNotFound, msg) }
func Internal(msg string) *AppError       { return New(CodeInternal, msg) }
func Validation(detail any) *AppError     { return &AppError{Code: CodeValidation, Message: "参数校验失败", Detail: detail} }
func ExternalAPI(err error, msg string) *AppError { return Wrap(err, CodeExternalAPI, msg) }

// As 是 errors.As 的快捷方式
func As(err error) (*AppError, bool) {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae, true
	}
	return nil, false
}
