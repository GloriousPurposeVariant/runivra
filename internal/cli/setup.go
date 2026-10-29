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
	"strings"

	"github.com/GloriousPurposeVariant/runivra/internal/postgres"
	"github.com/GloriousPurposeVariant/runivra/internal/python"
	"github.com/GloriousPurposeVariant/runivra/internal/setup"
	"github.com/GloriousPurposeVariant/runivra/internal/tui"
)

type runOptions struct {
	installDocker   bool
	installPostgres bool
	installPython   bool
	createDBUser    bool
	dbAdminUser     string
	dbAdminPassword string

	start bool
	ask   bool
}

const setupUsage = `Create an Odoo project folder.

Usage:
  runivra setup                                   open the wizard
  runivra setup --env dev --version 20.0 [options]

Required:
  --env ENV                 dev, staging or prod
  --version VERSION         16.0, 17.0, 18.0, 19.0 or 20.0

Location:
  --path DIR                where to build (default: the current folder)
  --name NAME               create a new folder with this name inside the path

Sources:
  --enterprise-token TOKEN  clone Odoo Enterprise with a Git token
  --enterprise-path DIR     copy Odoo Enterprise from a local folder
  --custom-repo URL         clone your custom addons
  --custom-branch BRANCH    branch to clone (default: the Odoo version)
  --custom-token TOKEN      token for a private custom addons repository

Where it runs:
  --stack STACK             docker (default): Odoo and PostgreSQL in containers
                            host-db: Odoo in Docker, PostgreSQL on this computer
                            host: Odoo and PostgreSQL on this computer, no Docker
  --db PLACE                where PostgreSQL is, when not the stack's usual place:
                            docker, local (this computer) or remote (your own
                            server, on-premise, or a service such as AWS RDS or Azure)

Database:
  --db-host HOST            remote: the server's address
  --db-port PORT            remote: its port (default: 5432)
  --db-sslmode MODE         remote: disable, prefer, require or verify-full;
                            managed services usually need require
  --db-user NAME            existing database user for Odoo
  --db-password PASSWORD    its password (or set RUNIVRA_DB_PASSWORD)
  --install-postgres        local: install PostgreSQL without asking when it is missing
  --db-create               create the database user instead of expecting it
  --db-admin-user NAME      remote, with --db-create: the server's main user
  --db-admin-password PW    with --db-create: the administrator's password, needed
                            on Windows and for remote (or set RUNIVRA_DB_ADMIN_PASSWORD)

Running:
  --port PORT               port for Odoo on this computer (default: 8069)
  --start                   start Odoo afterwards without asking
  --install-docker          install Docker without asking when it is missing

Tokens can also be set as RUNIVRA_ENTERPRISE_TOKEN and RUNIVRA_CUSTOM_TOKEN,
which keeps them out of your shell history.
`

