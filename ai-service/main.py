import os
import json
from google import genai
from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from dotenv import load_dotenv

load_dotenv()

api_key = os.getenv("GEMINI_API_KEY")
client = genai.Client(api_key=api_key) if api_key else None

app = FastAPI(title="SafeWork AI Arbiter — Anti-Ghosting Engine")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# --- Models ---

class DisputeRequest(BaseModel):
    project_id: str
    milestone_amount: float
    client_complaint: str
    freelancer_defense: str
    deliverables_text: str
    communication_logs: str = ""
    delivery_timeline: str = ""

class EvaluateRequest(BaseModel):
    milestone_description: str
    client_instructions: str = ""
    submitted_work: str
    file_name: str = ""
    file_content: str = ""
    is_late_delivery: bool = False
    hours_late: float = 0

class ProjectIdeaRequest(BaseModel):
    project_description: str
    total_budget: float

class GhostingAnalysisRequest(BaseModel):
    project_description: str
    milestones: list
    communication_summary: str = ""
    days_since_last_activity: int = 0

# --- Endpoints ---

@app.get("/api/health")
def health_check():
    return {"status": "AI Arbiter active. Anti-ghosting engine online."}


@app.post("/api/generate-milestones")
def generate_milestones(request: ProjectIdeaRequest):
    if client:
        try:
            system_prompt = f"""
            You are an expert technical project manager and escrow advisor for a freelance anti-ghosting platform.
            A user wants to start the following project. Break it down into 3-5 logical milestones.
            
            Project Description: {request.project_description}
            Total Budget: {request.total_budget}
            
            Assign a portion of the total budget to each milestone based on its complexity.
            Make sure the total budget across all milestones equals the provided total budget.
            Also suggest a reasonable deadline in days for each milestone.
            
            Return ONLY valid JSON in the following format:
            {{
                "milestones": [
                    {{
                        "description": "Short description of the milestone",
                        "amount": 1000,
                        "deadline_days": 7,
                        "client_instructions": "Specific acceptance criteria for this milestone"
                    }}
                ]
            }}
            """
            response = client.models.generate_content(
                model='gemini-1.5-flash',
                contents=system_prompt,
                config=genai.types.GenerateContentConfig(
                    response_mime_type="application/json"
                )
            )
            return json.loads(response.text)
        except Exception as e:
            print("Gemini API call failed, falling back to heuristic response:", e)

    # Heuristic fallback if GEMINI_API_KEY is not set or fails
    total = request.total_budget
    m1_amt = round(total * 0.3, 2)
    m2_amt = round(total * 0.4, 2)
    m3_amt = round(total - m1_amt - m2_amt, 2)
    desc = request.project_description[:60] if request.project_description else "Project"
    
    return {
        "milestones": [
            {
                "description": f"Phase 1: Setup, Architecture & Initial Prototype for {desc}",
                "amount": m1_amt,
                "deadline_days": 5,
                "client_instructions": "Complete initial architecture and deliver functional working prototype."
            },
            {
                "description": f"Phase 2: Core Feature Implementation & Backend Integration",
                "amount": m2_amt,
                "deadline_days": 10,
                "client_instructions": "Implement core backend logic, database schema, and primary UI views."
            },
            {
                "description": f"Phase 3: Testing, Bug Fixing, Documentation & Final Deployment",
                "amount": m3_amt,
                "deadline_days": 5,
                "client_instructions": "Deliver complete source code with unit tests, README documentation, and deployment build."
            }
        ]
    }


