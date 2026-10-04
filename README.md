# cf-daily

A Telegram bot that assigns one Codeforces problem per day and commits your
solutions to a GitHub repository.

- A scheduled job sends the day's problem at **11:30 AM IST**.
- A second job sends a reminder at **10:30 PM IST**.
- Reply to either message with `/submit` and your code. The solution is pushed
  to your GitHub repo via a GitHub App installation.

## Commands

| Command | What it does |
| --- | --- |
| `/start` | Register, or resume daily problems after `/stop` |
| `/stop` | Pause daily problems; saved solutions are kept |
| `/connect` | Install the GitHub App and link your account |
| `/help` | List the commands |
| `/submit` + code | Save a solution and commit it |
| `/edit` + code | Replace a solution |
| `/delete` | Delete a solution and remove the file |

Reply to a daily problem message to target that specific problem — including an
older one. Without a reply, the most recent problem sent to you is used.

## How a day works

1. The scheduler calls `POST /telegram/send-daily-problem`.
2. `EnsureToday` assigns a problem for the current IST date if one does not
   exist, skipping every problem assigned before, and stores it in
   `daily_problems`.
3. Each allowed user is messaged, and the message id is recorded in
   `telegram_problem_messages`. That row is what lets a `/submit` reply resolve
   back to a problem.
4. The scheduler calls `POST /telegram/send-reminder`, which reminds each user
   about the last problem actually sent to them. It never assigns a problem, so
   a late trigger can only remind about something already received.

## Endpoints

| Method | Path | Auth |
| --- | --- | --- |
| GET | `/health` | none |
| GET | `/problem` | none — random problem in range, no side effects |
| GET | `/problem/today` | none — read-only, 404 if nothing assigned yet |
| POST | `/telegram/webhook` | `X-Telegram-Bot-Api-Secret-Token` |
| POST | `/telegram/send-daily-problem` | `X-Cron-Secret` |
| POST | `/telegram/send-reminder` | `X-Cron-Secret` |
| GET | `/github/callback` | single-use OAuth state |

## Configuration

Copy `.env.example` to `.env`. Every variable without a default is required and
the process exits at startup if one is missing.

| Variable | Required | Notes |
| --- | --- | --- |
| `PORT` | no | defaults to `8080` |
| `LOG_LEVEL` | no | `debug`, `info` (default), `warn`, `error` |
| `DATABASE_URL` | yes | Postgres connection string |
| `MIN_RATING` / `MAX_RATING` | yes | problem rating range |
| `TELEGRAM_BOT_TOKEN` | yes | |
| `TELEGRAM_WEBHOOK_SECRET` | yes | must match the webhook registration |
| `TELEGRAM_ALLOWED_USER_IDS` | no | comma separated; **empty means nobody is messaged** |
| `CRON_SECRET` | yes | shared with the scheduler workflow |
| `FLUX_APP_ID` | yes | GitHub App id |
| `FLUX_APP_SLUG` | no | app slug in the install URL, defaults to `8pieces` |
| `FLUX_CLIENT_ID` / `FLUX_CLIENT_SECRET` | yes | GitHub App OAuth credentials |
| `FLUX_PRIVATE_KEY_B64` | yes | base64 of the App private key PEM |
| `FLUX_CALLBACK_URL` | yes | must match the App's callback URL |
| `FLUX_REPOSITORY` | yes | repository name created per user |

## Database

The schema lives in `migrations/`. Apply it with any Postgres client:

```sh
psql "$DATABASE_URL" -f migrations/0001_init.sql
```

`0001_init.sql` is idempotent and adds the unique indexes the code depends on.
`0002_fix_assigned_date.sql` is a one-time correction — read its header before
running it.

## Running locally

```sh
go build ./cmd/server
go test ./...
./cf-daily
```

## Scheduling

`.github/workflows/daily-problem.yml` drives both jobs. GitHub Actions
`schedule` is a best-effort queue and can run hours late, so the service is
written so that a late trigger is only late, never wrong. Use the workflow's
**Run workflow** button (`daily` or `reminder`) to trigger either job manually.
