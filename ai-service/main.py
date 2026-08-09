import os
import json
from google import genai
from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from dotenv import load_dotenv

load_dotenv()

client = genai.Client(api_key=os.getenv("GEMINI_API_KEY"))

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
            model='gemini-3.5-flash',
            contents=system_prompt,
            config=genai.types.GenerateContentConfig(
                response_mime_type="application/json"
            )
        )
        return json.loads(response.text)
    except Exception as e:
        import traceback
        traceback.print_exc()
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/api/evaluate-work")
def evaluate_work(request: EvaluateRequest):
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
        
        If a file is provided, thoroughly analyze its code/content.
        If the work violates ANY of the Client Rules, penalize the match_percentage heavily.
        Provide a confidence score on how sure you are about this evaluation.
        """
        response = client.models.generate_content(
            model='gemini-3.5-flash',
            contents=system_prompt,
            config=genai.types.GenerateContentConfig(
                response_mime_type="application/json"
            )
        )
        
        result = json.loads(response.text)
        return result
    except Exception as e:
        import traceback
        traceback.print_exc()
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/api/resolve-dispute")
def resolve_dispute(request: DisputeRequest):
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
        
        The percentages must add up to 100.
        If ghosting is detected, heavily penalize the ghosting party.
        """
        
        response = client.models.generate_content(
            model='gemini-3.5-flash',
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
        import traceback
        traceback.print_exc()
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/api/ghosting-analysis")
def ghosting_analysis(request: GhostingAnalysisRequest):
    """AI-powered ghosting risk analysis based on project context and communication patterns."""
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
            model='gemini-3.5-flash',
            contents=system_prompt,
            config=genai.types.GenerateContentConfig(
                response_mime_type="application/json"
            )
        )
        return json.loads(response.text)
    except Exception as e:
        import traceback
        traceback.print_exc()
        raise HTTPException(status_code=500, detail=str(e))