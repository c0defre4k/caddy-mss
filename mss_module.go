package mssmodule

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"golang.org/x/sys/unix"
)

// Define a private type for the context key to avoid collisions
type mssKeyType struct{}

var mssKey mssKeyType

func init() {
	caddy.RegisterModule(MSSMiddleware{})
	httpcaddyfile.RegisterHandlerDirective("mss_header", parseCaddyfile)
}

type MSSMiddleware struct{}

func (MSSMiddleware) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.mss_header",
		New: func() caddy.Module { return new(MSSMiddleware) },
	}
}

// Provision sets up the connection context callback
func (m *MSSMiddleware) Provision(ctx caddy.Context) error {
	// Find the parent HTTP app to register our context modifier
	// This is the "Modern Way" mentioned in the deprecation notice
	app, err := ctx.App("http")
	if err != nil {
		return err
	}
	server := app.(*caddyhttp.App).Servers

	// Register a callback for ALL servers in this app
	for _, srv := range server {
		srv.RegisterConnContext(func(ctx context.Context, c net.Conn) context.Context {
			var mss int
			if tc, ok := c.(*net.TCPConn); ok {
				raw, _ := tc.SyscallConn()
				raw.Control(func(fd uintptr) {
					mss, _ = unix.GetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_MAXSEG)
				})
			}
			// Inject the MSS directly into the context
			return context.WithValue(ctx, mssKey, mss)
		})
	}
	return nil
}

func (m MSSMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	// Retrieve the MSS from the context we populated in RegisterConnContext
	if mss, ok := r.Context().Value(mssKey).(int); ok && mss > 0 {
		r.Header.Set("X-Client-MSS", fmt.Sprintf("%d", mss))
	}
	return next.ServeHTTP(w, r)
}

// UnmarshalCaddyfile implements caddyfile.Unmarshaler.
func (m *MSSMiddleware) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	d.Next() // consume directive name

	// require an argument
	// if !d.NextArg() {
	// 	return d.ArgErr()
	// }

	// store the argument
	// m.Output = d.Val()
	return nil
}

// parseCaddyfile unmarshals tokens from h into a new Middleware.
func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var m MSSMiddleware
	err := m.UnmarshalCaddyfile(h.Dispenser)
	return m, err
}

// Interface guards
var (
	_ caddy.Provisioner           = (*MSSMiddleware)(nil)
	_ caddyhttp.MiddlewareHandler = (*MSSMiddleware)(nil)
)
