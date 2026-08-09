package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
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
	ID                 string    `json:"id"`
	Description        string    `json:"description"`
	Amount             float64   `json:"amount"`
	ClientInstructions string    `json:"client_instructions"`
	Status             string    `json:"status"` // pending, submitted, approved, rejected, disputed, released
	DeadlineDays       int       `json:"deadline_days"`
	DeadlineAt         time.Time `json:"deadline_at"`
	SubmittedAt        *time.Time `json:"submitted_at,omitempty"`
	SubmittedWork      string    `json:"submitted_work,omitempty"`
	AIMatchScore       *int      `json:"ai_match_score,omitempty"`
}

type AuditLog struct {
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"`
	Details   string    `json:"details"`
	Actor     string    `json:"actor"`
}

type DisputeResolution struct {
	Reasoning          string                 `json:"reasoning"`
	ConfidenceScore    int                    `json:"confidence_score"`
	BehavioralAnalysis map[string]interface{} `json:"behavioral_analysis"`
	EvidenceSummary    []interface{}          `json:"evidence_summary"`
	FreelancerPayout   float64                `json:"freelancer_payout"`
	ClientRefund       float64                `json:"client_refund"`
}

type Escrow struct {
	ID                string             `json:"id"`
	ProjectName       string             `json:"project_name"`
	ClientID          string             `json:"client_id"`
	FreelancerID      string             `json:"freelancer_id"`
	TotalAmount       float64            `json:"total_amount"`
	Milestones        []Milestone        `json:"milestones"`
	Status            string             `json:"status"` // active, completed, disputed, ghosted
	CreatedAt         time.Time          `json:"created_at"`
	AuditLogs         []AuditLog         `json:"audit_logs"`
	DisputeResolution *DisputeResolution `json:"dispute_resolution,omitempty"`
	GhostingRisk      string             `json:"ghosting_risk"` // none, low, medium, high, critical
}

type User struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	Reputation        float64 `json:"reputation"`
	Role              string  `json:"role"`
	CompletedProjects int     `json:"completed_projects"`
	DisputesWon       int     `json:"disputes_won"`
	DisputesLost      int     `json:"disputes_lost"`
	GhostingIncidents int     `json:"ghosting_incidents"`
	TrustTier         string  `json:"trust_tier"` // Newcomer, Bronze, Silver, Gold, Platinum
}

// --- Helpers ---

func appendAuditLog(escrow *Escrow, action, details, actor string) {
	escrow.AuditLogs = append(escrow.AuditLogs, AuditLog{
		Timestamp: time.Now(),
		Action:    action,
		Details:   details,
		Actor:     actor,
	})
}

func calculateTrustTier(u *User) string {
	score := u.Reputation
	completed := u.CompletedProjects
	if completed < 1 {
		return "Newcomer"
	}
	if score >= 4.5 && completed >= 5 && u.GhostingIncidents == 0 {
		return "Platinum"
	}
	if score >= 4.0 && completed >= 3 {
		return "Gold"
	}
	if score >= 3.5 && completed >= 2 {
		return "Silver"
	}
	if score >= 2.5 {
		return "Bronze"
	}
	return "Newcomer"
}

func clampReputation(rep float64) float64 {
	if rep < 0 {
		return 0
	}
	if rep > 5 {
		return 5
	}
	return math.Round(rep*100) / 100
}

func calculateGhostingRisk(escrow *Escrow) string {
	now := time.Now()
	highestRisk := "none"

	for _, m := range escrow.Milestones {
		if m.Status != "pending" {
			continue
		}
		if m.DeadlineAt.IsZero() {
			continue
		}
		hoursLeft := m.DeadlineAt.Sub(now).Hours()
		daysLeft := hoursLeft / 24

		risk := "none"
		if hoursLeft < 0 {
			risk = "critical" // Past deadline
		} else if daysLeft < 1 {
			risk = "high"
		} else if daysLeft < 3 {
			risk = "medium"
		} else if daysLeft < 7 {
			risk = "low"
		}

		// Promote risk level
		riskOrder := map[string]int{"none": 0, "low": 1, "medium": 2, "high": 3, "critical": 4}
		if riskOrder[risk] > riskOrder[highestRisk] {
			highestRisk = risk
		}
	}
	return highestRisk
}

