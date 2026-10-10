docker run -d -p 127.0.0.1:5432:5432 -e "POSTGRES_PASSWORD=postgres" -e "POSTGRES_INITDB_ARGS=--lc-collate C" postgres:15

./bin/tests-runner serve --server.addr=:9091 --server.auth.disable --db.host=127.0.0.1 --db.database=postgres --db.dialect=postgres --db.user=postgres --db.password=postgres
