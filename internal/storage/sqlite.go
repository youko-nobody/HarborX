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
	"harborx/internal/features/nodes"
	"harborx/internal/features/rules"
	"harborx/internal/features/subscriptions"
	"harborx/internal/features/templates"
)

//go:embed schema.sql seeds.sql
var migrationFiles embed.FS

type SQLiteStore struct {
	db *sql.DB
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
