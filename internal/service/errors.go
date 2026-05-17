package service

import "net/http"

type Error struct {
	Status  int
	Code    string
	Message string
}

func (e Error) Error() string {
	return e.Message
}

func newError(status int, code, message string) Error {
	return Error{Status: status, Code: code, Message: message}
}

func invalidRequest(message string) Error {
	return newError(http.StatusBadRequest, "invalid_request", message)
}
