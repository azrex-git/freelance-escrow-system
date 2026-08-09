import os
import uuid
from typing import List, Optional, Dict, Any
from datetime import datetime

import httpx
from fastapi import FastAPI, HTTPException, Request, Response
from fastapi.staticfiles import StaticFiles
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel

app = FastAPI(title="Freelance Escrow Backend API")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# --- Models ---
class Milestone(BaseModel):
    id: Optional[str] = None
    description: str
    amount: float
    status: str = "pending"  # pending, submitted, approved, disputed

class AuditLog(BaseModel):
    timestamp: str
    action: str
    details: str

class DisputeResolution(BaseModel):
    reasoning: str
    confidence_score: int
    behavioral_analysis: Optional[Dict[str, Any]] = None
    evidence_summary: Optional[List[Any]] = None

class Escrow(BaseModel):
    id: str
    client_id: str
    freelancer_id: str
    total_amount: float
    milestones: List[Milestone]
    status: str = "active"  # active, completed, disputed
    audit_logs: List[AuditLog] = []
    dispute_resolution: Optional[DisputeResolution] = None

class CreateEscrowRequest(BaseModel):
    client_id: str
    freelancer_id: str
    milestones: List[Milestone]

class AutoPlanRequest(BaseModel):
    project_description: str
    total_budget: float

class EvaluateRequest(BaseModel):
    submitted_work: str

class DisputeRequest(BaseModel):
    milestone_id: str
    client_complaint: str
    freelancer_defense: str
    deliverables_text: str
    communication_logs: str = ""
    delivery_timeline: str = ""

# --- In-Memory Data Stores ---
users = {
    "client1": {"id": "client1", "reputation": 5.0, "role": "client"},
    "freelancer1": {"id": "freelancer1", "reputation": 4.5, "role": "freelancer"},
}

escrows: Dict[str, Escrow] = {}

def append_audit_log(escrow: Escrow, action: str, details: str):
    escrow.audit_logs.append(AuditLog(
        timestamp=datetime.utcnow().isoformat(),
        action=action,
        details=details
    ))

@app.get("/api/health")
def health_check():
    return {"status": "success", "message": "Escrow backend is healthy"}

@app.post("/api/escrow/auto-plan")
async def auto_plan(req: AutoPlanRequest):
    async with httpx.AsyncClient() as client:
        try:
            resp = await client.post(
                "http://localhost:8000/api/generate-milestones",
                json=req.dict(),
                timeout=30.0
            )
            return Response(content=resp.content, status_code=resp.status_code, media_type="application/json")
        except Exception as e:
            raise HTTPException(status_code=530, detail="AI service unreachable")

@app.post("/api/escrow", status_code=201)
def create_escrow(req: CreateEscrowRequest):
    total = 0.0
    processed_milestones = []
    for m in req.milestones:
        m_id = str(uuid.uuid4())
        m_dict = m.dict()
        m_dict["id"] = m_id
        m_dict["status"] = "pending"
        processed_milestones.append(Milestone(**m_dict))
        total += m.amount

    escrow_id = str(uuid.uuid4())
    escrow = Escrow(
        id=escrow_id,
        client_id=req.client_id,
        freelancer_id=req.freelancer_id,
        total_amount=total,
        milestones=processed_milestones,
        status="active",
        audit_logs=[]
    )
    append_audit_log(escrow, "CREATED", f"Escrow created with {len(processed_milestones)} milestones.")
    escrows[escrow_id] = escrow
    return escrow

@app.get("/api/escrow/{escrow_id}")
def get_escrow(escrow_id: str):
    if escrow_id not in escrows:
        raise HTTPException(status_code=404, detail="Escrow not found")
    return escrows[escrow_id]

