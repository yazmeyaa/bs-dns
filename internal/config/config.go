package config

type DBConfig struct {
	FilePath string
}

type RedisConfig struct {
	Host     string
	Username string
	Password string
	Database int
}

type Config struct {
	DB    DBConfig
	Redis RedisConfig
}

func New() (*Config, error) {
	config := &Config{}

	return config, nil
}
