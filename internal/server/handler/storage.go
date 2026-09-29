package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/bluemir/grim/internal/server/backend/storage"
)

type createDiagramRequest struct {
	Source string `json:"source" binding:"required"`
}

type diagramResponse struct {
	*storage.Diagram
	Expiry storage.Expiry `json:"expiry"`
}

func CreateDiagram(c *gin.Context) error {
	s := backends(c).Storage
	if limit := s.MaxRequestBytes(); limit > 0 {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
	}

	var req createDiagramRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return storage.ErrTooLarge
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil
	}

	diagram, err := s.Create(c.Request.Context(), req.Source, c.ClientIP())
	if err != nil {
		return err
	}

	c.JSON(http.StatusCreated, diagramResponse{diagram, s.Expiry(diagram)})
	return nil
}

func GetDiagram(c *gin.Context) error {
	s := backends(c).Storage
	diagram, err := s.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, diagramResponse{diagram, s.Expiry(diagram)})
	return nil
}
