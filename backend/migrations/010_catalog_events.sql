CREATE TABLE IF NOT EXISTS aihub_catalog_event (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  app VARCHAR(64) NOT NULL DEFAULT 'aihub',
  event_type VARCHAR(64) NOT NULL,
  object VARCHAR(255) NOT NULL,
  resource_type VARCHAR(64) NOT NULL,
  resource_id VARCHAR(128) NOT NULL,
  skillset_name VARCHAR(128) DEFAULT NULL,
  version VARCHAR(64) DEFAULT NULL,
  revision VARCHAR(128) DEFAULT NULL,
  payload JSON DEFAULT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_catalog_event_app_id(app, id),
  KEY idx_catalog_event_skillset(skillset_name, id),
  KEY idx_catalog_event_resource(resource_type, resource_id, id),
  KEY idx_catalog_event_type(event_type, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS aihub_runtime_report (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  runtime_id VARCHAR(128) NOT NULL,
  hostname VARCHAR(255) DEFAULT NULL,
  skillset_name VARCHAR(128) DEFAULT NULL,
  revision VARCHAR(128) DEFAULT NULL,
  snapshot_id VARCHAR(128) DEFAULT NULL,
  skills_json JSON DEFAULT NULL,
  metadata JSON DEFAULT NULL,
  reported_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_runtime_report_runtime(runtime_id, reported_at),
  KEY idx_runtime_report_skillset(skillset_name, reported_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
