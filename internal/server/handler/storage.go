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

	expiry, err := s.Expiry(c.Request.Context(), diagram)
	if err != nil {
		return err
	}
	c.JSON(http.StatusCreated, diagramResponse{diagram, expiry})
	return nil
}

func GetDiagram(c *gin.Context) error {
	s := backends(c).Storage
	diagram, err := s.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}

	expiry, err := s.Expiry(c.Request.Context(), diagram)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, diagramResponse{diagram, expiry})
	return nil
}

type addContactRequest struct {
	Email string `json:"email" binding:"required"`
}

// AddDiagramContact registers an email on a diagram and mails a verification link.
func AddDiagramContact(c *gin.Context) error {
	var req addContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil
	}
	if err := backends(c).Storage.AddContact(c.Request.Context(), c.Param("id"), req.Email); err != nil {
		return err
	}
	c.Status(http.StatusAccepted)
	return nil
}

type contactTokenRequest struct {
	Token string `json:"token" binding:"required"`
}

// ConfirmDiagramContact handles the "verify" and "extend" links from mails.
func ConfirmDiagramContact(c *gin.Context) error {
	var req contactTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil
	}
	res, err := backends(c).Storage.ConfirmContact(c.Request.Context(), req.Token)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, res)
	return nil
}

// RemoveDiagramContact handles the "remove my email" link from mails.
func RemoveDiagramContact(c *gin.Context) error {
	var req contactTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil
	}
	res, err := backends(c).Storage.RemoveContact(c.Request.Context(), req.Token)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, res)
	return nil
}
