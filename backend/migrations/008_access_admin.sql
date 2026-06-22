CREATE TABLE IF NOT EXISTS aihub_role_mapping (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  provider VARCHAR(64) NOT NULL DEFAULT 'casdoor',
  external_role VARCHAR(128) NOT NULL,
  internal_role VARCHAR(128) NOT NULL,
  source VARCHAR(64) NOT NULL DEFAULT 'db',
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  description VARCHAR(512) DEFAULT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_provider_external_internal (provider, external_role, internal_role),
  KEY idx_external_role (external_role),
  KEY idx_internal_role (internal_role)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO aihub_role_mapping(provider, external_role, internal_role, source, enabled, description) VALUES
('casdoor', 'platform-admin', 'role:admin', 'seed', 1, 'Platform admin maps to AIHub admin'),
('casdoor', 'aihub-admin', 'role:admin', 'seed', 1, 'AIHub admin from Casdoor'),
('casdoor', 'admin', 'role:admin', 'seed', 1, 'Casdoor admin role fallback'),
('casdoor', 'aihub-developer', 'role:developer', 'seed', 1, 'AIHub developer from Casdoor'),
('casdoor', 'developer', 'role:developer', 'seed', 1, 'Casdoor developer role fallback'),
('casdoor', 'aihub-reviewer', 'role:reviewer', 'seed', 1, 'AIHub reviewer from Casdoor'),
('casdoor', 'reviewer', 'role:reviewer', 'seed', 1, 'Casdoor reviewer role fallback'),
('casdoor', 'aihub-agent', 'role:agent', 'seed', 1, 'AIHub agent from Casdoor'),
('casdoor', 'agent', 'role:agent', 'seed', 1, 'Casdoor agent role fallback'),
('casdoor', 'aihub-viewer', 'role:viewer', 'seed', 1, 'AIHub viewer from Casdoor'),
('casdoor', 'viewer', 'role:viewer', 'seed', 1, 'Casdoor viewer role fallback');