// --- In-Memory Data Stores ---

var (
	usersMutex sync.RWMutex
	users      = map[string]User{
		"client1": {
			ID: "client1", Name: "Alice (Client)", Reputation: 0, Role: "client",
			CompletedProjects: 0, DisputesWon: 0, DisputesLost: 0, GhostingIncidents: 0,
			TrustTier: "Newcomer",
		},
		"freelancer1": {
			ID: "freelancer1", Name: "Bob (Freelancer)", Reputation: 0, Role: "freelancer",
			CompletedProjects: 0, DisputesWon: 0, DisputesLost: 0, GhostingIncidents: 0,
			TrustTier: "Newcomer",
		},
	}

	escrowsMutex sync.RWMutex
	escrows      = map[string]*Escrow{}
)

func main() {
	app := fiber.New(fiber.Config{
		AppName: "SafeWork Anti-Ghosting Escrow API",
	})

	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New())

	app.Static("/", "../frontend")

	api := app.Group("/api")

	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "success", "message": "SafeWork backend is healthy"})
	})

	escrowRoutes := api.Group("/escrow")
	escrowRoutes.Post("/auto-plan", autoPlanHandler)
	escrowRoutes.Post("/", createEscrowHandler)
	escrowRoutes.Get("/all", listEscrowsHandler)
	escrowRoutes.Get("/:id", getEscrowHandler)
	escrowRoutes.Post("/:id/milestone/:milestoneId/submit", submitMilestoneHandler)
	escrowRoutes.Post("/:id/milestone/:milestoneId/approve", approveMilestoneHandler)
	escrowRoutes.Post("/:id/milestone/:milestoneId/reject", rejectMilestoneHandler)
	escrowRoutes.Post("/:id/milestone/:milestoneId/release", releaseMilestoneHandler)
	escrowRoutes.Post("/:id/dispute", resolveDisputeHandler)
	escrowRoutes.Get("/:id/ghosting-check", ghostingCheckHandler)

	reputationRoutes := api.Group("/reputation")
	reputationRoutes.Get("/:userId", getReputationHandler)

	log.Println("🛡️  SafeWork Anti-Ghosting Escrow running on :3000")
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
	client := &http.Client{Timeout: 45 * time.Second}
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
		ProjectName  string      `json:"project_name"`
		ClientID     string      `json:"client_id"`
		FreelancerID string      `json:"freelancer_id"`
		Milestones   []Milestone `json:"milestones"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	now := time.Now()
	total := 0.0
	for i := range req.Milestones {
		req.Milestones[i].ID = uuid.New().String()
		req.Milestones[i].Status = "pending"
		total += req.Milestones[i].Amount
		if req.Milestones[i].DeadlineDays > 0 {
			req.Milestones[i].DeadlineAt = now.Add(time.Duration(req.Milestones[i].DeadlineDays) * 24 * time.Hour)
		}
	}

	escrow := &Escrow{
		ID:           uuid.New().String(),
		ProjectName:  req.ProjectName,
		ClientID:     req.ClientID,
		FreelancerID: req.FreelancerID,
		TotalAmount:  total,
		Milestones:   req.Milestones,
		Status:       "active",
		CreatedAt:    now,
		AuditLogs:    []AuditLog{},
		GhostingRisk: "none",
	}

	appendAuditLog(escrow, "ESCROW_CREATED", fmt.Sprintf("Escrow created with %d milestones totalling ₹%.0f", len(req.Milestones), total), req.ClientID)

	escrowsMutex.Lock()
	escrows[escrow.ID] = escrow
	escrowsMutex.Unlock()

	return c.Status(fiber.StatusCreated).JSON(escrow)
}

func listEscrowsHandler(c *fiber.Ctx) error {
	escrowsMutex.RLock()
	defer escrowsMutex.RUnlock()

	list := make([]*Escrow, 0, len(escrows))
	for _, e := range escrows {
		// Update ghosting risk dynamically
		e.GhostingRisk = calculateGhostingRisk(e)
		list = append(list, e)
	}
	return c.JSON(list)
}

func getEscrowHandler(c *fiber.Ctx) error {
	id := c.Params("id")

	escrowsMutex.RLock()
	escrow, exists := escrows[id]
	escrowsMutex.RUnlock()

	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Escrow not found"})
	}
	escrow.GhostingRisk = calculateGhostingRisk(escrow)
	return c.JSON(escrow)
}

func submitMilestoneHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	mId := c.Params("milestoneId")

	var req struct {
		SubmittedWork string `json:"submitted_work"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	escrowsMutex.Lock()
	defer escrowsMutex.Unlock()

	escrow, exists := escrows[id]
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Escrow not found"})
	}

	for i := range escrow.Milestones {
		if escrow.Milestones[i].ID == mId {
			now := time.Now()
			escrow.Milestones[i].Status = "submitted"
			escrow.Milestones[i].SubmittedAt = &now
			escrow.Milestones[i].SubmittedWork = req.SubmittedWork

			lateStr := ""
			if !escrow.Milestones[i].DeadlineAt.IsZero() && now.After(escrow.Milestones[i].DeadlineAt) {
				hoursLate := now.Sub(escrow.Milestones[i].DeadlineAt).Hours()
				lateStr = fmt.Sprintf(" (%.0f hours late)", hoursLate)
			}

			appendAuditLog(escrow, "MILESTONE_SUBMITTED", fmt.Sprintf("Work submitted for milestone: %s%s", escrow.Milestones[i].Description, lateStr), escrow.FreelancerID)
			break
		}
	}

	escrow.GhostingRisk = calculateGhostingRisk(escrow)
	return c.JSON(escrow)
}

func approveMilestoneHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	mId := c.Params("milestoneId")

	escrowsMutex.Lock()
	defer escrowsMutex.Unlock()

	escrow, exists := escrows[id]
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Escrow not found"})
	}

	for i := range escrow.Milestones {
		if escrow.Milestones[i].ID == mId {
			escrow.Milestones[i].Status = "approved"
			appendAuditLog(escrow, "MILESTONE_APPROVED", fmt.Sprintf("Client approved '%s'", escrow.Milestones[i].Description), escrow.ClientID)
			break
		}
	}
	return c.JSON(escrow)
}

func rejectMilestoneHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	mId := c.Params("milestoneId")

	escrowsMutex.Lock()
	defer escrowsMutex.Unlock()

	escrow, exists := escrows[id]
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Escrow not found"})
	}

	for i := range escrow.Milestones {
		if escrow.Milestones[i].ID == mId {
			escrow.Milestones[i].Status = "disputed"
			appendAuditLog(escrow, "MILESTONE_REJECTED", fmt.Sprintf("Client rejected '%s' (Sent to Dispute Center)", escrow.Milestones[i].Description), escrow.ClientID)
			break
		}
	}
	return c.JSON(escrow)
}


func releaseMilestoneHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	mId := c.Params("milestoneId")

	escrowsMutex.Lock()
	escrow, exists := escrows[id]
	if !exists {
		escrowsMutex.Unlock()
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Escrow not found"})
	}

	var releasedAmount float64
	for i := range escrow.Milestones {
		if escrow.Milestones[i].ID == mId {
			if escrow.Milestones[i].Status != "approved" {
				escrowsMutex.Unlock()
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Milestone must be approved before release"})
			}
			escrow.Milestones[i].Status = "released"
			releasedAmount = escrow.Milestones[i].Amount
			appendAuditLog(escrow, "FUNDS_RELEASED", fmt.Sprintf("₹%.0f released for milestone: %s", releasedAmount, escrow.Milestones[i].Description), escrow.ClientID)
			break
		}
	}

	// Check if all milestones are released
	allReleased := true
	for _, m := range escrow.Milestones {
		if m.Status != "released" {
			allReleased = false
			break
		}
	}
	if allReleased {
		escrow.Status = "completed"
		appendAuditLog(escrow, "ESCROW_COMPLETED", "All milestones released. Project complete.", "SYSTEM")
	}
	freelancerId := escrow.FreelancerID
	clientId := escrow.ClientID
	escrowsMutex.Unlock()

	// Update reputation
	usersMutex.Lock()
	if f, exists := users[freelancerId]; exists {
		f.Reputation = clampReputation(f.Reputation + 0.25) // +0.25 per milestone
		if allReleased {
			f.CompletedProjects++
			f.Reputation = clampReputation(f.Reputation + 0.5) // bonus for completing full project
		}
		f.TrustTier = calculateTrustTier(&f)
		users[freelancerId] = f
	}
	if cl, exists := users[clientId]; exists {
		cl.Reputation = clampReputation(cl.Reputation + 0.15) // client gets rep for paying
		if allReleased {
			cl.CompletedProjects++
			cl.Reputation = clampReputation(cl.Reputation + 0.35) // bonus for completing project
		}
		cl.TrustTier = calculateTrustTier(&cl)
		users[clientId] = cl
	}
	usersMutex.Unlock()

	return c.JSON(fiber.Map{"status": "success", "released_amount": releasedAmount, "all_released": allReleased})
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
	appendAuditLog(escrow, "DISPUTE_RAISED", "Dispute raised: "+req.ClientComplaint[:min(60, len(req.ClientComplaint))]+"...", escrow.ClientID)
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

	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Post("http://localhost:8000/api/resolve-dispute", "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "AI service unreachable"})
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var aiResp map[string]interface{}
	if err := json.Unmarshal(body, &aiResp); err == nil {
		escrowsMutex.Lock()

		reasoning, _ := aiResp["reasoning"].(string)
		confScore, _ := aiResp["confidence_score"].(float64)
		fPayout, _ := aiResp["freelancer_payout"].(float64)
		cRefund, _ := aiResp["client_refund"].(float64)

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
			FreelancerPayout:   fPayout,
			ClientRefund:       cRefund,
		}
		appendAuditLog(escrow, "DISPUTE_RESOLVED", fmt.Sprintf("AI resolved with %d%% confidence. Freelancer: ₹%.0f, Client: ₹%.0f", int(confScore), fPayout, cRefund), "AI_ARBITER")
		escrowsMutex.Unlock()

		// Update reputation
		usersMutex.Lock()
		f := users[escrow.FreelancerID]
		cl := users[escrow.ClientID]

		if fPayout > cRefund {
			f.Reputation = clampReputation(f.Reputation + 0.1)
			f.DisputesWon++
			cl.Reputation = clampReputation(cl.Reputation - 0.2)
			cl.DisputesLost++
		} else {
			f.Reputation = clampReputation(f.Reputation - 0.3)
			f.DisputesLost++
			cl.Reputation = clampReputation(cl.Reputation + 0.1)
			cl.DisputesWon++
		}
		f.TrustTier = calculateTrustTier(&f)
		cl.TrustTier = calculateTrustTier(&cl)
		users[escrow.FreelancerID] = f
		users[escrow.ClientID] = cl
		usersMutex.Unlock()
	}

	c.Status(resp.StatusCode)
	c.Set("Content-Type", "application/json")
	return c.Send(body)
}

