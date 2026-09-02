# 🤖 Bot Commands, UI Keyboards & Advanced Admin Suite - @telefraudbot

This guide documents the complete user interface keyboard system, private message commands, automatic join/daily group scanners, advanced admin panel controls, developer branding, and HTML parse mode templates for **@telefraudbot**.

---

## 👨‍💻 Developer Attribution & System Branding

The bot features integrated developer attribution in all primary menus, `/start` welcome screens, and developer info interfaces.

- **Lead Developer**: `xspoilt`
- **Official Handle**: `@xspoilt`
- **Framework**: Go (`github.com/mymmrac/telego`)
- **Database**: PostgreSQL 15+

---

## 🎨 UI System: Reply Keyboards vs Inline Keyboards

`@telefraudbot` uses a hybrid user interface model combining persistent Reply Keyboards for primary navigation with dynamic Inline Keyboards for contextual actions.

### 1. Persistent Reply Keyboard (Private Chat Main Menu)
When a user sends `/start` in PM, the persistent bottom Reply Keyboard is activated:

```
+-------------------------------------------------------+
|  🛡️ Report Fraudster     |     🔍 Check Identifier   |
+-------------------------------------------------------+
|  📋 My Submissions        |     📊 Global Statistics  |
+-------------------------------------------------------+
|  ❓ Help & FAQ            |     👨‍💻 Developer Info     |
+-------------------------------------------------------+
```

### 2. Contextual Inline Keyboards
Inline keyboards are attached directly under messages for step-by-step flows:
- **Report Category Wizard**: `[ 💸 Financial ]`, `[ 👤 Impersonation ]`, `[ 🔐 Crypto/Phishing ]`, `[ 📦 Fake Store ]`
- **Admin Review Queue**: `[ ✅ Approve Fraud ]`, `[ ❌ Reject ]`, `[ ❓ Request Proof ]`, `[ 🚫 Ban Reporter ]`
- **Group Settings Panel**: `[ ⚙️ Auto-Ban: ON ]`, `[ 🗑️ Auto-Delete: ON ]`, `[ ⏰ Daily Scan: ON ]`
- **Group Clean Banner**: `[ ⚡ Clean Flagged Members Now ]`

---

## 👤 User PM Commands & Menu Flows

| Reply Button / Command | Arguments | Description |
| :--- | :--- | :--- |
| `🛡️ Report Fraudster` (`/report`) | None | Launches multi-step fraud reporting wizard with category selection & file proof upload. |
| `🔍 Check Identifier` (`/check`) | `<username \| id \| phone>` | Checks if an account/number is blacklisted in PostgreSQL. |
| `📋 My Submissions` (`/myreports`) | None | Views history of reports submitted by the user and their admin review statuses. |
| `📊 Global Statistics` (`/stats`) | None | Displays real-time database stats (total blacklisted, pending reports, protected groups). |
| `❓ Help & FAQ` (`/help`) | None | Displays reporting rules, proof requirements, and safety instructions. |
| `👨‍💻 Developer Info` (`/dev`) | None | Displays developer credentials, version information, and official contact links. |

### 👨‍💻 Developer Info HTML Output (`/dev`)
```html
<b>🤖 @telefraudbot System Info</b>

<b>Lead Developer:</b> xspoilt (<code>@xspoilt</code>)
<b>Uptime:</b> 4m 29s

<i>Designed &amp; Built for Telegram Group Safety.</i>
```

---

## 🛡️ Group Moderation & Automated Scanning Engine

### 1. ⚡ Real-Time New Member Join Check (`OnChatMemberUpdated` / `NewChatMembers`)
Whenever a user joins or is added to a monitored Telegram group:
1. Bot intercepts the join update instantly.
2. Queries PostgreSQL `fraud_records` by User ID, Username, and Phone (if available).
3. **If Flagged**:
   - Executes `BanChatMember` immediately.
   - Posts HTML alert in group: `⚠️ Flagged Fraudster Banned on Join!`.
   - Logs event to `moderation_logs`.

### 2. ⏰ Automated Daily Group Member Scan (Scheduled Cron Worker)
Every day at `03:00 AM UTC` (configurable):
1. Background worker iterates through all active groups where bot is Administrator.
2. Fetches/scans chat members against verified blacklisted accounts.
3. Automatically bans detected scammers or generates an HTML summary digest to Group Admins with an inline `[ ⚡ Purge Scammers ]` button.

### 3. Group Admin Commands

| Command | Scope | Description |
| :--- | :--- | :--- |
| `/scangroup` | Group Admin | Triggers an immediate manual scan of all group members against the fraud database. |
| `/checkuser` | Group Admin | `<@username \| reply>` Checks if a specific group member is flagged. |
| `/telefraud_settings` | Group Admin | Opens inline settings dashboard to configure auto-ban, auto-delete, and daily scan schedules. |

---

## 👑 Advanced Admin Control Suite

Superadmins (`ADMIN_IDS`) access an expanded management dashboard (`/admin`) with deep control over system parameters.

```
+-------------------------------------------------------+
|                 👑 ADMIN CONTROL PANEL                |
+-------------------------------------------------------+
|  📥 Pending Reports (12)  |  🚫 Blacklist Manager     |
+-------------------------------------------------------+
|  👥 Admin Management      |  📢 Global Broadcast      |
+-------------------------------------------------------+
|  📊 System Metrics        |  📂 Export/Import CSV     |
+-------------------------------------------------------+
|  ⚙️ Enforcement Mode      |  💾 Trigger DB Backup     |
+-------------------------------------------------------+
```

### 1. Administrative Commands Reference

| Command | Arguments | Description |
| :--- | :--- | :--- |
| `/admin` | None | Opens the interactive Admin Dashboard inline menu. |
| `/pending` | None | Views pending user report queue with quick approval/rejection inline buttons. |
| `/admin_add` | `<user_id>` | Grants Admin privileges to a user. |
| `/admin_remove` | `<user_id>` | Revokes Admin privileges. |
| `/admin_list` | None | Lists all active system administrators. |
| `/blacklist add` | `<target>` `<reason>` `<threat_level>` | Direct manual insertion into verified fraud database. |
| `/blacklist remove` | `<target>` | Removes an account from the blacklist. |
| `/broadcast` | `<HTML Message>` | Sends a global announcement to all registered bot users & groups. |
| `/admin_scangroup` | `<group_id>` | Remotely forces a full scan on any registered group. |
| `/export_blacklist` | `[csv \| json]` | Exports full blacklisted database as a downloadable file attachment. |
| `/import_blacklist` | `<Attach File>` | Bulk imports fraud records from a CSV/JSON file. |
| `/mode` | `[BAN \| KICK \| WARN \| SILENT]` | Sets global auto-moderation strictness level. |
| `/backup` | None | Executes immediate database backup and sends file to Admin Log Channel. |

### 2. Global Broadcast Engine (`/broadcast`)
Admins can send formatted HTML announcements across all bot users and groups:
- Features preview confirmation (`[ Send Now ]`, `[ Cancel ]`).
- Dynamic rate-limiting (30 msgs/sec to respect Telegram API limits).
- Displays live broadcast progress bar: `Progress: [████████░░] 80% (1,420/1,775)`.

### 3. Enforcement Modes (`/mode`)
Admins can switch the global group moderation enforcement level:
- **`BAN` (Default)**: Immediately bans flagged users and deletes scam messages.
- **`KICK`**: Kicks flagged users (allows rejoining if cleared).
- **`WARN`**: Deletes message and posts a warning banner without banning.
- **`SILENT`**: Logs detection to DB and alerts group admins silently without public message.
