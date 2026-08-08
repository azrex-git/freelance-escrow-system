import os
import json
from google import genai
from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field
from dotenv import load_dotenv

load_dotenv()

client = genai.Client(api_key=os.getenv("GEMINI_API_KEY"))

app = FastAPI(title="AntiLabs Escrow AI Arbiter")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

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
    submitted_work: str

class ProjectIdeaRequest(BaseModel):
    project_description: str
    total_budget: float

@app.get("/api/health")
def health_check():
    return {"status": "AI Service is active and analyzing."}

@app.post("/api/generate-milestones")
def generate_milestones(request: ProjectIdeaRequest):
    try:
        system_prompt = f"""
        You are an expert technical project manager and escrow advisor.
        A user wants to start the following project on an escrow platform. Break it down into 3-5 logical milestones.
        
        Project Description: {request.project_description}
        Total Budget: {request.total_budget}
        
        Assign a portion of the total budget to each milestone based on its complexity.
        Make sure the total budget across all milestones equals the provided total budget.
        
        Return ONLY valid JSON in the following format:
        {{
            "milestones": [
                {{
                    "description": "Short description of the milestone",
                    "amount": 1000
                }}
            ]
        }}
        """
        response = client.models.generate_content(
            model='gemini-3.6-flash',
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
        system_prompt = f"""
        You are an AI Judge for a freelance platform.
        Evaluate the following submitted work against the milestone description and determine its quality/completeness:
        
        Milestone Description: {request.milestone_description}
        Submitted Work: {request.submitted_work}
        
        Return ONLY valid JSON exactly matching this format:
        {{
            "match_percentage": <integer 0-100>,
            "feedback": "<detailed constructive feedback>",
            "confidence_score": <integer 0-100>,
            "behavioral_analysis": {{
                "client_professionalism": <integer 0-100>,
                "freelancer_professionalism": <integer 0-100>
            }},
            "evidence_summary": [
                "<evidence point 1>",
                "<evidence point 2>"
            ]
        }}
        
        If the work is a Github URL, assume it's valid code but penalize slightly if there is no description.
        If the work is just text, evaluate its completeness based on the milestone.
        Provide a confidence score on how sure you are about this evaluation.
        Infer professionalism scores from the way the submitted work was presented (e.g. detailed and polite vs abrupt).
        Provide 1-2 points in the evidence summary justifying the evaluation.
        """
        response = client.models.generate_content(
            model='gemini-3.6-flash',
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
        You are an impartial dispute resolution arbiter for a freelance escrow smart contract.
        Analyze the following dispute and determine a fair payout split. Pay special attention to the communication tone, clarity, and the delivery timeline.
        
        Client's Complaint: {request.client_complaint}
        Freelancer's Defense: {request.freelancer_defense}
        Deliverables Provided: {request.deliverables_text}
        Communication Logs: {request.communication_logs}
        Delivery Timeline context: {request.delivery_timeline}
        
        You must output ONLY a valid JSON object with no markdown formatting or extra text.
        Format required:
        {{
            "freelancer_percentage": <int 0-100>,
            "client_percentage": <int 0-100>,
            "reasoning": "<short string explaining the decision>",
            "confidence_score": <int 0-100>,
            "behavioral_analysis": {{
                "client_professionalism_score": <int 0-100>,
                "freelancer_professionalism_score": <int 0-100>,
                "tone_summary": "<short string>"
            }},
            "evidence_summary": [
                "<point 1>",
                "<point 2>"
            ]
        }}
        """
        
        response = client.models.generate_content(
            model='gemini-3.6-flash',
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
            "behavioral_analysis": result.get("behavioral_analysis", {}),
            "evidence_summary": result.get("evidence_summary", [])
        }

    except Exception as e:
        import traceback
        traceback.print_exc()
        raise HTTPException(status_code=500, detail=str(e))