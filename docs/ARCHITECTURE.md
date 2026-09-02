# 🏗️ System Architecture - @telefraudbot

This document details the high-level architecture, module decomposition, event flow, and sequence interactions for **@telefraudbot**.

---

## 📐 Overall Architecture Diagram

```mermaid
graph TD
    subgraph TG_INFRA["Telegram Infrastructure"]
        TG["Telegram API"]
    end

    subgraph BOT_LAYER["Bot Layer (Telego Framework)"]
        BH["Telego Bot Handler"]
        UPD["Update Listener / Long Polling"]
        ROUTER["Router & Middleware"]
    end

    subgraph UI_LAYER["User Interface Layer"]
        RKB["Persistent Reply Keyboard Engine"]
        IKB["Dynamic Inline Keyboard Router"]
    end

    subgraph CORE_SERVICES["Core Services"]
        RS["Report Wizard Service"]
        AS["Advanced Admin Control Suite"]
        MS["Group Moderation Engine"]
        NMS["New Member Join Scanner"]
        DGS["Daily Scheduled Group Scanner Worker"]
        BS["Blacklist Query Service"]
        BCAST["Global Broadcast Engine"]
        BK["Daily Backup Worker"]
    end

    subgraph STORAGE_LAYER["Storage Layer"]
        PG[("PostgreSQL Database")]
        FS[("Local Backup Storage")]
    end

    TG <-->|Long Polling / Webhook| UPD
    UPD --> ROUTER
    ROUTER --> RKB
    ROUTER --> IKB
    ROUTER --> BH

    BH --> RS
    BH --> AS
    BH --> MS
    BH --> NMS
    BH --> BS
    DGS --> MS
    AS --> BCAST

    RS <-->|Read / Write Reports| PG
    AS <-->|Approve / Config / Admins| PG
    MS <-->|Check Fraud Status| PG
    NMS <-->|Check Joiner Fraud Status| PG
    DGS <-->|Batch Member Scan| PG
    BS <-->|Fast Lookup Cache| PG
    BK -->|pg_dump Daily Dumps| PG
    BK -->|Save .sql.gz| FS
```

---

## 🧩 Component Decomposition

### 1. Bot & UI Layer (`internal/bot`, `internal/ui`)
- **Telego Framework (`github.com/mymmrac/telego`)**: High-performance Telegram Bot API framework.
- **Reply Keyboard System**: Renders persistent bottom navigation buttons (`🛡️ Report Fraudster`, `🔍 Check Identifier`, `📋 My Submissions`, `👨‍💻 Developer Info`).
- **Inline Keyboard System**: Handles stateful multi-step wizards, admin approval queues, group settings toggles, and report cleanup confirmations.
- **HTML Renderer & Sanitizer**: Ensures output formatting complies with `telego.ModeHTML` and escapes user input via `html.EscapeString()`.

### 2. Group Scanner & Protection Engines (`internal/services/moderation`)
- **Real-Time Message Interceptor**: Inspects sender ID, username, and message body text in group chats.
- **New Member Join Scanner (`OnChatMemberUpdated`)**: Triggers instantly when a user joins or is added to a group. Auto-bans verified fraudsters on entry.
- **Daily Automated Group Scanner (`Cron Worker`)**: Scheduled background worker running daily at 03:00 AM UTC. Scans member lists of all registered groups.

### 3. Advanced Admin Suite & Storage (`internal/services/admin`, `internal/database`)
- **Admin Management Service**: Dynamic admin role assignment, permission checks, and audit logging.
- **Global Broadcast Engine**: Asynchronous chunked mass messaging with live progress tracking and Telegram rate limit throttling.
- **Import/Export Service**: Bulk import/export of fraud records in CSV and JSON formats.
- **Database Pool (`pgx/v5`)**: High-concurrency PostgreSQL connection pool.
- **Daily Backup Manager**: Automated background worker running `pg_dump` with gzip compression and Telegram alert delivery.

---

## 🔄 Core Event Sequence Diagrams

### 1. New Member Join Scan Sequence (`OnChatMemberUpdated`)

```mermaid
sequenceDiagram
    autonumber
    actor Joiner as "New Group Member"
    participant TG as "Telegram API"
    participant Bot as "Telego Bot Listener"
    participant NMS as "New Member Scanner"
    participant DB as "PostgreSQL"
    participant Group as "Telegram Group"

    Joiner->>TG: Joins Group / Added by User
    TG->>Bot: Event: ChatMemberUpdated
    Bot->>NMS: Extract User ID, Username & Phone
    NMS->>DB: Query fraud_records by User ID
    alt Fraud Match Found
        DB-->>NMS: Scam Account Confirmed
        NMS->>Group: BanChatMember
        NMS->>Group: Send HTML Warning Banner
        NMS->>DB: Log Action in moderation_logs
    else User Clean
        DB-->>NMS: No Match Found
        NMS->>Bot: Allow Member
    end
```

---

### 2. Scheduled Daily Group Scan Sequence

```mermaid
sequenceDiagram
    autonumber
    participant Cron as "Daily Scan Worker (03:00 AM)"
    participant DGS as "Daily Group Scanner"
    participant DB as "PostgreSQL"
    participant TG as "Telegram API"
    participant Group as "Telegram Group"

    Cron->>DGS: Trigger Daily Group Member Audit
    DGS->>DB: Fetch Active Groups
    loop For Each Active Group
        DGS->>TG: GetChatAdministrators & Member List
        DGS->>DB: Bulk Match Member IDs against fraud_records
        alt Flagged Accounts Found
            DB-->>DGS: Flagged Accounts Detected
            DGS->>TG: Execute BanChatMember for Flagged Accounts
            DGS->>Group: Send Daily Audit Report HTML Summary
            DGS->>DB: Update last_scanned_at timestamp
        else Group Clean
            DGS->>DB: Update last_scanned_at timestamp
        end
    end
```

---

### 3. User PM Report & Proof Upload Flow

```mermaid
sequenceDiagram
    autonumber
    actor User as "Telegram User"
    participant Bot as "Telego Bot"
    participant RS as "Report Wizard Service"
    participant DB as "PostgreSQL"
    actor Admin as "System Admin"
    participant AC as "Admin Log Channel"

    User->>Bot: Click Report Fraudster
    Bot-->>User: Request Target Username / ID / Phone
    User->>Bot: Submit Target Info
    Bot-->>User: Select Category
    User->>Bot: Click Category Button
    Bot-->>User: Request Proof Screenshots
    User->>Bot: Upload Photo Proof
    RS->>DB: Save Report (Status PENDING)
    RS->>AC: Forward to Admin Review Queue
    Bot-->>User: Report submitted under admin review
```

---

### 4. Admin Approval & Global Broadcast Engine

```mermaid
sequenceDiagram
    autonumber
    actor Admin as "System Admin"
    participant Dashboard as "Admin Panel"
    participant AS as "Admin Service"
    participant DB as "PostgreSQL"
    participant BCAST as "Broadcast Engine"
    participant Users as "All Bot Users & Groups"

    Admin->>Dashboard: Click Global Broadcast
    Dashboard-->>Admin: Request Broadcast Text
    Admin->>Dashboard: Submits Message Text
    Dashboard-->>Admin: Render HTML Preview
    Admin->>Dashboard: Confirm Broadcast
    AS->>BCAST: Dispatch Mass Message
    loop Rate Limited Batch Dispatch
        BCAST->>Users: Send Message in HTML Parse Mode
    end
    BCAST-->>Admin: Broadcast Complete
```
