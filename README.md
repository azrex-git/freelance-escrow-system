# SafeWork: Reputation-Weighted Freelance Escrow

**Live Demo:** [https://freelance-escrow-system.vercel.app](https://freelance-escrow-system.vercel.app)

SafeWork is a milestone-based escrow platform designed to eliminate payment disputes, scope creep, and "ghosting" in freelance work. It leverages AI for automated dispute resolution and maintains a strict reputation system to ensure both clients and freelancers are held accountable.

This project was built to address the trust deficit in freelance marketplaces by ensuring funds are secured before work begins and payments are released only when milestones are completed.

## Architecture Overview

The platform is split into three main microservices to separate the core business logic from the AI processing:

1. **Frontend (Vercel)**
   - Built with raw HTML, CSS, and vanilla JavaScript for maximum performance and zero overhead.
   - Communicates dynamically with the Go Backend.
   
2. **Core Escrow Backend (Render - Go / Fiber)**
   - Acts as the simulated smart contract.
   - Manages the state of escrows, milestone progression, and user reputations.
   - Strictly enforces state transitions (e.g., funds cannot be released until a milestone is approved by the client or resolved by the AI).
   
3. **AI Dispute Arbiter (Render - Python / FastAPI)**
   - Powered by the Google Gemini API.
   - Evaluates disputed deliverables against the original milestone requirements.
   - Calculates fair split percentages for partial deliveries or late submissions.
   - Analyzes communication logs to detect ghosting behavior.

## Core Features

### Milestone-Based Escrow
Instead of funding an entire project upfront or relying on post-delivery invoices, projects are broken down into granular milestones. The escrow locks the budget and releases it milestone-by-milestone as the client approves the submitted work.

### AI Dispute Resolution
If a client rejects a freelancer's submitted work, the milestone enters a "Disputed" state. The AI Arbiter is then called to review the client's complaint, the freelancer's defense, and the actual submitted deliverables (e.g., GitHub links, documents). The AI returns a fair payout split based on the percentage of work completed, rather than a binary win/loss.

### Anti-Ghosting Protocol
If a party goes completely silent past a deadline, the system flags the project with a high ghosting risk. During a dispute, if the AI confirms a party has ghosted, that user faces a severe permanent reputation penalty, and the funds are routed to the active party.

### Permanent Reputation System
Every action on the platform impacts a user's Trust Tier. Successfully completing milestones incrementally increases reputation. Paying out on time increases client reputation. Disputes and ghosting heavily damage a user's score. Trust Tiers range from "Newcomer" up to "Elite", ensuring high-quality actors are easily identifiable.

- You earn +0.25 stars for successfully releasing a milestone (steady progress).
- You earn a massive +0.5 star bonus for successfully completing an entire multi-milestone project without disputes.
- Clients get a small bump (+0.15) just for paying on time to encourage good behavior.
- But if you lose a dispute or get caught ghosting by the AI Arbiter, you will lose up to -0.3 stars immediately.

## Running Locally

To run this project on your local machine for development or testing:

### Prerequisites
- Go 1.22+
- Python 3.10+
- A Google Gemini API Key

### 1. Start the AI Service
Navigate to the `ai-service` directory.
Install the dependencies:
```bash
pip install -r requirements.txt
```
Create a `.env` file and add your API key:
```
GEMINI_API_KEY=your_actual_api_key_here
```
Run the Python server:
```bash
uvicorn main:app --port 8000
```

### 2. Start the Go Backend
Open a new terminal and navigate to the `backend` directory.
Run the Go server:
```bash
go run main.go
```
The Go server will automatically serve the static frontend files.

### 3. Access the Application
Open your web browser and navigate to:
`http://localhost:3000`

## Deployment

This repository is configured for modern cloud deployment:
- The **Frontend** can be deployed directly to Vercel (point the root directory to `frontend`).
- The **Go Backend** can be deployed to Render as a Web Service (point root directory to `backend`, set the build command to `go build -o main main.go`, and start command to `./main`).
- The **AI Service** can be deployed to Render as a Web Service (point root directory to `ai-service`, set the build command to `pip install -r requirements.txt`, and start command to `uvicorn main:app --host 0.0.0.0 --port $PORT`). Ensure you set the `GEMINI_API_KEY` environment variable.

Make sure to update the `API_BASE_URL` in `frontend/script.js` and the Python service URLs in `backend/main.go` to match your live deployed URLs.