func runSetup(args []string) int {

	if len(args) == 0 {
		answers, ok, err := tui.Run(tui.Options{
			DockerMissing:      !setup.DetectDocker(context.Background()).Ready(),
			PostgresInstalled:  postgres.Detect().Installed(),
			NeedsAdminPassword: runtime.GOOS == "windows",
			CheckLogin:         checkLogin,
			CheckAdmin:         checkAdmin,
			RoleExists:         roleExists(),
			PythonMissing:      pythonMissing,
			PortIsFree:         setup.PortIsFree,
			Versions:           setup.SupportedVersions,
			CheckRemoteLogin:   checkRemoteLogin,
			CheckRemoteAdmin:   checkRemoteAdmin,
			RemoteRoleExists:   remoteRoleExists,
		})

		if err != nil {
			fmt.Println(paint(red, "runivra setup:"), err)
			return 1
		}
		if !ok {
			fmt.Println("Setup cancelled.")
			return 0
		}
		path := answers.Path
		if path == "" {
			path = "."
		}
		stack := setup.StackDocker
		if answers.HostOdoo {
			stack = setup.StackHost
		}

		return runPlan(setup.Request{
			Environment:     answers.Environment,
			Version:         answers.Version,
			Path:            path,
			Name:            answers.Name,
			EnterpriseToken: answers.EnterpriseToken,
			EnterprisePath:  answers.EnterprisePath,
			CustomRepo:      answers.CustomRepo,
			CustomBranch:    answers.CustomBranch,
			CustomToken:     answers.CustomToken,
			Port:            answers.Port,
			Stack:           stack,
			DB:              answers.DB,
			DBHost:          answers.DBHost,
			DBPort:          answers.DBPort,
			DBSSLMode:       answers.DBSSLMode,
			DBUser:          answers.DBUser,
			DBPassword:      answers.DBPassword,
		}, runOptions{installDocker: answers.InstallDocker, installPostgres: answers.InstallPostgres, installPython: answers.InstallPython,
			createDBUser: answers.CreateDBUser, dbAdminUser: answers.DBAdminUser, dbAdminPassword: answers.DBAdminPassword,
			start: answers.Start, ask: false})

	}

	var req setup.Request
	var installDocker bool
	var start bool
	var installPostgres bool
	var installPython bool
	var createDBUser bool
	var dbAdminUser string
	var dbAdminPassword string

	flags := flag.NewFlagSet("setup", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&req.Environment, "env", "", "dev, staging or production")
	flags.StringVar(&req.Version, "version", "", "Odoo version, for example 19.0")
	flags.StringVar(&req.Path, "path", ".", "project folder, created if it does not exist")
	flags.StringVar(&req.Name, "name", "", "create a folder with this name inside the path and build there")
	flags.StringVar(&req.EnterpriseToken, "enterprise-token", os.Getenv("RUNIVRA_ENTERPRISE_TOKEN"), "Git token used to clone Odoo Enterprise")
	flags.StringVar(&req.EnterprisePath, "enterprise-path", "", "copy Odoo Enterprise from this local folder")
	flags.StringVar(&req.CustomRepo, "custom-repo", "", "Git repository of your custom addons")
	flags.StringVar(&req.CustomBranch, "custom-branch", "", "branch of the custom addons (default: the Odoo version)")
	flags.StringVar(&req.CustomToken, "custom-token", os.Getenv("RUNIVRA_CUSTOM_TOKEN"), "Git token for a private custom addons repository")
	flags.BoolVar(&installDocker, "install-docker", false, "install Docker without asking when it is missing")
	flags.IntVar(&req.Port, "port", 8069, "port on this computer where Odoo will answer")
	flags.BoolVar(&start, "start", false, "start Odoo after setup without asking")
	flags.StringVar(&req.Stack, "stack", setup.StackDocker, "docker, host-db or host")
	flags.BoolVar(&installPython, "install-python", false, "install the needed Python without asking when it is missing")
	flags.StringVar(&req.DB, "db", "", "docker, local or remote")
	flags.StringVar(&req.DBHost, "db-host", "", "address of a remote database server")
	flags.IntVar(&req.DBPort, "db-port", 0, "port of a remote database server")
	flags.StringVar(&req.DBSSLMode, "db-sslmode", "", "SSL mode for a remote database server")
	flags.StringVar(&dbAdminUser, "db-admin-user", "", "main user of a remote database server")
	flags.StringVar(&req.DBUser, "db-user", "", "existing database user for Odoo")
	flags.StringVar(&req.DBPassword, "db-password", os.Getenv("RUNIVRA_DB_PASSWORD"), "password of that user")
	flags.BoolVar(&installPostgres, "install-postgres", false, "install PostgreSQL without asking when it is missing")
	flags.BoolVar(&createDBUser, "db-create", false, "create the database user in an existing PostgreSQL")
	flags.StringVar(&dbAdminPassword, "db-admin-password", os.Getenv("RUNIVRA_DB_ADMIN_PASSWORD"), "postgres administrator password")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Print(setupUsage)
			return 0
		}
		fmt.Println(paint(red, "runivra setup:"), err)
		fmt.Println("Run \"runivra setup --help\" to see the options.")
		return 2
	}

	return runPlan(req, runOptions{installDocker: installDocker, installPostgres: installPostgres, installPython: installPython,
		createDBUser: createDBUser, dbAdminUser: dbAdminUser, dbAdminPassword: dbAdminPassword, start: start, ask: true})

}

// plainLoginError turns PostgreSQL's message into one line for the wizard.
func plainLoginError(err error, who string) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "password authentication failed") {
		return errors.New("That password is not right for " + who + ".")
	}
	if errors.Is(err, postgres.ErrUnreachable) {
		return errors.New("PostgreSQL is not running on this computer.")
	}
	lines := strings.Split(strings.TrimSpace(err.Error()), "\n")
	return errors.New(lines[len(lines)-1])
}

// checkLogin and checkAdmin let the wizard test a login while the user is
// still on the question, instead of failing after the wizard has closed.
func checkLogin(user string, password string) error {
	err := postgres.Check(context.Background(), postgres.Local(user, password))
	return plainLoginError(err, "user "+user)
}

func checkAdmin(password string) error {
	err := postgres.LocalAdmin(postgres.Detect().Psql, password).Check(context.Background())
	return plainLoginError(err, "the postgres administrator")
}