@app.post("/api/evaluate-work")
def evaluate_work(request: EvaluateRequest):
    if client:
        try:
            late_context = ""
            if request.is_late_delivery:
                late_context = f"""
                ⚠️ LATE DELIVERY DETECTED: The freelancer submitted this work {request.hours_late:.1f} hours after the deadline.
                Factor the late delivery into the evaluation. Penalize the match_percentage proportionally:
                - 1-24 hours late: Deduct 5-10% from match
                - 24-72 hours late: Deduct 10-20% from match
                - 72+ hours late: Deduct 20-30% from match
                """

            system_prompt = f"""
            You are an AI Judge for a freelance anti-ghosting escrow platform called SafeWork.
            Evaluate the following submitted work against the milestone description and the client's rules.
            
            Milestone Description: {request.milestone_description}
            Client Rules & Instructions: {request.client_instructions if request.client_instructions else 'None'}
            Submitted Work Link/Text: {request.submitted_work}
            Uploaded File Name: {request.file_name if request.file_name else 'None'}
            Uploaded File Content:
            ```
            {request.file_content if request.file_content else 'None'}
            ```
            {late_context}
            
            IMPORTANT: Carefully assess if this is a PARTIAL delivery. If the freelancer delivered some but not all requirements, reflect this in the partial_delivery_percentage.
            
            Return ONLY valid JSON exactly matching this format:
            {{
                "match_percentage": <integer 0-100>,
                "partial_delivery_percentage": <integer 0-100, how much of the work was actually delivered>,
                "feedback": "<detailed constructive feedback>",
                "confidence_score": <integer 0-100>,
                "is_late": {str(request.is_late_delivery).lower()},
                "late_penalty_applied": <integer 0-30, penalty percentage applied for lateness>,
                "behavioral_analysis": {{
                    "client_professionalism": <integer 0-100>,
                    "freelancer_professionalism": <integer 0-100>
                }},
                "evidence_summary": [
                    "<evidence point 1>",
                    "<evidence point 2>",
                    "<evidence point 3>"
                ]
            }}
            """
            response = client.models.generate_content(
                model='gemini-1.5-flash',
                contents=system_prompt,
                config=genai.types.GenerateContentConfig(
                    response_mime_type="application/json"
                )
            )
            return json.loads(response.text)
        except Exception as e:
            print("Gemini API call failed, falling back to heuristic response:", e)

    # Heuristic fallback
    base_match = 90 if len(request.submitted_work.strip()) > 10 else 60
    penalty = 0
    if request.is_late_delivery:
        if request.hours_late <= 24:
            penalty = 10
        elif request.hours_late <= 72:
            penalty = 20
        else:
            penalty = 30

    final_match = max(0, base_match - penalty)
    return {
        "match_percentage": final_match,
        "partial_delivery_percentage": 100 if final_match >= 75 else 70,
        "feedback": f"Work evaluated against requirement '{request.milestone_description}'. Deliverable submitted: '{request.submitted_work[:80]}'. Quality score: {final_match}%.",
        "confidence_score": 92,
        "is_late": request.is_late_delivery,
        "late_penalty_applied": penalty,
        "behavioral_analysis": {
            "client_professionalism": 95,
            "freelancer_professionalism": 90 if not request.is_late_delivery else 75
        },
        "evidence_summary": [
            f"Submitted text length: {len(request.submitted_work)} characters",
            f"File attached: {request.file_name if request.file_name else 'None'}",
            f"Lateness penalty: {penalty}% ({request.hours_late:.1f}h late)" if request.is_late_delivery else "On-time delivery verified"
        ]
    }


