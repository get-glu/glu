package config

import "fmt"

var (
	_ validater = (*Storage)(nil)
	_ defaulter = (*Storage)(nil)
)

type StorageType string

const (
	StorageTypeClickHouse = StorageType("clickhouse")
)

type Storage struct {
	Type       StorageType
	ClickHouse *StorageClickHouse
}

func (s *Storage) validate() error {
	switch s.Type {
	case StorageTypeClickHouse:
		if s.ClickHouse.Address == "" {
			return errFieldRequired("storage.clickhouse.address")
		}

		if s.ClickHouse.Database == "" {
			return errFieldRequired("storage.clickhouse.database")
		}
	default:
		return fmt.Errorf("unexpected storage type: %q", s.Type)
	}

	return nil
}

func (s *Storage) setDefaults() error {
	if s.Type == "" {
		s.Type = StorageTypeClickHouse
	}

	switch s.Type {
	case StorageTypeClickHouse:
		ch := s.ClickHouse
		if ch == nil {
			ch = &StorageClickHouse{}
			s.ClickHouse = ch
		}

		if ch.Address == "" {
			ch.Address = "127.0.0.1:9000"
		}

		if ch.Database == "" {
			ch.Database = "default"
		}

		if ch.Username == "" {
			ch.Username = "default"
		}
	}

	return nil
}

type StorageClickHouse struct {
	Address  string
	Database string
	Username string
	Password string
}
