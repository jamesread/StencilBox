package iam

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jamesread/armature-iam/rbac"
	"github.com/jamesread/armature-iam/store"
)

// SetUserGroups replaces every group membership of one user in a single transaction. The
// Everyone group is always kept, and the change is refused if no user would keep superuser.
func SetUserGroups(ctx context.Context, db *sql.DB, userID int, groupIDs []int) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("set user groups begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err = replaceMemberships(ctx, tx, userID, groupIDs); err != nil {
		return err
	}
	if err = requireSuperuser(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

func replaceMemberships(ctx context.Context, tx *sql.Tx, userID int, groupIDs []int) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM user_group_memberships WHERE user_account_id = ?`, userID); err != nil {
		return fmt.Errorf("clear user groups: %w", err)
	}
	for _, gid := range groupIDs {
		if _, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO user_group_memberships (user_account_id, user_group_id) VALUES (?, ?)`,
			userID, gid); err != nil {
			return fmt.Errorf("add user to group %d: %w", gid, err)
		}
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO user_group_memberships (user_account_id, user_group_id)
		 SELECT ?, id FROM user_groups WHERE name = ?`, userID, rbac.GroupEveryone); err != nil {
		return fmt.Errorf("keep user in %s: %w", rbac.GroupEveryone, err)
	}
	return nil
}

// requireSuperuser refuses a change that would leave no user with the superuser role.
func requireSuperuser(ctx context.Context, tx *sql.Tx) error {
	var supers int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT ugm.user_account_id)
		 FROM user_group_memberships ugm
		 INNER JOIN rbac_group_roles gr ON gr.user_group_id = ugm.user_group_id
		 INNER JOIN rbac_roles r ON r.id = gr.role_id
		 WHERE r.name = ?`, rbac.RoleSuperuser).Scan(&supers); err != nil {
		return fmt.Errorf("count superusers: %w", err)
	}
	if supers == 0 {
		return store.ErrNoSuperuser
	}
	return nil
}
