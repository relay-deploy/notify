# notify

Email and Android push notifications for relay events.

Notify reads the events other relay services record in their Postgres outboxes and emails one configured address
about deploys and crashes. A crash loop produces one crashed email, reminders while it continues, and one
recovered email. Without a Resend API key, emails are logged instead of sent.

## Configuration

Environment keys use the .NET style.

| Key | Default | Notes |
|---|---|---|
| `Relay__Postgres__ConnectionString` | required | Npgsql format |
| `Relay__Endpoints__ManagementPort` | `8080` | serves `/health/live` and `/health/ready` |
| `Relay__Platform__Address` | required | e.g. `http://platform:80` |
| `Relay__Notify__Sources` | `runtime_docker,platform` | comma-separated producer schemas |
| `Relay__Notify__Email__To` | required | the one recipient |
| `Relay__Notify__Email__From` | `relay <onboarding@resend.dev>` | sender address |
| `Relay__Notify__Resend__ApiKey` | empty | empty means the log mailer |
| `Relay__Notify__Resend__BaseUrl` | `https://api.resend.com` | for testing |
| `Relay__Notify__TimeZone` | `UTC` | IANA name, used for times in emails |
| `Relay__Notify__ReminderInterval` | `30m` | between still-crashing reminders |
| `Relay__Notify__RecoveryQuietPeriod` | `5m` | how long an app stays up before recovered goes out |
| `Relay__Notify__NewLoopAfter` | `1h` | a crash this long after the last one starts a new alert |
| `Relay__Notify__NameFallbackAfter` | `2m` | wait for names before using short ids |
| `Relay__Notify__AlertExpiry` | `24h` | silent close of an alert with no new crash |
| `Relay__Notify__AlertPassInterval` | `1m` | how often alerts are checked |

## Run locally

```
docker compose -f ../install/compose.dev.yaml up -d --wait
Relay__Postgres__ConnectionString='Host=127.0.0.1;Port=5432;Username=relay;Password=relay;Database=relay' \
Relay__Platform__Address=http://127.0.0.1:5301 \
Relay__Notify__Email__To=you@example.invalid \
Relay__Endpoints__ManagementPort=5402 \
go run ./cmd/notify
curl 127.0.0.1:5402/health/ready
```
