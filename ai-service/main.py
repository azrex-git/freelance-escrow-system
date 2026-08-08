import os
import json
import google.generativeai as genai
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from dotenv import load_dotenv

load_dotenv()

genai.configure(api_key=os.getenv("GEMINI_API_KEY"))

model = genai.GenerativeModel('gemini-1.5-pro')

app = FastAPI(title="AntiLabs Escrow AI Arbiter")

class DisputeRequest(BaseModel):
    project_id: int
    milestone_amount: float
    client_complaint: str
    freelancer_defense: str
    deliverables_text: str

class EvaluateRequest(BaseModel):
    work_description: str

@app.get("/api/health")
def health_check():
    return {"status": "AI Service is active and analyzing."}

@app.post("/api/evaluate-work")
def evaluate_work(request: EvaluateRequest):
    try:
        system_prompt = f"""
        You are an AI Judge for a freelance platform.
        Evaluate the following submitted work or link and determine its quality/completeness:
        {request.work_description}
        
        Output ONLY a valid JSON object with no markdown formatting or extra text.
        Format required:
        {{
            "match_percentage": <int 0-100>,
            "feedback": "<short string explaining the evaluation>"
        }}
        """
        response = model.generate_content(system_prompt)
        
        # Clean potential markdown from response if Gemini includes it
        raw_text = response.text.replace('```json', '').replace('```', '').strip()
        result = json.loads(raw_text)
        return result
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/api/resolve-dispute")
def resolve_dispute(request: DisputeRequest):
    try:
        system_prompt = f"""
        You are an impartial dispute resolution arbiter for a freelance escrow smart contract.
        Analyze the following dispute and determine a fair payout split.
        
        Client's Complaint: {request.client_complaint}
        Freelancer's Defense: {request.freelancer_defense}
        Deliverables Provided: {request.deliverables_text}
        
        You must output ONLY a valid JSON object with no markdown formatting or extra text.
        Format required:
        {{
            "freelancer_percentage": <int 0-100>,
            "client_percentage": <int 0-100>,
            "reasoning": "<short string explaining the decision>"
        }}
        """
        
        response = model.generate_content(system_prompt)
        result = json.loads(response.text)
        
        freelancer_payout = (result["freelancer_percentage"] / 100) * request.milestone_amount
        client_refund = (result["client_percentage"] / 100) * request.milestone_amount
        
        return {
            "project_id": request.project_id,
            "freelancer_payout": freelancer_payout,
            "client_refund": client_refund,
            "reasoning": result["reasoning"]
        }

    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))