package clickhouse

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/get-glu/glu/internal/config"
	"github.com/get-glu/glu/internal/core"
	"github.com/get-glu/glu/internal/server"
)

type Storage struct {
	conn driver.Conn
}

func New(cfg *config.StorageClickHouse) (*Storage, error) {
	conn, err := connect(cfg)
	if err != nil {
		return nil, err
	}

	return &Storage{conn: conn}, nil
}

func conf(desc core.Descriptor, name string) string {
	value, _ := desc.Config[name].(string)
	return value
}

func (s *Storage) Status(ctx context.Context, desc core.Descriptor) (server.Status, error) {
	switch desc.Source.Kind {
	case "kubernetes":
		return s.statusKubernetes(ctx, desc)
	case "ci":
		scm := conf(desc, "scm")
		switch scm {
		case "github":
			return s.statusGitHub(ctx, desc)
		default:
			return nil, fmt.Errorf("unexpected scm: %q", scm)
		}

	case "oci":
		return nil, nil
	default:
		return nil, fmt.Errorf("unexpected source kind: %q", desc.Source.Kind)
	}
}

func (s *Storage) statusGitHub(ctx context.Context, desc core.Descriptor) (server.Status, error) {
	var (
		repository = conf(desc, "repository")
		branch     = conf(desc, "branch")
	)

	if branch == "" {
		branch = "main"
	}

	serviceName := ServiceName("github", repository)
	row := s.conn.QueryRow(ctx,
		`SELECT LogAttributes['vcs.repository.ref.revision'], LogAttributes['cicd.pipeline.name'],
		LogAttributes['cicd.pipeline.run.id'] FROM otel.otel_logs WHERE ServiceName = ? AND LogAttributes['vcs.repository.ref.name'] = ? ORDER BY Timestamp DESC LIMIT 1;`,
		serviceName,
		branch,
	)
	if err := row.Err(); err != nil {
		return nil, err
	}

	var (
		sha  string
		name string
		id   string
	)
	if err := row.Scan(&sha, &name, &id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return map[string]string{
		"sha":      sha,
		"pipeline": name,
		"run_id":   id,
	}, nil
}

func (s *Storage) statusKubernetes(ctx context.Context, desc core.Descriptor) (server.Status, error) {
	var (
		cluster   = desc.Source.Name
		namespace = conf(desc, "namespace")
		name      = conf(desc, "name")
		container = conf(desc, "container")
	)

	serviceName := ServiceName("kubernetes", cluster)
	row := s.conn.QueryRow(ctx,
		`SELECT ResourceAttributes['oci.manifest.digest'] FROM otel.otel_logs WHERE ServiceName = ? AND ResourceAttributes['k8s.namespace.name'] = ? AND ResourceAttributes['k8s.deployment.name'] = ? AND ResourceAttributes['k8s.container.name'] = ? ORDER BY Timestamp DESC LIMIT 1;`,
		serviceName,
		namespace,
		name,
		container,
	)
	if err := row.Err(); err != nil {
		return nil, err
	}

	var digest string
	if err := row.Scan(&digest); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return map[string]string{
		"digest": digest,
	}, nil
}

func connect(cfg *config.StorageClickHouse) (driver.Conn, error) {
	var (
		ctx       = context.Background()
		conn, err = clickhouse.Open(&clickhouse.Options{
			Addr: []string{cfg.Address},
			Auth: clickhouse.Auth{
				Database: cfg.Database,
				Username: cfg.Username,
				Password: cfg.Password,
			},
			ClientInfo: clickhouse.ClientInfo{
				Products: []struct {
					Name    string
					Version string
				}{
					{Name: "glu-go-client", Version: "0.1"},
				},
			},
			Debugf: func(format string, v ...interface{}) {
				fmt.Printf(format, v)
			},
		})
	)

	if err != nil {
		return nil, err
	}

	if err := conn.Ping(ctx); err != nil {
		if exception, ok := err.(*clickhouse.Exception); ok {
			fmt.Printf("Exception [%d] %s \n%s\n", exception.Code, exception.Message, exception.StackTrace)
		}
		return nil, err
	}

	return conn, nil
}

// TODO: move to package accessible by here and otel package
func ServiceName(prefix, name string) string {
	formattedName := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(name, "/", "-"), "_", "-"))
	return fmt.Sprintf("%s/%s", prefix, formattedName)
}
