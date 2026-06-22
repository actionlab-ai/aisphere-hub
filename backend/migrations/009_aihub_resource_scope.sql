-- Destructive branch schema extension for AIHub unified resources.
-- In test environments you can also rebuild the database from scratch.

ALTER TABLE ai_resource
  ADD COLUMN app VARCHAR(64) NOT NULL DEFAULT 'aihub' AFTER name,
  ADD COLUMN org_id VARCHAR(128) NULL AFTER app,
  ADD COLUMN project_id VARCHAR(128) NULL AFTER org_id,
  ADD COLUMN owner_subject VARCHAR(191) NULL AFTER owner;

CREATE INDEX idx_ai_resource_app_type ON ai_resource(app, type);
CREATE INDEX idx_ai_resource_org_project ON ai_resource(org_id, project_id);
CREATE INDEX idx_ai_resource_owner_subject ON ai_resource(owner_subject);

-- Future canonical table names. The current Go store still writes ai_resource_group
-- for compatibility inside this transition branch; these tables are reserved for
-- the next cut when SkillSet storage is fully detached from legacy group naming.
CREATE TABLE IF NOT EXISTS aihub_skillset (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  namespace_id VARCHAR(128) NOT NULL DEFAULT '_global',
  app VARCHAR(64) NOT NULL DEFAULT 'aihub',
  org_id VARCHAR(128) NULL,
  project_id VARCHAR(128) NULL,
  name VARCHAR(128) NOT NULL,
  display_name VARCHAR(255),
  description TEXT,
  status VARCHAR(32) NOT NULL DEFAULT 'enable',
  visibility VARCHAR(32) NOT NULL DEFAULT 'private',
  owner_subject VARCHAR(191),
  labels JSON,
  metadata JSON,
  download_count BIGINT DEFAULT 0,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  UNIQUE KEY uk_skillset_name (namespace_id, name),
  KEY idx_scope (org_id, project_id),
  KEY idx_owner (owner_subject)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS aihub_skillset_member (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  namespace_id VARCHAR(128) NOT NULL DEFAULT '_global',
  skillset_name VARCHAR(128) NOT NULL,
  resource_type VARCHAR(64) NOT NULL DEFAULT 'skill',
  resource_name VARCHAR(128) NOT NULL,
  version VARCHAR(64),
  label VARCHAR(64),
  required_flag TINYINT(1) NOT NULL DEFAULT 1,
  sort_order INT NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  UNIQUE KEY uk_skillset_resource (namespace_id, skillset_name, resource_type, resource_name),
  KEY idx_resource_reverse (namespace_id, resource_type, resource_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS aihub_agent (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  namespace_id VARCHAR(128) NOT NULL DEFAULT '_global',
  app VARCHAR(64) NOT NULL DEFAULT 'aihub',
  org_id VARCHAR(128) NULL,
  project_id VARCHAR(128) NULL,
  agent_id VARCHAR(128) NOT NULL,
  display_name VARCHAR(255),
  description TEXT,
  owner_subject VARCHAR(191),
  status VARCHAR(32) NOT NULL DEFAULT 'enable',
  metadata JSON,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  UNIQUE KEY uk_agent_id (namespace_id, agent_id),
  KEY idx_scope (org_id, project_id),
  KEY idx_owner (owner_subject)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS aihub_workflow (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  namespace_id VARCHAR(128) NOT NULL DEFAULT '_global',
  app VARCHAR(64) NOT NULL DEFAULT 'aihub',
  org_id VARCHAR(128) NULL,
  project_id VARCHAR(128) NULL,
  workflow_id VARCHAR(128) NOT NULL,
  agent_id VARCHAR(128),
  display_name VARCHAR(255),
  description TEXT,
  owner_subject VARCHAR(191),
  status VARCHAR(32) NOT NULL DEFAULT 'enable',
  definition JSON,
  metadata JSON,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  UNIQUE KEY uk_workflow_id (namespace_id, workflow_id),
  KEY idx_agent (namespace_id, agent_id),
  KEY idx_scope (org_id, project_id),
  KEY idx_owner (owner_subject)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS aihub_run (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  namespace_id VARCHAR(128) NOT NULL DEFAULT '_global',
  app VARCHAR(64) NOT NULL DEFAULT 'aihub',
  org_id VARCHAR(128) NULL,
  project_id VARCHAR(128) NULL,
  run_id VARCHAR(128) NOT NULL,
  workflow_id VARCHAR(128),
  agent_id VARCHAR(128),
  status VARCHAR(32) NOT NULL,
  created_by VARCHAR(191),
  metadata JSON,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  UNIQUE KEY uk_run_id (namespace_id, run_id),
  KEY idx_workflow (namespace_id, workflow_id),
  KEY idx_agent (namespace_id, agent_id),
  KEY idx_scope (org_id, project_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
