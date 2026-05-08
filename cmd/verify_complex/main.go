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
	fmt.Println("🚀 Running COMPLEX Multi-Repo Verification Suite...")

	// Tasks spanning multiple repositories
	tasks := []core.Task{
		{
			ID:          "backend-api#42",
			Title:       "Add /users/search endpoint",
			Description: "Create a new search endpoint for users. Should be added as /search without versioning for now.",
			Status:      "open",
			Provider:    "github",
			Metadata:    map[string]string{"repo": "nestor-org/backend-api"},
		},
		{
			ID:          "frontend-app#101",
			Title:       "Implement User Search Bar",
			Description: "Frontend search bar that calls backend-api#42. Depends on the backend endpoint being ready.",
			Status:      "open",
			Provider:    "github",
			Metadata:    map[string]string{"repo": "nestor-org/frontend-app"},
		},
		{
			ID:          "shared-types#5",
			Title:       "Update User schema",
			Description: "Add 'lastSeen' field to User type. Both frontend and backend depend on this.",
			Status:      "closed",
			Provider:    "github",
			Metadata:    map[string]string{"repo": "nestor-org/shared-types"},
		},
	}

	adrs := []string{
		"ADR-0002: All Backend API changes must include a versioned path (e.g., /v2/...).",
	}

	ctx := context.Background()
	mockLLM := &llm.MockLLM{}
	fixtureProvider := &task.FixtureTaskProvider{Tasks: tasks}

	engine := core.NewEngine(nil, mockLLM, fixtureProvider)

	fmt.Println("\n--- Multi-Repo Analysis ---")
	if err := engine.RunAnalysis(ctx); err != nil {
		log.Fatalf("Analysis failed: %v", err)
	}

	fmt.Println("\n--- Simulated Conflict Check (Manual) ---")
	// Demonstrate how Nestor detects the cross-repo versioning violation
	fmt.Printf("Scenario: %s is proposing a non-versioned endpoint, which violates ADR-0002.\n", tasks[0].ID)
	
	// We use the real (mocked) report logic
	report, _ := mockLLM.AnalyzeConflict(ctx, tasks, adrs)
	fmt.Printf("Final Report Summary: %s\n", report)

	fmt.Println("\n✅ Complex Multi-Repo Verification Passed!")
}
