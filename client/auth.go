package client

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/titpetric/atkins/colors"
)

// MinPasswordLength mirrors what the server accepts, so a typo is
// caught at the prompt rather than after a round trip.
const MinPasswordLength = 8

// RunLogin drives `atkins --login https://domain`.
//
// It prompts for an email and password, exchanges them for tokens and
// writes them to ~/.atkins/credentials.json. From then on every atkins
// run in a git repository dispatches to this server.
func RunLogin(ctx context.Context, server string) error {
	server = NormalizeServer(server)
	if server == "" {
		return errors.New("a server URL is required, e.g. atkins --login https://ci.example.com")
	}
	if !strings.HasPrefix(server, "http://") && !strings.HasPrefix(server, "https://") {
		server = "https://" + server
	}

	prompt := NewPrompter()

	email, err := prompt.Line("Email", EnvEmail)
	if err != nil {
		return err
	}
	if email == "" {
		return errors.New("email is required")
	}

	password, err := prompt.Password("Password")
	if err != nil {
		return err
	}
	if password == "" {
		return errors.New("password is required")
	}

	hostname, _ := os.Hostname()

	credential, err := New(server).Login(ctx, LoginRequest{
		Email:    email,
		Password: password,
		Hostname: hostname,
	})
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	return report(credential, "Logged in")
}

// RunRegister drives `atkins --register https://domain`.
//
// Registration collects a username, an email and a password, and logs
// the new account in on success, so a fresh instance goes from nothing
// to a usable credential in one command.
func RunRegister(ctx context.Context, server string) error {
	server = NormalizeServer(server)
	if server == "" {
		return errors.New("a server URL is required, e.g. atkins --register https://ci.example.com")
	}
	if !strings.HasPrefix(server, "http://") && !strings.HasPrefix(server, "https://") {
		server = "https://" + server
	}

	prompt := NewPrompter()

	username, err := prompt.Line("Username", EnvUsername)
	if err != nil {
		return err
	}
	if username == "" {
		return errors.New("username is required")
	}

	email, err := prompt.Line("Email", EnvEmail)
	if err != nil {
		return err
	}
	if email == "" {
		return errors.New("email is required")
	}

	password, err := prompt.Password("Password")
	if err != nil {
		return err
	}
	if len(password) < MinPasswordLength {
		return fmt.Errorf("password must be at least %d characters", MinPasswordLength)
	}

	// Only confirm when we actually prompted; a scripted registration
	// driven by ATKINS_PASSWORD has nothing to mistype.
	if os.Getenv(EnvPassword) == "" {
		confirm, err := prompt.Password("Confirm password")
		if err != nil {
			return err
		}
		if confirm != password {
			return errors.New("passwords do not match")
		}
	}

	credential, err := New(server).Register(ctx, RegisterRequest{
		Username: username,
		Email:    email,
		Password: password,
	})
	if err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}

	return report(credential, "Registered")
}

// RunLogout drives `atkins --logout`. An empty server logs out of the
// default server.
func RunLogout(ctx context.Context, server string) error {
	c, err := Open(server)
	if err != nil {
		return err
	}

	target := c.Server()
	if err := c.Logout(ctx); err != nil {
		// The local credential is already gone; say so rather than
		// leaving the user unsure which side succeeded.
		fmt.Fprintf(os.Stderr, "%s local credential removed, server reported: %v\n", colors.BrightYellow("WARN:"), err)
		return nil
	}

	fmt.Fprintf(os.Stderr, "%s Logged out of %s\n", colors.BrightGreen("✓"), target)
	return nil
}

// report prints the outcome of a successful login or registration.
func report(credential *Credential, action string) error {
	path, err := CredentialsPath()
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "%s %s to %s as %s\n",
		colors.BrightGreen("✓"), action, credential.Server, colors.BrightCyan(credential.Username))
	fmt.Fprintf(os.Stderr, "  Credentials saved to %s\n", colors.Dim(path))

	return nil
}
