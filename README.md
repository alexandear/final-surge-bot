# Final Surge Bot

Telegram bot to communicate with Final Surge platform.

<img src="web\static\final_surge_bot.jpg" width="700" alt="Telegram Bot Screenshot">

## Development

### Local

Start PostgreSQL:

```
docker run --rm --name pgtest -e POSTGRES_HOST_AUTH_METHOD=trust -p 5432:5432 postgres
```

Set environment:

```
PUBLIC_URL=https://final-surge-bot.herokuapp.com/;BOT_API_KEY=<BOT_API_KEY>;PORT=8080;DATABASE_URL=postgresql://postgres:@localhost:5432/postgres
```

Delete webhook:

```
curl https://api.telegram.org/bot<BOT_API_KEY>/deleteWebhook
```
