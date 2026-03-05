# Telegram Quiz Bot: Refactoring & Feature Plan

This document outlines the step-by-step tasks required to refactor the current Telegram Quiz Bot and implement new features, including a Web UI and an MCP (Model Context Protocol) server.

## Phase 1: Domain Refactoring & Repository Pattern [COMPLETED]

The goal of this phase is to reduce complexity, improve extensibility, and introduce a structured learning curriculum.

*   **Task 1.1: Define Core Domain Models.** [COMPLETED] Refactor the data model to support a hierarchical structure:
    *   **`Segment`**: A high-level learning category (e.g., "Basics", "Intermediate"). Needs a title, a description of what will be learned, and an order index.
    *   **`Quiz`**: A specific topic within a segment (e.g., "Greetings", "Numbers 1-10"). Needs a title, a description of the content, and an order index.
    *   **`Question`**: An individual multiple-choice question belonging to a Quiz.
*   **Task 1.2: Implement Repository Interfaces.** [COMPLETED] Create interface definitions for the data access layer to decouple the database implementation from the business logic.
    *   `SegmentRepository`
    *   `QuizRepository`
    *   `QuestionRepository`
    *   `UserRepository`
*   **Task 1.3: Update SQLite Implementation.** [COMPLETED] Implement the repository interfaces using SQLite. Write migration scripts to update the current schema to the new hierarchical structure (`segments` -> `quizzes` -> `questions`).
*   **Task 1.4: Update Bot & Core Logic.** [COMPLETED] Refactor the Telegram bot handlers, scheduler, and LLM clients to use the new domain models and repository interfaces.
    *   Replace `LEARNINGPLAN.md` with database queries that dynamically pull the current user's learning progress from the `Segment` -> `Quiz` hierarchy.

## Phase 2: Web Server & API Backend

This phase introduces an embedded HTTP server to serve the management API and the frontend UI.

*   **Task 2.1: Setup `net/http` Server.** Initialize an HTTP server within the Go application that runs concurrently with the Telegram bot and scheduler.
*   **Task 2.2: Implement REST APIs.** Create endpoints for CRUD operations on the domain models:
    *   `GET/POST/PUT/DELETE /api/segments`
    *   `GET/POST/PUT/DELETE /api/segments/{id}/quizzes`
    *   `GET/POST/PUT/DELETE /api/quizzes/{id}/questions`
    *   `GET /api/plan` (View the overall learning curriculum)
*   **Task 2.3: Integrate API with Repositories.** Connect the HTTP handlers to the new repository layer for data persistence.

## Phase 3: Clean React + Vite + Tailwind Frontend

This phase provides a user-friendly interface to manage the learning curriculum.

*   **Task 3.1: Initialize Frontend Project.** Set up a new Vite + React + Tailwind CSS project in a `frontend/` directory.
*   **Task 3.2: Build UI Components.** Develop a super clean, modern UI to manage the learning curriculum:
    *   Dashboard to view the overall plan.
    *   Segment management (add, edit descriptions, change ordering).
    *   Quiz management within segments (add, edit topics and descriptions, change ordering).
    *   Question curation (view, add, edit, delete generated questions for a quiz).
*   **Task 3.3: API Integration.** Connect the React frontend to the Go backend APIs using standard `fetch` or a library like `axios`.
*   **Task 3.4: Serve Frontend from Go.** Configure the Go `net/http` server to statically serve the built React frontend files (e.g., from a `dist/` folder) so the entire application runs from a single binary.

## Phase 4: MCP (Model Context Protocol) Integration

This phase enables other AIs to programmatically interact with and curate the learning plan.

*   **Task 4.1: Setup MCP Server.** Integrate an MCP server into the Go application.
*   **Task 4.2: Expose MCP Tools.** Implement tools/functions that other AIs can call to manage the curriculum:
    *   `get_plan` (returns the full hierarchy of segments and quizzes)
    *   `add_segment` & `update_segment`
    *   `add_quiz` & `update_quiz`
    *   `add_question` & `update_question`
*   **Task 4.3: Secure & Test MCP.** Ensure the MCP integration correctly utilizes the repository layer to persist data and that the tools function reliably.
