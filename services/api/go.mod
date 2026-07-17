module github.com/devopsaccess-in/devopsaccess.in/services/api

go 1.25.0

require (
	github.com/devopsaccess-in/devopsaccess.in/services/shared v0.0.0
	github.com/go-chi/chi/v5 v5.2.0
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/jackc/pgx/v5 v5.7.2
	github.com/rs/zerolog v1.33.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
)

// Resolved by the go.work workspace during development; the replace keeps
// standalone `go build ./services/api` working in CI.
replace github.com/devopsaccess-in/devopsaccess.in/services/shared => ../shared
