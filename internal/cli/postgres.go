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
  --create             create that user instead of expecting it to exist
  --admin-password PW  Windows only, with --create: the password of the postgres
                       administrator; RUNIVRA_DB_ADMIN_PASSWORD is read when not given
`

// databaseLogin is what Odoo needs to connect to PostgreSQL on this computer.
type databaseLogin struct {
	User     string
	Password string
	Port     int
}

// postgresRequest is everything ensurePostgres may need to know.
type postgresRequest struct {
	install       bool   // install PostgreSQL without asking when it is missing
	ask           bool   // questions may be asked on the terminal
	user          string // the database user for Odoo
	password      string // its password; empty means "generate" when the user is created
	create        bool   // create the user in an existing PostgreSQL
	adminPassword string // Windows only: the postgres administrator's password
}

func runPostgres(args []string) int {
	flags := flag.NewFlagSet("postgres", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	request := postgresRequest{ask: true}
	flags.BoolVar(&request.install, "install", false, "")
	flags.StringVar(&request.user, "user", "", "")
	flags.StringVar(&request.password, "password", os.Getenv("RUNIVRA_DB_PASSWORD"), "")
	flags.BoolVar(&request.create, "create", false, "")
	flags.StringVar(&request.adminPassword, "admin-password", os.Getenv("RUNIVRA_DB_ADMIN_PASSWORD"), "")
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

	_, ok := ensurePostgres(ctx, request)
	if !ok {
		return 1
	}
	return 0
}

// ensurePostgres makes sure a PostgreSQL server on this computer can be used
// by Odoo, and returns the login to use. It returns false when it cannot.
func ensurePostgres(ctx context.Context, request postgresRequest) (databaseLogin, bool) {
	install, ask, user, password := request.install, request.ask, request.user, request.password
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

	create := request.create
	if !create && user == "" && ask && fancy {
		create = !askYes("Does Odoo already have a database user in this PostgreSQL?")
	}
	if create {
		return createLogin(ctx, status, request)
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
		fmt.Println(paint(red, "That login cannot be used:"), err)
		fmt.Println(paint(dim, "To have Runivra create a new user instead, run this again with --create."))
		return databaseLogin{}, false
	}
	fmt.Println(paint(green, "Login works for user"), paint(cyan, user))
	return databaseLogin{User: user, Password: password, Port: postgres.Port}, true
}

// createLogin adds a database user for Odoo to a PostgreSQL that is already
// installed. That needs the administrator once.
func createLogin(ctx context.Context, status postgres.Status, request postgresRequest) (databaseLogin, bool) {
	login := databaseLogin{User: request.user, Password: request.password, Port: postgres.Port}
	if login.User == "" && request.ask && fancy {
		login.User = askText("Name for the new user", postgres.Role)
	}
	if login.User == "" {
		login.User = postgres.Role
	}
	generated := login.Password == ""
	if generated {
		login.Password = postgres.NewPassword()
	}

	adminPassword := request.adminPassword
	if runtime.GOOS == "windows" && adminPassword == "" {
		if request.ask && fancy {
			fmt.Println(paint(dim, "Creating a user needs the password of the postgres administrator, set when PostgreSQL was installed."))
			adminPassword = askText("Administrator password", "")
		}
		if adminPassword == "" {
			fmt.Println(paint(red, "runivra postgres:"), "the postgres administrator password is needed to create a user; give --admin-password or set RUNIVRA_DB_ADMIN_PASSWORD")
			return databaseLogin{}, false
		}
	}

	exists, err := postgres.RoleExists(ctx, status.Psql, adminPassword, login.User)
	if err != nil {
		fmt.Println(paint(red, "The existing database users could not be checked:"), err)
		return databaseLogin{}, false
	}
	if exists {
		fmt.Println(paint(red, "A database user named"), paint(cyan, login.User), paint(red, "already exists, so nothing was created."))
		fmt.Println(paint(dim, "Choose another name, or use that user with its own password."))
		return databaseLogin{}, false
	}
	if err := postgres.CreateRole(ctx, status.Psql, adminPassword, login.User, login.Password); err != nil {
		fmt.Println(paint(red, "The database user could not be created:"), err)
		return databaseLogin{}, false
	}

	if err := postgres.Check(ctx, status.Psql, login.User, login.Password); err != nil {
		fmt.Println(paint(red, "The new database user cannot log in:"), err)
		return databaseLogin{}, false
	}

	fmt.Println(paint(green, "Created database user"), paint(cyan, login.User))
	if generated {
		fmt.Println("  " + paint(dim, "Its password") + " " + login.Password)
		fmt.Println(paint(dim, "  Keep it somewhere safe. It is also written to the project's odoo.conf."))
	}
	return login, true
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
