package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/GloriousPurposeVariant/runivra/internal/postgres"
	"github.com/GloriousPurposeVariant/runivra/internal/setup"
	"github.com/GloriousPurposeVariant/runivra/internal/tui"
)

// remoteServer is the database server of a request, logged in to as user.
func remoteServer(req setup.Request, user string, password string) postgres.Server {
	port := req.DBPort
	if port == 0 {
		port = postgres.Port
	}
	return postgres.Server{Host: req.DBHost, Port: port, User: user, Password: password, SSLMode: req.DBSSLMode}
}

// explainRemote prints why a remote server could not be used, with a hint
// when it did not answer at all.
func explainRemote(message string, err error) {
	fmt.Println(paint(red, message), err)
	if errors.Is(err, postgres.ErrUnreachable) {
		fmt.Println(paint(dim, "Check the address and port, and that the server accepts connections from this computer:"))
		fmt.Println(paint(dim, "on AWS the security group, on Azure the firewall rules, on your own server pg_hba.conf."))
	}
}

// ensureRemote checks, or creates, the login Odoo will use on a database
// server elsewhere. Nothing is installed: the server is someone else's.
func ensureRemote(ctx context.Context, req setup.Request, options runOptions) (setup.Request, bool) {
	fmt.Println(paint(dim, "Database server:"), paint(cyan, remoteServer(req, "", "").Address()))

	create := options.createDBUser
	if !create && req.DBUser == "" && options.ask && fancy {
		create = !askYes("Does Odoo already have a database user on this server?")
	}
	if create {
		return createRemoteLogin(ctx, req, options)
	}

	if req.DBUser == "" && options.ask && fancy {
		fmt.Println(paint(dim, "Give the database user Odoo should connect as. It must be allowed to create databases."))
		req.DBUser = askText("User", postgres.Role)
	}
	if req.DBUser == "" {
		req.DBUser = postgres.Role
	}
	if req.DBPassword == "" && options.ask && fancy {
		req.DBPassword = askText("Password", "")
	}
	if req.DBPassword == "" {
		fmt.Println(paint(red, "runivra setup:"), "a password is needed for user", req.DBUser+"; give --db-password or set RUNIVRA_DB_PASSWORD")
		return req, false
	}
	if err := postgres.Check(ctx, remoteServer(req, req.DBUser, req.DBPassword)); err != nil {
		explainRemote("That login cannot be used:", err)
		return req, false
	}
	fmt.Println(paint(green, "Login works for user"), paint(cyan, req.DBUser))
	return req, true
}

// createRemoteLogin adds Odoo's user through the server's main user, the one
// a provider such as AWS or Azure gives you when the server is created.
func createRemoteLogin(ctx context.Context, req setup.Request, options runOptions) (setup.Request, bool) {
	adminUser, adminPassword := options.dbAdminUser, options.dbAdminPassword
	if options.ask && fancy {
		if adminUser == "" {
			fmt.Println(paint(dim, "Creating a user needs the server's main user, the one set when the server was created."))
			adminUser = askText("Main user", "postgres")
		}
		if adminPassword == "" {
			adminPassword = askText("Its password", "")
		}
	}
	if adminUser == "" || adminPassword == "" {
		fmt.Println(paint(red, "runivra setup:"), "creating a user needs the server's main user; give --db-admin-user and --db-admin-password")
		return req, false
	}
	admin := postgres.Admin{Server: remoteServer(req, adminUser, adminPassword)}

	if req.DBUser == "" && options.ask && fancy {
		req.DBUser = askText("Name for the new user", postgres.Role)
	}
	if req.DBUser == "" {
		req.DBUser = postgres.Role
	}
	generated := req.DBPassword == ""
	if generated {
		req.DBPassword = postgres.NewPassword()
	}

	exists, err := admin.RoleExists(ctx, req.DBUser)
	if err != nil {
		explainRemote("The server could not be used as "+adminUser+":", err)
		return req, false
	}
	if exists {
		fmt.Println(paint(red, "A database user named"), paint(cyan, req.DBUser), paint(red, "already exists, so nothing was created."))
		fmt.Println(paint(dim, "Choose another name, or use that user with its own password."))
		return req, false
	}
	if err := admin.CreateRole(ctx, req.DBUser, req.DBPassword); err != nil {
		explainRemote("The database user could not be created:", err)
		return req, false
	}
	if err := postgres.Check(ctx, remoteServer(req, req.DBUser, req.DBPassword)); err != nil {
		explainRemote("The new database user cannot log in:", err)
		return req, false
	}

	fmt.Println(paint(green, "Created database user"), paint(cyan, req.DBUser))
	if generated {
		fmt.Println("  " + paint(dim, "Its password") + " " + req.DBPassword)
		fmt.Println(paint(dim, "  Keep it somewhere safe. It is also written to the project's odoo.conf."))
	}
	return req, true
}

// wizardServer is the server the wizard describes, logged in to as user.
func wizardServer(remote tui.Remote, user string, password string) postgres.Server {
	return postgres.Server{Host: remote.Host, Port: remote.Port, User: user, Password: password, SSLMode: remote.SSLMode}
}

// plainRemoteError explains a server that did not answer in one line for the
// wizard. PostgreSQL's own messages are already short.
func plainRemoteError(err error, server postgres.Server) error {
	if errors.Is(err, postgres.ErrUnreachable) {
		return errors.New("Nothing answered at " + server.Address() + ". Check the address and port, and that the server accepts connections from this computer.")
	}
	return err
}

// checkRemoteLogin, checkRemoteAdmin and remoteRoleExists let the wizard test
// a remote server while the user is still on the question.
func checkRemoteLogin(remote tui.Remote, user string, password string) error {
	server := wizardServer(remote, user, password)
	return plainRemoteError(postgres.Check(context.Background(), server), server)
}

func checkRemoteAdmin(remote tui.Remote, user string, password string) error {
	server := wizardServer(remote, user, password)
	return plainRemoteError(postgres.Admin{Server: server}.Check(context.Background()), server)
}

func remoteRoleExists(remote tui.Remote, adminUser string, adminPassword string, name string) (bool, error) {
	admin := postgres.Admin{Server: wizardServer(remote, adminUser, adminPassword)}
	return admin.RoleExists(context.Background(), name)
}
