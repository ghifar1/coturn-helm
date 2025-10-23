# Repository Guidelines

## Project Structure & Module Organization
Core sources live in `src/`, with `apps/relay/` providing the TURN server entry point, `client/` and `client++/` exposing C/C++ client libraries, and `server/` holding shared networking primitives. Operational assets sit in `docker/` (container builds), `scripts/` (packaging and release helpers), and `examples/scripts/` for runnable TURN/STUN scenarios. Reference material—including flow diagrams, configuration primers, and database guides—is in `docs/`, while manual pages are maintained under `man/`.

## Build, Test, and Development Commands
- `./configure && make` generates and compiles the Autotools build (preferred when matching packaged layouts).  
- `make install` or `sudo make install` deploys binaries, configs, and the sample SQLite DB; use `make clean` before switching toolchains.  
- `cmake -S . -B build` followed by `cmake --build build` offers an out-of-tree alternative; append `--target install-runtime` to install only runtime artifacts.  
- `docker build docker/coturn -t coturn/local` reproduces the published container image for integration testing.

## Coding Style & Naming Conventions
Follow the existing C style: four-space indentation, braces on their own line, and `lower_snake_case` for functions and variables. Keep headers guarded, prefer `const` correctness, and reuse helpers from `src/server/` instead of duplicating socket logic. Macros stay uppercase with succinct names (`TURNDBDIR`, `TLS_SUPPORTED`). No automatic formatter runs in CI, so match nearby code and update SPDX headers when creating new files.

## Testing Guidelines
Run protocol checks with `examples/scripts/rfc5769.sh` to validate STUN/TURN message encoding. Exercise the minimal TURN flow via `examples/scripts/basic/relay.sh` (server) plus `examples/scripts/basic/udp_c2c_client.sh` (client); extend to long-term credential scenarios with the scripts in `examples/scripts/longtermsecure/`. Capture regressions with Wireshark traces when modifying packet parsing, and document any new scenarios in `docs/Testing.md`.

## Commit & Pull Request Guidelines
Recent history favors concise, capitalized summaries such as `Update Debian "trixie" …` or `Fix memory leak using libevent`; mirror that style, referencing CVE IDs or script names when relevant. Each PR should describe the change, list manual or scripted tests, link GitHub issues, and include configuration or deployment implications. Screenshots are only expected for docs with rendered diagrams; otherwise attach logs when altering network flows.

## Security & Configuration Tips
Avoid committing runtime secrets; rely on the sample configs in `examples/` and point operators to `docs/Configuration.md` and `docs/PostInstall.md`. When touching authentication or database code, cross-check encryptions against the guidance in `docs/Mongo.md`, `docs/PostgreSQL.md`, and `docs/Redis.md`, and note any schema migrations in `turndb/`.
