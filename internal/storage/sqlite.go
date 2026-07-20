package storage

import (
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"harborx/internal/config"
	"harborx/internal/features/auth"
	"harborx/internal/features/backups"
	"harborx/internal/features/certificates"
	"harborx/internal/features/dns"
	"harborx/internal/features/nodes"
	"harborx/internal/features/notifications"
	"harborx/internal/features/proxygroups"
	"harborx/internal/features/remote"
	"harborx/internal/features/rules"
	"harborx/internal/features/subscriptions"
	"harborx/internal/features/system"
	"harborx/internal/features/templates"
	"harborx/internal/features/traffic"
	"harborx/internal/features/users"
)

//go:embed schema.sql seeds.sql
var migrationFiles embed.FS

type SQLiteStore struct {
	db *sql.DB
}

func (s *SQLiteStore) ListUsers() ([]users.User, error) {
	rows, err := s.db.Query(`
		SELECT id, username, role, status, display_name, email, created_at, updated_at
		FROM users
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []users.User
	for rows.Next() {
		var item users.User
		if err := rows.Scan(&item.ID, &item.Username, &item.Role, &item.Status, &item.DisplayName, &item.Email, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SQLiteStore) CreateUser(input users.CreateInput, passwordHash string) (users.User, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	role := input.Role
	if role == "" {
		role = "member"
	}
	item := users.User{
		ID:          newID("user"),
		Username:    strings.TrimSpace(input.Username),
		Role:        role,
		Status:      "active",
		DisplayName: input.DisplayName,
		Email:       input.Email,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_, err := s.db.Exec(`
		INSERT INTO users (id, username, password_hash, role, status, display_name, email, totp_secret, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, '', ?, ?)
	`, item.ID, item.Username, passwordHash, item.Role, item.Status, item.DisplayName, item.Email, item.CreatedAt, item.UpdatedAt)
	if err != nil {
		return users.User{}, err
	}
	return item, nil
}

func (s *SQLiteStore) UpdateUser(id string, input users.UpdateInput, passwordHash string) (users.User, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	role := input.Role
	if role == "" {
		role = "member"
	}
	status := input.Status
	if status == "" {
		status = "active"
	}

	var result sql.Result
	var err error
	if passwordHash == "" {
		result, err = s.db.Exec(`
			UPDATE users
			SET role = ?, status = ?, display_name = ?, email = ?, updated_at = ?
			WHERE id = ?
		`, role, status, input.DisplayName, input.Email, now, id)
	} else {
		result, err = s.db.Exec(`
			UPDATE users
			SET password_hash = ?, role = ?, status = ?, display_name = ?, email = ?, updated_at = ?
			WHERE id = ?
		`, passwordHash, role, status, input.DisplayName, input.Email, now, id)
	}
	if err != nil {
		return users.User{}, err
	}
	if rows, err := result.RowsAffected(); err != nil {
		return users.User{}, err
	} else if rows == 0 {
		return users.User{}, errors.New("user not found")
	}
	return s.findUser(id)
}

func (s *SQLiteStore) DeleteUser(id string) error {
	result, err := s.db.Exec(`DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if rows, err := result.RowsAffected(); err != nil {
		return err
	} else if rows == 0 {
		return errors.New("user not found")
	}
	return nil
}

func (s *SQLiteStore) findUser(id string) (users.User, error) {
	var item users.User
	err := s.db.QueryRow(`
		SELECT id, username, role, status, display_name, email, created_at, updated_at
		FROM users
		WHERE id = ?
	`, id).Scan(&item.ID, &item.Username, &item.Role, &item.Status, &item.DisplayName, &item.Email, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return users.User{}, errors.New("user not found")
		}
		return users.User{}, err
	}
	return item, nil
}

func (s *SQLiteStore) GetUserByUsername(username string) (auth.User, error) {
	var user auth.User
	err := s.db.QueryRow(`
		SELECT id, username, password_hash, role, status, display_name
		FROM users
		WHERE username = ?
	`, username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.Status, &user.DisplayName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return auth.User{}, errors.New("user not found")
		}
		return auth.User{}, err
	}
	return user, nil
}

func (s *SQLiteStore) UpdateUserPasswordHash(userID string, passwordHash string) error {
	_, err := s.db.Exec(`UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`, passwordHash, time.Now().UTC().Format(time.RFC3339), userID)
	return err
}

