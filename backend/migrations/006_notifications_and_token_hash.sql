ALTER TABLE aihub_token ADD COLUMN token_hash VARCHAR(128) NULL;
ALTER TABLE aihub_token ADD UNIQUE KEY uk_token_hash_2(token_hash);

CREATE TABLE IF NOT EXISTS aihub_notification (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  notification_id VARCHAR(128) NOT NULL,
  namespace_id VARCHAR(128),
  subject_id VARCHAR(128) NOT NULL,
  target_type VARCHAR(32),
  target_name VARCHAR(128),
  event_type VARCHAR(64) NOT NULL,
  title VARCHAR(255),
  message TEXT,
  payload JSON,
  read_flag BOOLEAN NOT NULL DEFAULT FALSE,
  created_at DATETIME NOT NULL,
  UNIQUE KEY uk_notification_id(notification_id),
  KEY idx_subject_read(subject_id, read_flag),
  KEY idx_target(namespace_id, target_type, target_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
