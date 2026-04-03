package doctor

import (
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"
)

type dbTarget struct {
	driver  string
	dialect string
	dsn     string
}

func detectDBTarget(conn string) dbTarget {
	v := strings.TrimSpace(conn)
	switch {
	case strings.HasPrefix(v, "postgres://"), strings.HasPrefix(v, "postgresql://"):
		return dbTarget{driver: "pgx", dialect: "postgres", dsn: v}
	case strings.HasPrefix(v, "mysql://"):
		return dbTarget{driver: "mysql", dialect: "mysql", dsn: strings.TrimPrefix(v, "mysql://")}
	case strings.HasPrefix(v, "mysql:"):
		return dbTarget{driver: "mysql", dialect: "mysql", dsn: strings.TrimPrefix(v, "mysql:")}
	default:
		return dbTarget{driver: "sqlite", dialect: "sqlite", dsn: v}
	}
}

func validateConnectionString(t dbTarget) error {
	switch t.dialect {
	case "postgres":
		_, err := pgconn.ParseConfig(t.dsn)
		if err != nil {
			return fmt.Errorf("invalid Postgres DSN: %w", err)
		}
		return nil
	case "mysql":
		_, err := mysql.ParseDSN(t.dsn)
		if err != nil {
			return fmt.Errorf("invalid MySQL DSN: %w", err)
		}
		return nil
	default:
		if strings.TrimSpace(t.dsn) == "" {
			return fmt.Errorf("sqlite path is empty")
		}
		return nil
	}
}
