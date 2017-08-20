package pm

import (
	"fmt"
	"net/http"
)

type RunError interface {
	Code() int
	Cause() interface{}
}

type errorImpl struct {
	code  int
	cause interface{}
}

func (e *errorImpl) Error() string {
	return fmt.Sprintf("[%d] %v", e.code, e.cause)
}

func Error(code int, cause interface{}) error {
	return &errorImpl{code: code, cause: cause}
}

func BadRequest(cause interface{}) error {
	return Error(http.StatusBadRequest, cause)
}

func InternalError(cause interface{}) error {
	return Error(http.StatusInternalServerError, cause)
}
