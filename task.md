# Task: Create a Telegram AI Quiz Bot

The goal is to develop a Telegram bot that provides a gamified Spanish learning experience using an AI (LLM) to generate content dynamically.

Based on the existing codebase, here are the main tasks required to build this system from scratch or understand its implementation:

## 1. Setup and Infrastructure

*   **Telegram Bot Setup:** Create a bot via BotFather and obtain an API token.
*   **Database Design:** Set up a SQLite database with tables for:
    *   `users`: Telegram ID, username, score.
    *   `quizzes`: Topic, question, options (JSON), correct answer.
    *   `user_answers`: Tracks which user answered which quiz and if they were correct (to prevent double voting).
    *   `settings`: To store global state like the current learning step.
    *   `topic_lessons` & `user_lessons`: To store and track mini-lessons before quizzes.
*   **LLM Integration:** Set up a local Ollama instance with an appropriate model (e.g., `qwen3.5:4b`). Implement a client in Go to send prompts and parse JSON responses.

## 2. Core Bot Logic

*   **Command Handling:** Implement handlers for basic commands (`/start`, `/leaderboard`, `/plan`, `/nextstep`).
*   **Quiz Flow:**
    *   Implement the `/quiz` command to fetch the next unanswered question for the user's current topic.
    *   Present the question using Telegram inline keyboards.
    *   Handle callback queries when a user clicks an option.
    *   Verify the answer, update the database (score and `user_answers`), and provide immediate feedback.
    *   Automatically serve the next question after a brief delay.

## 3. Learning Plan and Content Generation

*   **Learning Plan Management:** Implement a system to read topics sequentially from a file (e.g., `LEARNINGPLAN.md`). Track progress and automatically advance topics after a set number of generated quizzes.
*   **Dynamic Generation:** Write robust prompts for the LLM to generate high-quality, topic-specific multiple-choice questions in a strict JSON format. Implement logic to retry on malformed responses and exclude recently generated questions to ensure variety.

## 4. Automation and User Experience

*   **Adaptive Seeding:** Implement a background process that monitors a user's pool of unanswered questions. If they are running low on questions for the current topic, trigger the LLM to generate more in the background to ensure they don't have to wait.
*   **Cron Scheduler (Optional but recommended):** Add a scheduler to periodically generate batches of questions to keep the global pool fresh.
*   **Pre-Quiz Lessons:** Implement a feature to show a brief lesson on a topic before the user takes their first quiz on that topic.

## 5. Gamification

*   **Scoring System:** Award points for correct answers.
*   **Leaderboard:** Create a command to display the top users based on their scores to encourage competition.