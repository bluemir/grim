package handler

import (
	"bytes"
	"io/fs"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"

	"github.com/bluemir/grim/assets"
)

var validSlug = regexp.MustCompile(`^[a-z0-9-]+(/[a-z0-9-]+)?$`)

func GuideManifest(c *gin.Context) error {
	guideFS, err := fs.Sub(assets.GuideFS, "guide")
	if err != nil {
		return err
	}
	data, err := fs.ReadFile(guideFS, "manifest.json")
	if err != nil {
		return err
	}
	c.Data(http.StatusOK, "application/json", data)
	return nil
}

func GuidePage(c *gin.Context) error {
	slug := strings.TrimPrefix(c.Param("slug"), "/")

	if !validSlug.MatchString(slug) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid slug"})
		return nil
	}

	guideFS, err := fs.Sub(assets.GuideFS, "guide")
	if err != nil {
		return err
	}

	mdData, err := fs.ReadFile(guideFS, slug+".md")
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "page not found"})
		return nil
	}

	md := goldmark.New(goldmark.WithExtensions(extension.GFM))
	var buf bytes.Buffer
	if err := md.Convert(mdData, &buf); err != nil {
		return err
	}

	c.JSON(http.StatusOK, gin.H{
		"slug": slug,
		"html": buf.String(),
	})
	return nil
}
