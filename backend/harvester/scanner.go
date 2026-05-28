package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/google/go-github/v50/github"
)

// getFileTree fetches the repository tree using the GitHub API
func getFileTree(ctx context.Context, client *github.Client, owner, repo string) (*github.Tree, error) {
	// First, let's get the default branch so we have a reliable tree SHA
	repository, _, err := client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository info: %w", err)
	}

	branch := repository.GetDefaultBranch()
	if branch == "" {
		branch = "main" // fallback
	}

	// We can use the branch name as the SHA/ref to fetch the recursive tree
	log.Printf("Fetching file tree for %s/%s (branch: %s)", owner, repo, branch)
	tree, _, err := client.Git.GetTree(ctx, owner, repo, branch, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get git tree: %w", err)
	}

	if tree.Truncated != nil && *tree.Truncated {
		log.Printf("Warning: The Git tree for %s/%s is truncated due to size.", owner, repo)
	}

	return tree, nil
}

// extractDependencies scans the tree for manifest files and extracts dependencies
func extractDependencies(ctx context.Context, client *github.Client, owner, repo string, tree *github.Tree) ([]string, error) {
	var allDeps []string
	var foundPackageJSON, foundRequirements bool

	for _, entry := range tree.Entries {
		path := entry.GetPath()

		// Look for top-level dependency files
		if path == "package.json" && !foundPackageJSON {
			foundPackageJSON = true
			log.Printf("Found package.json in %s/%s", owner, repo)
			content, err := getFileContent(ctx, client, owner, repo, path)
			if err == nil {
				allDeps = append(allDeps, parsePackageJSON(content)...)
			}
		} else if path == "requirements.txt" && !foundRequirements {
			foundRequirements = true
			log.Printf("Found requirements.txt in %s/%s", owner, repo)
			content, err := getFileContent(ctx, client, owner, repo, path)
			if err == nil {
				allDeps = append(allDeps, parseRequirementsTxt(content)...)
			}
		}
	}

	// Also detect stacks based on file/folder names
	allDeps = append(allDeps, detectTechStack(tree)...)

	return uniqueStrings(allDeps), nil
}