func ghostingCheckHandler(c *fiber.Ctx) error {
	id := c.Params("id")

	escrowsMutex.RLock()
	escrow, exists := escrows[id]
	escrowsMutex.RUnlock()

	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Escrow not found"})
	}

	risk := calculateGhostingRisk(escrow)
	now := time.Now()

	overdueCount := 0
	var overdueMilestones []map[string]interface{}
	for _, m := range escrow.Milestones {
		if m.Status == "pending" && !m.DeadlineAt.IsZero() && now.After(m.DeadlineAt) {
			overdueCount++
			overdueMilestones = append(overdueMilestones, map[string]interface{}{
				"id":          m.ID,
				"description": m.Description,
				"hours_overdue": now.Sub(m.DeadlineAt).Hours(),
			})
		}
	}

	return c.JSON(fiber.Map{
		"escrow_id":           id,
		"ghosting_risk":       risk,
		"overdue_milestones":  overdueCount,
		"overdue_details":     overdueMilestones,
		"recommendation":      getGhostingRecommendation(risk),
	})
}

func getGhostingRecommendation(risk string) string {
	switch risk {
	case "critical":
		return "🚨 CRITICAL: Deadline has passed with no submission. Consider auto-refunding the client and flagging the freelancer's reputation."
	case "high":
		return "⚠️ HIGH RISK: Less than 24 hours to deadline with no submission. Send an urgent reminder."
	case "medium":
		return "🔶 MODERATE: Deadline is approaching. Consider checking in with the freelancer."
	case "low":
		return "📋 LOW: Deadline is approaching within a week. Monitoring."
	default:
		return "✅ No ghosting risk detected."
	}
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
