CREATE TABLE IF NOT EXISTS ai_resource (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  namespace_id VARCHAR(128) NOT NULL,
  type VARCHAR(64) NOT NULL,
  name VARCHAR(128) NOT NULL,
  display_name VARCHAR(255),
  description TEXT,
  status VARCHAR(32) NOT NULL,
  scope VARCHAR(32) DEFAULT 'private',
  owner VARCHAR(128),
  latest_version VARCHAR(64),
  stable_version VARCHAR(64),
  gray_version VARCHAR(64),
  biz_tags JSON,
  metadata JSON,
  download_count BIGINT DEFAULT 0,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  UNIQUE KEY uk_ns_type_name (namespace_id, type, name),
  KEY idx_ns_type (namespace_id, type),
  KEY idx_status (status)
);

CREATE TABLE IF NOT EXISTS ai_resource_version (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  namespace_id VARCHAR(128) NOT NULL,
  type VARCHAR(64) NOT NULL,
  name VARCHAR(128) NOT NULL,
  version VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL,
  author VARCHAR(128),
  commit_msg TEXT,
  resource_card JSON,
  storage JSON NOT NULL,
  publish_pipeline_info JSON,
  download_count BIGINT DEFAULT 0,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  UNIQUE KEY uk_ns_type_name_version (namespace_id, type, name, version),
  KEY idx_ns_type_name (namespace_id, type, name),
  KEY idx_status (status)
);

CREATE TABLE IF NOT EXISTS ai_resource_group (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  namespace_id VARCHAR(128) NOT NULL,
  name VARCHAR(128) NOT NULL,
  display_name VARCHAR(255),
  description TEXT,
  status VARCHAR(32) NOT NULL DEFAULT 'enable',
  scope VARCHAR(32) DEFAULT 'private',
  owner VARCHAR(128),
  labels JSON,
  metadata JSON,
  download_count BIGINT DEFAULT 0,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  UNIQUE KEY uk_ns_group_name (namespace_id, name)
);

CREATE TABLE IF NOT EXISTS ai_resource_group_member (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  namespace_id VARCHAR(128) NOT NULL,
  group_name VARCHAR(128) NOT NULL,
  resource_type VARCHAR(64) NOT NULL DEFAULT 'skill',
  resource_name VARCHAR(128) NOT NULL,
  version VARCHAR(64),
  label VARCHAR(64),
  required_flag TINYINT(1) NOT NULL DEFAULT 1,
  sort_order INT NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  UNIQUE KEY uk_group_resource (namespace_id, group_name, resource_type, resource_name),
  KEY idx_resource_reverse (namespace_id, resource_type, resource_name)
);

CREATE TABLE IF NOT EXISTS ai_resource_audit_log (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  namespace_id VARCHAR(128) NOT NULL,
  type VARCHAR(64),
  name VARCHAR(128),
  version VARCHAR(64),
  action VARCHAR(64) NOT NULL,
  operator VARCHAR(128),
  detail JSON,
  created_at DATETIME NOT NULL,
  KEY idx_ns_type_name (namespace_id, type, name),
  KEY idx_action (action),
  KEY idx_created_at (created_at)
);

CREATE TABLE IF NOT EXISTS ai_skill_proposal (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  proposal_id VARCHAR(128) NOT NULL,
  namespace_id VARCHAR(128) NOT NULL,
  skill_name VARCHAR(128) NOT NULL,
  base_version VARCHAR(64) NOT NULL,
  candidate_version VARCHAR(64),
  proposal_type VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL,
  source_agent_id VARCHAR(128),
  source_session_id VARCHAR(128),
  source_run_id VARCHAR(128),
  source_task_id VARCHAR(128),
  reason TEXT,
  delta_json JSON,
  evidence_json JSON,
  overlay_ref VARCHAR(512),
  created_by VARCHAR(128),
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  UNIQUE KEY uk_proposal_id (proposal_id),
  KEY idx_skill (namespace_id, skill_name),
  KEY idx_status (status)
);

CREATE TABLE IF NOT EXISTS ai_skill_overlay (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  overlay_ref VARCHAR(512) NOT NULL,
  namespace_id VARCHAR(128) NOT NULL,
  skill_name VARCHAR(128) NOT NULL,
  base_version VARCHAR(64) NOT NULL,
  proposal_id VARCHAR(128) NOT NULL,
  overlay_json JSON NOT NULL,
  status VARCHAR(32) NOT NULL,
  expires_at DATETIME NULL,
  created_at DATETIME NOT NULL,
  UNIQUE KEY uk_overlay_ref (overlay_ref),
  KEY idx_skill (namespace_id, skill_name),
  KEY idx_proposal_id (proposal_id)
);

CREATE TABLE IF NOT EXISTS ai_skill_proposal_validation (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  proposal_id VARCHAR(128) NOT NULL,
  validation_status VARCHAR(32) NOT NULL,
  score DECIMAL(8,4),
  check_result JSON,
  test_result JSON,
  created_at DATETIME NOT NULL,
  KEY idx_proposal_id (proposal_id)
);
