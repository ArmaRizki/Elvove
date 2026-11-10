package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct{ Pool *pgxpool.Pool }

func Open(ctx context.Context, url string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}
	cfg.MaxConns = 8
	cfg.MinConns = 1
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute
	// Java runs with TimeZone.setDefault(Asia/Jakarta), so every timestamp it
	// writes is WIB wall clock. SQL NOW() must produce the same value, or a
	// column filled by NOW() lands seven hours before one Java wrote.
	if cfg.ConnConfig.RuntimeParams == nil {
		cfg.ConnConfig.RuntimeParams = map[string]string{}
	}
	cfg.ConnConfig.RuntimeParams["timezone"] = "Asia/Jakarta"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open database pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &Store{Pool: pool}, nil
}

func (s *Store) Close() { s.Pool.Close() }

func (s *Store) QueryRows(ctx context.Context, query string, args ...any) ([]map[string]any, error) {
	rows, err := s.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	fields := rows.FieldDescriptions()
	result := make([]map[string]any, 0)
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}
		item := make(map[string]any, len(fields))
		for i, field := range fields {
			item[field.Name] = normalize(values[i])
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Store) QueryOne(ctx context.Context, query string, args ...any) (map[string]any, error) {
	rows, err := s.QueryRows(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, pgx.ErrNoRows
	}
	return rows[0], nil
}

func (s *Store) Exec(ctx context.Context, query string, args ...any) (int64, error) {
	tag, err := s.Pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// jakartaZone mirrors Java's TimeZone.setDefault(Asia/Jakarta): every timestamp
// the API emits is WIB wall-clock with no offset.
var jakartaZone = time.FixedZone("Asia/Jakarta", 7*60*60)

// Now returns the current time prepared for a `timestamp without time zone`
// column; see WallClock.
func Now() time.Time { return WallClock(time.Now()) }

// WallClock re-tags a moment as WIB while keeping its wall-clock fields.
//
// The datetime columns are `timestamp without time zone` and pgx renders a
// time.Time in the session timezone (UTC here), so passing the raw instant would
// store the server's own timezone — UTC inside the container image — while Java
// stores WIB because it calls TimeZone.setDefault(Asia/Jakarta). Re-tagging the
// WIB fields as UTC makes pgx write "2026-11-10 13:59" verbatim, and reading the
// row back yields those same WIB fields.
func WallClock(t time.Time) time.Time {
	wib := t.In(jakartaZone)
	return time.Date(wib.Year(), wib.Month(), wib.Day(), wib.Hour(), wib.Minute(), wib.Second(), wib.Nanosecond(), time.UTC)
}

func (s *Store) Exists(ctx context.Context, query string, args ...any) (bool, error) {
	var exists bool
	err := s.Pool.QueryRow(ctx, query, args...).Scan(&exists)
	return exists, err
}

func normalize(value any) any {
	switch v := value.(type) {
	case time.Time:
		// The column is `timestamp without time zone` holding WIB wall-clock (see
		// Now) and pgx returns it tagged UTC. Do NOT convert the zone here: that
		// adds another +7 and renders 13:54 as 20:54. A date column arrives as
		// midnight, which is how Java's LocalDate ("2026-12-10") is told apart.
		if v.Hour() == 0 && v.Minute() == 0 && v.Second() == 0 && v.Nanosecond() == 0 {
			return v.Format("2006-01-02")
		}
		return v.Format("2006-01-02T15:04:05.999999999")
	case []byte:
		return string(v)
	default:
		return v
	}
}

func JSONValue(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage("null")
	}
	return encoded
}

func IsNotFound(err error) bool { return errors.Is(err, pgx.ErrNoRows) }
