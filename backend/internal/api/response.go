package api

import (
	"errors"
	"net/http"

	"github.com/awu123-create/http-bench/internal/task"
	"github.com/gin-gonic/gin"
)

type Response struct {
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func OK(c *gin.Context, data any) {
	c.JSON(200, Response{Message: "ok", Data: data})
}

func Created(c *gin.Context, data any) {
	c.JSON(201, Response{Message: "created", Data: data})
}

func Error(c *gin.Context, status int, message string) {
	c.JSON(status, Response{Message: message})
}

func RespondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, task.ErrInvalidConfig), errors.Is(err, task.ErrInvalidTaskID):
		Error(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, task.ErrTaskNotFound):
		Error(c, http.StatusNotFound, err.Error())
	case errors.Is(err, task.ErrTaskAlreadyRunning):
		Error(c, http.StatusConflict, err.Error())
	case errors.Is(err, task.ErrTaskNotRunning):
		Error(c, http.StatusConflict, err.Error())
	default:
		Error(c, http.StatusInternalServerError, err.Error())
	}
}
