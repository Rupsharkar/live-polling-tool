# Live Polling Tool

A small, polished live polling application built for the internship task.

## Stack

- Frontend: React + Vite
- Backend: Go + Gin
- Database: MongoDB
- Realtime: Redis (INCR + Pub/Sub)
- WebSocket: Gorilla WebSocket
- Authentication: JWT + bcrypt

## Architecture

A vote follows this path:

React -> Go/Gin -> validation -> MongoDB vote -> Redis INCR -> Redis Pub/Sub -> Go WebSocket -> all React clients

MongoDB permanently stores users, polls and votes. Redis maintains the fast live counters and publishes changes to every connected poll viewer.

## Local setup

### 1. Backend

Create `backend/.env` from `backend/.env.example`.

Required values:

- `MONGO_URI`
- `MONGO_DB`
- `REDIS_URL`
- `JWT_SECRET`
- `FRONTEND_URL`
- `PORT`

Then:

```bash
cd backend
go mod tidy
go run ./cmd/server
```

Backend runs on `http://localhost:8080`.

### 2. Frontend

```bash
cd frontend
npm install
npm run dev
```

Frontend runs on the Vite URL shown in the terminal.

Create `frontend/.env`:

```env
VITE_API_URL=http://localhost:8080/api
VITE_WS_URL=ws://localhost:8080
```

## Live demo test

Open the same poll URL in two browser windows. Vote in one window. The other window updates without refreshing.

## Deployment

Suggested simple deployment:

- Frontend: Vercel
- Backend: Render
- MongoDB: MongoDB Atlas
- Redis: Redis Cloud

Set the production environment variables from the example files. Do not commit secrets.

## Security decisions

- Passwords are bcrypt hashed.
- Poll creation/edit/close endpoints require a JWT.
- Backend validates question, options, IDs and vote input.
- MongoDB has a unique compound index on `(pollId, voterId)` to prevent duplicate votes.
- JWT secret and database credentials are environment variables.
- CORS is restricted to the configured frontend origin.
- The frontend never directly accesses MongoDB or Redis.

## AI disclosure

AI assistance was used for scaffolding, code review, debugging ideas and documentation. The project should only be submitted after the author has run it, tested the live voting flow, and understands the authentication, MongoDB, Redis Pub/Sub, and WebSocket flow.
