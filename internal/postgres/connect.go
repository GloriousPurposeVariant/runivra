package postgres

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrUnreachable means nothing answered at the server's address.
var ErrUnreachable = errors.New("PostgreSQL is not answering")

// Server is a PostgreSQL server and the login used to reach it. It can be on
// this computer, on another one, or a service such as AWS RDS or Azure.
type Server struct {
	Host     string
	Port     int
	User     string
	Password string
	SSLMode  string // disable, prefer, require, verify-full; "" means prefer
}

// Local is PostgreSQL on this computer, logged in to as user.
func Local(user string, password string) Server {
	return Server{Host: "127.0.0.1", Port: Port, User: user, Password: password}
}

func (s Server) Address() string {
	return net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
}

// url builds the address the driver connects to. url.UserPassword escapes
// characters such as @ and : that a password may contain.
func (s Server) url() string {
	query := url.Values{"connect_timeout": {"10"}}
	if s.SSLMode != "" {
		query.Set("sslmode", s.SSLMode)
	}
	address := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(s.User, s.Password),
		Host:     s.Address(),
		Path:     "/postgres",
		RawQuery: query.Encode(),
	}
	return address.String()
}

// explain turns the driver's error into PostgreSQL's own message, or
// ErrUnreachable when the server could not be reached at all.
func (s Server) explain(err error) error {
	var serverErr *pgconn.PgError
	if errors.As(err, &serverErr) {
		return errors.New(serverErr.Message)
	}
	var networkErr *net.OpError
	if errors.As(err, &networkErr) {
		return fmt.Errorf("%w at %s", ErrUnreachable, s.Address())
	}
	return err
}

// Strings runs a query that returns one column and gives back its values.
func (s Server) Strings(ctx context.Context, sql string, args ...any) ([]string, error) {
	conn, err := pgx.Connect(ctx, s.url())
	if err != nil {
		return nil, s.explain(err)
	}
	defer conn.Close(ctx)

	rows, err := conn.Query(ctx, sql, args...)
	if err != nil {
		return nil, s.explain(err)
	}
	values, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, s.explain(err)
	}
	return values, nil
}

// Exec runs a statement that returns nothing.
func (s Server) Exec(ctx context.Context, sql string) error {
	conn, err := pgx.Connect(ctx, s.url())
	if err != nil {
		return s.explain(err)
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx, sql); err != nil {
		return s.explain(err)
	}
	return nil
}
