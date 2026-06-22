CREATE TABLE IF NOT EXISTS aihub_namespace (
  namespace_id VARCHAR(128) PRIMARY KEY,
  display_name VARCHAR(255),
  description TEXT,
  owner VARCHAR(128),
  visibility VARCHAR(32) DEFAULT 'PRIVATE',
  metadata JSON,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);
CREATE TABLE IF NOT EXISTS aihub_namespace_member (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  namespace_id VARCHAR(128) NOT NULL,
  subject_id VARCHAR(128) NOT NULL,
  subject_type VARCHAR(32),
  display_name VARCHAR(255),
  roles JSON,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  UNIQUE KEY uk_ns_subject(namespace_id, subject_id),
  KEY idx_subject(subject_id)
);
CREATE TABLE IF NOT EXISTS aihub_star (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  namespace_id VARCHAR(128) NOT NULL,
  skill_name VARCHAR(128) NOT NULL,
  subject_id VARCHAR(128) NOT NULL,
  created_at DATETIME NOT NULL,
  UNIQUE KEY uk_star(namespace_id, skill_name, subject_id),
  KEY idx_skill(namespace_id, skill_name)
);
CREATE TABLE IF NOT EXISTS aihub_rating (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  namespace_id VARCHAR(128) NOT NULL,
  skill_name VARCHAR(128) NOT NULL,
  subject_id VARCHAR(128) NOT NULL,
  rating INT NOT NULL,
  comment TEXT,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  UNIQUE KEY uk_rating(namespace_id, skill_name, subject_id),
  KEY idx_skill(namespace_id, skill_name)
);
CREATE TABLE IF NOT EXISTS aihub_subscription (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  namespace_id VARCHAR(128) NOT NULL,
  target_type VARCHAR(32) NOT NULL,
  target_name VARCHAR(128) NOT NULL,
  subject_id VARCHAR(128) NOT NULL,
  created_at DATETIME NOT NULL,
  UNIQUE KEY uk_sub(namespace_id, target_type, target_name, subject_id),
  KEY idx_subject(subject_id)
);
CREATE TABLE IF NOT EXISTS aihub_audit_log (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  log_id VARCHAR(128) NOT NULL,
  namespace_id VARCHAR(128),
  resource_type VARCHAR(64),
  resource_name VARCHAR(128),
  version VARCHAR(64),
  action VARCHAR(128) NOT NULL,
  operator VARCHAR(128),
  detail JSON,
  request_id VARCHAR(128),
  created_at DATETIME NOT NULL,
  UNIQUE KEY uk_log_id(log_id),
  KEY idx_resource(namespace_id, resource_type, resource_name),
  KEY idx_action(action),
  KEY idx_created_at(created_at)
);
CREATE TABLE IF NOT EXISTS aihub_token (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  key_id VARCHAR(128) NOT NULL,
  name VARCHAR(128),
  subject_id VARCHAR(128) NOT NULL,
  subject_type VARCHAR(32),
  roles JSON,
  permissions JSON,
  namespaces JSON,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  token_hash VARCHAR(128),
  expires_at DATETIME,
  last_used_at DATETIME,
  created_at DATETIME NOT NULL,
  UNIQUE KEY uk_key_id(key_id),
  UNIQUE KEY uk_token_hash(token_hash),
  KEY idx_subject(subject_id)
);
CREATE TABLE IF NOT EXISTS aihub_idempotency (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  idempotency_key VARCHAR(191) NOT NULL,
  method VARCHAR(16) NOT NULL,
  path VARCHAR(512) NOT NULL,
  request_hash VARCHAR(128),
  status_code INT,
  response_body MEDIUMTEXT,
  created_at DATETIME NOT NULL,
  expires_at DATETIME NOT NULL,
  UNIQUE KEY uk_idempotency_key(idempotency_key),
  KEY idx_expires(expires_at)
);
