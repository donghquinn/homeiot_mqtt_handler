package postgres

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/donghquinn/gqbd"
	_ "github.com/lib/pq"
	"org.donghyuns.com/mqtt/listner/configs"
)

type PostgresService struct {
	Client *sql.DB
}

func NewPostgresConnector(cfg configs.PostgresConfig) (*PostgresService, error) {
	sslMode := "disable"
	maxConn := 20
	maxIdleconn := 10
	maxLifeTime := 30 * time.Minute
	maxIdleLifeTime := 5 * time.Minute

	port, err := strconv.Atoi(cfg.DbPort)
	if err != nil {
		port = 5432
	}

	conn, err := sql.Open("postgres", gqbd.BuildConnectionString(gqbd.PostgreSQL, gqbd.DBConfig{
		Host:     cfg.DbHost,
		Port:     port,
		Password: cfg.DbPasswd,
		User:     cfg.DbUser,
		DBName:   cfg.DbName,
		SSLMode:  sslMode,
	}))
	if err != nil {
		return nil, fmt.Errorf("postgres connector init err: %v", err)
	}

	conn.SetMaxOpenConns(maxConn)
	conn.SetMaxIdleConns(maxIdleconn)
	conn.SetConnMaxLifetime(maxLifeTime)
	conn.SetConnMaxIdleTime(maxIdleLifeTime)

	return &PostgresService{
		Client: conn,
	}, nil
}

func (p *PostgresService) CheckConnection() error {
	defer p.Client.Close()
	if err := p.Client.Ping(); err != nil {
		return fmt.Errorf("check database connection err: %v", err)
	}
	return nil
}
