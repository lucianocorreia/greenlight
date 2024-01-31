package main

import (
	"context"
	"database/sql"
	"expvar"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/lucianocorreia/greenlight/internal/data"
	"github.com/lucianocorreia/greenlight/internal/jsonlog"
	"github.com/lucianocorreia/greenlight/internal/mailer"
)

const (
	version = "1.0.0"
)

var buildTime string

type config struct {
	port int
	env  string
	db   struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  string
	}

	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}

	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}

	cors struct {
		trustedOrigins []string
	}
}

type application struct {
	config config
	logger *jsonlog.Logger
	models data.Models
	mailer mailer.Mailer
	wg     sync.WaitGroup
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	var cfg config

	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")

	dsn := os.Getenv("GREENLIGHT_DB_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost/greenlight?sslmode=disable"
	}

	flag.StringVar(&cfg.db.dsn, "db-dsn", dsn, "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")

	// rate limiter
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")

	// smtp

	mailerPort := os.Getenv("MAILER_PORT")
	if mailerPort == "" {
		mailerPort = "25"
	}
	p, error := strconv.Atoi(mailerPort)
	if error != nil {
		log.Fatal("Error converting mailer port to int")
	}

	flag.StringVar(&cfg.smtp.host, "smtp-host", os.Getenv("MAILER_HOST"), "SMTP server hostname")
	flag.IntVar(&cfg.smtp.port, "smtp-port", p, "SMTP server port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", os.Getenv("MAILER_USERNAME"), "SMTP server username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", os.Getenv("MAILER_PASSWORD"), "SMTP server password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "no-replay@greenlight.example", "SMTP sender email address")

	var trustedOrigins string
	flag.StringVar(&trustedOrigins, "cors-trusted-origins", "", "CORS trusted origins (space separated)")
	cfg.cors.trustedOrigins = strings.Fields(trustedOrigins)

	displayVersion := flag.Bool("version", false, "Display version and exit")

	flag.Parse()

	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	if *displayVersion {
		fmt.Printf("Version:\t%s\n", version)
		fmt.Printf("Build time:\t%s\n", buildTime)
		os.Exit(0)
	}

	// Connect to the database
	db, err := openDB(cfg)
	if err != nil {
		logger.Fatal(err, nil)
	}
	defer db.Close()
	logger.Info("database connection pool established", nil)

	expvar.NewString("version").Set(version)

	expvar.Publish("goroutines", expvar.Func(func() interface{} {
		return runtime.NumGoroutine()
	}))

	expvar.Publish("database", expvar.Func(func() interface{} {
		return db.Stats()
	}))

	expvar.Publish("memory", expvar.Func(func() interface{} {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		return m
	}))

	expvar.Publish("timestamp", expvar.Func(func() interface{} {
		return time.Now().Unix()
	}))

	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
	}

	// srv := &http.Server{
	// 	Addr:         fmt.Sprintf(":%d", cfg.port),
	// 	Handler:      app.routes(),
	// 	ErrorLog:     log.New(logger, "", 0),
	// 	IdleTimeout:  time.Minute,
	// 	ReadTimeout:  10 * time.Second,
	// 	WriteTimeout: 30 * time.Second,
	// }

	// logger.Info("Starting server", map[string]string{
	// 	"addr": srv.Addr,
	// 	"env":  cfg.env,
	// })
	// err = srv.ListenAndServe()

	err = app.serve()
	if err != nil {
		logger.Fatal(err, nil)
	}
}

func openDB(cfg config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	// setup connection pool
	db.SetMaxOpenConns(cfg.db.maxOpenConns)
	db.SetMaxIdleConns(cfg.db.maxIdleConns)
	duration, err := time.ParseDuration(cfg.db.maxIdleTime)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxIdleTime(duration)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}