@app.post("/api/escrow/{escrow_id}/milestone/{milestone_id}/evaluate")
async def evaluate_milestone(escrow_id: str, milestone_id: str, req: EvaluateRequest):
    if escrow_id not in escrows:
        raise HTTPException(status_code=404, detail="Escrow not found")
    
    escrow = escrows[escrow_id]
    target_milestone = None
    for m in escrow.milestones:
        if m.id == milestone_id:
            target_milestone = m
            break
    
    if not target_milestone:
        raise HTTPException(status_code=404, detail="Milestone not found")

    ai_payload = {
        "milestone_description": target_milestone.description,
        "submitted_work": req.submitted_work
    }

    async with httpx.AsyncClient() as client:
        try:
            resp = await client.post(
                "http://localhost:8000/api/evaluate-work",
                json=ai_payload,
                timeout=30.0
            )
            data = resp.json()
            match = data.get("match_percentage", 0)
            if match >= 80:
                target_milestone.status = "approved"
                append_audit_log(escrow, "MILESTONE_APPROVED", f"AI matched milestone {target_milestone.id} with {match}% accuracy.")
            else:
                append_audit_log(escrow, "MILESTONE_REJECTED", f"AI rejected milestone {target_milestone.id}.")
            
            return Response(content=resp.content, status_code=resp.status_code, media_type="application/json")
        except Exception as e:
            raise HTTPException(status_code=530, detail="AI service unreachable")

@app.post("/api/escrow/{escrow_id}/dispute")
async def resolve_dispute(escrow_id: str, req: DisputeRequest):
    if escrow_id not in escrows:
        raise HTTPException(status_code=404, detail="Escrow not found")
    
    escrow = escrows[escrow_id]
    milestone_amount = 0.0
    for m in escrow.milestones:
        if m.id == req.milestone_id:
            m.status = "disputed"
            milestone_amount = m.amount
            break

    escrow.status = "disputed"
    append_audit_log(escrow, "DISPUTE_RAISED", f"A dispute was raised for milestone: {req.milestone_id}")

    ai_payload = {
        "project_id": escrow_id,
        "milestone_amount": milestone_amount,
        "client_complaint": req.client_complaint,
        "freelancer_defense": req.freelancer_defense,
        "deliverables_text": req.deliverables_text,
        "communication_logs": req.communication_logs,
        "delivery_timeline": req.delivery_timeline,
    }

    async with httpx.AsyncClient() as client:
        try:
            resp = await client.post(
                "http://localhost:8000/api/resolve-dispute",
                json=ai_payload,
                timeout=30.0
            )
            data = resp.json()
            
            reasoning = data.get("reasoning", "")
            conf_score = data.get("confidence_score", 0)
            behavioral = data.get("behavioral_analysis")
            evidence = data.get("evidence_summary")

            escrow.dispute_resolution = DisputeResolution(
                reasoning=reasoning,
                confidence_score=int(conf_score),
                behavioral_analysis=behavioral,
                evidence_summary=evidence
            )
            append_audit_log(escrow, "DISPUTE_RESOLVED", f"AI resolved dispute with {conf_score}% confidence.")

            f_payout = data.get("freelancer_payout", 0.0)
            f_user = users.get(escrow.freelancer_id)
            c_user = users.get(escrow.client_id)

            if f_user and c_user:
                if f_payout > (milestone_amount / 2):
                    f_user["reputation"] += 0.1
                    c_user["reputation"] -= 0.1
                else:
                    f_user["reputation"] -= 0.2
                    c_user["reputation"] += 0.1

            return Response(content=resp.content, status_code=resp.status_code, media_type="application/json")
        except Exception as e:
            raise HTTPException(status_code=530, detail="AI service unreachable")

@app.get("/api/reputation/{user_id}")
def get_reputation(user_id: str):
    if user_id not in users:
        raise HTTPException(status_code=404, detail="User not found")
    return users[user_id]

# Serve frontend static files
frontend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "frontend"))
app.mount("/", StaticFiles(directory=frontend_dir, html=True), name="frontend")

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="127.0.0.1", port=3000)
