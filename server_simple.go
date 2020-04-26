package usvc

type ServerSimple struct {
	*Server
}

func NewServerSimple(c *Config) *ServerSimple {
	s := &ServerSimple{
		NewServer(c),
	}
	WithLiveliness("/health")(s.Server)
	return s
}
