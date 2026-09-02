# 🛡️ @telefraudbot - Telegram Anti-Fraud & Group Protection Bot

`@telefraudbot` is an enterprise-grade Telegram bot built in **Go** using the [`github.com/mymmrac/telego`](https://github.com/mymmrac/telego) framework, powered by **PostgreSQL** with automated **daily backups**, dynamic **Reply & Inline Keyboards**, and strictly formatted with **HTML Parse Mode**.

Developed by **xspoilt** (`@xspoilt`), `@telefraudbot` enables Telegram users to report scam accounts while equipping group administrators and bot admins with automated real-time auto-banning, member join checks, daily background member audits, and an expanded administrative control suite.

---

## ⚡ Feature Matrix

- **🔍 Multi-Identifier Reporting**:
  - Report fraudsters using Telegram Username (`@scammer`), User ID (`123456789`), or Phone Number (`+1234567890`).
  - Interactive multi-step report wizard (Category selection, description, proof upload with photos, documents, and videos).

- **🧬 Identity Linking & Merge**:
  - Every scammer is a canonical entity linked to all known identifiers — User ID, `@username`, and phone number — so an account that changes its username is still caught by its immutable User ID, and vice versa.
  - Newly discovered identifiers (e.g. a username mapped to a user id) are linked automatically; separately-reported identities that share an identifier are merged.

- **⌨️ Hybrid Reply & Inline Keyboard System**:
  - **Persistent Reply Keyboard** in PM for main navigation (`🛡️ Report Fraudster`, `🔍 Check Identifier`, `📋 My Submissions`, `👨‍💻 Developer Info`).
  - **Contextual Inline Keyboards** for report wizards, admin approval queues, group settings toggles, and scammer purge buttons.

- **🛡️ Real-Time Group Protection & Scanning Engines**:
  - **⚡ Instant Join Gate (`OnChatMemberUpdated`)**: Auto-scans every user joining or added to a group. Immediately auto-bans blacklisted scammers upon entry.
  - **⏰ Daily Group Scanner Worker**: Background worker running every day at `03:00 AM UTC` to audit all member lists in protected groups.
  - **Real-Time Message Interceptor**: Inspects sender IDs, mentioned handles, and phone numbers in group chat messages.

- **👑 Advanced Admin Control Suite**:
  - **Admin Role Management**: Add/remove system admins (`/admin_add`, `/admin_remove`, `/admin_list`).
  - **📢 Global Broadcast Engine**: Rate-limited HTML mass announcements across all users & groups with live progress indicators (`/broadcast`).
  - **📂 CSV/JSON Import & Export**: Bulk import or download blacklisted database records (`/export_blacklist`, `/import_blacklist`).
  - **⚙️ Dynamic Enforcement Modes**: Toggle between `BAN`, `KICK`, `WARN`, or `SILENT` global moderation modes (`/mode`).

- **👨‍💻 Developer Attribution**:
  - Built-in credits placement for lead developer **xspoilt** (`@xspoilt`).
  - `/dev` command and main menu keyboard button for system information and developer info.

- **💾 Enterprise PostgreSQL & Automated Daily Backup**:
  - Relational schema optimized with indexing for instant identifier lookups.
  - Integrated daily automated `pg_dump` worker with compressed `.sql.gz` backups and retention rotation.

---

## 🛠️ Technology Stack

| Component | Technology | Description |
| :--- | :--- | :--- |
| **Language** | Go 1.25+ | High performance, concurrency, strong typing |
| **Telegram Framework** | `github.com/mymmrac/telego` | Fast Telegram Bot API framework |
| **Database** | PostgreSQL 15+ (`pgx/v5`) | Relational storage with sub-millisecond query indexes |
| **Parse Mode** | HTML (`telego.ModeHTML`) | Rich formatted notifications & UI strings |
| **Developer** | xspoilt (`@xspoilt`) | Project architecture & design |
| **Backup Engine** | `pg_dump` + Cron Worker | Daily compressed database dumps & retention management |

---

## 📁 Repository Structure

```tree
telefraud/
├── cmd/
│   └── telefraud/
│       └── main.go             # Application entrypoint & wiring
├── config/
│   └── config.go               # Configuration loader & env parser
├── internal/
│   ├── bot/
│   │   └── bot.go              # Telego client & update loop
│   ├── database/
│   │   ├── database.go         # pgx pool + embedded migration runner
│   │   ├── store.go            # PostgreSQL identity.Store
│   │   └── migrations/
│   │       └── 0001_init.sql   # Identity-aware schema
│   ├── handlers/
│   │   ├── handlers.go         # Message dispatch & reply keyboard
│   │   ├── user.go             # /start, /dev, /check, /help
│   │   └── group.go            # Join-gate scan & auto-ban
│   ├── models/
│   │   ├── models.go           # Domain structs & constants
│   │   └── ident.go            # Identifier normalization
│   └── services/
│       └── identity/
│           └── identity.go     # Resolve / link / merge logic
├── docs/                       # Project documentation
│   ├── ARCHITECTURE.md         # Architecture design & system sequence
│   ├── DATABASE_SCHEMA.md      # PostgreSQL DDL & index strategies
│   ├── BOT_COMMANDS.md         # Comprehensive bot command & UI manual
│   └── BACKUP_STRATEGY.md      # Daily backup execution & recovery guide
├── scripts/
│   ├── backup.sh               # Shell script for manual/cron pg_dump
│   └── init_db.sql             # SQL initialization (mirrors migrations)
├── Dockerfile                  # Multi-stage build → minimal runtime image
├── docker-compose.yml          # Postgres + bot stack (pgdata/backups volumes)
├── .dockerignore               # Excludes secrets from the build context
├── .gitignore                  # Excludes .env and backups from git
├── .env.example                # Environment template (copy to .env)
├── go.mod                      # Go module definition
├── go.sum                      # Dependency checksums
└── README.md                   # Project overview & documentation index
```

---

## 🚀 Quick Start Guide

### Option A: Docker Compose (recommended)

```bash
# 1. Create your env file and set BOT_TOKEN.
cp .env.example .env

# 2. Start just PostgreSQL (for testing / psql access).
docker compose up -d db

# 3. Start the full stack — Postgres + bot.
#    The schema is applied automatically on boot (embedded migrations).
docker compose up -d --build

# Follow logs / stop.
docker compose logs -f bot
docker compose down
```

Data persists in the `pgdata` volume; backups persist in `backups`. The `db`
service publishes `5432` for local `psql` access.

### Option B: Run on the host (development)

The bot auto-loads a `.env` file from the working directory, so after
`cp .env.example .env` and setting `BOT_TOKEN`, you can run with a single
command against a local PostgreSQL 15+:

```bash
# Initialize the database once.
psql -U telefraud -d telefraud_db -f scripts/init_db.sql

# Run (the main package lives under cmd/telefraud, not the repo root).
go run ./cmd/telefraud
```

Or use the Makefile targets:

```bash
make dev          # go run ./cmd/telefraud
make test         # go test ./...
make build        # compile to ./bin/telefraud
make docker-up    # Postgres + bot via Docker Compose
```

---

## 📖 Complete Documentation Index

- [🏗️ System Architecture & Event Flow](docs/ARCHITECTURE.md)
- [🗄️ PostgreSQL Database Schema](docs/DATABASE_SCHEMA.md)
- [🤖 Bot Command, UI & Admin Manual](docs/BOT_COMMANDS.md)
- [💾 Daily Backup & Recovery Strategy](docs/BACKUP_STRATEGY.md)
