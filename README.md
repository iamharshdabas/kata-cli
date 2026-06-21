# 🔮 KATA-CLI

`kata-cli` is a minimalist, keyboard-driven terminal dashboard built for developers to track coding practice (katas) using spaced repetition. It is styled using standard terminal theme ANSI colors (`1`–`8`) so it automatically matches your terminal config.

---

## 🚀 Setup & Installation

### 1. Install Dependencies

```bash
go mod tidy
```

### 2. Build & Run

```bash
# Run directly
go run ./cmd/kata-cli

# Compile and run
go build -o kata-cli ./cmd/kata-cli
./kata-cli
```

You can also use `just` if you have it installed:

```bash
# Compile and run via justfile
just build
./kata-cli
```

---

## 💾 Git Remote Setup (Upstream/Fork)

When you first clone this repository, the remote named `origin` points to the upstream creator's repository. To sync your progress to your own fork while keeping the ability to pull updates:

### 1. Configure Remotes (Fork Setup)

```bash
# Rename creator remote to 'upstream'
git remote rename origin upstream

# Add your personal repository as 'origin'
git remote add origin git@github.com:YOUR_USERNAME/YOUR_KATA_REPO.git

# Push your history to your personal fork
git push -u origin main
```

_Note: If you skip this, the CLI will run local commits only, avoiding pushes to the upstream repository._

### 2. Pulling Creator Updates

To pull new features and fixes without losing your personal spacing database (`db.json`):

```bash
# Fetch updates from the creator
git fetch upstream

# Merge updates into your main branch
git merge upstream/main
```

If any code conflicts arise, resolve them, add them, and commit. Your personal logs in `db.json` will be preserved!

---

## 💡 What it is & Core Features

- **Spaced Repetition Practice**: Tracks your daily coding katas using the SM-2 scheduling algorithm. Calculates next review intervals and brain ease factors automatically based on your rating.
- **LeetCode API Fetching & Caching**: Automatically queries LeetCode problem titles and difficulty tiers based on the problem number, with a local 7-day cache.
- **Automatic Git Syncing**: Stashes and commits your practice logs (`db.json`) automatically to Git after changes, keeping your progress synced upstream.
- **Minimalist Keyboard UI**: Highly responsive terminal interface built with Charmbracelet Bubble Tea, featuring tab cycling, live filtering/searching, and direct links to open problems in your default browser.
