package clickhouse

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

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

func (s *Storage) Status(ctx context.Context, desc core.Descriptor) (server.Status, error) {
	switch desc.Source.Kind {
	case "kubernetes":
		conf := func(name string) string {
			value, _ := desc.Config[name].(string)
			return value
		}

		return s.statusKubernetes(ctx, desc.Source.Name, conf("namespace"), conf("name"), conf("container"))
	case "oci", "ci":
		return nil, nil
	default:
		return nil, fmt.Errorf("unexpected source kind: %q", desc.Source.Kind)
	}
}

func (s *Storage) statusKubernetes(ctx context.Context, cluster, namespace, name, container string) (server.Status, error) {
	serviceName := fmt.Sprintf("kubernetes/%s", cluster)
	row := s.conn.QueryRow(ctx,
		`select ResourceAttributes['oci.manifest.digest'] from otel.otel_logs where ServiceName = ? AND ResourceAttributes['k8s.namespace.name'] = ? AND ResourceAttributes['k8s.deployment.name'] = ? AND ResourceAttributes['k8s.container.name'] = ? order by Timestamp desc limit 1;`,
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
		"Digest": digest,
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
