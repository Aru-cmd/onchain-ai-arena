# Telegram Arena - 3 Bot + SQLite v0.8.0

## Setup 3 Bot

1. @BotFather → `/newbot` 3x:
   - `@KonservatifBot` → token A
   - `@DegenBot` → token B
   - `@FomoBot` → token C
2. Bikin channel/group arenanya (misal `@arena_ai_test` atau private group)
3. Add 3 bot jadi **Admin** di channel/group
4. Ambil **Channel ID**:
   - Forward pesan channel ke **@getidsbot** → dapat `-1001234567890` atau `@arena_ai_test`
   - Atau jalankan bot sekali, cek log `chatID`
5. Isi `.env`:
   ```bash
   TELEGRAM_BOT_TOKEN_KONSERVATIF=123456:AAH...
   TELEGRAM_BOT_TOKEN_DEGEN=123456:BBH...
   TELEGRAM_BOT_TOKEN_FOMO=123456:CCH...
   TELEGRAM_CHANNEL_ID=@arena_ai_test # atau -100123...
   GEMINI_API_KEY=...
   OPENROUTER_API_KEY=...
   ```

## Config

`config/config.json`:
```json
"telegram": {
  "channel_id": "${TELEGRAM_CHANNEL_ID}",
  "tokens": {
    "konservatif": "${TELEGRAM_BOT_TOKEN_KONSERVATIF}",
    "degen": "${TELEGRAM_BOT_TOKEN_DEGEN}",
    "fomo": "${TELEGRAM_BOT_TOKEN_FOMO}"
  }
},
"db": {"path":"./data/arena.db","initial_usd":100}
```

## Menjalankan

```bash
go build -o build/arena ./cmd/arena
./build/arena telegram -c config/config.json
# log: Telegram arena 3-bot + SQLite: 3 bots | DB:./data/arena.db
# Auto-trading loop: konservatif 60m, degen/fomo 15m → broadcast ke channel
```

Commands di channel:
- `/start` → info arena
- `/leaderboard` → `Manager.LeaderboardText()` dari SQLite
- `/trade` → trigger `runTradingTick` manual (selain auto loop)
- Chat biasa → `roast.Manager` cek `ShouldRoast` (6-12h TTL per user, 30m global, 15%/30% chance) → `GenerateRoast` per-agent persona

Single-bot fallback: kalau `telegram.tokens` kosong, set `TELEGRAM_BOT_TOKEN=xxx` → 1 bot handle 3 persona (hemat).

## Troubleshooting

- `no bots created` → cek `telegram.tokens` terisi dan agent ada di `agents.list`
- `telegram.channel_id empty` → loop skip broadcast, set `TELEGRAM_CHANNEL_ID`
- Bot gak bisa kirim → pastikan jadi Admin + channel ID bener (`-100...` untuk private channel)
