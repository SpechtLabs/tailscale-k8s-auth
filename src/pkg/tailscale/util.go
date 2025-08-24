package tailscale

import (
	"crypto/tls"
	"net/http"

	"tailscale.com/ipn"
)

// CtxConnKey is a key to look up a net.Conn stored in an HTTP request's context.
// Exported for tests and integration points that need to retrieve the connection.
type CtxConnKey struct{}

// IsFunnelRequest checks if an HTTP request is coming over Tailscale Funnel.
func IsFunnelRequest(r *http.Request) bool {
	// If we're funneling through the local tailscaled, it will set this HTTP
	// header.
	if r.Header.Get("Tailscale-Funnel-Request") != "" {
		return true
	}

	// If the funneled connection is from tsnet, then the net.Conn will be of
	// type ipn.FunnelConn.
	netConn := r.Context().Value(CtxConnKey{})

	// if the conn is wrapped inside TLS, unwrap it
	if tlsConn, ok := netConn.(*tls.Conn); ok {
		netConn = tlsConn.NetConn()
	}

	if _, ok := netConn.(*ipn.FunnelConn); ok {
		return true
	}

	return false
}
