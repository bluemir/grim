package cache

import (
	"github.com/gin-gonic/gin"

	"github.com/bluemir/grim/internal/buildinfo"
)

const (
	REVVED = "__REVVED__"
)

func CacheBusting(c *gin.Context) {
	c.Set(REVVED, Rev())
}

func Rev() string {
	return buildinfo.Signature[:16]
}

// in html.
// <script src="/static/{{ .GetString "__REVVED__" }}/js/index.js"></script>
