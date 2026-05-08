package main

import (
	"context"
	"fmt"
	"log"

	"github.com/markkovari/nestor/internal/adapters/llm"
	"github.com/markkovari/nestor/internal/adapters/task"
	"github.com/markkovari/nestor/internal/core"
)

func main() {
	fmt.Println("🚀 Running Nestor Verification Suite...")

	tasks := []core.Task{
		{
			ID:          "T-101",
			Title:       "Define User Interface",
			Description: "Create the core user repository methods.",
			Status:      "Completed",
		},
		{
			ID:          "T-102",
			Title:       "Implement User Profile",
			Description: "Use User repository to fetch profile data. Depends on T-101.",
			Status:      "In Progress",
		},
		{
			ID:          "T-103",
			Title:       "Quick DB Hack",
			Description: "Directly query the 'users' table in the controller for speed. (Violates ADR-0001)",
			Status:      "Todo",
		},
	}

	ctx := context.Background()
	mockLLM := &llm.MockLLM{}
	fixtureProvider := &task.FixtureTaskProvider{Tasks: tasks}

	engine := core.NewEngine(nil, mockLLM, fixtureProvider)

	fmt.Println("\n--- Step 1: Integrated Analysis ---")
	// The engine uses ADRs internally (currently hardcoded or mocked in RunAnalysis)
	if err := engine.RunAnalysis(ctx); err != nil {
		log.Fatalf("Engine RunAnalysis failed: %v", err)
	}

	fmt.Println("\n✅ Verification Suite Passed!")
}
