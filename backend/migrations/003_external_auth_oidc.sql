ALTER TABLE iam_subject ADD COLUMN email VARCHAR(255) NULL;
ALTER TABLE iam_subject ADD COLUMN external_provider VARCHAR(128) NULL;
ALTER TABLE iam_subject ADD COLUMN external_issuer VARCHAR(512) NULL;
ALTER TABLE iam_subject ADD COLUMN external_subject VARCHAR(255) NULL;
ALTER TABLE iam_subject ADD COLUMN external_username VARCHAR(255) NULL;
ALTER TABLE iam_subject ADD KEY idx_external_identity (external_provider, external_subject);

CREATE TABLE IF NOT EXISTS iam_auth_provider (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  provider_name VARCHAR(128) NOT NULL,
  provider_type VARCHAR(64) NOT NULL,
  issuer VARCHAR(512),
  audience VARCHAR(255),
  jwks_url VARCHAR(512),
  introspection_url VARCHAR(512),
  client_id VARCHAR(255),
  scopes VARCHAR(512),
  claim_mapping JSON,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  UNIQUE KEY uk_provider_name (provider_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS iam_role_mapping (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  provider_name VARCHAR(128) NOT NULL,
  external_group VARCHAR(255),
  external_role VARCHAR(255),
  subject_type VARCHAR(32),
  internal_roles JSON,
  permissions JSON,
  namespaces JSON,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  KEY idx_provider (provider_name),
  KEY idx_external_group (external_group),
  KEY idx_external_role (external_role)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
