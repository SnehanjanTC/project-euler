package service

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// DataSourceConfig holds connection details
type DataSourceConfig struct {
	Type     string // "postgres", "mysql"
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string // "disable", "require"
}

// DataSource defines the interface for data sources
type DataSource interface {
	Connect(config DataSourceConfig) error
	Close() error
	ListTables() ([]string, error)
	PreviewData(tableName string, limit int) ([]map[string]interface{}, error)
}

// PostgresDataSource implements DataSource for PostgreSQL
type PostgresDataSource struct {
	db *sql.DB
}

func (p *PostgresDataSource) Connect(config DataSourceConfig) error {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return err
	}

	if err := db.Ping(); err != nil {
		return err
	}

	p.db = db
	return nil
}

func (p *PostgresDataSource) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

func (p *PostgresDataSource) ListTables() ([]string, error) {
	query := `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public'
		ORDER BY table_name;
	`
	rows, err := p.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}
	return tables, nil
}

func (p *PostgresDataSource) PreviewData(tableName string, limit int) ([]map[string]interface{}, error) {
	// WARNING: VULNERABLE TO SQL INJECTION IF tableName IS UNTRUSTED
	// In a real app, validate tableName against ListTables() whitelist
	query := fmt.Sprintf("SELECT * FROM %s LIMIT %d", tableName, limit)

	rows, err := p.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}

	for rows.Next() {
		// Prepare a slice of interface{} to hold values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		// Convert to map
		rowMap := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]

			// Handle byte slices (common for strings in DB drivers)
			if b, ok := val.([]byte); ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = val
			}
		}
		result = append(result, rowMap)
	}

	return result, nil
}
