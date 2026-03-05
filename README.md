# Telegram Quiz Bot

A scalable Go application that provides a daily Spanish quiz to Telegram users for gamification. It uses the `qwen3.5:4b` local LLM via Ollama to dynamically generate the quiz questions.

## Architecture

*   **SQLite DB**: Stores users, quizzes, and answers to calculate leaderboards and prevent double-voting.
*   **Cron Scheduler**: Periodically triggers the generation of a new quiz.
*   **Learning Plan Manager**: Tracks the current topic and progress in the instructional sequence (defined in `LEARNINGPLAN.md`).
*   **Ollama Client**: Creates robust JSON requests to Ollama with deduplication logic.
*   **Telebot**: robust Telegram API framework for message handling and inline keyboards.

## Features

### Adaptive Seeding & Personalized Pools
The bot maintains a personalized pool of questions for each user. It proactively seeds more questions from the current learning plan topic when a user's pool runs low (0 or 1 questions left). This ensures a continuous learning experience without repeats.

### Learning Plan Alignment
Questions are automatically generated based on the active step in the learning plan. After a certain number of questions are generated for a topic, the bot automatically advances to the next step.

## Prerequisites

1.  **Go 1.20+** via your local toolchain.
2.  **Ollama**: Installed and running locally.
3.  **Telegram Bot Token**: Obtain one from [@BotFather](https://t.me/BotFather) on Telegram.

## Setup Instructions

1.  **Pull the Model**
    Ensure you have the required model pulled in Ollama.
    ```bash
    ollama run qwen3.5:4b
    ```

2.  **Install Dependencies**
    ```bash
    go mod tidy
    ```

3.  **Run the Bot**
    Export your token and run the application.

    ```bash
    export TELEGRAM_BOT_TOKEN="your_token_here"
    export OLLAMA_URL="http://localhost:11434" # Optional, defaults to this
    export OLLAMA_MODEL="qwen3.5:4b"           # Optional, defaults to this
    
    go run cmd/bot/main.go
    ```

## Usage

*   Send `/start` to the bot to register and get a welcome message.
*   Send `/quiz` to get your next unanswered quiz from the current topic in the learning plan.
*   Tap an inline keyboard option to answer. The bot will tell you if you're right. You can only answer once per question!
*   Send `/plan` to see your current topic and progress.
*   Send `/nextstep` to manually advance the learning plan.
*   Send `/leaderboard` to view the top scorers.

## Extensibility

This bot natively stores an `audio_file_id` column in the `quizzes` table in preparation for sending voice notes (TTS) in the future.
