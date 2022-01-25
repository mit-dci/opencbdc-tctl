package http

import (
	"fmt"
	"net/http"
)

func (srv *HttpServer) UserFromThumbprint(thumbprint string) *SystemUser {
	for _, u := range srv.users {
		if u.Thumbprint == thumbprint {
			return u
		}
	}
	return nil
}

func (srv *HttpServer) UserFromRequest(r *http.Request) (*SystemUser, error) {
	if r.TLS == nil {
		return nil, fmt.Errorf("Request contains no TLS info")
	}

	if len(r.TLS.PeerCertificates) == 0 {
		return nil, fmt.Errorf("Request contains no TLS peer certificate")
	}

	return srv.UserFromCert(r.TLS.PeerCertificates[0]), nil
}
