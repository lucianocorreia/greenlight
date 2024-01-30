# connect to db
.PHONY: psql
psql:
	/Applications/Postgres.app/Contents/Versions/16/bin/psql -p5432 "greenlight"
