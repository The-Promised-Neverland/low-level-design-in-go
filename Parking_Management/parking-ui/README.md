# Parking control room

Live polling UI for the Go Automated Parking System. Read-only apart from `POST /api/strategy`; the backend is the source of truth.

```bash
npm install
cp .env.example .env.local   # NEXT_PUBLIC_API_BASE=http://localhost:8080
npm run dev                  # http://localhost:3000
```

Polls `/api/health`, `/api/state`, `/api/events` every 700 ms. Backend must allow CORS from `http://localhost:3000`.
