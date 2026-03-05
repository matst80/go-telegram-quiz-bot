# Telegram AI Quiz Bot

This repository contains the source code for a Telegram bot designed to create a gamified Spanish learning experience. It leverages a local LLM (Ollama running `qwen3.5:4b`) to dynamically generate personalized quizzes based on a structured learning plan.

## Key Components

*   **Telegram Bot Interface:** Handles user interactions, commands (`/start`, `/quiz`, `/plan`, `/leaderboard`, etc.), and inline keyboards for answering quizzes. It's built using the `telebot.v3` framework.
*   **Local LLM Integration:** Connects to a local Ollama instance to generate quiz questions dynamically. It includes prompt engineering to ensure the LLM returns well-structured JSON questions with four options and a correct answer. It also tries to avoid generating duplicate questions.
*   **Learning Plan Manager:** Reads topics from `LEARNINGPLAN.md` and sequentially guides the user through them. After a configurable number of quizzes (default 5) are generated for a topic, it automatically advances to the next one.
*   **Database (SQLite):** Stores user profiles, generated quizzes, user answers, scores (for leaderboards), and learning plan progress. It ensures users don't answer the same question twice and tracks their progress.
*   **Scheduler & Seeding System:** A cron job periodically generates new quizzes. More importantly, an adaptive seeding system monitors each user's pool of unanswered questions for the current topic. If the pool drops below a threshold (2 questions), it proactively triggers the LLM to generate more, ensuring a continuous flow.

## Gamification Features

*   **Leaderboard:** Users earn points for correct answers, and a `/leaderboard` command displays the top scorers, adding a competitive element.
*   **Progress Tracking:** The bot guides users through a structured curriculum, giving a sense of progression.
*   **Immediate Feedback:** Users receive instant "Correct" or "Incorrect" messages upon answering.

## Workflow

1.  A user starts the bot and requests a quiz.
2.  The bot checks the current topic in the learning plan.
3.  It retrieves an unanswered question for that user and topic from the database.
4.  If the user's pool of questions is running low, it asynchronously asks the LLM to generate more.
5.  The user answers the question via an inline keyboard.
6.  The bot records the answer, updates the score if correct, and automatically sends the next question.
