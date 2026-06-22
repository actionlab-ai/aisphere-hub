CREATE TABLE IF NOT EXISTS aihub_model_profile (
  namespace_id VARCHAR(128) NOT NULL,
  id VARCHAR(191) NOT NULL,
  payload_json JSON NOT NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY(namespace_id, id),
  KEY idx_aihub_model_profile_updated(updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
