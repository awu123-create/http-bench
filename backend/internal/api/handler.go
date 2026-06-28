package api

import (
	"io"
	"strconv"

	"github.com/awu123-create/http-bench/internal/config"
	"github.com/awu123-create/http-bench/internal/task"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	manager *task.Manager
}

func NewHandler(manager *task.Manager) *Handler {
	return &Handler{
		manager: manager,
	}
}

func getTaskID(c *gin.Context) (int64, error) {
	taskID := c.Param("id")
	id, err := strconv.ParseInt(taskID, 10, 64)
	if err != nil {
		return 0, task.ErrInvalidTaskID
	}
	return id, nil
}

func (h *Handler) CreateTest(c *gin.Context) {
	var cfg config.TestConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		RespondError(c, err)
		return
	}

	task, err := h.manager.Create(c.Request.Context(), cfg)
	if err != nil {
		RespondError(c, err)
		return
	}

	Created(c, gin.H{
		"task_id": task.ID,
	})
}

func (h *Handler) StartTest(c *gin.Context) {
	// 从URL参数中获取任务ID
	taskID, err := getTaskID(c)
	if err != nil {
		RespondError(c, err)
		return
	}

	err = h.manager.Start(c.Request.Context(), taskID)
	if err != nil {
		RespondError(c, err)
		return
	}

	OK(c, gin.H{
		"result": "Task started",
	})
}

func (h *Handler) GetTest(c *gin.Context) {
	taskID, err := getTaskID(c)
	if err != nil {
		RespondError(c, err)
		return
	}

	task, err := h.manager.Get(c.Request.Context(), taskID)
	if err != nil {
		RespondError(c, err)
		return
	}

	OK(c, gin.H{
		"task": task,
	})
}

func (h *Handler) StopTest(c *gin.Context) {
	taskID, err := getTaskID(c)
	if err != nil {
		RespondError(c, err)
		return
	}

	err = h.manager.Stop(c.Request.Context(), taskID)
	if err != nil {
		RespondError(c, err)
		return
	}

	OK(c, gin.H{
		"result": "Task stopped",
	})
}

func (h *Handler) GetMetrics(c *gin.Context) {
	taskID, err := getTaskID(c)
	if err != nil {
		RespondError(c, err)
		return
	}

	_, err = h.manager.Get(c.Request.Context(), taskID)
	if err != nil {
		RespondError(c, err)
		return
	}

	metrics, err := h.manager.GetAllMetrics(c.Request.Context(), taskID)
	if err != nil {
		RespondError(c, err)
		return
	}

	OK(c, gin.H{
		"metrics": metrics,
	})
}

func (h *Handler) StreamMetrics(c *gin.Context) {
	taskID, err := getTaskID(c)
	if err != nil {
		RespondError(c, err)
		return
	}

	ch, unsubscribe, err := h.manager.SubscribeMetrics(c.Request.Context(), taskID)
	if err != nil {
		RespondError(c, err)
		return
	}
	defer unsubscribe()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	c.Stream(func(w io.Writer) bool {
		select {
		case point, ok := <-ch:
			if !ok {
				return false
			}
			c.SSEvent("metric", point)
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}

func (h *Handler) ListTests(c *gin.Context) {
	tasks, err := h.manager.ListTasks(c.Request.Context())
	if err != nil {
		RespondError(c, err)
		return
	}

	OK(c, gin.H{
		"tasks": tasks,
	})
}

func (h *Handler) GetTestResult(c *gin.Context) {
	taskID, err := getTaskID(c)
	if err != nil {
		RespondError(c, err)
		return
	}

	result, err := h.manager.GetTestResult(c.Request.Context(), taskID)
	if err != nil {
		RespondError(c, err)
		return
	}

	OK(c, gin.H{
		"result": result,
	})
}
