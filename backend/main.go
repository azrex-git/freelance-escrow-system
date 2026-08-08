package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
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
