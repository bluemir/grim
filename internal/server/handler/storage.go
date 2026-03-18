package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type createDiagramRequest struct {
	Source string `json:"source" binding:"required"`
}

func CreateDiagram(c *gin.Context) error {
	var req createDiagramRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil
	}

	diagram, err := backends(c).Storage.Create(c.Request.Context(), req.Source)
	if err != nil {
		return err
	}

	c.JSON(http.StatusCreated, diagram)
	return nil
}

func GetDiagram(c *gin.Context) error {
	id := c.Param("id")

	diagram, err := backends(c).Storage.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return nil
	}

	c.JSON(http.StatusOK, diagram)
	return nil
}

type updateDiagramRequest struct {
	Source string `json:"source" binding:"required"`
}

func UpdateDiagram(c *gin.Context) error {
	id := c.Param("id")

	var req updateDiagramRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil
	}

	diagram, err := backends(c).Storage.Update(c.Request.Context(), id, req.Source)
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, diagram)
	return nil
}
