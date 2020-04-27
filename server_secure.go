package usvc

import "crypto/tls"

type ServerSecure struct {
	*Server
}

func NewServerSecure(c *Config, certFile, keyFile string) (*ServerSecure, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	s := &ServerSecure{
		NewServer(c),
	}
	WithLiveliness("/health")(s.Server)
	WithTLS(cert)(s.Server)
	return s, nil
}
