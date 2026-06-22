CREATE TABLE IF NOT EXISTS iam_subject (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  subject_id VARCHAR(128) NOT NULL,
  subject_type VARCHAR(32) NOT NULL,
  display_name VARCHAR(255),
  organization_id VARCHAR(128),
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  metadata JSON,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  UNIQUE KEY uk_subject_id (subject_id),
  KEY idx_org (organization_id),
  KEY idx_type_status (subject_type, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS iam_api_key (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  key_id VARCHAR(128) NOT NULL,
  subject_id VARCHAR(128) NOT NULL,
  token_hash VARCHAR(128) NOT NULL,
  name VARCHAR(255),
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  expires_at DATETIME NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  UNIQUE KEY uk_key_id (key_id),
  UNIQUE KEY uk_token_hash (token_hash),
  KEY idx_subject (subject_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS iam_role_binding (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  subject_id VARCHAR(128) NOT NULL,
  namespace_id VARCHAR(128) NOT NULL,
  role_name VARCHAR(128) NOT NULL,
  created_at DATETIME NOT NULL,
  UNIQUE KEY uk_subject_ns_role (subject_id, namespace_id, role_name),
  KEY idx_subject (subject_id),
  KEY idx_ns_role (namespace_id, role_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS iam_policy (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  policy_id VARCHAR(128) NOT NULL,
  subject_id VARCHAR(128) NOT NULL,
  namespace_id VARCHAR(128) NOT NULL,
  resource_type VARCHAR(64) NOT NULL,
  resource_name VARCHAR(128) NOT NULL DEFAULT '*',
  action VARCHAR(128) NOT NULL,
  effect VARCHAR(16) NOT NULL DEFAULT 'allow',
  created_at DATETIME NOT NULL,
  UNIQUE KEY uk_policy_id (policy_id),
  KEY idx_subject (subject_id),
  KEY idx_ns_resource (namespace_id, resource_type, resource_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
