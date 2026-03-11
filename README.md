# GitMatch

GitMatch is a full-stack web application that helps developers discover open source repositories matching their skills and interests. Users authenticate with their GitHub username, and the application aggregates their profile data -- languages, topics, followers, and repository activity -- to present a curated feed of repositories with open issues ready for contribution. The interface draws inspiration from swipe-based discovery patterns: users browse repository cards, mark those they find interesting, and build a personal list of saved issues for later contribution.

---

## Architecture Overview

The project is composed of three independently running services:

**Frontend** -- A single-page React application built with Vite. It handles user interaction, skill selection, issue browsing, pagination, and saved-issue management. All state is maintained client-side with React hooks; user session data is persisted in `localStorage`.

**Harvester API** (Go, port 8082) -- Accepts skill-based search queries from the frontend, translates them into GitHub Search API calls, enriches results with per-repository language breakdowns, and returns structured JSON. Results are cached in MongoDB with a 24-hour TTL to reduce API rate-limit pressure.

**User Info Harvester API** (Go, port 8084) -- Accepts a GitHub username and returns an aggregated profile: bio, location, follower/following counts, per-language repository tallies, and topic frequencies. This data powers the profile overview and the "Your Languages" section of the skill selector.

### Data Flow

1. The user enters their GitHub username on the login screen. The frontend sends `GET /user?username=<name>` to the User Info Harvester.
2. The returned profile is stored in `localStorage` and rendered in the profile overview panel.
3. When the user selects one or more skills, the frontend service layer (`src/app/services/harvester.ts`) constructs a GitHub-compatible search query (combining `language:` and `topic:` qualifiers) and sends `GET /harvest?q=<query>&page=<n>` to the Harvester API.
4. The Harvester API checks MongoDB for a cached response. On a cache miss, it queries the GitHub Search API, fetches language breakdowns for each repository, filters out repositories with zero open issues, caches the result, and responds with JSON.
5. The frontend transforms each repository result into a display card with computed difficulty, labels, and description, then renders them in a paginated grid.

---

## Project Structure

```
gitmatch/
  index.html                    Application entry point (Vite)
  package.json                  Frontend dependencies and scripts
  vite.config.ts                Vite configuration with React and Tailwind CSS plugins
  postcss.config.mjs            PostCSS configuration (empty; Tailwind v4 handles plugins)
  backend/
    harvester/
      main.go                   Harvester API server (repository search and caching)
      go.mod                    Go module definition and dependencies
    userinfoharvester/
      main.go                   User Info Harvester API server (profile aggregation)
      go.mod                    Go module definition and dependencies
  src/
    main.tsx                    React DOM entry point
    app/
      App.tsx                   Root component; routing, state management, layout
      services/
        harvester.ts            Frontend client for the Harvester API
      types/
        user-profile.ts         TypeScript interface for user profile data
      components/
        github-login.tsx        Login form; calls User Info Harvester API
        issue-card.tsx          Repository/issue card with action buttons
        saved-issues.tsx        List view for bookmarked repositories
        skill-selector.tsx      Skill/language filter panel
        user-profile-overview.tsx  GitHub-style profile display with language bar
        ui/                     Reusable UI primitives (shadcn/ui components)
    styles/
      index.css                 Style entry point (imports fonts, Tailwind, theme)
      fonts.css                 Font-face declarations
      tailwind.css              Tailwind CSS v4 base import
      theme.css                 Custom CSS variables and theme tokens
  guidelines/
    Guidelines.md               AI and design system guidelines
  ATTRIBUTIONS.md               Third-party license attributions
```

---

## Prerequisites

- **Node.js** (v18 or later) and a package manager (`npm`, `pnpm`, or `yarn`)
- **Go** (1.21 or later)
- **MongoDB** (optional; the Harvester API operates without caching if MongoDB is unavailable)
- **GitHub Personal Access Token** (optional but strongly recommended to avoid rate limits)

---

## Environment Variables

Create a `.env` file in the `backend/harvester/` directory (or in the project root; the harvester falls back to `../.env`). The User Info Harvester reads `GITHUB_TOKEN` from the process environment directly.

```
GITHUB_TOKEN=ghp_your_personal_access_token
MONGODB_URI=mongodb://localhost:27017
MONGODB_DB=gitmatch
```

- `GITHUB_TOKEN` -- Used by both backend services to authenticate with the GitHub API. Without it, requests are subject to the unauthenticated rate limit of 60 requests per hour.
- `MONGODB_URI` -- Connection string for MongoDB. If unset, the Harvester API logs a message and proceeds without caching.
- `MONGODB_DB` -- Database name for the cache collection. Defaults to `gitmatch` if unset.

---

## Getting Started

### 1. Install frontend dependencies

From the project root:

```bash
npm install
```

### 2. Start the Harvester API

```bash
cd backend/harvester
go run main.go
```

The server starts on port **8082** and exposes a single endpoint: `GET /harvest`.

### 3. Start the User Info Harvester API

In a separate terminal:

