package server

import (
	"bytes"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/bluemir/grim/assets"
	"github.com/bluemir/grim/internal/server/handler"
	"github.com/bluemir/grim/internal/server/middleware/cache"
)

// @title grim
// @version 0.1.0
// @description
func (server *Server) routes(app gin.IRouter, noRoute func(...gin.HandlerFunc)) {
	var (
	//requireLogin = handler.RequireLogin
	//can          = handler.Can
	)

	// API
	{
		v1 := app.Group("/api/v1", markAcceptJSON)

		v1.GET("/ping", api(handler.Ping))

		// WebSocket
		//v1.GET("/ws", handler.Websocket)
		// Server Sent Event
		//v1.GET("/stream", sse(handler.Stream))
		// http2 Server Push
		//v1.GET("/push", api(handler.Push))
	}

	// Static Pages
	{
		// js, css, etc.
		app.Group("/static").Group(cache.Rev(), cache.Set(cache.ForRevvedResource)).StaticFS("/", http.FS(assets.Static()))

		app.GET("/", html("index.html"))
		app.GET("/editor", html("editor.html"))
	}

	noRoute(func(c *gin.Context) {
		for accept := range strings.SplitSeq(c.Request.Header.Get("Accept"), ",") {
			t, _, e := mime.ParseMediaType(accept)
			if e != nil {
				continue
			}

			switch t {
			case "application/json":
				c.Status(http.StatusNotFound)
				return
			case "text/html", "*/*":
				c.HTML(http.StatusNotFound, "errors/not-found.html", c)
				return
			case "text/plain":
				c.String(http.StatusNotFound, "not found")
				return
			}
		}
	})
}
func api(fn func(c *gin.Context) error) gin.HandlerFunc {

	return func(c *gin.Context) {
		if err := fn(c); err != nil {
			c.Error(err)
			c.Abort()
		}
	}
}

type ServerSentEventErrorData struct {
	Type       string         `json:"type"`
	Title      string         `json:"title"`
	Status     int            `json:"status,omitempty"`
	Detail     string         `json:"detail,omitempty"`
	Instance   string         `json:"instance,omitempty"`
	Extensions map[string]any `json:"-"`
}

func sse(fn func(c *gin.Context) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		rc := http.NewResponseController(c.Writer)
		rc.SetWriteDeadline(time.Time{})
		rc.SetReadDeadline(time.Time{})

		if err := fn(c); err != nil {
			c.SSEvent("error", ServerSentEventErrorData{
				Type:   "about:blank",
				Title:  "Error",
				Detail: err.Error(),
			})
		}
	}
}

func bodyReaderTweak(c *gin.Context) {
	// body 를 여러번 읽지 못하는 것을 대응 하기 위한 tweak
	body := c.Request.Body

	buf := bytes.NewBuffer(nil)

	if _, err := io.Copy(buf, body); err != nil {
		c.Error(err)
		c.Abort()
		return
	}

	// should i use `gin.BodyBytesKey`? https://stackoverflow.com/questions/62736851/go-gin-read-request-body-many-times

	// Data 는 buf에서 읽고, close 는 원래것에서 호출
	c.Request.Body = struct {
		io.Reader
		io.Closer
	}{
		Reader: bytes.NewReader(buf.Bytes()),
		Closer: body,
	}

}
