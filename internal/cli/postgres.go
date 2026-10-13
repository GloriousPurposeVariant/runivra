package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"

	"github.com/GloriousPurposeVariant/runivra/internal/postgres"
)

const postgresUsage = `Check, install or prepare PostgreSQL on this computer, outside Docker.

Usage:
  runivra postgres [options]

When PostgreSQL is missing, this offers to install it and creates a database
user for Odoo. When it is already installed, it asks for an existing user and
password and checks that they work.

Options:
  --install            install without asking when PostgreSQL is missing
  --user NAME          existing database user to check (default: odoo)
  --password PASSWORD  its password; RUNIVRA_DB_PASSWORD is read when this is not given
`

// databaseLogin is what Odoo needs to connect to PostgreSQL on this computer.
type databaseLogin struct {
	User     string
	Password string
	Port     int
}

func runPostgres(args []string) int {
	flags := flag.NewFlagSet("postgres", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var install bool
	var user, password string
	flags.BoolVar(&install, "install", false, "")
	flags.StringVar(&user, "user", "", "")
	flags.StringVar(&password, "password", os.Getenv("RUNIVRA_DB_PASSWORD"), "")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Print(postgresUsage)
			return 0
		}
		fmt.Println(paint(red, "runivra postgres:"), err)
		return 2
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	_, ok := ensurePostgres(ctx, install, true, user, password)
	if !ok {
		return 1
	}
	return 0
}

// ensurePostgres makes sure a PostgreSQL server on this computer can be used
// by Odoo, and returns the login to use. It returns false when it cannot.
func ensurePostgres(ctx context.Context, install bool, ask bool, user string, password string) (databaseLogin, bool) {
	status := postgres.Detect()

	if !status.Installed() {
		fmt.Println(paint(yellow, "PostgreSQL is not installed on this computer."))
		if !install && ask && fancy {
			install = askYes("Install PostgreSQL " + postgres.Version + " now?")
		}
		if !install {
			fmt.Println(paint(dim, "Nothing was installed. Run \"runivra postgres --install\" when you are ready."))
			return databaseLogin{}, false
		}
		return installPostgres(ctx, user, password)
	}

	fmt.Println(paint(green, "PostgreSQL is installed:"), paint(cyan, status.Psql))
	if !status.Running {
		fmt.Println(paint(yellow, fmt.Sprintf("It is not accepting connections on port %d. Start the PostgreSQL service and run this again.", postgres.Port)))
		return databaseLogin{}, false
	}

	if user == "" && ask && fancy {
		fmt.Println(paint(dim, "Give the database user Odoo should connect as. It must be allowed to create databases."))
		user = askText("User", postgres.Role)
	}
	if user == "" {
		user = postgres.Role
	}
	if password == "" && ask && fancy {
		password = askText("Password", "")
	}
	if password == "" {
		fmt.Println(paint(red, "runivra postgres:"), "a password is needed for user", user+"; give --password or set RUNIVRA_DB_PASSWORD")
		return databaseLogin{}, false
	}
	if err := postgres.Check(ctx, status.Psql, user, password); err != nil {
		fmt.Println(paint(red, "That login did not work:"), err)
		return databaseLogin{}, false
	}
	fmt.Println(paint(green, "Login works for user"), paint(cyan, user))
	return databaseLogin{User: user, Password: password, Port: postgres.Port}, true
}

func installPostgres(ctx context.Context, user string, password string) (databaseLogin, bool) {
	adminPassword := postgres.NewPassword()
	fmt.Println(paint(dim, "Installing PostgreSQL. You may be asked for your password or for permission."))
	fmt.Println()
	if err := postgres.Install(ctx, os.Stdin, os.Stdout, adminPassword); err != nil {
		fmt.Println(paint(red, "PostgreSQL was not installed:"), err)
		return databaseLogin{}, false
	}

	status := postgres.Detect()
	if !status.Installed() {
		fmt.Println(paint(red, "runivra postgres:"), "the install finished but psql could not be found; open a new terminal and run this again")
		return databaseLogin{}, false
	}

	login := databaseLogin{User: user, Password: password, Port: postgres.Port}
	if login.User == "" {
		login.User = postgres.Role
	}
	chosen := login.Password != ""
	if !chosen {
		login.Password = postgres.NewPassword()
	}
	if err := postgres.CreateRole(ctx, status.Psql, adminPassword, login.User, login.Password); err != nil {
		fmt.Println(paint(red, "The database user could not be created:"), err)
		return databaseLogin{}, false
	}
	if err := postgres.Check(ctx, status.Psql, login.User, login.Password); err != nil {
		fmt.Println(paint(red, "The new database user cannot log in:"), err)
		return databaseLogin{}, false
	}

	fmt.Println()
	fmt.Println(paint(green, "PostgreSQL is installed and ready."))
	row := func(label string, value string) {
		fmt.Println("  " + paint(dim, fmt.Sprintf("%-24s", label)) + " " + value)
	}
	row("Port", fmt.Sprint(login.Port))
	row("Database user for Odoo", login.User)
	if chosen {
		row("Its password", paint(dim, "the one you gave"))
	} else {
		row("Its password", login.Password)
	}

	if runtime.GOOS == "windows" {
		row("Administrator (postgres)", adminPassword)
	}
	fmt.Println(paint(dim, "  Keep these somewhere safe. They are not stored by Runivra and will not be shown again."))

	changed, err := postgres.AddToPath(ctx, status.Psql)
	switch {
	case err != nil:
		fmt.Println(paint(yellow, "  psql could not be added to your PATH: "+err.Error()))
		fmt.Println(paint(dim, "  You can still run it by its full location: "+status.Psql))
	case changed:
		fmt.Println(paint(dim, "  psql was added to your PATH. Open a new terminal, then connect with:"))
		fmt.Println("    psql -U " + login.User + " -h localhost -d postgres")
	}

	return login, true
}
