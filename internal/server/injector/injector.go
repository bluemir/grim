package injector

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/xid"

	"github.com/bluemir/grim/internal/server/backend"
)

var keyBackend = xid.New().String()

func Inject(b *backend.Backends) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(keyBackend, b)
	}
}
func Backends(c *gin.Context) *backend.Backends {
	return c.MustGet(keyBackend).(*backend.Backends)
}

type FrontendConfig struct {
	UseCDN         bool `json:"useCDN"`
	StorageEnabled bool `json:"storageEnabled"`
}

var keyConfig = xid.New().String()

func InjectConfig(conf *FrontendConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(keyConfig, conf)
	}
}
func Config(c *gin.Context) *FrontendConfig {
	return c.MustGet(keyConfig).(*FrontendConfig)
}
