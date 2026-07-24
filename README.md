# Theresa

An AI tutor that teaches your course material on a visual board while explaining it out
loud — paste a problem, upload a PDF or a photo of a page, and get taught.

- **Backend**: Go (chi router, MongoDB, Gemini)
- **Frontend**: Next.js (App Router, TypeScript, Tailwind) styled after ChatGPT's UI
- **Database**: MongoDB
- **AI**: Google Gemini (text understanding + Gemini Live for real-time voice)

## Status

Milestone 1: project scaffold, health check, and themed frontend shell. See
`/Users/mac/.claude/plans/swift-purring-iverson.md` for the full milestone roadmap.

## Running locally

### 1. Local MongoDB (optional — you can point at Atlas instead)

```
docker compose up -d
```

### 2. Backend

```
cd backend
cp .env.example .env   # fill in real values, never commit this file
go run ./cmd/server
```

Health check: `curl http://localhost:8080/healthz`

### 3. Frontend

```
cd frontend
cp .env.example .env.local
npm install
npm run dev
```

Open http://localhost:3000
