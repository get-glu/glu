package config

import "os"

var (
	_ validater = (*Server)(nil)
	_ defaulter = (*Server)(nil)
)

type Protocol string

const (
	ProtocolHTTP  = Protocol("http")
	ProtocolHTTPS = Protocol("https")
)

type Server struct {
	Port     int      `glu:"port"`
	Host     string   `glu:"host"`
	Protocol Protocol `glu:"protocol"`
	CertFile string   `glu:"cert_file"`
	KeyFile  string   `glu:"key_file"`
}

func (s *Server) validate() error {
	if s.Port < 0 {
		return errFieldPositiveNonZero("port")
	}

	if s.Protocol != ProtocolHTTP && s.Protocol != ProtocolHTTPS {
		return errFieldRequired("protocol")
	}

	if s.Protocol == ProtocolHTTPS {
		if s.CertFile == "" {
			return errFieldRequired("server.cert_file")
		}

		if s.KeyFile == "" {
			return errFieldRequired("server.key_file")
		}

		if _, err := os.Stat(s.CertFile); err != nil {
			return errFieldWrap("server.cert_file", err)
		}

		if _, err := os.Stat(s.KeyFile); err != nil {
			return errFieldWrap("server.key_file", err)
		}
	}

	return nil
}

func (s *Server) setDefaults() error {
	if s.Port <= 0 {
		s.Port = 8080
	}

	if s.Protocol == "" {
		s.Protocol = ProtocolHTTP
	}

	if s.Host == "" {
		s.Host = "0.0.0.0"
	}

	return nil
}
