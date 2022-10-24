package bot

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
)

type Postgres struct {
	dbPool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{dbPool: pool}
}

func (p *Postgres) Init(ctx context.Context) error {
	if _, err := p.dbPool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS user_tokens (
    user_name char(40) primary key,
    user_key char(40) not null,
    token char(40) not null
);`); err != nil {
		return fmt.Errorf("failed to create table user_tokens")
	}

	return nil
}

func (p *Postgres) UserToken(ctx context.Context, userName string) (UserToken, error) {
	var userToken UserToken

	rows, err := p.dbPool.Query(ctx, `SELECT user_key, token FROM user_tokens WHERE user_name=$1`, userName)
	if err != nil {
		return UserToken{}, fmt.Errorf("failed to query: %w", err)
	}

	for rows.Next() {
		if errScan := rows.Scan(&userToken.UserKey, &userToken.Token); errScan != nil {
			return UserToken{}, fmt.Errorf("failed during scan: %w", errScan)
		}
	}

	if rows.Err() != nil {
		return UserToken{}, fmt.Errorf("failed rows: %w", rows.Err())
	}

	if userToken.UserKey == "" {
		return UserToken{}, ErrNotFound
	}

	return userToken, nil
}

func (p *Postgres) UpdateUserToken(ctx context.Context, userName string, userToken UserToken) error {
	if _, err := p.dbPool.Exec(ctx, `
INSERT INTO user_tokens(user_name, user_key, token) VALUES ($1, $2, $3) ON CONFLICT (user_name)
	DO UPDATE SET user_key=excluded.user_key, token=excluded.token`,
		userName, userToken.UserKey, userToken.Token); err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}

	return nil
}
