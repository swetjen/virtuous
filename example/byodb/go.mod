module github.com/swetjen/virtuous/example/byodb

go 1.25.0

require (
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/jackc/pgx/v5 v5.8.0
	github.com/swetjen/virtuous v0.0.0
	golang.org/x/crypto v0.49.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/text v0.35.0 // indirect
)

replace github.com/swetjen/virtuous => ../..