func (s *SQLiteStore) CreateAPIToken(userID string, name string, tokenHash string) error {
	_, err := s.db.Exec(`
		INSERT INTO api_tokens (id, user_id, name, token_hash, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, newID("token"), userID, name, tokenHash, time.Now().UTC().Format(time.RFC3339))
	return err
}

func (s *SQLiteStore) FindAPITokenByHash(tokenHash string) (auth.User, error) {
	var user auth.User
	err := s.db.QueryRow(`
		SELECT u.id, u.username, u.password_hash, u.role, u.status, u.display_name
		FROM api_tokens t
		JOIN users u ON u.id = t.user_id
		WHERE t.token_hash = ?
	`, tokenHash).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.Status, &user.DisplayName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return auth.User{}, errors.New("invalid token")
		}
		return auth.User{}, err
	}
	if user.Status != "active" {
		return auth.User{}, errors.New("user is disabled")
	}
	_, _ = s.db.Exec(`UPDATE api_tokens SET last_used_at = ? WHERE token_hash = ?`, time.Now().UTC().Format(time.RFC3339), tokenHash)
	user.PasswordHash = ""
	return user, nil
}

func OpenSQLite(cfg config.Config) (*SQLiteStore, error) {
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	db, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	store := &SQLiteStore{db: db}
	if _, err := store.db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *SQLiteStore) migrate() error {
	schemaSQL, err := migrationFiles.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("read schema.sql: %w", err)
	}

	seedsSQL, err := migrationFiles.ReadFile("seeds.sql")
	if err != nil {
		return fmt.Errorf("read seeds.sql: %w", err)
	}

	if _, err := s.db.Exec(string(schemaSQL)); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	if _, err := s.db.Exec(string(seedsSQL)); err != nil {
		return fmt.Errorf("apply seeds: %w", err)
	}

	return nil
}

func (s *SQLiteStore) ListNodes() ([]nodes.Node, error) {
	rows, err := s.db.Query(`
		SELECT id, name, source_kind, protocol, server_host, server_port, tags_json, metadata_json, enabled, created_at, updated_at
		FROM nodes
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []nodes.Node
	for rows.Next() {
		var item nodes.Node
		var tagsJSON string
		var metadataJSON string
		var enabled int
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.SourceKind,
			&item.Protocol,
			&item.ServerHost,
			&item.ServerPort,
			&tagsJSON,
			&metadataJSON,
			&enabled,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.Enabled = enabled == 1
		item.Tags = decodeStringSlice(tagsJSON)
		item.Metadata = decodeMap(metadataJSON)
		items = append(items, item)
	}

	return items, rows.Err()
}

func (s *SQLiteStore) CreateNode(input nodes.CreateInput) (nodes.Node, error) {
	if strings.TrimSpace(input.Name) == "" {
		return nodes.Node{}, errors.New("node name is required")
	}
	if strings.TrimSpace(input.Protocol) == "" {
		return nodes.Node{}, errors.New("node protocol is required")
	}
	if strings.TrimSpace(input.ServerHost) == "" {
		return nodes.Node{}, errors.New("node server host is required")
	}
	if input.ServerPort <= 0 {
		return nodes.Node{}, errors.New("node server port must be greater than 0")
	}
	if strings.TrimSpace(input.SourceKind) == "" {
		input.SourceKind = "manual"
	}

	now := time.Now().UTC().Format(time.RFC3339)
	item := nodes.Node{
		ID:         newID("node"),
		Name:       strings.TrimSpace(input.Name),
		SourceKind: input.SourceKind,
		Protocol:   input.Protocol,
		ServerHost: input.ServerHost,
		ServerPort: input.ServerPort,
		Tags:       cloneStringSlice(input.Tags),
		Metadata:   cloneMap(input.Metadata),
		Enabled:    input.Enabled,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	_, err := s.db.Exec(`
		INSERT INTO nodes (id, name, source_kind, protocol, server_host, server_port, tags_json, metadata_json, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		item.ID,
		item.Name,
		item.SourceKind,
		item.Protocol,
		item.ServerHost,
		item.ServerPort,
		encodeJSON(item.Tags),
		encodeJSON(item.Metadata),
		boolToInt(item.Enabled),
		item.CreatedAt,
		item.UpdatedAt,
	)
	if err != nil {
		return nodes.Node{}, err
	}

	return item, nil
}

func (s *SQLiteStore) DeleteNode(id string) error {
	result, err := s.db.Exec(`DELETE FROM nodes WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("node not found")
	}
	return nil
}

func (s *SQLiteStore) UpdateNode(id string, input nodes.CreateInput) (nodes.Node, error) {
	if strings.TrimSpace(input.Name) == "" {
		return nodes.Node{}, errors.New("node name is required")
	}
	if strings.TrimSpace(input.Protocol) == "" {
		return nodes.Node{}, errors.New("node protocol is required")
	}
	if strings.TrimSpace(input.ServerHost) == "" {
		return nodes.Node{}, errors.New("node server host is required")
	}
	if input.ServerPort <= 0 {
		return nodes.Node{}, errors.New("node server port must be greater than 0")
	}
	if strings.TrimSpace(input.SourceKind) == "" {
		input.SourceKind = "manual"
	}

	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.Exec(`
		UPDATE nodes
		SET name = ?, source_kind = ?, protocol = ?, server_host = ?, server_port = ?, tags_json = ?, metadata_json = ?, enabled = ?, updated_at = ?
		WHERE id = ?
	`,
		strings.TrimSpace(input.Name),
		input.SourceKind,
		input.Protocol,
		input.ServerHost,
		input.ServerPort,
		encodeJSON(cloneStringSlice(input.Tags)),
		encodeJSON(cloneMap(input.Metadata)),
		boolToInt(input.Enabled),
		now,
		id,
	)
	if err != nil {
		return nodes.Node{}, err
	}
	if rows, err := result.RowsAffected(); err != nil {
		return nodes.Node{}, err
	} else if rows == 0 {
		return nodes.Node{}, errors.New("node not found")
	}
	return s.findNode(id)
}

func (s *SQLiteStore) findNode(id string) (nodes.Node, error) {
	var item nodes.Node
	var tagsJSON string
	var metadataJSON string
	var enabled int
	err := s.db.QueryRow(`
		SELECT id, name, source_kind, protocol, server_host, server_port, tags_json, metadata_json, enabled, created_at, updated_at
		FROM nodes
		WHERE id = ?
	`, id).Scan(
		&item.ID,
		&item.Name,
		&item.SourceKind,
		&item.Protocol,
		&item.ServerHost,
		&item.ServerPort,
		&tagsJSON,
		&metadataJSON,
		&enabled,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nodes.Node{}, errors.New("node not found")
		}
		return nodes.Node{}, err
	}
	item.Enabled = enabled == 1
	item.Tags = decodeStringSlice(tagsJSON)
	item.Metadata = decodeMap(metadataJSON)
	return item, nil
}

func (s *SQLiteStore) ListRuleSets() ([]rules.RuleSet, error) {
	rows, err := s.db.Query(`
		SELECT id, name, scope, description, created_at, updated_at
		FROM rule_sets
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []rules.RuleSet
	for rows.Next() {
		var item rules.RuleSet
		if err := rows.Scan(&item.ID, &item.Name, &item.Scope, &item.Description, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		ruleItems, err := s.listRulesForRuleSet(item.ID)
		if err != nil {
			return nil, err
		}
		item.Rules = ruleItems
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SQLiteStore) CreateRuleSet(input rules.CreateRuleSetInput) (rules.RuleSet, error) {
	if strings.TrimSpace(input.Name) == "" {
		return rules.RuleSet{}, errors.New("rule set name is required")
	}
	if strings.TrimSpace(input.Scope) == "" {
		input.Scope = "global"
	}

	now := time.Now().UTC().Format(time.RFC3339)
	item := rules.RuleSet{
		ID:          newID("ruleset"),
		Name:        strings.TrimSpace(input.Name),
		Scope:       input.Scope,
		Description: input.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	tx, err := s.db.Begin()
	if err != nil {
		return rules.RuleSet{}, err
	}

	if _, err := tx.Exec(`
		INSERT INTO rule_sets (id, name, scope, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, item.ID, item.Name, item.Scope, item.Description, item.CreatedAt, item.UpdatedAt); err != nil {
		_ = tx.Rollback()
		return rules.RuleSet{}, err
	}

	for index, ruleItem := range input.Rules {
		if strings.TrimSpace(ruleItem.RuleType) == "" {
			_ = tx.Rollback()
			return rules.RuleSet{}, errors.New("rule type is required")
		}
		if strings.TrimSpace(ruleItem.Policy) == "" {
			_ = tx.Rollback()
			return rules.RuleSet{}, errors.New("rule policy is required")
		}
		if ruleItem.SortOrder == 0 {
			ruleItem.SortOrder = index + 1
		}
		if _, err := tx.Exec(`
			INSERT INTO rules (id, rule_set_id, rule_type, pattern, policy, sort_order, enabled, note, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			newID("rule"),
			item.ID,
			ruleItem.RuleType,
			ruleItem.Pattern,
			ruleItem.Policy,
			ruleItem.SortOrder,
			boolToInt(ruleItem.Enabled),
			ruleItem.Note,
			now,
			now,
		); err != nil {
			_ = tx.Rollback()
			return rules.RuleSet{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return rules.RuleSet{}, err
	}

	createdRules, err := s.listRulesForRuleSet(item.ID)
	if err != nil {
		return rules.RuleSet{}, err
	}
	item.Rules = createdRules
	return item, nil
}

func (s *SQLiteStore) UpdateRuleSet(id string, input rules.CreateRuleSetInput) (rules.RuleSet, error) {
	if strings.TrimSpace(input.Name) == "" {
		return rules.RuleSet{}, errors.New("rule set name is required")
	}
	if strings.TrimSpace(input.Scope) == "" {
		input.Scope = "global"
	}

	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.db.Begin()
	if err != nil {
		return rules.RuleSet{}, err
	}

	result, err := tx.Exec(`
		UPDATE rule_sets
		SET name = ?, scope = ?, description = ?, updated_at = ?
		WHERE id = ?
	`, strings.TrimSpace(input.Name), input.Scope, input.Description, now, id)
	if err != nil {
		_ = tx.Rollback()
		return rules.RuleSet{}, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return rules.RuleSet{}, err
	}
	if rowsAffected == 0 {
		_ = tx.Rollback()
		return rules.RuleSet{}, errors.New("rule set not found")
	}

	if _, err := tx.Exec(`DELETE FROM rules WHERE rule_set_id = ?`, id); err != nil {
		_ = tx.Rollback()
		return rules.RuleSet{}, err
	}

	for index, ruleItem := range input.Rules {
		if ruleItem.SortOrder == 0 {
			ruleItem.SortOrder = index + 1
		}
		if _, err := tx.Exec(`
			INSERT INTO rules (id, rule_set_id, rule_type, pattern, policy, sort_order, enabled, note, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			newID("rule"),
			id,
			ruleItem.RuleType,
			ruleItem.Pattern,
			ruleItem.Policy,
			ruleItem.SortOrder,
			boolToInt(ruleItem.Enabled),
			ruleItem.Note,
			now,
			now,
		); err != nil {
			_ = tx.Rollback()
			return rules.RuleSet{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return rules.RuleSet{}, err
	}

	return s.findRuleSet(id)
}

func (s *SQLiteStore) DeleteRuleSet(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM rules WHERE rule_set_id = ?`, id); err != nil {
		_ = tx.Rollback()
		return err
	}
	result, err := tx.Exec(`DELETE FROM rule_sets WHERE id = ?`, id)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if rowsAffected == 0 {
		_ = tx.Rollback()
		return errors.New("rule set not found")
	}
	return tx.Commit()
}

func (s *SQLiteStore) findRuleSet(id string) (rules.RuleSet, error) {
	var item rules.RuleSet
	err := s.db.QueryRow(`
		SELECT id, name, scope, description, created_at, updated_at
		FROM rule_sets
		WHERE id = ?
	`, id).Scan(&item.ID, &item.Name, &item.Scope, &item.Description, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return rules.RuleSet{}, errors.New("rule set not found")
		}
		return rules.RuleSet{}, err
	}
	ruleItems, err := s.listRulesForRuleSet(id)
	if err != nil {
		return rules.RuleSet{}, err
	}
	item.Rules = ruleItems
	return item, nil
}

func (s *SQLiteStore) listRulesForRuleSet(ruleSetID string) ([]rules.Rule, error) {
	rows, err := s.db.Query(`
		SELECT id, rule_type, pattern, policy, sort_order, enabled, note
		FROM rules
		WHERE rule_set_id = ?
		ORDER BY sort_order ASC, created_at ASC
	`, ruleSetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []rules.Rule
	for rows.Next() {
		var item rules.Rule
		var enabled int
		if err := rows.Scan(&item.ID, &item.RuleType, &item.Pattern, &item.Policy, &item.SortOrder, &enabled, &item.Note); err != nil {
			return nil, err
		}
		item.Enabled = enabled == 1
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SQLiteStore) ListTemplates() ([]templates.Template, error) {
	rows, err := s.db.Query(`
		SELECT id, name, kind, description, variables_json, content, locked
		FROM templates
		ORDER BY kind ASC, name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []templates.Template
	for rows.Next() {
		var item templates.Template
		var variablesJSON string
		var locked int
		if err := rows.Scan(&item.ID, &item.Name, &item.Kind, &item.Description, &variablesJSON, &item.Content, &locked); err != nil {
			return nil, err
		}
		item.Variables = decodeStringSlice(variablesJSON)
		item.Locked = locked == 1
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SQLiteStore) CreateTemplate(input templates.CreateInput) (templates.Template, error) {
	if strings.TrimSpace(input.Name) == "" {
		return templates.Template{}, errors.New("template name is required")
	}
	if strings.TrimSpace(input.Kind) == "" {
		input.Kind = "private"
	}
	if strings.TrimSpace(input.Content) == "" {
		return templates.Template{}, errors.New("template content is required")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	item := templates.Template{
		ID:          newID("template"),
		Name:        strings.TrimSpace(input.Name),
		Kind:        input.Kind,
		Description: input.Description,
		Variables:   cloneStringSlice(input.Variables),
		Content:     input.Content,
		Locked:      false,
	}

	_, err := s.db.Exec(`
		INSERT INTO templates (id, name, kind, description, engine, variables_json, content, locked, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'text-template', ?, ?, 0, ?, ?)
	`,
		item.ID,
		item.Name,
		item.Kind,
		item.Description,
		encodeJSON(item.Variables),
		item.Content,
		now,
		now,
	)
	if err != nil {
		return templates.Template{}, err
	}

	return item, nil
}

func (s *SQLiteStore) UpdateTemplate(id string, input templates.CreateInput) (templates.Template, error) {
	if strings.TrimSpace(input.Name) == "" {
		return templates.Template{}, errors.New("template name is required")
	}
	if strings.TrimSpace(input.Kind) == "" {
		input.Kind = "private"
	}
	if strings.TrimSpace(input.Content) == "" {
		return templates.Template{}, errors.New("template content is required")
	}
	if locked, err := s.templateLocked(id); err != nil {
		return templates.Template{}, err
	} else if locked {
		return templates.Template{}, errors.New("built-in templates are locked")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.Exec(`
		UPDATE templates
		SET name = ?, kind = ?, description = ?, variables_json = ?, content = ?, updated_at = ?
		WHERE id = ?
	`, strings.TrimSpace(input.Name), input.Kind, input.Description, encodeJSON(cloneStringSlice(input.Variables)), input.Content, now, id)
	if err != nil {
		return templates.Template{}, err
	}
	if rows, err := result.RowsAffected(); err != nil {
		return templates.Template{}, err
	} else if rows == 0 {
		return templates.Template{}, errors.New("template not found")
	}
	return s.findTemplate(id)
}

func (s *SQLiteStore) DeleteTemplate(id string) error {
	if locked, err := s.templateLocked(id); err != nil {
		return err
	} else if locked {
		return errors.New("built-in templates are locked")
	}
	result, err := s.db.Exec(`DELETE FROM templates WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if rows, err := result.RowsAffected(); err != nil {
		return err
	} else if rows == 0 {
		return errors.New("template not found")
	}
	return nil
}

func (s *SQLiteStore) findTemplate(id string) (templates.Template, error) {
	var item templates.Template
	var variablesJSON string
	var locked int
	err := s.db.QueryRow(`
		SELECT id, name, kind, description, variables_json, content, locked
		FROM templates
		WHERE id = ?
	`, id).Scan(&item.ID, &item.Name, &item.Kind, &item.Description, &variablesJSON, &item.Content, &locked)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return templates.Template{}, errors.New("template not found")
		}
		return templates.Template{}, err
	}
	item.Variables = decodeStringSlice(variablesJSON)
	item.Locked = locked == 1
	return item, nil
}

func (s *SQLiteStore) templateLocked(id string) (bool, error) {
	var locked int
	err := s.db.QueryRow(`SELECT locked FROM templates WHERE id = ?`, id).Scan(&locked)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, errors.New("template not found")
		}
		return false, err
	}
	return locked == 1, nil
}

func (s *SQLiteStore) ListSubscriptions() ([]subscriptions.Subscription, error) {
	rows, err := s.db.Query(`
		SELECT id, name, owner_user_id, output_format, template_id, source_json, options_json, created_at, updated_at
		FROM subscriptions
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []subscriptions.Subscription
	for rows.Next() {
		var item subscriptions.Subscription
		var sourceJSON string
		var optionsJSON string
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.OwnerUserID,
			&item.OutputFormat,
			&item.TemplateID,
			&sourceJSON,
			&optionsJSON,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.Sources = decodeStringSlice(sourceJSON)
		item.Options = decodeMap(optionsJSON)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SQLiteStore) CreateSubscription(input subscriptions.CreateInput) (subscriptions.Subscription, error) {
	if strings.TrimSpace(input.Name) == "" {
		return subscriptions.Subscription{}, errors.New("subscription name is required")
	}
	if strings.TrimSpace(input.OutputFormat) == "" {
		return subscriptions.Subscription{}, errors.New("subscription output format is required")
	}
	if strings.TrimSpace(input.TemplateID) == "" {
		return subscriptions.Subscription{}, errors.New("template id is required")
	}
	if strings.TrimSpace(input.OwnerUserID) == "" {
		input.OwnerUserID = "local-admin"
	}

	now := time.Now().UTC().Format(time.RFC3339)
	item := subscriptions.Subscription{
		ID:           newID("subscription"),
		Name:         strings.TrimSpace(input.Name),
		OwnerUserID:  input.OwnerUserID,
		OutputFormat: input.OutputFormat,
		TemplateID:   input.TemplateID,
		Sources:      cloneStringSlice(input.Sources),
		Options:      cloneMap(input.Options),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	_, err := s.db.Exec(`
		INSERT INTO subscriptions (id, name, owner_user_id, output_format, template_id, source_json, options_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		item.ID,
		item.Name,
		item.OwnerUserID,
		item.OutputFormat,
		item.TemplateID,
		encodeJSON(item.Sources),
		encodeJSON(item.Options),
		item.CreatedAt,
		item.UpdatedAt,
	)
	if err != nil {
		return subscriptions.Subscription{}, err
	}

	return item, nil
}

func (s *SQLiteStore) UpdateSubscription(id string, input subscriptions.CreateInput) (subscriptions.Subscription, error) {
	if strings.TrimSpace(input.Name) == "" {
		return subscriptions.Subscription{}, errors.New("subscription name is required")
	}
	if strings.TrimSpace(input.OutputFormat) == "" {
		return subscriptions.Subscription{}, errors.New("subscription output format is required")
	}
	if strings.TrimSpace(input.TemplateID) == "" {
		return subscriptions.Subscription{}, errors.New("template id is required")
	}
	if strings.TrimSpace(input.OwnerUserID) == "" {
		input.OwnerUserID = "local-admin"
	}

	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.Exec(`
		UPDATE subscriptions
		SET name = ?, owner_user_id = ?, output_format = ?, template_id = ?, source_json = ?, options_json = ?, updated_at = ?
		WHERE id = ?
	`,
		strings.TrimSpace(input.Name),
		input.OwnerUserID,
		input.OutputFormat,
		input.TemplateID,
		encodeJSON(cloneStringSlice(input.Sources)),
		encodeJSON(cloneMap(input.Options)),
		now,
		id,
	)
	if err != nil {
		return subscriptions.Subscription{}, err
	}
	if rows, err := result.RowsAffected(); err != nil {
		return subscriptions.Subscription{}, err
	} else if rows == 0 {
		return subscriptions.Subscription{}, errors.New("subscription not found")
	}
	return s.findSubscriptionRecord(id)
}

func (s *SQLiteStore) DeleteSubscription(id string) error {
	result, err := s.db.Exec(`DELETE FROM subscriptions WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if rows, err := result.RowsAffected(); err != nil {
		return err
	} else if rows == 0 {
		return errors.New("subscription not found")
	}
	return nil
}

func (s *SQLiteStore) findSubscriptionRecord(id string) (subscriptions.Subscription, error) {
	var item subscriptions.Subscription
	var sourceJSON string
	var optionsJSON string
	err := s.db.QueryRow(`
		SELECT id, name, owner_user_id, output_format, template_id, source_json, options_json, created_at, updated_at
		FROM subscriptions
		WHERE id = ?
	`, id).Scan(
		&item.ID,
		&item.Name,
		&item.OwnerUserID,
		&item.OutputFormat,
		&item.TemplateID,
		&sourceJSON,
		&optionsJSON,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return subscriptions.Subscription{}, errors.New("subscription not found")
		}
		return subscriptions.Subscription{}, err
	}
	item.Sources = decodeStringSlice(sourceJSON)
	item.Options = decodeMap(optionsJSON)
	return item, nil
}

func (s *SQLiteStore) ListRemoteServers() ([]remote.RemoteServer, error) {
	rows, err := s.db.Query(`
		SELECT id, name, host, connection_mode, status, metadata_json, created_at, updated_at
		FROM remote_servers
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []remote.RemoteServer
	for rows.Next() {
		var item remote.RemoteServer
		var metadataJSON string
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Host,
			&item.ConnectionMode,
			&item.Status,
			&metadataJSON,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.Metadata = decodeMap(metadataJSON)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SQLiteStore) CreateRemoteServer(input remote.CreateServerInput, serverTokenHash string, agentTokenHash string) (remote.RemoteServer, error) {
	if strings.TrimSpace(input.Name) == "" {
		return remote.RemoteServer{}, errors.New("remote server name is required")
	}
	if strings.TrimSpace(input.Host) == "" {
		return remote.RemoteServer{}, errors.New("remote server host is required")
	}
	if strings.TrimSpace(input.ConnectionMode) == "" {
		input.ConnectionMode = "pull"
	}

	now := time.Now().UTC().Format(time.RFC3339)
	item := remote.RemoteServer{
		ID:             newID("remote"),
		Name:           strings.TrimSpace(input.Name),
		Host:           strings.TrimSpace(input.Host),
		ConnectionMode: input.ConnectionMode,
		Status:         "pending",
		Metadata:       cloneMap(input.Metadata),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	_, err := s.db.Exec(`
		INSERT INTO remote_servers (id, name, host, connection_mode, status, server_token_hash, agent_token_hash, metadata_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		item.ID,
		item.Name,
		item.Host,
		item.ConnectionMode,
		item.Status,
		serverTokenHash,
		agentTokenHash,
		encodeJSON(item.Metadata),
		item.CreatedAt,
		item.UpdatedAt,
	)
	if err != nil {
		return remote.RemoteServer{}, err
	}

	return item, nil
}

func (s *SQLiteStore) UpdateRemoteServer(id string, input remote.UpdateServerInput) (remote.RemoteServer, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.Exec(`
		UPDATE remote_servers
		SET name = ?, host = ?, connection_mode = ?, status = ?, metadata_json = ?, updated_at = ?
		WHERE id = ?
	`,
		strings.TrimSpace(input.Name),
		strings.TrimSpace(input.Host),
		input.ConnectionMode,
		input.Status,
		encodeJSON(cloneMap(input.Metadata)),
		now,
		id,
	)
	if err != nil {
		return remote.RemoteServer{}, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return remote.RemoteServer{}, err
	}
	if rowsAffected == 0 {
		return remote.RemoteServer{}, errors.New("remote server not found")
	}

	return s.findRemoteServer(id)
}

func (s *SQLiteStore) DeleteRemoteServer(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM remote_tasks WHERE remote_server_id = ?`, id); err != nil {
		_ = tx.Rollback()
		return err
	}
	result, err := tx.Exec(`DELETE FROM remote_servers WHERE id = ?`, id)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if rowsAffected == 0 {
		_ = tx.Rollback()
		return errors.New("remote server not found")
	}
	return tx.Commit()
}

func (s *SQLiteStore) ListRemoteTasks(serverID string) ([]remote.RemoteTask, error) {
	rows, err := s.db.Query(`
		SELECT id, remote_server_id, task_kind, status, payload_json, output_text, created_at, updated_at
		FROM remote_tasks
		WHERE remote_server_id = ?
		ORDER BY created_at DESC
	`, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []remote.RemoteTask
	for rows.Next() {
		var item remote.RemoteTask
		var payloadJSON string
		if err := rows.Scan(
			&item.ID,
			&item.RemoteServerID,
			&item.TaskKind,
			&item.Status,
			&payloadJSON,
			&item.OutputText,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.Payload = decodeMap(payloadJSON)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SQLiteStore) CreateRemoteTask(serverID string, input remote.CreateTaskInput) (remote.RemoteTask, error) {
	if _, err := s.findRemoteServer(serverID); err != nil {
		return remote.RemoteTask{}, err
	}
	if strings.TrimSpace(input.TaskKind) == "" {
		return remote.RemoteTask{}, errors.New("remote task kind is required")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	item := remote.RemoteTask{
		ID:             newID("task"),
		RemoteServerID: serverID,
		TaskKind:       strings.TrimSpace(input.TaskKind),
		Status:         "queued",
		Payload:        cloneMap(input.Payload),
		OutputText:     "",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	_, err := s.db.Exec(`
		INSERT INTO remote_tasks (id, remote_server_id, task_kind, status, payload_json, output_text, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		item.ID,
		item.RemoteServerID,
		item.TaskKind,
		item.Status,
		encodeJSON(item.Payload),
		item.OutputText,
		item.CreatedAt,
		item.UpdatedAt,
	)
	if err != nil {
		return remote.RemoteTask{}, err
	}

	return item, nil
}

func (s *SQLiteStore) findRemoteServer(id string) (remote.RemoteServer, error) {
	var item remote.RemoteServer
	var metadataJSON string
	err := s.db.QueryRow(`
		SELECT id, name, host, connection_mode, status, metadata_json, created_at, updated_at
		FROM remote_servers
		WHERE id = ?
	`, id).Scan(
		&item.ID,
		&item.Name,
		&item.Host,
		&item.ConnectionMode,
		&item.Status,
		&metadataJSON,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return remote.RemoteServer{}, errors.New("remote server not found")
		}
		return remote.RemoteServer{}, err
	}
	item.Metadata = decodeMap(metadataJSON)
	return item, nil
}

func (s *SQLiteStore) ListProxyGroups() ([]proxygroups.ProxyGroup, error) {
	rows, err := s.db.Query(`
		SELECT id, name, group_kind, config_json, sort_order, created_at, updated_at
		FROM proxy_groups
		ORDER BY sort_order ASC, name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []proxygroups.ProxyGroup
	for rows.Next() {
		var item proxygroups.ProxyGroup
		var configJSON string
		if err := rows.Scan(&item.ID, &item.Name, &item.GroupKind, &configJSON, &item.SortOrder, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Config = decodeMap(configJSON)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SQLiteStore) CreateProxyGroup(input proxygroups.CreateInput) (proxygroups.ProxyGroup, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	item := proxygroups.ProxyGroup{
		ID:        newID("proxygroup"),
		Name:      strings.TrimSpace(input.Name),
		GroupKind: input.GroupKind,
		Config:    cloneMap(input.Config),
		SortOrder: input.SortOrder,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := s.db.Exec(`
		INSERT INTO proxy_groups (id, name, group_kind, config_json, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, item.ID, item.Name, item.GroupKind, encodeJSON(item.Config), item.SortOrder, item.CreatedAt, item.UpdatedAt)
	if err != nil {
		return proxygroups.ProxyGroup{}, err
	}
	return item, nil
}

func (s *SQLiteStore) UpdateProxyGroup(id string, input proxygroups.CreateInput) (proxygroups.ProxyGroup, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.Exec(`
		UPDATE proxy_groups
		SET name = ?, group_kind = ?, config_json = ?, sort_order = ?, updated_at = ?
		WHERE id = ?
	`, strings.TrimSpace(input.Name), input.GroupKind, encodeJSON(cloneMap(input.Config)), input.SortOrder, now, id)
	if err != nil {
		return proxygroups.ProxyGroup{}, err
	}
	if rows, err := result.RowsAffected(); err != nil {
		return proxygroups.ProxyGroup{}, err
	} else if rows == 0 {
		return proxygroups.ProxyGroup{}, errors.New("proxy group not found")
	}
	return s.findProxyGroup(id)
}

func (s *SQLiteStore) DeleteProxyGroup(id string) error {
	result, err := s.db.Exec(`DELETE FROM proxy_groups WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if rows, err := result.RowsAffected(); err != nil {
		return err
	} else if rows == 0 {
		return errors.New("proxy group not found")
	}
	return nil
}

func (s *SQLiteStore) findProxyGroup(id string) (proxygroups.ProxyGroup, error) {
	var item proxygroups.ProxyGroup
	var configJSON string
	err := s.db.QueryRow(`
		SELECT id, name, group_kind, config_json, sort_order, created_at, updated_at
		FROM proxy_groups
		WHERE id = ?
	`, id).Scan(&item.ID, &item.Name, &item.GroupKind, &configJSON, &item.SortOrder, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return proxygroups.ProxyGroup{}, errors.New("proxy group not found")
		}
		return proxygroups.ProxyGroup{}, err
	}
	item.Config = decodeMap(configJSON)
	return item, nil
}

func (s *SQLiteStore) ListDNSProviders() ([]dns.Provider, error) {
	rows, err := s.db.Query(`
		SELECT id, provider_kind, name, credentials_json, created_at, updated_at
		FROM dns_providers
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []dns.Provider
	for rows.Next() {
		var item dns.Provider
		var credentialsJSON string
		if err := rows.Scan(&item.ID, &item.ProviderKind, &item.Name, &credentialsJSON, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Credentials = decodeMap(credentialsJSON)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SQLiteStore) CreateDNSProvider(input dns.CreateProviderInput) (dns.Provider, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	item := dns.Provider{
		ID:           newID("dns"),
		ProviderKind: input.ProviderKind,
		Name:         strings.TrimSpace(input.Name),
		Credentials:  cloneMap(input.Credentials),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	_, err := s.db.Exec(`
		INSERT INTO dns_providers (id, provider_kind, name, credentials_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, item.ID, item.ProviderKind, item.Name, encodeJSON(item.Credentials), item.CreatedAt, item.UpdatedAt)
	if err != nil {
		return dns.Provider{}, err
	}
	return item, nil
}

func (s *SQLiteStore) UpdateDNSProvider(id string, input dns.CreateProviderInput) (dns.Provider, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.Exec(`
		UPDATE dns_providers
		SET provider_kind = ?, name = ?, credentials_json = ?, updated_at = ?
		WHERE id = ?
	`, input.ProviderKind, strings.TrimSpace(input.Name), encodeJSON(cloneMap(input.Credentials)), now, id)
	if err != nil {
		return dns.Provider{}, err
	}
	if rows, err := result.RowsAffected(); err != nil {
		return dns.Provider{}, err
	} else if rows == 0 {
		return dns.Provider{}, errors.New("dns provider not found")
	}
	return s.findDNSProvider(id)
}

func (s *SQLiteStore) DeleteDNSProvider(id string) error {
	result, err := s.db.Exec(`DELETE FROM dns_providers WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if rows, err := result.RowsAffected(); err != nil {
		return err
	} else if rows == 0 {
		return errors.New("dns provider not found")
	}
	return nil
}

func (s *SQLiteStore) findDNSProvider(id string) (dns.Provider, error) {
	var item dns.Provider
	var credentialsJSON string
	err := s.db.QueryRow(`
		SELECT id, provider_kind, name, credentials_json, created_at, updated_at
		FROM dns_providers
		WHERE id = ?
	`, id).Scan(&item.ID, &item.ProviderKind, &item.Name, &credentialsJSON, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dns.Provider{}, errors.New("dns provider not found")
		}
		return dns.Provider{}, err
	}
	item.Credentials = decodeMap(credentialsJSON)
	return item, nil
}

func (s *SQLiteStore) ListCertificates() ([]certificates.Certificate, error) {
	rows, err := s.db.Query(`
		SELECT id, name, domain, provider_id, cert_pem, key_pem, auto_renew, auto_deploy, expires_at, created_at, updated_at
		FROM certificates
		ORDER BY domain ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []certificates.Certificate
	for rows.Next() {
		var item certificates.Certificate
		var autoRenew int
		var autoDeploy int
		if err := rows.Scan(&item.ID, &item.Name, &item.Domain, &item.ProviderID, &item.CertPEM, &item.KeyPEM, &autoRenew, &autoDeploy, &item.ExpiresAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.AutoRenew = autoRenew == 1
		item.AutoDeploy = autoDeploy == 1
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SQLiteStore) CreateCertificate(input certificates.CreateInput) (certificates.Certificate, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	item := certificates.Certificate{
		ID:         newID("cert"),
		Name:       strings.TrimSpace(input.Name),
		Domain:     strings.TrimSpace(input.Domain),
		ProviderID: input.ProviderID,
		CertPEM:    input.CertPEM,
		KeyPEM:     input.KeyPEM,
		AutoRenew:  input.AutoRenew,
		AutoDeploy: input.AutoDeploy,
		ExpiresAt:  input.ExpiresAt,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	_, err := s.db.Exec(`
		INSERT INTO certificates (id, name, domain, provider_id, cert_pem, key_pem, auto_renew, auto_deploy, expires_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, item.ID, item.Name, item.Domain, item.ProviderID, item.CertPEM, item.KeyPEM, boolToInt(item.AutoRenew), boolToInt(item.AutoDeploy), item.ExpiresAt, item.CreatedAt, item.UpdatedAt)
	if err != nil {
		return certificates.Certificate{}, err
	}
	return item, nil
}

func (s *SQLiteStore) UpdateCertificate(id string, input certificates.CreateInput) (certificates.Certificate, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.Exec(`
		UPDATE certificates
		SET name = ?, domain = ?, provider_id = ?, cert_pem = ?, key_pem = ?, auto_renew = ?, auto_deploy = ?, expires_at = ?, updated_at = ?
		WHERE id = ?
	`, strings.TrimSpace(input.Name), strings.TrimSpace(input.Domain), input.ProviderID, input.CertPEM, input.KeyPEM, boolToInt(input.AutoRenew), boolToInt(input.AutoDeploy), input.ExpiresAt, now, id)
	if err != nil {
		return certificates.Certificate{}, err
	}
	if rows, err := result.RowsAffected(); err != nil {
		return certificates.Certificate{}, err
	} else if rows == 0 {
		return certificates.Certificate{}, errors.New("certificate not found")
	}
	return s.findCertificate(id)
}

func (s *SQLiteStore) DeleteCertificate(id string) error {
	result, err := s.db.Exec(`DELETE FROM certificates WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if rows, err := result.RowsAffected(); err != nil {
		return err
	} else if rows == 0 {
		return errors.New("certificate not found")
	}
	return nil
}

func (s *SQLiteStore) findCertificate(id string) (certificates.Certificate, error) {
	var item certificates.Certificate
	var autoRenew int
	var autoDeploy int
	err := s.db.QueryRow(`
		SELECT id, name, domain, provider_id, cert_pem, key_pem, auto_renew, auto_deploy, expires_at, created_at, updated_at
		FROM certificates
		WHERE id = ?
	`, id).Scan(&item.ID, &item.Name, &item.Domain, &item.ProviderID, &item.CertPEM, &item.KeyPEM, &autoRenew, &autoDeploy, &item.ExpiresAt, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return certificates.Certificate{}, errors.New("certificate not found")
		}
		return certificates.Certificate{}, err
	}
	item.AutoRenew = autoRenew == 1
	item.AutoDeploy = autoDeploy == 1
	return item, nil
}

func (s *SQLiteStore) ListNotificationChannels() ([]notifications.Channel, error) {
	rows, err := s.db.Query(`
		SELECT id, channel_kind, name, config_json, enabled, created_at, updated_at
		FROM notifications
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []notifications.Channel
	for rows.Next() {
		var item notifications.Channel
		var configJSON string
		var enabled int
		if err := rows.Scan(&item.ID, &item.ChannelKind, &item.Name, &configJSON, &enabled, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Config = decodeMap(configJSON)
		item.Enabled = enabled == 1
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SQLiteStore) CreateNotificationChannel(input notifications.CreateInput) (notifications.Channel, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	item := notifications.Channel{
		ID:          newID("notify"),
		ChannelKind: input.ChannelKind,
		Name:        strings.TrimSpace(input.Name),
		Config:      cloneMap(input.Config),
		Enabled:     input.Enabled,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_, err := s.db.Exec(`
		INSERT INTO notifications (id, channel_kind, name, config_json, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, item.ID, item.ChannelKind, item.Name, encodeJSON(item.Config), boolToInt(item.Enabled), item.CreatedAt, item.UpdatedAt)
	if err != nil {
		return notifications.Channel{}, err
	}
	return item, nil
}

func (s *SQLiteStore) UpdateNotificationChannel(id string, input notifications.CreateInput) (notifications.Channel, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.Exec(`
		UPDATE notifications
		SET channel_kind = ?, name = ?, config_json = ?, enabled = ?, updated_at = ?
		WHERE id = ?
	`, input.ChannelKind, strings.TrimSpace(input.Name), encodeJSON(cloneMap(input.Config)), boolToInt(input.Enabled), now, id)
	if err != nil {
		return notifications.Channel{}, err
	}
	if rows, err := result.RowsAffected(); err != nil {
		return notifications.Channel{}, err
	} else if rows == 0 {
		return notifications.Channel{}, errors.New("notification channel not found")
	}
	return s.findNotificationChannel(id)
}

func (s *SQLiteStore) DeleteNotificationChannel(id string) error {
	result, err := s.db.Exec(`DELETE FROM notifications WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if rows, err := result.RowsAffected(); err != nil {
		return err
	} else if rows == 0 {
		return errors.New("notification channel not found")
	}
	return nil
}

func (s *SQLiteStore) findNotificationChannel(id string) (notifications.Channel, error) {
	var item notifications.Channel
	var configJSON string
	var enabled int
	err := s.db.QueryRow(`
		SELECT id, channel_kind, name, config_json, enabled, created_at, updated_at
		FROM notifications
		WHERE id = ?
	`, id).Scan(&item.ID, &item.ChannelKind, &item.Name, &configJSON, &enabled, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return notifications.Channel{}, errors.New("notification channel not found")
		}
		return notifications.Channel{}, err
	}
	item.Config = decodeMap(configJSON)
	item.Enabled = enabled == 1
	return item, nil
}

func (s *SQLiteStore) ListBackups() ([]backups.Backup, error) {
	rows, err := s.db.Query(`
		SELECT id, backup_kind, file_path, summary, created_at
		FROM backups
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []backups.Backup
	for rows.Next() {
		var item backups.Backup
		if err := rows.Scan(&item.ID, &item.BackupKind, &item.FilePath, &item.Summary, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SQLiteStore) CreateBackup(input backups.CreateInput) (backups.Backup, error) {
	item := backups.Backup{
		ID:         newID("backup"),
		BackupKind: input.BackupKind,
		FilePath:   strings.TrimSpace(input.FilePath),
		Summary:    input.Summary,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	_, err := s.db.Exec(`
		INSERT INTO backups (id, backup_kind, file_path, summary, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, item.ID, item.BackupKind, item.FilePath, item.Summary, item.CreatedAt)
	if err != nil {
		return backups.Backup{}, err
	}
	return item, nil
}

func (s *SQLiteStore) DeleteBackup(id string) error {
	result, err := s.db.Exec(`DELETE FROM backups WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if rows, err := result.RowsAffected(); err != nil {
		return err
	} else if rows == 0 {
		return errors.New("backup not found")
	}
	return nil
}

func (s *SQLiteStore) ListSystemSettings() ([]system.Setting, error) {
	rows, err := s.db.Query(`
		SELECT key, value_json, updated_at
		FROM system_settings
		ORDER BY key ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []system.Setting
	for rows.Next() {
		var item system.Setting
		var valueJSON string
		if err := rows.Scan(&item.Key, &valueJSON, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Value = decodeMap(valueJSON)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SQLiteStore) UpsertSystemSetting(key string, input system.UpsertSettingInput) (system.Setting, error) {
	item := system.Setting{
		Key:       strings.TrimSpace(key),
		Value:     cloneMap(input.Value),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	_, err := s.db.Exec(`
		INSERT INTO system_settings (key, value_json, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value_json = excluded.value_json, updated_at = excluded.updated_at
	`, item.Key, encodeJSON(item.Value), item.UpdatedAt)
	if err != nil {
		return system.Setting{}, err
	}
	return item, nil
}

func (s *SQLiteStore) DeleteSystemSetting(key string) error {
	result, err := s.db.Exec(`DELETE FROM system_settings WHERE key = ?`, key)
	if err != nil {
		return err
	}
	if rows, err := result.RowsAffected(); err != nil {
		return err
	} else if rows == 0 {
		return errors.New("system setting not found")
	}
	return nil
}

func (s *SQLiteStore) ListTrafficSamples(scope string, scopeID string) ([]traffic.Sample, error) {
	query := `
		SELECT id, sample_scope, scope_id, rx_bytes, tx_bytes, rate_json, recorded_at
		FROM traffic_samples
	`
	var args []any
	var filters []string
	if strings.TrimSpace(scope) != "" {
		filters = append(filters, "sample_scope = ?")
		args = append(args, scope)
	}
	if strings.TrimSpace(scopeID) != "" {
		filters = append(filters, "scope_id = ?")
		args = append(args, scopeID)
	}
	if len(filters) > 0 {
		query += " WHERE " + strings.Join(filters, " AND ")
	}
	query += " ORDER BY recorded_at DESC LIMIT 500"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []traffic.Sample
	for rows.Next() {
		var item traffic.Sample
		var rateJSON string
		if err := rows.Scan(&item.ID, &item.SampleScope, &item.ScopeID, &item.RXBytes, &item.TXBytes, &rateJSON, &item.RecordedAt); err != nil {
			return nil, err
		}
		item.Rate = decodeMap(rateJSON)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SQLiteStore) CreateTrafficSample(input traffic.CreateSampleInput) (traffic.Sample, error) {
	item := traffic.Sample{
		ID:          newID("traffic"),
		SampleScope: strings.TrimSpace(input.SampleScope),
		ScopeID:     strings.TrimSpace(input.ScopeID),
		RXBytes:     input.RXBytes,
		TXBytes:     input.TXBytes,
		Rate:        cloneMap(input.Rate),
		RecordedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	_, err := s.db.Exec(`
		INSERT INTO traffic_samples (id, sample_scope, scope_id, rx_bytes, tx_bytes, rate_json, recorded_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, item.ID, item.SampleScope, item.ScopeID, item.RXBytes, item.TXBytes, encodeJSON(item.Rate), item.RecordedAt)
	if err != nil {
		return traffic.Sample{}, err
	}
	return item, nil
}

func newID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UTC().UnixNano())
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func encodeJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func decodeStringSlice(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []string{}
	}
	return items
}

func decodeMap(raw string) map[string]any {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}
	}
	var item map[string]any
	if err := json.Unmarshal([]byte(raw), &item); err != nil {
		return map[string]any{}
	}
	return item
}

func cloneStringSlice(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}
	cloned := make([]string, len(items))
	copy(cloned, items)
	return cloned
}

func cloneMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
