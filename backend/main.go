package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// Initialize a new Fiber app
	// Fiber is highly optimized for performance and is a great fit for a Go microservice.
	app := fiber.New(fiber.Config{
		AppName: "Freelance Escrow Backend API",
	})

	// Middleware
	app.Use(recover.New()) // Recover from panics to prevent the app from crashing
	app.Use(logger.New())  // Log HTTP requests
	app.Use(cors.New())    // Allow cross-origin requests

	// Serve Frontend Static Files
	app.Static("/", "../frontend")

	// Base API route group
	api := app.Group("/api")

	// --- Health Check Endpoint ---
	// Endpoint: GET /api/health
	// Used by orchestrators (like Kubernetes or Docker) to verify if the service is up and running.
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "success",
			"message": "Escrow backend is healthy and running",
		})
	})

	// --- Escrow Routes Group ---
	// Base path: /api/escrow
	// Handlers for creating escrows, releasing funds, dispute resolution, etc.
	escrow := api.Group("/escrow")
	escrow.Post("/", createEscrowHandler)
	escrow.Get("/:id", getEscrowHandler)
	escrow.Post("/evaluate", evaluateWorkHandler)
	// Example endpoints to be implemented:
	// escrow.Post("/:id/release", releaseFundsHandler)
	// escrow.Post("/:id/dispute", raiseDisputeHandler)

	// --- Reputation Routes Group ---
	// Base path: /api/reputation
	// Handlers for fetching or updating the reputation scores of freelancers and clients.
	reputation := api.Group("/reputation")
	reputation.Get("/:userId", getReputationHandler)
	reputation.Post("/:userId/review", addReviewHandler)

	// Start the web server on port 3000
	log.Println("Starting the server on :3000")
	log.Fatal(app.Listen(":3000"))
}

// --- Placeholder Handlers ---
// These functions are stubs for the actual business logic to be implemented later.

func createEscrowHandler(c *fiber.Ctx) error {
	// TODO: Parse request body, interact with the smart contract via Web3, save to DB
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Escrow created successfully",
	})
}

type EvaluateRequest struct {
	WorkDescription string `json:"work_description"`
}

func evaluateWorkHandler(c *fiber.Ctx) error {
	var req EvaluateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	payloadBytes, err := json.Marshal(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to encode request"})
	}

	// Make HTTP POST request to Python AI Service with a timeout
	client := &http.Client{
		Timeout: 30 * time.Second, // 30 second timeout for Gemini AI
	}
	
	resp, err := client.Post("http://localhost:8000/api/evaluate-work", "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "AI service is unreachable"})
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read AI response"})
	}

	// Forward the exact status code from the Python AI Service
	c.Status(resp.StatusCode)
	c.Set("Content-Type", "application/json")
	return c.Send(body)
}

func getEscrowHandler(c *fiber.Ctx) error {
	// id := c.Params("id")
	// TODO: Fetch escrow details from DB or Smart Contract
	return c.JSON(fiber.Map{
		"message": "Escrow details fetched successfully",
	})
}

func getReputationHandler(c *fiber.Ctx) error {
	// userId := c.Params("userId")
	// TODO: Fetch user reputation from DB
	return c.JSON(fiber.Map{
		"message": "Reputation details fetched successfully",
	})
}

func addReviewHandler(c *fiber.Ctx) error {
	// userId := c.Params("userId")
	// TODO: Parse review payload, update reputation score
	return c.JSON(fiber.Map{
		"message": "Review added successfully",
	})
}
