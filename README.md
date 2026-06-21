# 🔮 KATA-CLI

`kata-cli` is a minimalist, keyboard-driven terminal dashboard built for developers to track coding practice (katas) using spaced repetition. It is styled using standard terminal theme ANSI colors (`1`–`8`) so it automatically matches your terminal config.

---

## 🚀 Setup & Installation

### 1. Clone & Navigate
Clone the upstream repository into your local practice folder (e.g., `katas`):
```bash
git clone https://github.com/iamharshdabas/kata-cli.git katas
cd katas
```

### 2. Install Dependencies
```bash
go mod tidy
```

### 3. Build & Run
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

## 💾 Git Remote Setup (Upstream / Standalone Repository)

When you first clone this repository, the remote named `origin` points to the upstream creator's repository.

> [!TIP]
> **Get Green Squares (GitHub Contributions):** Commits to a *forked* repository on GitHub do not count towards your contribution activity graph (green squares) unless they are merged into the upstream repository's default branch. Since you want to track your daily practice, it is highly recommended to create a **standalone repository** instead of a fork.

### Method 1: Standalone Repository using GitHub CLI (Recommended)

If you have the [GitHub CLI (`gh`)](https://cli.github.com/) installed, you can create a standalone repository in your account and link it:

1. **Authenticate and set up Git credentials**:
   ```bash
   gh auth login
   gh auth setup-git
   ```

2. **Create a new standalone repository on GitHub**:
   ```bash
   # Create a private or public repository (e.g. named 'katas')
   gh repo create katas --private
   ```

3. **Configure your remotes & push**:
   ```bash
   # Rename the creator's remote to 'upstream'
   git remote rename origin upstream

   # Add your new standalone repository as 'origin'
   git remote add origin https://github.com/YOUR_USERNAME/katas.git

   # Push your history and set the tracking branch
   git push -u origin main
   ```

---

### Method 2: Automated Forking via GitHub CLI (Alternative)

If you prefer to fork the repository directly and do not mind that your daily practice commits won't count as contributions on your profile:

```bash
# Fork the repository and configure remotes automatically
gh repo fork iamharshdabas/kata-cli --clone=false
```
*This command automatically forks the repository to your account, renames the creator's remote to `upstream`, and configures your personal fork as `origin`.*

---

### Method 3: Standalone Repository via GitHub Website (If not using GitHub CLI)

If you do not have the GitHub CLI installed, you can manually create a repository on the web:

1. **Create a new repository**: Go to [github.com/new](https://github.com/new) and create a new repository (e.g., `katas`). Do **not** initialize it with a README, license, or .gitignore.
2. **Configure your local remotes**:
   ```bash
   # Rename the creator's remote to 'upstream'
   git remote rename origin upstream

   # Add your personal repository as 'origin' (using HTTPS URL)
   git remote add origin https://github.com/YOUR_USERNAME/YOUR_KATA_REPO.git

   # Push your history to your personal repository
   git push -u origin main
   ```

_Note: If you skip this setup, the CLI will run local commits only and will not push updates online._

### 3. Pulling Creator Updates
To pull new features and fixes from the creator without losing your personal spacing database (`db.json`):

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
