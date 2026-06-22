package mysqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	localprovider "github.com/actionlab-ai/aisphere-hub/backend/internal/auth/providers/local"
)

func (s *Store) ListLocalAccounts(ctx context.Context) ([]*localprovider.Account, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT username,password_hash,subject_id,subject_type,display_name,email,organization,roles,permissions,namespaces,status,created_at,updated_at FROM iam_local_account ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*localprovider.Account{}
	for rows.Next() {
		var a localprovider.Account
		var displayName, email, org sql.NullString
		var rolesJSON, permsJSON, nsJSON sql.NullString
		var status string
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&a.Username, &a.PasswordHash, &a.SubjectID, &a.SubjectType, &displayName, &email, &org, &rolesJSON, &permsJSON, &nsJSON, &status, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		if displayName.Valid {
			a.DisplayName = displayName.String
		}
		if email.Valid {
			a.Email = email.String
		}
		if org.Valid {
			a.Organization = org.String
		}
		if rolesJSON.Valid && rolesJSON.String != "" {
			_ = json.Unmarshal([]byte(rolesJSON.String), &a.Roles)
		}
		if permsJSON.Valid && permsJSON.String != "" {
			_ = json.Unmarshal([]byte(permsJSON.String), &a.Permissions)
		}
		if nsJSON.Valid && nsJSON.String != "" {
			_ = json.Unmarshal([]byte(nsJSON.String), &a.Namespaces)
		}
		a.Disabled = status == "disabled"
		a.CreatedAt = createdAt
		a.UpdatedAt = updatedAt
		out = append(out, &a)
	}
	return out, rows.Err()
}

func (s *Store) SaveLocalAccount(ctx context.Context, acc *localprovider.Account) error {
	if acc == nil || acc.Username == "" {
		return nil
	}
	roles, _ := json.Marshal(acc.Roles)
	perms, _ := json.Marshal(acc.Permissions)
	ns, _ := json.Marshal(acc.Namespaces)
	status := "active"
	if acc.Disabled {
		status = "disabled"
	}
	now := time.Now()
	created := acc.CreatedAt
	if created.IsZero() {
		created = now
	}
	updated := acc.UpdatedAt
	if updated.IsZero() {
		updated = now
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO iam_local_account(username,subject_id,subject_type,display_name,email,organization,password_hash,roles,permissions,namespaces,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE subject_id=VALUES(subject_id),subject_type=VALUES(subject_type),display_name=VALUES(display_name),email=VALUES(email),organization=VALUES(organization),password_hash=VALUES(password_hash),roles=VALUES(roles),permissions=VALUES(permissions),namespaces=VALUES(namespaces),status=VALUES(status),updated_at=VALUES(updated_at)`, acc.Username, acc.SubjectID, acc.SubjectType, acc.DisplayName, acc.Email, acc.Organization, acc.PasswordHash, string(roles), string(perms), string(ns), status, created, updated)
	return err
}