func detectTechStack(tree *github.Tree) []string {
	var tech []string
	techSet := make(map[string]bool)

	for _, entry := range tree.Entries {
		path := strings.ToLower(entry.GetPath())

		// Infrastructure, Containerization & CI/CD
		if strings.Contains(path, "docker-compose") {
			techSet["Docker Compose"] = true
		}
		if strings.Contains(path, "dockerfile") {
			techSet["Docker"] = true
		}
		if strings.Contains(path, "kubernetes") || strings.Contains(path, "k8s") || strings.Contains(path, "helm") {
			techSet["Kubernetes"] = true
		}
		if strings.HasSuffix(path, ".tf") {
			techSet["Terraform"] = true
		}
		if strings.Contains(path, ".github/workflows") {
			techSet["GitHub Actions"] = true
		}
		if strings.Contains(path, ".gitlab-ci.yml") {
			techSet["GitLab CI"] = true
		}
		if path == ".travis.yml" {
			techSet["Travis CI"] = true
		}
		if path == "jenkinsfile" {
			techSet["Jenkins"] = true
		}
		if path == "circle.yml" || strings.Contains(path, ".circleci/config.yml") {
			techSet["CircleCI"] = true
		}
		if strings.Contains(path, "ansible") {
			techSet["Ansible"] = true
		}
		if path == "vagrantfile" {
			techSet["Vagrant"] = true
		}

		// Backend, Ecosystems & Manifests
		if strings.Contains(path, "manage.py") {
			techSet["Django"] = true
			techSet["Python"] = true
		}
		if strings.Contains(path, "requirements.txt") || strings.Contains(path, "pipfile") || strings.Contains(path, "pyproject.toml") || strings.Contains(path, "setup.py") {
			techSet["Python"] = true
		}
		if strings.Contains(path, "pom.xml") || strings.Contains(path, "build.gradle") {
			techSet["Java/JVM"] = true
		}
		if strings.Contains(path, "cargo.toml") {
			techSet["Rust"] = true
		}
		if strings.Contains(path, "go.mod") {
			techSet["Go"] = true
		}
		if strings.Contains(path, "gemfile") {
			techSet["Ruby"] = true
		}
		if strings.Contains(path, "composer.json") {
			techSet["PHP"] = true
		}
		if strings.Contains(path, "artisan") {
			techSet["Laravel (PHP)"] = true
		}
		if strings.Contains(path, "mix.exs") {
			techSet["Elixir"] = true
		}
		if strings.Contains(path, "rebar.config") {
			techSet["Erlang"] = true
		}
		if strings.HasSuffix(path, ".csproj") || strings.HasSuffix(path, ".sln") {
			techSet["C# / .NET"] = true
		}
		if path == "cmakelists.txt" || path == "makefile" || path == "configure.ac" {
			techSet["C/C++"] = true
		}
		if path == "pubspec.yaml" {
			techSet["Dart / Flutter"] = true
		}
		if path == "package.swift" {
			techSet["Swift"] = true
		}
		if strings.HasSuffix(path, ".scala") || path == "build.sbt" {
			techSet["Scala"] = true
		}
		if strings.Contains(path, "project.clj") || strings.HasSuffix(path, ".edn") {
			techSet["Clojure"] = true
		}

		// Frontend Frameworks, Tooling, & JS Ecosystem
		if strings.Contains(path, "next.config") {
			techSet["Next.js"] = true
			techSet["React"] = true
		}
		if strings.Contains(path, "nuxt.config") {
			techSet["Nuxt"] = true
			techSet["Vue.js"] = true
		}
		if strings.Contains(path, "vue.config") {
			techSet["Vue.js"] = true
		}
		if strings.Contains(path, "angular.json") {
			techSet["Angular"] = true
		}
		if strings.Contains(path, "svelte.config") {
			techSet["Svelte"] = true
		}
		if strings.Contains(path, "astro.config") {
			techSet["Astro"] = true
		}
		if strings.Contains(path, "vite.config") {
			techSet["Vite"] = true
		}
		if strings.Contains(path, "webpack.config") {
			techSet["Webpack"] = true
		}
		if strings.Contains(path, "rollup.config") {
			techSet["Rollup"] = true
		}
		if strings.Contains(path, "tailwind.config") {
			techSet["Tailwind CSS"] = true
		}
		if strings.Contains(path, "tsconfig.json") {
			techSet["TypeScript"] = true
		}
		if strings.Contains(path, "stencil.config") {
			techSet["Stencil"] = true
		}
		if strings.Contains(path, "gatsby-config") {
			techSet["Gatsby"] = true
		}

		// Mobile Frameworks
		if path == "android/app/build.gradle" || path == "ios/podfile" {
			techSet["Mobile (iOS/Android)"] = true
		}
		if strings.Contains(path, "capacitor.config") {
			techSet["Capacitor"] = true
		}
		if strings.Contains(path, "ionic.config") {
			techSet["Ionic"] = true
		}

		// ORMs and Databases
		if strings.Contains(path, "schema.prisma") {
			techSet["Prisma"] = true
		}
		if strings.Contains(path, "typeorm") {
			techSet["TypeORM"] = true
		}
		if strings.Contains(path, "alembic") {
			techSet["Alembic"] = true
		}
		if strings.Contains(path, "flyway") {
			techSet["Flyway"] = true
		}

		// Testing
		if strings.Contains(path, "jest.config") {
			techSet["Jest"] = true
		}
		if strings.Contains(path, "cypress.json") || strings.Contains(path, "cypress.config") {
			techSet["Cypress"] = true
		}
		if strings.Contains(path, "playwright.config") {
			techSet["Playwright"] = true
		}
		if strings.Contains(path, "vitest.config") {
			techSet["Vitest"] = true
		}
		if strings.Contains(path, "pytest.ini") {
			techSet["PyTest"] = true
		}
		if strings.Contains(path, "karma.conf") {
			techSet["Karma"] = true
		}

		// Folders/Structure
		if entry.GetType() == "tree" {
			if path == "pages" || path == "app" || path == "src/pages" || path == "src/app" {
				techSet["Server-side Routing (App/Pages)"] = true
			}
			if path == "components" || path == "src/components" {
				techSet["Component-based UI"] = true
			}
			if path == ".devcontainer" {
				techSet["DevContainers"] = true
			}
		}

		// Extensions (Fallback/Strong Hints)
		if strings.HasSuffix(path, ".tsx") || strings.HasSuffix(path, ".jsx") {
			techSet["React"] = true
			techSet["TypeScript/JavaScript"] = true
		}
		if strings.HasSuffix(path, ".vue") {
			techSet["Vue.js"] = true
		}
		if strings.HasSuffix(path, ".ts") {
			techSet["TypeScript"] = true
		}
		if strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".mjs") || strings.HasSuffix(path, ".cjs") {
			techSet["JavaScript"] = true
		}
		if strings.HasSuffix(path, ".py") {
			techSet["Python"] = true
		}
		if strings.HasSuffix(path, ".rb") {
			techSet["Ruby"] = true
		}
		if strings.HasSuffix(path, ".php") {
			techSet["PHP"] = true
		}
		if strings.HasSuffix(path, ".go") {
			techSet["Go"] = true
		}
		if strings.HasSuffix(path, ".rs") {
			techSet["Rust"] = true
		}
		if strings.HasSuffix(path, ".sh") || strings.HasSuffix(path, ".bash") {
			techSet["Shell Script"] = true
		}
		if strings.HasSuffix(path, ".kt") || strings.HasSuffix(path, ".kts") {
			techSet["Kotlin"] = true
		}
		if strings.HasSuffix(path, ".swift") {
			techSet["Swift"] = true
		}
		if strings.HasSuffix(path, ".dart") {
			techSet["Dart / Flutter"] = true
		}
		if strings.HasSuffix(path, ".c") || strings.HasSuffix(path, ".cpp") || strings.HasSuffix(path, ".h") || strings.HasSuffix(path, ".hpp") {
			techSet["C/C++"] = true
		}
		if strings.HasSuffix(path, ".cs") {
			techSet["C# / .NET"] = true
		}
		if strings.HasSuffix(path, ".ex") || strings.HasSuffix(path, ".exs") {
			techSet["Elixir"] = true
		}
		if strings.HasSuffix(path, ".java") {
			techSet["Java"] = true
		}
		if strings.HasSuffix(path, ".r") {
			techSet["R"] = true
		}
		if strings.HasSuffix(path, ".pl") || strings.HasSuffix(path, ".pm") {
			techSet["Perl"] = true
		}
		if strings.HasSuffix(path, ".lua") {
			techSet["Lua"] = true
		}
		if strings.HasSuffix(path, ".zig") {
			techSet["Zig"] = true
		}
	}

	for t := range techSet {
		tech = append(tech, t)
	}
	return tech
}

