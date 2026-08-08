package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// --- Models ---
type Milestone struct {
	ID          string  `json:"id"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Status      string  `json:"status"` // pending, submitted, approved, disputed
}

type AuditLog struct {
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"`
	Details   string    `json:"details"`
}

type DisputeResolution struct {
	Reasoning          string                 `json:"reasoning"`
	ConfidenceScore    int                    `json:"confidence_score"`
	BehavioralAnalysis map[string]interface{} `json:"behavioral_analysis"`
	EvidenceSummary    []interface{}          `json:"evidence_summary"`
}

type Escrow struct {
	ID                string             `json:"id"`
	ClientID          string             `json:"client_id"`
	FreelancerID      string             `json:"freelancer_id"`
	TotalAmount       float64            `json:"total_amount"`
	Milestones        []Milestone        `json:"milestones"`
	Status            string             `json:"status"` // active, completed, disputed
	AuditLogs         []AuditLog         `json:"audit_logs"`
	DisputeResolution *DisputeResolution `json:"dispute_resolution,omitempty"`
}

type User struct {
	ID         string  `json:"id"`
	Reputation float64 `json:"reputation"`
	Role       string  `json:"role"` // client, freelancer
}

// --- Helpers ---
func appendAuditLog(escrow *Escrow, action string, details string) {
	escrow.AuditLogs = append(escrow.AuditLogs, AuditLog{
		Timestamp: time.Now(),
		Action:    action,
		Details:   details,
	})
}

// --- In-Memory Data Stores ---
var (
	usersMutex sync.RWMutex
	users      = map[string]User{
		"client1":     {ID: "client1", Reputation: 5.0, Role: "client"},
		"freelancer1": {ID: "freelancer1", Reputation: 4.5, Role: "freelancer"},
	}

	escrowsMutex sync.RWMutex
	escrows      = map[string]*Escrow{}
)

func main() {
	app := fiber.New(fiber.Config{
		AppName: "Freelance Escrow Backend API",
	})

	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New())

	app.Static("/", "../frontend")

	api := app.Group("/api")

	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "success", "message": "Escrow backend is healthy"})
	})

	escrowRoutes := api.Group("/escrow")
	escrowRoutes.Post("/auto-plan", autoPlanHandler)
	escrowRoutes.Post("/", createEscrowHandler)
	escrowRoutes.Get("/:id", getEscrowHandler)
	escrowRoutes.Post("/:id/milestone/:milestoneId/evaluate", evaluateMilestoneHandler)
	escrowRoutes.Post("/:id/dispute", resolveDisputeHandler)

	reputationRoutes := api.Group("/reputation")
	reputationRoutes.Get("/:userId", getReputationHandler)

	log.Println("Starting the server on :3000")
	log.Fatal(app.Listen(":3000"))
}

// --- Handlers ---

func autoPlanHandler(c *fiber.Ctx) error {
	var req struct {
		ProjectDescription string  `json:"project_description"`
		TotalBudget        float64 `json:"total_budget"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	payloadBytes, _ := json.Marshal(req)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post("http://localhost:8000/api/generate-milestones", "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "AI service unreachable"})
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	c.Status(resp.StatusCode)
	c.Set("Content-Type", "application/json")
	return c.Send(body)
}

func createEscrowHandler(c *fiber.Ctx) error {
	var req struct {
		ClientID     string      `json:"client_id"`
		FreelancerID string      `json:"freelancer_id"`
		Milestones   []Milestone `json:"milestones"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	total := 0.0
	for i := range req.Milestones {
		req.Milestones[i].ID = uuid.New().String()
		req.Milestones[i].Status = "pending"
		total += req.Milestones[i].Amount
	}

	escrow := &Escrow{
		ID:           uuid.New().String(),
		ClientID:     req.ClientID,
		FreelancerID: req.FreelancerID,
		TotalAmount:  total,
		Milestones:   req.Milestones,
		Status:       "active",
		AuditLogs:    []AuditLog{},
	}

	appendAuditLog(escrow, "CREATED", fmt.Sprintf("Escrow created with %d milestones.", len(req.Milestones)))

	escrowsMutex.Lock()
	escrows[escrow.ID] = escrow
	escrowsMutex.Unlock()

	return c.Status(fiber.StatusCreated).JSON(escrow)
}

func getEscrowHandler(c *fiber.Ctx) error {
	id := c.Params("id")

	escrowsMutex.RLock()
	escrow, exists := escrows[id]
	escrowsMutex.RUnlock()

	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Escrow not found"})
	}
	return c.JSON(escrow)
}

func evaluateMilestoneHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	mId := c.Params("milestoneId")

	var req struct {
		SubmittedWork string `json:"submitted_work"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	escrowsMutex.RLock()
	escrow, exists := escrows[id]
	escrowsMutex.RUnlock()

	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Escrow not found"})
	}

	var milestone *Milestone
	for i := range escrow.Milestones {
		if escrow.Milestones[i].ID == mId {
			milestone = &escrow.Milestones[i]
			break
		}
	}

	if milestone == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Milestone not found"})
	}

	// Call AI Service
	aiReq := map[string]string{
		"milestone_description": milestone.Description,
		"submitted_work":        req.SubmittedWork,
	}
	payloadBytes, _ := json.Marshal(aiReq)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post("http://localhost:8000/api/evaluate-work", "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "AI service unreachable"})
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Update state if match > 80 as a naive example
	var aiResp map[string]interface{}
	json.Unmarshal(body, &aiResp)
	
	escrowsMutex.Lock()
	if match, ok := aiResp["match_percentage"].(float64); ok && match >= 80 {
		milestone.Status = "approved"
		appendAuditLog(escrow, "MILESTONE_APPROVED", fmt.Sprintf("AI matched milestone %s with %.0f%% accuracy.", milestone.ID, match))
	} else {
		appendAuditLog(escrow, "MILESTONE_REJECTED", fmt.Sprintf("AI rejected milestone %s.", milestone.ID))
	}
	escrowsMutex.Unlock()

	c.Status(resp.StatusCode)
	c.Set("Content-Type", "application/json")
	return c.Send(body)
}

func resolveDisputeHandler(c *fiber.Ctx) error {
	id := c.Params("id")

	var req struct {
		MilestoneID       string `json:"milestone_id"`
		ClientComplaint   string `json:"client_complaint"`
		FreelancerDefense string `json:"freelancer_defense"`
		DeliverablesText  string `json:"deliverables_text"`
		CommunicationLogs string `json:"communication_logs"`
		DeliveryTimeline  string `json:"delivery_timeline"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	escrowsMutex.RLock()
	escrow, exists := escrows[id]
	escrowsMutex.RUnlock()

	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Escrow not found"})
	}

	var milestoneAmount float64
	for i := range escrow.Milestones {
		if escrow.Milestones[i].ID == req.MilestoneID {
			escrow.Milestones[i].Status = "disputed"
			milestoneAmount = escrow.Milestones[i].Amount
			break
		}
	}

	escrowsMutex.Lock()
	escrow.Status = "disputed"
	appendAuditLog(escrow, "DISPUTE_RAISED", "A dispute was raised for milestone: "+req.MilestoneID)
	escrowsMutex.Unlock()

	// Call AI Service
	aiReq := map[string]interface{}{
		"project_id":         id,
		"milestone_amount":   milestoneAmount,
		"client_complaint":   req.ClientComplaint,
		"freelancer_defense": req.FreelancerDefense,
		"deliverables_text":  req.DeliverablesText,
		"communication_logs": req.CommunicationLogs,
		"delivery_timeline":  req.DeliveryTimeline,
	}
	payloadBytes, _ := json.Marshal(aiReq)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post("http://localhost:8000/api/resolve-dispute", "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "AI service unreachable"})
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Parse rich AI response
	var aiResp map[string]interface{}
	if err := json.Unmarshal(body, &aiResp); err == nil {
		escrowsMutex.Lock()
		
		reasoning, _ := aiResp["reasoning"].(string)
		confScore, _ := aiResp["confidence_score"].(float64)
		
		var behavioral map[string]interface{}
		if val, ok := aiResp["behavioral_analysis"].(map[string]interface{}); ok {
			behavioral = val
		}
		
		var evidence []interface{}
		if val, ok := aiResp["evidence_summary"].([]interface{}); ok {
			evidence = val
		}

		escrow.DisputeResolution = &DisputeResolution{
			Reasoning:          reasoning,
			ConfidenceScore:    int(confScore),
			BehavioralAnalysis: behavioral,
			EvidenceSummary:    evidence,
		}
		appendAuditLog(escrow, "DISPUTE_RESOLVED", fmt.Sprintf("AI resolved dispute with %d%% confidence.", int(confScore)))
		escrowsMutex.Unlock()

		fPayout, fOk := aiResp["freelancer_payout"].(float64)
		if fOk {
			usersMutex.Lock()
			f := users[escrow.FreelancerID]
			clientUser := users[escrow.ClientID]

			if fPayout > (milestoneAmount / 2) {
				// Freelancer won most of it
				f.Reputation += 0.1
				clientUser.Reputation -= 0.1
			} else {
				// Client won most of it
				f.Reputation -= 0.2
				clientUser.Reputation += 0.1
			}
			users[escrow.FreelancerID] = f
			users[escrow.ClientID] = clientUser
			usersMutex.Unlock()
		}
	}

	c.Status(resp.StatusCode)
	c.Set("Content-Type", "application/json")
	return c.Send(body)
}

func getReputationHandler(c *fiber.Ctx) error {
	userId := c.Params("userId")

	usersMutex.RLock()
	user, exists := users[userId]
	usersMutex.RUnlock()

	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}
	return c.JSON(user)
}
