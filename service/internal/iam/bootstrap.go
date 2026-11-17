package iam

import (
	"context"
	"fmt"
	"os"

	armpassword "github.com/jamesread/armature-iam/password"
	"github.com/jamesread/armature-iam/store"
	log "github.com/sirupsen/logrus"
)

// DefaultAdminUsername and DefaultAdminPassword are created on an empty database.
const (
	DefaultAdminUsername = "admin"
	DefaultAdminPassword = "admin"
)

// MinPasswordLength applies to every password set through the API.
const MinPasswordLength = 8

// Bootstrap creates the default admin on an empty database, optionally resets its password
// (STENCILBOX_RESET_ADMIN_PASSWORD), and repairs system groups and memberships.
func Bootstrap(ctx context.Context, st store.Store) error {
	adminID, err := ensureDefaultAdmin(ctx, st)
	if err != nil {
		return err
	}
	if err = resetAdminPasswordIfRequested(ctx, st); err != nil {
		return err
	}
	// Puts the first user in Administrators, and users with no groups in Everyone.
	if err = st.EnsureRBACBootstrap(ctx); err != nil {
		return fmt.Errorf("rbac bootstrap: %w", err)
	}
	return ensureAdminInEveryone(ctx, st, adminID)
}

func resetAdminPasswordIfRequested(ctx context.Context, st store.Store) error {
	if os.Getenv("STENCILBOX_RESET_ADMIN_PASSWORD") == "" {
		return nil
	}
	return resetAdminPassword(ctx, st)
}

func resetAdminPassword(ctx context.Context, st store.Store) error {
	user, err := st.GetUserByUsername(ctx, DefaultAdminUsername)
	if err != nil {
		return err
	}
	if user == nil {
		log.Warnf("STENCILBOX_RESET_ADMIN_PASSWORD is set, but there is no %q user", DefaultAdminUsername)
		return nil
	}
	if err = setDefaultAdminPassword(ctx, st, user.ID); err != nil {
		return err
	}
	log.Warnf("STENCILBOX_RESET_ADMIN_PASSWORD is set: %q password reset to %q and added to Administrators. Unset it and restart.", DefaultAdminUsername, DefaultAdminPassword)
	return nil
}

func setDefaultAdminPassword(ctx context.Context, st store.Store, userID int) error {
	hash, err := armpassword.Hash(DefaultAdminPassword)
	if err != nil {
		return err
	}
	if err = st.UpdateUserPassword(ctx, userID, hash); err != nil {
		return fmt.Errorf("reset admin password: %w", err)
	}
	return st.EnsureUserInGroup(ctx, userID, "Administrators")
}

// ensureAdminInEveryone adds a newly created default admin to Everyone: it is in Administrators
// by now, so the bootstrap skipped Everyone. An adminID of 0 means no admin was created.
func ensureAdminInEveryone(ctx context.Context, st store.Store, adminID int) error {
	if adminID <= 0 {
		return nil
	}
	if err := st.EnsureUserInEveryoneGroup(ctx, adminID); err != nil {
		return fmt.Errorf("add default admin to Everyone: %w", err)
	}
	return nil
}

// ensureDefaultAdmin creates the default admin on an empty database and returns its ID, or 0
// when users already exist.
func ensureDefaultAdmin(ctx context.Context, st store.Store) (int, error) {
	count, err := st.CountUserAccounts(ctx)
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	if count > 0 {
		return 0, nil
	}
	hash, err := armpassword.Hash(DefaultAdminPassword)
	if err != nil {
		return 0, err
	}
	id, err := st.CreateUserAccount(ctx, DefaultAdminUsername, hash, store.UserCreatedByAdmin)
	if err != nil {
		return 0, fmt.Errorf("create default admin: %w", err)
	}
	log.Warnf("No users found: created %q with password %q. Log in and change it now.", DefaultAdminUsername, DefaultAdminPassword)
	return id, nil
}

// UsesDefaultAdminPassword reports whether the default admin still has the default password.
func UsesDefaultAdminPassword(ctx context.Context, st store.Store) bool {
	user, err := st.GetUserByUsername(ctx, DefaultAdminUsername)
	if err != nil || user == nil || user.PasswordHash == "" {
		return false
	}
	ok, err := armpassword.Verify(user.PasswordHash, DefaultAdminPassword)
	return err == nil && ok
}
