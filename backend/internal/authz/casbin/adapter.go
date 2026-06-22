package casbin

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"

	casbinmodel "github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	_ "github.com/go-sql-driver/mysql"
)

// MySQLAdapter stores Casbin p/g rules in aihub_casbin_rule.
type MySQLAdapter struct {
	dsn string
	db  *sql.DB
}

func NewMySQLAdapter(dsn string) (*MySQLAdapter, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &MySQLAdapter{dsn: dsn, db: db}, nil
}

func (a *MySQLAdapter) Close() error { return a.db.Close() }

func (a *MySQLAdapter) LoadPolicy(model casbinmodel.Model) error {
	rows, err := a.db.QueryContext(context.Background(), `SELECT ptype,v0,v1,v2,v3,v4,v5 FROM aihub_casbin_rule ORDER BY id ASC`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var ptype string
		vals := make([]sql.NullString, 6)
		if err := rows.Scan(&ptype, &vals[0], &vals[1], &vals[2], &vals[3], &vals[4], &vals[5]); err != nil {
			return err
		}
		line := ptype
		for _, v := range vals {
			if v.Valid && strings.TrimSpace(v.String) != "" {
				line += ", " + v.String
			}
		}
		if err := persist.LoadPolicyLine(line, model); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (a *MySQLAdapter) SavePolicy(model casbinmodel.Model) error {
	tx, err := a.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(context.Background(), `DELETE FROM aihub_casbin_rule`); err != nil {
		return err
	}
	for sec, ast := range model {
		if sec != "p" && sec != "g" {
			continue
		}
		for ptype, policy := range ast {
			for _, rule := range policy.Policy {
				if err := insertRuleTx(tx, ptype, rule); err != nil {
					return err
				}
			}
		}
	}
	return tx.Commit()
}

func (a *MySQLAdapter) AddPolicy(sec string, ptype string, rule []string) error {
	return insertRule(a.db, ptype, rule)
}

func (a *MySQLAdapter) RemovePolicy(sec string, ptype string, rule []string) error {
	vals := normalizeRule(rule)
	_, err := a.db.ExecContext(context.Background(), `DELETE FROM aihub_casbin_rule WHERE ptype=? AND v0<=>? AND v1<=>? AND v2<=>? AND v3<=>? AND v4<=>? AND v5<=>?`, ptype, vals[0], vals[1], vals[2], vals[3], vals[4], vals[5])
	return err
}

func (a *MySQLAdapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	if fieldIndex < 0 || fieldIndex > 5 {
		return fmt.Errorf("invalid fieldIndex %d", fieldIndex)
	}
	where := []string{"ptype=?"}
	args := []any{ptype}
	for i, v := range fieldValues {
		idx := fieldIndex + i
		if idx > 5 {
			break
		}
		where = append(where, fmt.Sprintf("v%d=?", idx))
		args = append(args, v)
	}
	_, err := a.db.ExecContext(context.Background(), `DELETE FROM aihub_casbin_rule WHERE `+strings.Join(where, " AND "), args...)
	return err
}

func insertRule(db *sql.DB, ptype string, rule []string) error {
	vals := normalizeRule(rule)
	_, err := db.ExecContext(context.Background(), `INSERT IGNORE INTO aihub_casbin_rule(ptype,v0,v1,v2,v3,v4,v5,rule_hash) VALUES(?,?,?,?,?,?,?,?)`, ptype, vals[0], vals[1], vals[2], vals[3], vals[4], vals[5], ruleHash(ptype, rule))
	return err
}
func insertRuleTx(tx *sql.Tx, ptype string, rule []string) error {
	vals := normalizeRule(rule)
	_, err := tx.ExecContext(context.Background(), `INSERT IGNORE INTO aihub_casbin_rule(ptype,v0,v1,v2,v3,v4,v5,rule_hash) VALUES(?,?,?,?,?,?,?,?)`, ptype, vals[0], vals[1], vals[2], vals[3], vals[4], vals[5], ruleHash(ptype, rule))
	return err
}
func ruleHash(ptype string, rule []string) string {
	parts := make([]string, 0, 7)
	parts = append(parts, ptype)
	for i := 0; i < 6; i++ {
		if i < len(rule) {
			parts = append(parts, rule[i])
		} else {
			parts = append(parts, "")
		}
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x1f")))
	return hex.EncodeToString(sum[:])
}

func normalizeRule(rule []string) []any {
	vals := make([]any, 6)
	for i := 0; i < 6 && i < len(rule); i++ {
		if strings.TrimSpace(rule[i]) != "" {
			vals[i] = rule[i]
		}
	}
	return vals
}
