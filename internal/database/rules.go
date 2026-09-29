package database

import (
	"context"
	"database/sql"

	"humi/entity"
)

// Rules returns the rules of one node, in a stable order.
func (s *SQLite) Rules(ctx context.Context, nodeID int64) ([]entity.Rule, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, metric, op, threshold, for_min, enabled FROM rules
		 WHERE node_id = ? ORDER BY metric, op`, nodeID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	rules := make([]entity.Rule, 0, len(defaultRules))
	for rows.Next() {
		var r entity.Rule
		if err = rows.Scan(&r.ID, &r.Metric, &r.Op, &r.Threshold, &r.ForMin, &r.Enabled); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

// ReplaceRules swaps the whole rule set of a node in one transaction.
func (s *SQLite) ReplaceRules(ctx context.Context, nodeID int64, rules []entity.Rule) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err = tx.ExecContext(ctx, `DELETE FROM rules WHERE node_id = ?`, nodeID); err != nil {
		return err
	}
	if err = insertRules(ctx, tx, nodeID, rules); err != nil {
		return conflict(err)
	}
	return tx.Commit()
}

func insertRules(ctx context.Context, tx *sql.Tx, nodeID int64, rules []entity.Rule) error {
	for _, r := range rules {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO rules (node_id, metric, op, threshold, for_min, enabled) VALUES (?, ?, ?, ?, ?, ?)`,
			nodeID, r.Metric, r.Op, r.Threshold, r.ForMin, r.Enabled); err != nil {
			return err
		}
	}
	return nil
}
