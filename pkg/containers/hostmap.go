package containers

import (
	"context"
	"fmt"
	"iter"
	"net"
	"net/url"
	"slices"
	"strings"

	"go.flipt.io/flipt/errors"
	"google.golang.org/grpc/metadata"
)

type HostMap[T any] struct {
	hosts   []string
	entries map[string]T
}

func NewHostMap[T any]() *HostMap[T] {
	return &HostMap[T]{
		entries: map[string]T{},
	}
}

func (m *HostMap[T]) Add(host string, t T) error {
	if !strings.Contains(host, "://") {
		host = "http://" + host
	}

	u, err := url.Parse(host)
	if err != nil {
		return err
	}

	if u.Host == "" {
		return fmt.Errorf("unexpected host: %q", host)
	}

	if m.entries == nil {
		m.entries = map[string]T{}
	}

	hostname := u.Hostname()
	if _, ok := m.entries[hostname]; ok {
		return fmt.Errorf("entry already exists for host %q", hostname)
	}

	// insert hostname into hosts slice in order
	i, _ := slices.BinarySearch(m.hosts, hostname)
	m.hosts = slices.Insert(m.hosts, i, hostname)
	// store t by hostname (host excluding port)
	m.entries[hostname] = t

	return nil
}

func (m *HostMap[T]) All(context.Context) iter.Seq2[string, T] {
	return func(yield func(string, T) bool) {
		for _, host := range m.hosts {
			t, ok := m.entries[host]
			if !ok {
				panic("host missing from entries")
			}

			if !yield(host, t) {
				return
			}
		}
	}
}

func (m *HostMap[T]) Get(ctx context.Context) (t T, err error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return t, errors.ErrNotFound("host missing")
	}

	if hosts := md.Get("X-Forwarded-Host"); len(hosts) > 0 {
		host := parseHost(hosts[0])
		if host == "" {
			return t, errors.ErrNotFoundf("forwarded host: %q", hosts[0])
		}

		t, ok = m.entries[host]
		if !ok {
			return t, errors.ErrNotFoundf("forwarded host: %q", hosts[0])
		}

		return t, nil
	}

	if hosts := md.Get("Host"); len(hosts) > 0 {
		host := parseHost(hosts[0])
		if host == "" {
			return t, errors.ErrNotFoundf("host: %q", hosts[0])
		}

		t, ok = m.entries[host]
		if !ok {
			return t, errors.ErrNotFoundf("host: %q", hosts[0])
		}

		return t, nil
	}

	return t, errors.ErrNotFound("host missing")
}

func parseHost(host string) string {
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}

	return host
}
