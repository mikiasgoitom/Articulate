# G6 Blog Project: Collaboration & Workflow Guidelines

This document outlines the development workflow, branching strategy, and conventions our team will follow to ensure a smooth, consistent, and high-quality development process.

## 1. Initial Setup: Forking the Repository

To get started, you need to create your own copy of the main project repository.

### Step 1: Fork the Repository

1. Navigate to the main project repository: [https://github.com/mikiasgoitom/Articulate](https://github.com/mikiasgoitom/Articulate)
2. Click the **"Fork"** button in the top-right corner. This will create a copy of the repository under your own GitHub account.

### Step 2: Clone Your Fork

Clone the forked repository (your copy) to your local machine. Replace `<Your-GitHub-Username>` with your actual username.

```bash
git clone https://github.com/<Your-GitHub-Username>/Articulate.git
cd Articulate
```

### Step 3: Add the `upstream` Remote

Add the original project repository as a remote named `upstream`. This allows you to pull in changes from the main project to keep your fork updated.

```bash
git remote add upstream https://github.com/mikiasgoitom/Articulate.git
```

### Step 4: Verify Remotes

Check that you have both `origin` (your fork) and `upstream` (the original repo) configured correctly.

```bash
git remote -v
```

You should see output similar to this:

```bash
origin    https://github.com/<Your-GitHub-Username>/Articulate.git (fetch)
origin    https://github.com/<Your-GitHub-Username>/Articulate.git (push)
upstream  https://github.com/mikiasgoitom/Articulate.git (fetch)
upstream  https://github.com/mikiasgoitom/Articulate.git (push)
```

### Step 5: Keep Your Fork Synced

Before starting any new work, always sync your `develop` branch with the `upstream` repository.

```bash
git fetch upstream
git checkout develop
git merge upstream/develop
```

## 2. Core Branching Strategy

Our workflow is based on two primary, long-lived branches:

- **`main`**: This branch represents the production-ready, stable codebase. No one should ever commit directly to `main`.
- **`develop`**: This is our main development branch. It contains the latest delivered development changes and is the base for all new work. All feature branches are merged into `develop`.

## 3. Development Workflow: Adding a New Feature

Follow these steps every time you start working on a new feature, bugfix, or task.

### Step 1: Sync with `develop`

Before starting any new work, make sure your local `develop` branch is up-to-date with the `upstream` repository.

```bash
git checkout develop
git pull upstream develop
```

### Step 2: Create Your Branch

Create a new branch from `develop`. Your branch name **must** follow our naming convention.

**Branch Naming Convention:** `<username>-<type>/<short-description>`

- **`<username>`**: Your GitHub username (e.g., `Tesfamichael12`).
- **`<type>`**: The type of work (`feature`, `bugfix`, `refactor`, `docs`).
- **`<short-description>`**: A few words describing the task, using hyphens (`-`).

**Example:**

```bash
# For a new feature
git checkout -b Tesfamichael12-feature/user-authentication

# For a bug fix
git checkout -b Tesfamichael12-bugfix/fix-pagination-error
```

### Step 3: Code and Commit

Work on your task in the new branch. Make small, frequent, and logical commits. Each commit message **must** follow the **Conventional Commits** standard.

**Commit Message Convention:** `<type>(<scope>): <description>`

- **`type`**: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`.
- **`scope` (optional)**: The part of the codebase affected (e.g., `auth`, `blog`, `db`).
- **`description`**: A short, present-tense summary of the change.

**Example Commit:**

```bash
git add .
git commit -m "feat(auth): implement user registration endpoint"
```

### Step 4: Push Your Branch

Once your work is ready for review (or you want to share it), push your branch to the remote repository (`origin`).

```bash
git push -u origin Tesfamichael12-feature/user-authentication
```

### Step 5: Create a Pull Request (PR)

Go to the GitHub repository and create a new Pull Request (PR) from your branch **to the `develop` branch** of the `upstream` (original) repository.

## 4. Pull Request (PR) Guidelines

Your PR is your request to merge your code into the `develop` branch. Make it easy for others to review.

- **Title**: The PR title should follow the Conventional Commits format (e.g., `feat(auth): Add JWT generation and validation`).
- **Description**:
  - Clearly explain **what** problem the PR solves and **why** you made these changes.
  - Link to any relevant issues or tasks (e.g., `Closes #42`).
  - Describe how you tested your changes.
- **Keep it Focused**: A PR should address one single task or feature. Avoid large, multi-purpose PRs.
- **Code Review**:
  - At least **one** team member must review and approve your PR.
  - Address all comments and feedback from your reviewers. Push new commits to your branch to update the PR.
- **Merging Strategy**: Once approved, the PR will be merged into `develop`. The choice of merge method depends on the situation.
  - **Squash and Merge (Default for Feature Branches)**:
    - **What it does**: Combines all of your PR's commits into a single commit on the `develop` branch.
    - **When to use**: This is our default method for merging feature and bugfix branches. It keeps the `develop` branch history clean and easy to read, as each PR is represented by a single, meaningful commit.

  - **Rebase and Merge**:
    - **What it does**: Re-writes your PR's commits on top of the `develop` branch, creating a perfectly linear history.
    - **When to use**: Use this for small, simple PRs where preserving the individual commit history is valuable. It requires you to first rebase your branch against the latest `develop` before merging. This can be more complex and is best reserved for those comfortable with `git rebase`.

- **Branch Cleanup**: Your branch will be deleted automatically after the merge.

## 5. Code Style & Linting

To maintain a consistent and readable codebase, all Go code must be formatted and linted before committing.

### Manual Formatting & Linting

**1. Formatting with `gofmt` (Built-in)**

`gofmt` is Go's official code formatter. It's included with Go, so no installation is needed.

- **How to run**: Before creating a pull request, run this command from the project root to format all your code automatically.
  ```bash
  gofmt -w .
  ```

**2. Linting with `golangci-lint`**

`golangci-lint` is a powerful tool that checks your code for common bugs, style issues, and mistakes.

- **Installation**: Each team member needs to install it once.
  ```bash
  # Installs the linter to your Go bin directory
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.57.2
  ```
- **How to run**: Run this command from the project root. You must fix all reported issues before your code can be merged.
  ```bash
  golangci-lint run
  ```

### Automated Checks with Pre-commit (Recommended)

To make this process effortless, we use `pre-commit` hooks to run these checks automatically every time you make a commit.

- **Step 1: Install `pre-commit`**
  You'll need Python and pip installed.
  ```bash
  pip install pre-commit
  ```
- **Step 2: Set up the Git Hooks**
  Run this command once from the project root. It will set up the hooks based on the `.pre-commit-config.yaml` file in our repository.
  ```bash
  pre-commit install
  ```

From now on, `gofmt` and `golangci-lint` will run automatically on the files you've changed before every commit. If they find any issues, the commit will be stopped, allowing you to fix them first.

## 6. Testing

Quality is a shared responsibility. All new features and bug fixes should be accompanied by tests.

- **Unit Tests**: For any new logic in use cases or services, please provide unit tests.
- **Integration Tests**: For changes affecting database interactions or API endpoints, consider adding integration tests.
- **Run Tests**: Always run the full test suite locally (`go test ./...`) and ensure all tests pass before pushing your changes.

## 7. Communication & Issue Tracking

Clear communication is key to effective collaboration.

- **GitHub Issues**: All tasks, features, and bugs should be tracked as GitHub Issues. Before starting work, assign yourself to the relevant issue.
- **Pull Requests**: For code-specific discussions, use comments within the Pull Request.
- **General Discussion**: For broader questions or team-wide discussions, we will use our designated chat channel (e.g., Slack/Discord).