func getFileContent(ctx context.Context, client *github.Client, owner, repo, path string) (string, error) {
	fileContent, _, _, err := client.Repositories.GetContents(ctx, owner, repo, path, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get contents for %s: %w", path, err)
	}
	if fileContent == nil {
		return "", fmt.Errorf("no content returned for %s", path)
	}
	content, err := fileContent.GetContent()
	if err != nil {
		return "", fmt.Errorf("failed to decode content for %s: %w", path, err)
	}
	return content, nil
}

func parsePackageJSON(content string) []string {
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	var deps []string

	if err := json.Unmarshal([]byte(content), &pkg); err != nil {
		log.Printf("Warning: failed to parse package.json: %v", err)
		return deps
	}

	for k := range pkg.Dependencies {
		deps = append(deps, k)
	}
	for k := range pkg.DevDependencies {
		deps = append(deps, k)
	}
	return deps
}

func parseRequirementsTxt(content string) []string {
	var deps []string
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Ignore comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Split by typical version delimiters
		parts := strings.FieldsFunc(line, func(r rune) bool {
			return r == '=' || r == '>' || r == '<' || r == '~' || r == '!' || r == ';'
		})
		if len(parts) > 0 {
			pkgName := strings.TrimSpace(parts[0])
			if pkgName != "" {
				deps = append(deps, pkgName)
			}
		}
	}
	return deps
}

func uniqueStrings(input []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range input {
		if entry != "" && !keys[entry] {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
