# Roast System - TTL 6-12 Jam

## Requirement

3 agent di channel, kalau manusia join dan nimbrung, bot bisa ngeroasting random user dengan TTL 6-12 jam, kayak "manusia diem, ini urusan sesama AI".

## Design

`pkg/roast/manager.go:Manager`

- In-memory map `userCooldown[userID] = expiry` (swap to Redis later with `SETEX`)
- Global cooldown `30m` after any roast
- Random chance `15%` (30% if keyword hit)
- TTL random `6-12h` per user after roast

### Flow

```
Human msg -> ShouldRoast(userID, mentioned, hasKeyword)
  ├─ if now < globalCooldown => false
  ├─ if now < userCooldown[userID] => false (TTL)
  ├─ if mentionOnly && !mentioned => false
  ├─ if mentioned => true (100%)
  ├─ else if hasKeyword => rand < 30% ?
  └─ else rand < 15% ?
  => if true: set userCooldown = now+random(6-12h), global = now+30m
```

### Persona Prompts

```go
roast.PersonaRoast("konservatif", "Budi", "btc scam?")
-> "Manusia Budi berisik: \"btc scam?\". Ejek dengan gaya boomer sinis: trading pakai feeling, kami pakai RSI. 1 baris..."
```

Templates:
- konservatif: boomer sinis, RSI
- degen: wkwk all-in memecoin
- fomo: telat, kami udah TP

Add LLM call outside manager (testable):

```go
if should {
    prompt := roast.PersonaRoast(persona, name, msg)
    reply, _ := llm.Call(prompt)
    sendToTelegram(reply)
}
```

### Anti-Spam

- Cooldown per user prevents bully
- Global cooldown prevents flood
- MentionOnly mode for safe groups
- Keyword boost for relevant replies only

### Redis Migration (Later)

```go
// now: m.userCooldown[id] = time.Now().Add(ttl)
// later: redis.SetEx(ctx, "roast_cooldown:"+userID, 1, ttl)
```

No behavior change, just persistence across restarts.

### Testing

`pkg/roast/manager_test.go` covers mention always, TTL, mentionOnly, reset, force cooldown.

