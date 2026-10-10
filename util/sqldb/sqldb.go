package sqldb

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"

	"tests-runner/util/logger"
	"tests-runner/util/util/errutil"
)

// ConnectionFlags contains command line flags required to connect to a database
type ConnectionFlags struct {
	Dialect         string `json:"dialect" arg:"env:DB_DIALECT"`
	Host            string `json:"host" arg:"env:DB_HOST"`
	User            string `json:"user" arg:"env:DB_USER"`
	Password        string `json:"password" arg:"env:DB_PASSWORD"`
	Database        string `json:"database" arg:"env:DB_DATABASE"`
	CredentialsFile string `json:"credentials-file" arg:"env:DB_CREDENTIALS_FILE"`
	// SSLMode is a postgres-specific parameter to configure
	// encryption on the connection.
	//
	// For possible values see: https://www.postgresql.org/docs/current/libpq-ssl.html#LIBPQ-SSL-PROTECTION
	//
	// By default it is "disable".
	// Recommended value: "verify-full".
	SSLMode string `json:"sslmode" arg:"env:DB_SSLMODE"`
}

// ConfigFlags contains command line flags describing DB configuration
type ConfigFlags struct {
	MaxOpenConnections    int           `json:"max_open_connections" arg:"env:DB_MAX_OPEN_CONNECTIONS"`
	MaxConnectionLifetime time.Duration `json:"max_connection_lifetime" arg:"env:DB_MAX_CONNECTION_LIFETIME"`
	NoTracing             bool          `json:"no_tracing" arg:"env:DB_NO_TRACING"`
}

// LoggingFlags contains flags needed to setup DB logging
type LoggingFlags struct {
	// LogEnable configures if DB can print anything to logs at all.
	// If set to false, then all logs from database are silenced.
	LogEnable bool `json:"log_enable" arg:"env:DB_LOG_ENABLE"`
	// LogLevel configures what severity is interesting for user
	// to see in logs. For example, if set to logger.Error, then
	// only errors occurring within database layer will be printed.
	//
	// Default level is info.
	//
	// To see all sql queries in logs user should use logger.Debug
	// level.
	LogLevel              logger.Level  `json:"log_level" arg:"env:DB_LOG_LEVEL"`
	LogSlowQueryThreshold time.Duration `json:"log_slow_query_threshold" arg:"env:DB_LOG_SLOW_QUERY_THRESHOLD"`
}

// Flags contains command line flags needed to instantiate SQL DB
type Flags struct {
	ConnectionFlags
	ConfigFlags
	LoggingFlags
}

func defaultInt(val int, env string) int {
	if env = os.Getenv(env); env == "" {
		return val
	}

	v, err := strconv.ParseInt(env, 10, 32)
	if err != nil {
		return val
	}
	return int(v)
}

func defaultDur(val time.Duration, env string) time.Duration {
	if env = os.Getenv(env); env == "" {
		return val
	}

	v, err := time.ParseDuration(env)
	if err != nil {
		return val
	}
	return v
}

func defaultBool(val bool, env string) bool {
	if env = os.Getenv(env); env == "" {
		return val
	}

	v, err := strconv.ParseBool(env)
	if err != nil {
		return val
	}
	return v
}

// InjectFlags injects sqldb related flags into the given flag set.
func InjectFlags(fs *pflag.FlagSet, f *Flags) {
	dialects := gDialects.getRegisteredDialectsNames()

	fs.StringVarP(&f.Dialect, "db.dialect", "", os.Getenv("DB_DIALECT"), "One of the supported SQL dialects: "+strings.Join(dialects, ","))
	fs.StringVarP(&f.Host, "db.host", "", os.Getenv("DB_HOST"), "Database server host address or filename for local DB")
	fs.StringVarP(&f.User, "db.user", "", os.Getenv("DB_USER"), "Database server username")
	fs.StringVarP(&f.Password, "db.password", "", os.Getenv("DB_PASSWORD"), "Database server password")
	fs.StringVarP(&f.Database, "db.database", "", os.Getenv("DB_DATABASE"), "Database name")
	fs.StringVarP(&f.SSLMode, "db.sslmode", "", os.Getenv("DB_SSLMODE"), "Postgres sslmode")
	fs.StringVarP(&f.CredentialsFile, "db.credentials-file", "", os.Getenv("DB_CREDENTIALS_FILE"), "Database cloud credentials file")

	fs.IntVarP(&f.MaxOpenConnections, "db.max-open-connections", "", defaultInt(4, "DB_MAX_OPEN_CONNECTIONS"),
		"Maximum number of simultaneous connections to DB server")
	fs.DurationVarP(&f.MaxConnectionLifetime, "db.max-connection-lifetime", "", defaultDur(1*time.Hour, "DB_MAX_CONNECTION_LIFETIME"),
		"Maximum amount of time a connection may be reused")

	fs.BoolVarP(&f.LogEnable, "db.log.enable", "", defaultBool(false, "DB_LOG_ENABLE"), "Enable database logging")

	if logLevel := os.Getenv("DB_LOG_LEVEL"); logLevel != "" {
		err := f.LogLevel.Set(logLevel)
		errutil.FatalIf(err)
	}
	fs.VarP(&f.LogLevel, "db.log.level", "", "Database log level")

	fs.DurationVarP(&f.LogSlowQueryThreshold, "db.log.slow-query-threshold", "", defaultDur(0, "DB_LOG_SLOW_QUERY_THRESHOLD"),
		"Log queries that took longer than the specified threshold")
}
