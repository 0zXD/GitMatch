import type { GitHubIssue } from "../components/issue-card";

const HARVESTER_URL = "http://localhost:8082";

interface RepoResult {
  name: string;
  stars: number;
  forks: number;
  url: string;
  description: string;
  primary_language: string;
  open_issues: number;
  language_breakdown?: Record<string, number>;
  valid_tags?: string[];
}

// GitHub-recognized programming languages (lowercase)
const GITHUB_LANGUAGES = new Set([
  "javascript", "typescript", "python", "java", "go", "rust",
  "c++", "c#", "c", "ruby", "php", "lua", "shell", "nix", "tex",
  "jupyter notebook", "html", "css", "haskell", "scala", "kotlin",
  "swift", "objective-c", "dart", "r", "matlab", "perl",
  "elixir", "erlang", "clojure", "f#", "ocaml", "zig",
  "groovy", "powershell",
]);

// Special skill-to-query mappings for non-standard names
const SKILL_QUERY_MAP: Record<string, string> = {
  "html/css": "language:HTML language:CSS",
  ".net": "topic:dotnet",
  "rest api": "topic:rest-api",
  "node.js": "topic:nodejs",
};

function buildSearchQuery(skills: string[]): string {
  const parts: string[] = [];

  for (const skill of skills) {
    const lower = skill.toLowerCase();

    if (SKILL_QUERY_MAP[lower]) {
      parts.push(SKILL_QUERY_MAP[lower]);
    } else if (GITHUB_LANGUAGES.has(lower)) {
      parts.push(`language:"${skill}"`);
    } else {
      // Frameworks, tools, etc. → treat as topic
      parts.push(`topic:${lower.replace(/[.\s]+/g, "-")}`);
    }
  }

  return parts.join(" ");
}

function hashString(str: string): number {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    hash = ((hash << 5) - hash) + str.charCodeAt(i);
    hash |= 0;
  }
  return Math.abs(hash);
}

function getDifficulty(repo: RepoResult): "beginner" | "intermediate" | "advanced" {
  if (repo.stars < 1000) return "beginner";
  if (repo.stars < 5000) return "intermediate";
  return "advanced";
}

function generateLabels(repo: RepoResult): string[] {
  const labels: string[] = [];
  const difficulty = getDifficulty(repo);

  if (difficulty === "beginner" || repo.open_issues > 5) {
    labels.push("good first issue");
  }
  if (repo.open_issues > 20) {
    labels.push("help wanted");
  }

  // Add a category label based on primary language
  const lang = (repo.primary_language || "").toLowerCase();
  const frontendLangs = ["javascript", "typescript", "html", "css", "dart"];
  const backendLangs = ["python", "java", "go", "ruby", "php", "c#", "elixir", "erlang"];
  const systemsLangs = ["rust", "c", "c++", "zig"];

  if (frontendLangs.includes(lang)) labels.push("frontend");
  else if (backendLangs.includes(lang)) labels.push("backend");
  else if (systemsLangs.includes(lang)) labels.push("systems");

  if (repo.stars > 1000) labels.push("popular");
  if (labels.length === 0) labels.push("open source");

  return labels.slice(0, 3);
}

function buildDescription(repo: RepoResult): string {
  const lang = repo.primary_language || "Open Source";
  const parts: string[] = [];

  if (repo.description) {
    parts.push(repo.description);
  }

  // Mention language combo if repo uses multiple languages
  if (repo.valid_tags && repo.valid_tags.length > 1) {
    parts.push(
      `Uses ${repo.valid_tags.join(", ")} — ${repo.open_issues} open issues ready for contributions.`
    );
  } else {
    parts.push(
      `${lang} project with ${repo.open_issues} open issues ready for contributions.`
    );
  }

  return parts.join(" ");
}

function repoToIssue(repo: RepoResult): GitHubIssue {
  const lang = repo.primary_language || "Open Source";
  const languageTags = repo.valid_tags?.length
    ? repo.valid_tags
    : lang !== "Open Source"
      ? [lang]
      : [];

  return {
    id: hashString(repo.url || repo.name),
    title: repo.description
      ? repo.description.length > 80
        ? repo.description.slice(0, 77) + "..."
        : repo.description
      : `Contribute to ${repo.name}`,
    repository: repo.name,
    description: buildDescription(repo),
    labels: generateLabels(repo),
    language: lang,
    stars: repo.stars,
    forks: repo.forks || 0,
    comments: 0,
    difficulty: getDifficulty(repo),
    url: repo.url,
    openIssues: repo.open_issues,
    languageTags,
  };
}

export interface HarvestResult {
  issues: GitHubIssue[];
  hasMore: boolean;
  page: number;
  endCursor?: string;
}

interface IssueResultApi {
  id: number;
  title: string;
  url: string;
  number: number;
  state: string;
  body: string;
  comments: number;
  labels: string[];
  created_at: string;
  name: string;
  repo_url: string;
  stars: number;
  description: string;
  primary_language: string;
  language_breakdown?: Record<string, number>;
  valid_tags?: string[];
}

interface HarvestResponse {
  results: IssueResultApi[];
  has_more: boolean;
  page: number;
  end_cursor?: string;
}

export async function fetchIssuesForSkills(
  skills: string[],
  experience: "beginner" | "intermediate" | "advanced" = "beginner",
  repoCount: number = 0,
  signal?: AbortSignal,
  page: number = 1,
  cursor?: string,
): Promise<HarvestResult> {
  if (skills.length === 0) return { issues: [], hasMore: false, page: 1 };

  const query = buildSearchQuery(skills);
  const params = new URLSearchParams({ 
    q: query, 
    page: String(page),
    experience: experience,
    repoCount: String(repoCount)
  });
  if (cursor) {
    params.append('after', cursor);
  }
  const response = await fetch(`${HARVESTER_URL}/issues?${params}`, { signal });

  if (!response.ok) {
    throw new Error(`Harvester API error: ${response.status}`);
  }

  const data: HarvestResponse = await response.json();
  const rawIssues = data.results ?? [];
  const issues: GitHubIssue[] = rawIssues.map((apiIssue) => {
    const lang = apiIssue.primary_language || "Open Source";
    const languageTags = apiIssue.valid_tags?.length
      ? apiIssue.valid_tags
      : lang !== "Open Source"
        ? [lang]
        : [];

    const apiLabels = apiIssue.labels || [];
    let difficulty: "beginner" | "intermediate" | "advanced" = "intermediate";
    const titleLower = apiIssue.title ? apiIssue.title.toLowerCase() : "";
    const isBeginner = apiLabels.some((l) =>
      l.toLowerCase().includes("good first") || l.toLowerCase().includes("beginner") || l.toLowerCase().includes("easy")
    );
    if (isBeginner || titleLower.includes("good first") || titleLower.includes("easy")) {
      difficulty = "beginner";
    }

    return {
      id: apiIssue.id || Math.random(),
      title: apiIssue.title || "Untitled Issue",
      repository: apiIssue.name || "Unknown Repo",
      description: apiIssue.body ? apiIssue.body.substring(0, 150) + "..." : (apiIssue.description || "No description provided."),
      labels: apiLabels.slice(0, 4),
      language: lang,
      stars: apiIssue.stars || 0,
      forks: 0,
      comments: apiIssue.comments || 0,
      difficulty,
      url: apiIssue.url || "#",
      openIssues: 1, // This represents 1 specific issue now
      languageTags,
    };
  });

  return {
    issues,
    hasMore: data.has_more ?? false,
    page: data.page ?? page,
    endCursor: data.end_cursor,
  };
}
