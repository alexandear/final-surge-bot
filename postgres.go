package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
)

type Postgres struct {
	conn *pgx.Conn
}

func NewPostgres(conn *pgx.Conn) (*Postgres, error) {
	p := &Postgres{
		conn: conn,
	}

	if _, err := p.conn.Exec(context.Background(), `
CREATE TABLE IF NOT EXISTS user_tokens (
    user_name char(40) primary key,
    user_key char(40) not null,
    token char(40) not null
);`); err != nil {
		return nil, fmt.Errorf("failed to create table user_tokens")
	}

	return p, nil
}

func (p *Postgres) UserToken(ctx context.Context, userName string) (UserToken, error) {
	var userToken UserToken

	rows, err := p.conn.Query(ctx, `SELECT user_key, token FROM user_tokens WHERE user_name=$1`, userName)
	if err != nil {
		return UserToken{}, fmt.Errorf("failed to query: %w", err)
	}

	for rows.Next() {
		if errScan := rows.Scan(&userToken.UserKey, &userToken.Token); errScan != nil {
			return UserToken{}, fmt.Errorf("failed during scan: %w", errScan)
		}
	}

	if rows.Err() != nil {
		return UserToken{}, fmt.Errorf("failed rows: %w", err)
	}

	return userToken, nil
}

func (p *Postgres) UpdateUserToken(ctx context.Context, userName string, userToken UserToken) error {
	if _, err := p.conn.Exec(ctx, `
INSERT INTO user_tokens(user_name, user_key, token) VALUES ($1, $2, $3) ON CONFLICT (user_name)
	DO UPDATE SET user_key=excluded.user_key, token=excluded.token`,
		userName, userToken.UserKey, userToken.Token); err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}

	return nil
}