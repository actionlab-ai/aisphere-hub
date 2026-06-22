package objectstore

// S3Config is used by both AWS S3 and MinIO. The concrete adapter should be
// implemented behind the ObjectStore port, not leaked into service code.
type S3Config struct {
	Endpoint       string `yaml:"endpoint" json:"endpoint"`
	Region         string `yaml:"region" json:"region"`
	AccessKey      string `yaml:"accessKey" json:"accessKey"`
	SecretKey      string `yaml:"secretKey" json:"secretKey"`
	Bucket         string `yaml:"bucket" json:"bucket"`
	UseSSL         bool   `yaml:"useSSL" json:"useSSL"`
	ForcePathStyle bool   `yaml:"forcePathStyle" json:"forcePathStyle"`
	PresignBaseURL string `yaml:"presignBaseURL" json:"presignBaseURL"`
}
