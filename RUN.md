docker run -d -p 127.0.0.1:5432:5432 -e "POSTGRES_PASSWORD=postgres" -e "POSTGRES_INITDB_ARGS=--lc-collate C" postgres:15

./bin/tests-runner serve --server.addr=:9091 --server.auth.disable --db.host=127.0.0.1 --db.database=postgres --db.dialect=postgres --db.user=postgres --db.password=postgres

# Rust solutions

Solutions written in Rust (see exercises/RUST.md) are built on the grading host
by `make -f Makefile.rust build`, like C solutions. The host needs, besides gcc and make:

- rustc 1.85 or newer on PATH (Debian 13's rustc works)
- bindgen 0.71 or newer (Debian 13's package `bindgen`) and the Valgrind headers (`/usr/include/valgrind`)
- the headers the C builds use: liburing (03), libext2fs (04, 05, 06, 08, 10) and ntfs-3g (07);
  bindgen generates the Rust bindings for them at build time
- curl, tar and sha256sum, and network access to static.crates.io.

The crates listed in exercises/src/stdlib/crates.mk are downloaded and compiled on
demand into $XDG_CACHE_HOME/fs101-rust (default ~/.cache/fs101-rust of the user that
runs `tests-runner serve`; override with FS101_RUST_CACHE)