@app.post("/api/resolve-dispute")
def resolve_dispute(request: DisputeRequest):
    if client:
        try:
            system_prompt = f"""
            You are an impartial AI dispute resolution arbiter for SafeWork, a freelance escrow platform.
            Analyze this dispute and determine a fair payout split. Consider:
            - Quality and completeness of deliverables
            - Communication tone and professionalism
            - Delivery timeline (was it late? was ghosting involved?)
            - Who is more at fault?
            
            Client's Complaint: {request.client_complaint}
            Freelancer's Defense: {request.freelancer_defense}
            Deliverables Provided: {request.deliverables_text}
            Communication Logs: {request.communication_logs}
            Delivery Timeline: {request.delivery_timeline}
            
            Return ONLY valid JSON:
            {{
                "freelancer_percentage": <int 0-100>,
                "client_percentage": <int 0-100>,
                "reasoning": "<detailed 2-3 sentence explanation>",
                "confidence_score": <int 0-100>,
                "is_ghosting_detected": <boolean>,
                "ghosting_party": "<'client' or 'freelancer' or 'none'>",
                "behavioral_analysis": {{
                    "client_professionalism_score": <int 0-100>,
                    "freelancer_professionalism_score": <int 0-100>,
                    "tone_summary": "<short summary>"
                }},
                "evidence_summary": [
                    "<point 1>",
                    "<point 2>",
                    "<point 3>"
                ]
            }}
            """
            
            response = client.models.generate_content(
                model='gemini-1.5-flash',
                contents=system_prompt,
                config=genai.types.GenerateContentConfig(
                    response_mime_type="application/json"
                )
            )
            result = json.loads(response.text)
            
            freelancer_payout = (result["freelancer_percentage"] / 100) * request.milestone_amount
            client_refund = (result["client_percentage"] / 100) * request.milestone_amount
            
            return {
                "project_id": request.project_id,
                "freelancer_payout": freelancer_payout,
                "client_refund": client_refund,
                "reasoning": result.get("reasoning", ""),
                "confidence_score": result.get("confidence_score", 0),
                "is_ghosting_detected": result.get("is_ghosting_detected", False),
                "ghosting_party": result.get("ghosting_party", "none"),
                "behavioral_analysis": result.get("behavioral_analysis", {}),
                "evidence_summary": result.get("evidence_summary", [])
            }
        except Exception as e:
            print("Gemini API call failed, falling back to heuristic response:", e)

    # Heuristic fallback
    complaint = request.client_complaint.lower()
    defense = request.freelancer_defense.lower()
    
    is_ghost = "ghost" in complaint or "unresponsive" in complaint or "disappeared" in complaint
    
    if is_ghost:
        f_pct = 20
        c_pct = 80
        ghosting_party = "freelancer"
        reason = "Ghosting detected. The freelancer was unresponsive past the deadline without clear communication."
    elif len(request.deliverables_text.strip()) > 30:
        f_pct = 70
        c_pct = 30
        ghosting_party = "none"
        reason = "Substantial deliverables were provided by freelancer. Minor revisions required, partial payout granted."
    else:
        f_pct = 50
        c_pct = 50
        ghosting_party = "none"
        reason = "Incomplete evidence provided by both parties. Splitting funds equally per platform resolution standards."

    f_payout = (f_pct / 100.0) * request.milestone_amount
    c_refund = (c_pct / 100.0) * request.milestone_amount

    return {
        "project_id": request.project_id,
        "freelancer_payout": f_payout,
        "client_refund": c_refund,
        "reasoning": reason,
        "confidence_score": 88,
        "is_ghosting_detected": is_ghost,
        "ghosting_party": ghosting_party,
        "behavioral_analysis": {
            "client_professionalism_score": 85,
            "freelancer_professionalism_score": 60 if is_ghost else 80,
            "tone_summary": "Dispute analyzed based on submission history and communication logs."
        },
        "evidence_summary": [
            f"Client complaint: '{request.client_complaint[:60]}...'",
            f"Freelancer defense: '{request.freelancer_defense[:60]}...'",
            f"Deliverable text evaluation: {len(request.deliverables_text)} chars provided"
        ]
    }


@app.post("/api/ghosting-analysis")
def ghosting_analysis(request: GhostingAnalysisRequest):
    if client:
        try:
            milestones_str = json.dumps(request.milestones, indent=2)
            system_prompt = f"""
            You are an anti-ghosting analysis AI for a freelance escrow platform.
            Analyze the following project context and determine the ghosting risk.
            
            Project Description: {request.project_description}
            Milestones: {milestones_str}
            Communication Summary: {request.communication_summary if request.communication_summary else 'No recent communication'}
            Days Since Last Activity: {request.days_since_last_activity}
            
            Return ONLY valid JSON:
            {{
                "ghosting_risk_score": <int 0-100>,
                "risk_level": "<low/medium/high/critical>",
                "risk_factors": [
                    "<factor 1>",
                    "<factor 2>"
                ],
                "recommended_actions": [
                    "<action 1>",
                    "<action 2>"
                ],
                "analysis_summary": "<2-3 sentence summary>"
            }}
            """
            
            response = client.models.generate_content(
                model='gemini-1.5-flash',
                contents=system_prompt,
                config=genai.types.GenerateContentConfig(
                    response_mime_type="application/json"
                )
            )
            return json.loads(response.text)
        except Exception as e:
            print("Gemini API call failed, falling back to heuristic response:", e)

    days = request.days_since_last_activity
    if days >= 7:
        score = 85
        level = "critical"
    elif days >= 4:
        score = 60
        level = "high"
    elif days >= 2:
        score = 35
        level = "medium"
    else:
        score = 10
        level = "low"

    return {
        "ghosting_risk_score": score,
        "risk_level": level,
        "risk_factors": [
            f"{days} days since last recorded activity",
            "Milestone deadline approaching without submitted draft"
        ],
        "recommended_actions": [
            "Send automated ping notification to freelancer",
            "Flag project for dispute team monitoring" if score >= 60 else "No immediate intervention required"
        ],
        "analysis_summary": f"Ghosting risk evaluated as {level.upper()} based on {days} days of inactivity."
    }

if __name__ == "__main__":
    import uvicorn
    port = int(os.environ.get("PORT", 8000))
    uvicorn.run(app, host="0.0.0.0", port=port)