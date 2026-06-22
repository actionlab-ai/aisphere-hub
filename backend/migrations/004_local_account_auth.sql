-- Local AIHub account mode. The current implementation can use config/file storage;
-- these tables are reserved for DB-backed local accounts and service accounts.
CREATE TABLE IF NOT EXISTS iam_local_account (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  username VARCHAR(128) NOT NULL,
  subject_id VARCHAR(128) NOT NULL,
  subject_type VARCHAR(32) NOT NULL DEFAULT 'human',
  display_name VARCHAR(255),
  email VARCHAR(255),
  organization VARCHAR(128),
  password_hash VARCHAR(512) NOT NULL,
  roles JSON,
  permissions JSON,
  namespaces JSON,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  last_login_at DATETIME,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  UNIQUE KEY uk_username (username),
  UNIQUE KEY uk_subject_id (subject_id),
  KEY idx_org (organization),
  KEY idx_status (status)
);

CREATE TABLE IF NOT EXISTS iam_service_account (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  account_id VARCHAR(128) NOT NULL,
  account_type VARCHAR(32) NOT NULL DEFAULT 'agent',
  display_name VARCHAR(255),
  organization VARCHAR(128),
  roles JSON,
  permissions JSON,
  namespaces JSON,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  UNIQUE KEY uk_account_id (account_id),
  KEY idx_org (organization),
  KEY idx_status (status)
);
