module github.com/swetjen/virtuous/pgtypetest

go 1.25

require (
	github.com/jackc/pgtype v1.14.4
	github.com/jackc/pgx/v5 v5.8.0
	github.com/swetjen/virtuous v0.0.0
)

require github.com/jackc/pgio v1.0.0 // indirect

replace github.com/swetjen/virtuous => ../
