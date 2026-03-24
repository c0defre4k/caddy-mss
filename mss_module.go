package mssmodule

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"
)

type mssKeyType struct{}
type connKeyType struct{}

var mssKey mssKeyType
var connKey connKeyType

func init() {
	caddy.RegisterModule(MSSMiddleware{})
	httpcaddyfile.RegisterHandlerDirective("mss_header", parseCaddyfile)
}

type MSSMiddleware struct {
	logger *zap.Logger
}

func (MSSMiddleware) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.mss_header",
		New: func() caddy.Module { return new(MSSMiddleware) },
	}
}

// unwrapConn extracts the underlying TCP connection from TLS or other wrappers
func unwrapConn(c net.Conn) *net.TCPConn {
	// Try to unwrap TLS
	if tlsConn, ok := c.(*tls.Conn); ok {
		c = tlsConn.NetConn()
	}

	// Now check if it's TCP
	if tcpConn, ok := c.(*net.TCPConn); ok {
		return tcpConn
	}

	return nil
}

func (m *MSSMiddleware) Provision(ctx caddy.Context) error {
	m.logger = ctx.Logger(m)
	m.logger.Info("MSSMiddleware provisioning started")

	app, err := ctx.App("http")
	if err != nil {
		m.logger.Error("failed to get HTTP app", zap.Error(err))
		return err
	}

	server := app.(*caddyhttp.App).Servers
	m.logger.Info("registering connection context", zap.Int("server_count", len(server)))

	for name, srv := range server {
		m.logger.Info("registering for server", zap.String("server_name", name))

		srv.RegisterConnContext(func(ctx context.Context, c net.Conn) context.Context {
			// Store the connection itself for later use
			ctx = context.WithValue(ctx, connKey, c)

			var mss int
			tc := unwrapConn(c)
			if tc != nil {
				var err error
				mss, err = getMSS(tc, m.logger)
				if err != nil {
					m.logger.Error("getMSS failed in RegisterConnContext", zap.Error(err))
				} else if mss > 0 {
					m.logger.Info("MSS extracted in RegisterConnContext", zap.Int("mss", mss))
				} else {
					m.logger.Warn("MSS is zero in RegisterConnContext")
				}
			} else {
				m.logger.Warn("Could not unwrap to TCP connection", zap.String("type", fmt.Sprintf("%T", c)))
			}

			return context.WithValue(ctx, mssKey, mss)
		})
	}

	m.logger.Info("MSSMiddleware provisioning complete")
	return nil
}

func (m MSSMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	mss, ok := r.Context().Value(mssKey).(int)

	if !ok || mss == 0 {
		if conn, ok := r.Context().Value(connKey).(net.Conn); ok {
			tc := unwrapConn(conn)
			if tc != nil {
				var err error
				mss, err = getMSS(tc, m.logger)
				if err != nil {
					m.logger.Error("getMSS failed in ServeHTTP", zap.Error(err))
				} else {
					m.logger.Info("MSS extracted in ServeHTTP", zap.Int("mss", mss))
				}
			}
		}
	}

	if mss > 0 {
		r.Header.Set("X-Client-MSS", fmt.Sprintf("%d", mss))
	}

	return next.ServeHTTP(w, r)
}

// UnmarshalCaddyfile implements caddyfile.Unmarshaler.
func (m *MSSMiddleware) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	d.Next()
	return nil
}

func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var m MSSMiddleware
	err := m.UnmarshalCaddyfile(h.Dispenser)
	return m, err
}

var (
	_ caddy.Provisioner           = (*MSSMiddleware)(nil)
	_ caddyhttp.MiddlewareHandler = (*MSSMiddleware)(nil)
)