```bash
cd backend/userinfoharvester
go run main.go
```

The server starts on port **8084** and exposes a single endpoint: `GET /user`.

### 4. Start the frontend development server

From the project root:

```bash
npm run dev
```

Vite will start a development server (typically on `http://localhost:5173`). Open the URL in a browser.

### 5. Build for production

```bash
npm run build
```

The output is written to `dist/`.

---

## Backend API Reference

### Harvester API -- `GET /harvest`

Searches GitHub for repositories matching a skill-based query.

**Query Parameters:**

- `q` (string, required unless `topic` is provided) -- A GitHub search query string composed of `language:` and `topic:` qualifiers. Example: `language:"Python" topic:django`.
- `topic` (string, optional) -- Shorthand for a single topic search. Used when `q` is absent.
- `page` (integer, optional, default 1) -- The page number for paginated results.
- `lite` (string, optional) -- When set to `"true"`, skips per-repository language breakdown fetching for faster responses.

**Response:**

```json
{
  "results": [
    {
      "name": "owner/repo",
      "stars": 1234,
      "forks": 56,
      "url": "https://github.com/owner/repo",
      "description": "A brief description",
      "primary_language": "Python",
      "open_issues": 42,
      "language_breakdown": { "Python": 78.5, "JavaScript": 21.5 },
      "valid_tags": ["Python", "JavaScript"]
    }
  ],
  "has_more": true,
  "page": 1
}
```

The `valid_tags` array contains languages making up 10% or more of the repository's code. The `X-Cache` response header indicates `HIT` or `MISS`.

### User Info Harvester API -- `GET /user`

Retrieves an aggregated GitHub user profile.

**Query Parameters:**

- `username` (string, required) -- The GitHub username to look up.

**Response:**

```json
{
  "name": "Jane Doe",
  "username": "janedoe",
  "bio": "Open source enthusiast",
  "location": "San Francisco, CA",
  "company": "@example",
  "twitter": "janedoe",
  "blog": "https://janedoe.dev",
  "public_repos": 42,
  "followers": 150,
  "following": 30,
  "created_at": "2015-03-21",
  "languages": { "Python": 12, "TypeScript": 8, "Go": 5 },
  "topics": { "machine-learning": 4, "web": 3 }
}
```

The `languages` and `topics` maps contain counts of how many of the user's repositories use each language or topic.

---

## Frontend Service Layer

The file `src/app/services/harvester.ts` serves as the frontend client for the Harvester API. It performs the following:

- **Query construction** -- Translates an array of user-selected skill strings into a GitHub-compatible query. Programming languages map to `language:"X"` qualifiers; frameworks and tools map to `topic:X` qualifiers. Special-case mappings handle names like "HTML/CSS", ".NET", "REST API", and "Node.js".
- **Result transformation** -- Converts raw `RepoResult` objects from the API into `GitHubIssue` display objects consumed by the UI components. This includes computing difficulty tiers (beginner, intermediate, advanced) based on star count, generating contextual labels (good first issue, help wanted, frontend/backend/systems), and building human-readable descriptions.
- **Pagination** -- Exposes page-based fetching with abort signal support for request cancellation on rapid filter changes.

The User Info Harvester is called directly from the `GitHubLogin` component without a separate service module.

---

## Key Frontend Components

- **App** -- Root component managing global state: user session, selected skills, fetched issues, pagination, dark mode, and tab navigation between Discover and Saved views.
- **GitHubLogin** -- Login form with username validation. Calls the User Info Harvester API and passes the resulting profile upstream.
- **SkillSelector** -- Displays the user's own languages (derived from their profile) and a broader set of predefined skills. Supports multi-select toggling.
- **IssueCard** -- Renders a single repository as a card showing title, description, difficulty badge, labels, language tags, star/fork/issue counts, and action buttons (Interested / Skip / View on GitHub).
- **SavedIssues** -- Lists all bookmarked repositories with external links and a remove button.
- **UserProfileOverview** -- Displays the authenticated user's GitHub profile: avatar, bio, metadata, a language distribution bar chart, and top topics.

---

## Caching

The Harvester API uses MongoDB as an optional response cache. Each unique query-plus-page combination is stored as a document keyed by a normalized, sorted representation of the query string. A TTL index on the `cached_at` field automatically removes entries after 24 hours. If MongoDB is unreachable or unconfigured, the API continues to function by querying the GitHub API directly on every request.

---

## Technology Stack

**Frontend:**
- React 18 with TypeScript
- Vite 6 (build tooling and dev server)
- Tailwind CSS v4 (utility-first styling via @tailwindcss/vite)
- Radix UI primitives (shadcn/ui component library)
- Lucide React (icon set)
- Motion (animations)

**Backend:**
- Go 1.25
- go-github v50 (GitHub API client)
- mongo-driver v2 (MongoDB driver, Harvester only)
- oauth2 (GitHub token authentication)
- godotenv (environment variable loading, Harvester only)

**Infrastructure:**
- MongoDB (optional response cache with TTL-based expiration)
