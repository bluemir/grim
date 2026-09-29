package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"os"
	"path/filepath"

	"github.com/cockroachdb/errors"
	"github.com/gin-gonic/gin"
	"github.com/hjson/hjson-go/v4"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"

	"github.com/bluemir/grim/assets"
	"github.com/bluemir/grim/internal/server/backend"
	"github.com/bluemir/grim/internal/server/controller"
	"github.com/bluemir/grim/internal/server/injector"
	"github.com/bluemir/grim/internal/server/store"
)

type Args struct {
	ServiceHttpBind string
	Cert            CertConfig
	AdminHttpBind   string
	ConfigFilePath  string

	DBPath string
	Salt   string
	UseCDN bool
}

type Config struct {
	HTTP    HTTPConfig `yaml:"http"`
	Backend backend.Config
}

type HTTPConfig struct {
	// BaseURL is the public URL of this server (e.g. https://grim.example.com),
	// used for links in mails.
	BaseURL string `yaml:"baseURL"`
	// TrustedProxies lists proxy IPs/CIDRs whose forwarding headers are
	// believed when resolving the client IP. Empty means trust none and use
	// the connection's remote address.
	TrustedProxies []string `yaml:"trustedProxies"`
	// RemoteIPHeaders overrides which headers carry the client IP
	// (default: X-Forwarded-For, X-Real-IP).
	RemoteIPHeaders []string `yaml:"remoteIPHeaders"`
}

func DefaultConfig() Config {
	return Config{
		Backend: backend.DefaultConfig(),
	}
}

type Server struct {
	httpConfig     HTTPConfig
	backends       *backend.Backends
	frontendConfig *injector.FrontendConfig
}

func Run(ctx context.Context, args *Args) error {
	if err := assets.CheckDevAssets(); err != nil {
		return err
	}

	conf, err := ReadConfigFile(args.ConfigFilePath)
	if err != nil {
		return errors.Wrapf(err, "config file not exist. path: %s", args.ConfigFilePath)
	}

	// pass cmd to config
	conf.Backend.Auth.Salt = args.Salt
	conf.Backend.Storage.BaseURL = conf.HTTP.BaseURL

	db, err := store.Initialize(ctx, args.DBPath)
	if err != nil {
		return err
	}

	bs, err := backend.Initialize(ctx, &conf.Backend, db)
	if err != nil {
		return err
	}

	server := &Server{
		httpConfig: conf.HTTP,
		backends:   bs,
		frontendConfig: &injector.FrontendConfig{
			UseCDN:         args.UseCDN,
			StorageEnabled: conf.Backend.Storage.Enabled,
		},
	}

	if !logrus.IsLevelEnabled(logrus.DebugLevel) {
		gin.SetMode(gin.ReleaseMode)
	}

	certs, err := args.Cert.Load()
	if err != nil {
		return err
	}

	tlsConfig, err := getTLSConfig(certs, nil)
	if err != nil {
		return err
	}

	// run servers
	eg, nCtx := errgroup.WithContext(ctx)
	eg.Go(server.RunServiceHTTPServer(nCtx, args.ServiceHttpBind, tlsConfig))
	eg.Go(server.RunAdminHTTPServer(nCtx, args.AdminHttpBind))
	eg.Go(server.RunController(nCtx))

	// TODO run grpc, http, https, http2https redirect servers by config

	if err := eg.Wait(); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

type CertConfig struct {
	CertFile string
	KeyFile  string
}

func (cert *CertConfig) Load() (*tls.Certificate, error) {
	if cert == nil {
		return nil, nil
	}
	if cert.CertFile == "" || cert.KeyFile == "" {
		return nil, nil
	}
	c, err := tls.LoadX509KeyPair(cert.CertFile, cert.KeyFile)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &c, nil
}
func getTLSConfig(serverCert *tls.Certificate, clientAuthCACert *tls.Certificate) (*tls.Config, error) {
	if serverCert == nil {
		return nil, nil
	}
	conf := tls.Config{
		Certificates: []tls.Certificate{*serverCert},
	}

	if clientAuthCACert != nil {
		conf.ClientAuth = tls.VerifyClientCertIfGiven
		conf.ClientCAs = x509.NewCertPool()

		for _, der := range clientAuthCACert.Certificate {
			c, err := x509.ParseCertificate(der)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			conf.ClientCAs.AddCert(c)
		}
	}

	return &conf, nil
}

// RunController returns a function that runs the Controller event loop.
func (s *Server) RunController(ctx context.Context) func() error {
	return func() error {
		ctrl := controller.New(ctx, s.backends)
		return ctrl.Run(ctx)
	}
}

func ReadConfigFile(configFilePath string) (*Config, error) {
	conf := DefaultConfig()

	buf, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch filepath.Ext(configFilePath) {
	case ".yaml", ".yml":
		logrus.Info("parse as yaml")
		if err := yaml.Unmarshal(buf, &conf); err != nil {
			return nil, errors.WithStack(err)
		}
	case ".json", ".hjson":
		logrus.Info("parse as hjson")
		if err := hjson.Unmarshal(buf, &conf); err != nil {
			return nil, errors.WithStack(err)
		}
	default:
		return nil, errors.Errorf("unknown ext: %s", filepath.Ext(configFilePath))
	}

	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		buf, _ := hjson.Marshal(conf)
		logrus.Debugf("\n%s", string(buf))
	}

	return &conf, nil
}