// roleExists returns the check the wizard uses to see whether a database user
// name is taken. It returns nil on Linux, where asking the administrator
// would stop to prompt for the sudo password behind the wizard's screen.
func roleExists() func(adminPassword string, name string) (bool, error) {
	if runtime.GOOS == "linux" {
		return nil
	}
	return func(adminPassword string, name string) (bool, error) {
		return postgres.LocalAdmin(postgres.Detect().Psql, adminPassword).RoleExists(context.Background(), name)
	}
}

// pythonMissing tells the wizard whether the Python an Odoo version needs is
// absent, so it can ask about installing it.
func pythonMissing(odooVersion string) bool {
	version, err := python.VersionFor(odooVersion)
	if err != nil {
		return false
	}
	return python.Find(context.Background(), version) == ""
}

func runPlan(req setup.Request, options runOptions) int {
	problems := req.Problems()
	if len(problems) > 0 {
		fmt.Println("runivra setup: problems found:")
		for _, problem := range problems {
			fmt.Println("  ", problem)
		}
		return 2
	}

	if err := setup.CheckFolder(req.Folder()); err != nil {
		fmt.Println("runivra setup:", err)
		return 2
	}

	if err := setup.CheckSources(req); err != nil {
		fmt.Println(paint(red, "runivra setup:"), err)
		return 2
	}

	if req.IsDevelopment() && !setup.PortIsFree(req.Port) {
		fmt.Println(paint(red, "runivra setup:"), fmt.Sprintf("port %d is already in use; choose another with --port", req.Port))
		return 2
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if req.HostOdoo() {
		path, ok := ensurePython(ctx, req.Version, options.installPython, options.ask)
		if !ok {
			fmt.Println(paint(red, "runivra setup:"), "this stack needs that Python on this computer; nothing was created.")
			return 1
		}
		req.PythonPath = path
	} else {
		ensureDocker(ctx, options)
	}

	if req.HostDatabase() {
		login, ok := ensurePostgres(ctx, postgresRequest{
			install: options.installPostgres, ask: options.ask,
			user: req.DBUser, password: req.DBPassword,
			create: options.createDBUser, adminPassword: options.dbAdminPassword,
		})
		if !ok {
			fmt.Println(paint(red, "runivra setup:"), "this stack needs PostgreSQL on this computer; nothing was created.")
			return 1
		}
		req.DBUser, req.DBPassword, req.DBPort = login.User, login.Password, login.Port
		fmt.Println()
	}
	if req.DatabasePlace() == setup.DBRemote {
		var ok bool
		req, ok = ensureRemote(ctx, req, options)
		if !ok {
			fmt.Println(paint(red, "runivra setup:"), "the database server could not be used; nothing was created.")
			return 1
		}
		fmt.Println()
	}

	checklistRoot = req.Folder()

	steps := setup.BuildPlan(req)
	fmt.Println(paint(bold, "Runivra setup"), paint(dim, "·"), req.Environment, paint(dim, "·"), "Odoo", req.Version)
	fmt.Println(paint(dim, "Project folder:"), paint(cyan, req.Folder()))
	fmt.Println()
	if fancy {
		printChecklist(steps)
		fmt.Print(hideCursor)
		defer fmt.Print(showCursor)
	}

	later := 0
	var warnings []*setup.Warning
	for index, step := range steps {

		err := runStep(len(steps), index, step, func(report func(setup.Progress)) error {
			return setup.Apply(ctx, step, req, report)
		})

		if errors.Is(err, setup.ErrNotBuiltYet) {
			later++
			continue
		}
		var warning *setup.Warning
		if errors.As(err, &warning) {
			warnings = append(warnings, warning)
			continue
		}
		if err != nil {

			fmt.Println(paint(red, "runivra setup:"), err)
			return 1
		}
	}

	fmt.Print(showCursor)
	fmt.Println()
	if later > 0 {
		fmt.Println(paint(green, "Done."), paint(dim, "Steps marked – arrive in later milestones."))
		return 0
	}
	fmt.Println(paint(green, "Done."))
	for _, warning := range warnings {
		fmt.Println()
		fmt.Println(paint(yellow, "! "+warning.Summary))
		for _, detail := range warning.Details {
			fmt.Println(paint(dim, "    "+detail))
		}
	}
	if req.HostOdoo() {
		return startHostOdoo(ctx, req, options)
	}
	if req.IsDevelopment() {
		return startOdoo(ctx, req, options)
	}
	return 0
}
