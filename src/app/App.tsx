import { useState, useMemo, useEffect, useRef, useCallback } from "react";
import { IssueCard, GitHubIssue } from "./components/issue-card";
import { SkillSelector } from "./components/skill-selector";
import { SavedIssues } from "./components/saved-issues";
import { GitHubLogin } from "./components/github-login";
import { UserProfileOverview } from "./components/user-profile-overview";
import { Button } from "./components/ui/button";
import { Github, Sparkles, BookMarked, Search, Filter, Moon, Sun, LogOut, Loader2 } from "lucide-react";
import { fetchReposForSkills, type HarvestResult } from "./services/harvester";
import { Input } from "./components/ui/input";
import type { UserProfile } from "./types/user-profile";

export default function App() {
  const [userProfile, setUserProfile] = useState<UserProfile | null>(() => {
    // Clean up old login key from previous implementation
    localStorage.removeItem("githubUser");
    const stored = localStorage.getItem("userProfile");
    try {
      return stored ? JSON.parse(stored) : null;
    } catch {
      localStorage.removeItem("userProfile");
      return null;
    }
  });
  const [selectedSkills, setSelectedSkills] = useState<string[]>([]);
  const [currentIndex, setCurrentIndex] = useState(0);
  const [savedIssues, setSavedIssues] = useState<GitHubIssue[]>([]);
  const [activeTab, setActiveTab] = useState<"discover" | "saved">("discover");
  const [showSkillSelector, setShowSkillSelector] = useState(false);
  const [darkMode, setDarkMode] = useState(false);
  const [apiIssues, setApiIssues] = useState<GitHubIssue[]>([]);
  const [loading, setLoading] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [apiError, setApiError] = useState<string | null>(null);
  const [apiPage, setApiPage] = useState(1);
  const [hasMorePages, setHasMorePages] = useState(false);
  const [apiCursor, setApiCursor] = useState<string | undefined>();
  const abortRef = useRef<AbortController | null>(null);

  const githubUser = userProfile?.username ?? null;

  // Derive sorted user languages from profile (most-used first)
  const userLanguages = useMemo(() => {
    if (!userProfile?.languages) return [];
    return Object.entries(userProfile.languages)
      .sort(([, a], [, b]) => b - a)
      .map(([lang]) => lang);
  }, [userProfile]);

  const handleLogin = (profile: UserProfile) => {
    localStorage.setItem("userProfile", JSON.stringify(profile));
    setUserProfile(profile);
  };

  const handleLogout = () => {
    localStorage.removeItem("userProfile");
    setUserProfile(null);
    setSelectedSkills([]);
    setApiCursor(undefined);
  };

  useEffect(() => {
    // Check for saved preference
    const savedMode = localStorage.getItem("darkMode");
    if (savedMode === "true") {
      setDarkMode(true);
    }
  }, []);

  useEffect(() => {
    // Save preference
    localStorage.setItem("darkMode", darkMode.toString());
  }, [darkMode]);

  // Fetch repos from harvester API when skills change
  useEffect(() => {
    if (selectedSkills.length === 0) {
      setApiIssues([]);
      setApiError(null);
      setLoading(false);
      setApiPage(1);
      setHasMorePages(false);
      setApiCursor(undefined);
      return;
    }

    // Abort any in-flight request
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;

    const debounce = setTimeout(async () => {
      setLoading(true);
      setApiError(null);
      try {
        const result = await fetchReposForSkills(selectedSkills, controller.signal, 1);
        if (!controller.signal.aborted) {
          setApiIssues(result.issues);
          setApiPage(1);
          setHasMorePages(result.hasMore);
          setCurrentIndex(0);
          setApiCursor(result.endCursor);
        }
      } catch (err: unknown) {
        if (!controller.signal.aborted) {
          const msg = err instanceof Error ? err.message : "Failed to fetch";
          if (err instanceof DOMException && err.name === "AbortError") return;
          setApiError(msg);
        }
      } finally {
        if (!controller.signal.aborted) {
          setLoading(false);
        }
      }
    }, 500);

    return () => {
      clearTimeout(debounce);
      controller.abort();
    };
  }, [selectedSkills]);

  const filteredIssues = useMemo(() => {
    if (selectedSkills.length > 0 && apiIssues.length > 0) {
      return apiIssues.filter((issue) => !savedIssues.find((saved) => saved.id === issue.id));
    }
    return [];
  }, [selectedSkills, savedIssues, apiIssues]);

  const currentIssues = filteredIssues.slice(currentIndex, currentIndex + 6);

  const handleSkillToggle = (skill: string) => {
    setSelectedSkills((prev) =>
      prev.includes(skill) ? prev.filter((s) => s !== skill) : [...prev, skill]
    );
    setCurrentIndex(0);
  };

  const handleInterested = (issue: GitHubIssue) => {
    setSavedIssues((prev) => [...prev, issue]);
  };

  const handleSkip = () => {
    // Just move to next issue
  };

  const handleRemoveSaved = (id: number) => {
    setSavedIssues((prev) => prev.filter((issue) => issue.id !== id));
  };

  const handleLoadMore = () => {
    setCurrentIndex((prev) => prev + 6);
  };

  const handleFetchNextPage = useCallback(async () => {
    if (loadingMore || !hasMorePages) return;
    const nextPage = apiPage + 1;
    setLoadingMore(true);
    try {
      const result = await fetchReposForSkills(selectedSkills, undefined, nextPage, apiCursor);
      // Deduplicate by id
      const existingIds = new Set(apiIssues.map((i) => i.id));
      const newIssues = result.issues.filter((i) => !existingIds.has(i.id));
      setApiIssues((prev) => [...prev, ...newIssues]);
      setApiPage(nextPage);
      setHasMorePages(result.hasMore);
      setApiCursor(result.endCursor);
    } catch (err: unknown) {
      if (err instanceof DOMException && err.name === "AbortError") return;
      const msg = err instanceof Error ? err.message : "Failed to fetch more";
      setApiError(msg);
    } finally {
      setLoadingMore(false);
    }
  }, [loadingMore, hasMorePages, apiPage, selectedSkills, apiIssues, apiCursor]);

  const handleGoBack = () => {
    setCurrentIndex((prev) => Math.max(0, prev - 6));
  };

  const toggleDarkMode = () => {
    setDarkMode(!darkMode);
  };

  if (!userProfile) {
    return <GitHubLogin onSubmit={handleLogin} />;
  }

  return (
    <div className={darkMode ? "dark" : ""}>
      <div className="min-h-screen bg-[#f6f8fa] dark:bg-[#0d1117] transition-colors">
        {/* Header - GitHub Style */}
        <header className="bg-[#24292f] border-b border-[#d0d7de] dark:border-[#30363d]">
          <div className="max-w-7xl mx-auto px-4 py-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-4">
                <div className="flex items-center gap-2">
                  <Github className="w-8 h-8 text-white" />
                  <span className="text-xl font-semibold text-white">GitMatch</span>
                </div>
                <div className="relative hidden md:block">
                  <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-[#656d76]" />
                  <Input
                    placeholder="Search issues..."
                    className="pl-9 w-80 bg-[#0d1117] border-[#30363d] text-white placeholder:text-[#656d76]"
                  />
                </div>
              </div>
              <div className="flex items-center gap-3">
                <button
                  onClick={toggleDarkMode}
                  className="p-2 rounded-lg hover:bg-[#30363d] transition-colors"
                  aria-label="Toggle dark mode"
                >
                  {darkMode ? (
                    <Sun className="w-5 h-5 text-[#f0883e]" />
                  ) : (
                    <Moon className="w-5 h-5 text-[#8b949e]" />
                  )}
                </button>
                <div className="flex items-center gap-2 bg-[#21262d] text-[#e6edf3] px-3 py-1.5 rounded-full text-sm font-medium">
                  <img
                    src={`https://github.com/${githubUser}.png?size=24`}
                    alt={githubUser ?? ""}
                    className="w-5 h-5 rounded-full"
                    onError={(e) => { (e.target as HTMLImageElement).style.display = 'none'; }}
                  />
                  {githubUser}
                </div>
                <div className="bg-[#0969da] text-white px-3 py-1.5 rounded-full text-sm font-medium">
                  {savedIssues.length} Saved
                </div>
                <button
                  onClick={handleLogout}
                  className="p-2 rounded-lg hover:bg-[#30363d] transition-colors"
                  aria-label="Log out"
                  title="Log out"
                >
                  <LogOut className="w-5 h-5 text-[#8b949e]" />
                </button>
              </div>
            </div>
          </div>
        </header>

        {/* Navigation Tabs */}
        <div className="bg-white dark:bg-[#0d1117] border-b border-[#d0d7de] dark:border-[#30363d] transition-colors">
          <div className="max-w-7xl mx-auto px-4">
            <div className="flex items-center gap-4">
              <button
                onClick={() => setActiveTab("discover")}
                className={`px-4 py-3 text-sm font-medium border-b-2 transition-colors ${
                  activeTab === "discover"
                    ? "border-[#fd8c73] text-[#24292f] dark:text-[#e6edf3]"
                    : "border-transparent text-[#656d76] dark:text-[#8b949e] hover:text-[#24292f] dark:hover:text-[#e6edf3] hover:border-[#d0d7de] dark:hover:border-[#30363d]"
                }`}
              >
                <Sparkles className="inline w-4 h-4 mr-2" />
                Discover Issues
              </button>
              <button
                onClick={() => setActiveTab("saved")}
                className={`px-4 py-3 text-sm font-medium border-b-2 transition-colors ${
                  activeTab === "saved"
                    ? "border-[#fd8c73] text-[#24292f] dark:text-[#e6edf3]"
                    : "border-transparent text-[#656d76] dark:text-[#8b949e] hover:text-[#24292f] dark:hover:text-[#e6edf3] hover:border-[#d0d7de] dark:hover:border-[#30363d]"
                }`}
              >
                <BookMarked className="inline w-4 h-4 mr-2" />
                Saved Issues ({savedIssues.length})
              </button>
            </div>
          </div>
        </div>

        <main className="max-w-7xl mx-auto px-4 py-6">
          {activeTab === "discover" ? (
            <div>
              {/* Filter Bar */}
              <div className="bg-white dark:bg-[#161b22] rounded-lg border border-[#d0d7de] dark:border-[#30363d] p-4 mb-6 transition-colors">
                <div className="flex items-center justify-between mb-3">
                  <div className="flex items-center gap-2">
                    <Filter className="w-4 h-4 text-[#656d76] dark:text-[#8b949e]" />
                    <span className="text-sm font-semibold text-[#24292f] dark:text-[#e6edf3]">
                      Filter by your skills
                    </span>
                    {selectedSkills.length > 0 && (
                      <span className="text-xs text-[#656d76] dark:text-[#8b949e]">
                        ({selectedSkills.length} selected)
                      </span>
                    )}
                  </div>
                  <Button
                    onClick={() => setShowSkillSelector(!showSkillSelector)}
                    variant="outline"
                    size="sm"
                    className="border-[#d0d7de] dark:border-[#30363d] text-[#24292f] dark:text-[#e6edf3] hover:bg-[#f6f8fa] dark:hover:bg-[#21262d]"
                  >
                    {showSkillSelector ? "Hide Filters" : "Show Filters"}
                  </Button>
                </div>

                {showSkillSelector && (
                  <div className="pt-3 border-t border-[#d0d7de] dark:border-[#30363d]">
                    <SkillSelector
                      selectedSkills={selectedSkills}
                      onSkillToggle={handleSkillToggle}
                      darkMode={darkMode}
                      userLanguages={userLanguages}
                    />
                    {selectedSkills.length > 0 && (
                      <div className="mt-3 flex items-center justify-between">
                        <span className="text-sm text-[#656d76] dark:text-[#8b949e]">
                          {loading ? (
                            <span className="flex items-center gap-2">
                              <Loader2 className="w-3.5 h-3.5 animate-spin" />
                              Searching GitHub repositories...
                            </span>
                          ) : apiError ? (
                            <span className="text-[#cf222e] dark:text-[#f85149]">
                              API unavailable — showing local results ({filteredIssues.length})
                            </span>
                          ) : (
                            `Showing ${filteredIssues.length} matching ${filteredIssues.length === 1 ? "repository" : "repositories"}`
                          )}
                        </span>
                        <Button
                          onClick={() => setSelectedSkills([])}
                          variant="link"
                          size="sm"
                          className="text-[#0969da] dark:text-[#58a6ff]"
                        >
                          Clear all filters
                        </Button>
                      </div>
                    )}
                  </div>
                )}
              </div>

              {/* Issue Grid */}
              {selectedSkills.length === 0 && userProfile ? (
                <UserProfileOverview profile={userProfile} darkMode={darkMode} />
              ) : loading ? (
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
                  {Array.from({ length: 4 }).map((_, i) => (
                    <div
                      key={i}
                      className="rounded-lg border bg-white dark:bg-[#161b22] border-[#d0d7de] dark:border-[#30363d] p-6 animate-pulse"
                    >
                      <div className="h-4 bg-[#d0d7de] dark:bg-[#30363d] rounded w-1/3 mb-3"></div>
                      <div className="h-5 bg-[#d0d7de] dark:bg-[#30363d] rounded w-3/4 mb-4"></div>
                      <div className="h-3 bg-[#d0d7de] dark:bg-[#30363d] rounded w-full mb-2"></div>
                      <div className="h-3 bg-[#d0d7de] dark:bg-[#30363d] rounded w-2/3 mb-4"></div>
                      <div className="flex gap-2 mb-4">
                        <div className="h-5 bg-[#d0d7de] dark:bg-[#30363d] rounded-full w-16"></div>
                        <div className="h-5 bg-[#d0d7de] dark:bg-[#30363d] rounded-full w-20"></div>
                      </div>
                      <div className="flex gap-4">
                        <div className="h-3 bg-[#d0d7de] dark:bg-[#30363d] rounded w-12"></div>
                        <div className="h-3 bg-[#d0d7de] dark:bg-[#30363d] rounded w-12"></div>
                      </div>
                    </div>
                  ))}
                </div>
              ) : currentIssues.length > 0 ? (
                <>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
                    {currentIssues.map((issue) => (
                      <IssueCard
                        key={issue.id}
                        issue={issue}
                        onInterested={() => handleInterested(issue)}
                        onSkip={handleSkip}
                        darkMode={darkMode}
                      />
                    ))}
                  </div>

                  {/* Pagination Controls */}
                  <div className="flex items-center justify-center gap-3 mt-2">
                    {currentIndex > 0 && (
                      <Button
                        onClick={handleGoBack}
                        variant="outline"
                        className="border-[#d0d7de] dark:border-[#30363d] text-[#24292f] dark:text-[#e6edf3] hover:bg-[#f6f8fa] dark:hover:bg-[#21262d]"
                      >
                        ← Previous
                      </Button>
                    )}
                    {filteredIssues.length > 0 && (
                      <span className="text-xs text-[#656d76] dark:text-[#8b949e]">
                        {currentIndex + 1}–{Math.min(currentIndex + 6, filteredIssues.length)} of {filteredIssues.length}
                      </span>
                    )}
                    {currentIndex + 6 < filteredIssues.length && (
                      <Button
                        onClick={handleLoadMore}
                        className="bg-[#0969da] hover:bg-[#0860ca] dark:bg-[#1f6feb] dark:hover:bg-[#1a5cd7] text-white"
                      >
                        Load More →
                      </Button>
                    )}
                  </div>

                  {currentIndex + 6 >= filteredIssues.length && filteredIssues.length > 0 && (
                    <div className="bg-white dark:bg-[#161b22] rounded-lg border border-[#d0d7de] dark:border-[#30363d] p-8 text-center transition-colors mt-4">
                      {hasMorePages ? (
                        <>
                          <p className="text-[#656d76] dark:text-[#8b949e] mb-3">
                            You've seen all loaded results. Want to find more repositories?
                          </p>
                          <Button
                            onClick={handleFetchNextPage}
                            disabled={loadingMore}
                            className="bg-[#0969da] hover:bg-[#0860ca] dark:bg-[#1f6feb] dark:hover:bg-[#1a5cd7] text-white"
                          >
                            {loadingMore ? (
                              <>
                                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                                Fetching more…
                              </>
                            ) : (
                              "Load More Repositories"
                            )}
                          </Button>
                        </>
                      ) : (
                        <>
                          <p className="text-[#656d76] dark:text-[#8b949e] mb-2">
                            You've seen all available repositories matching your filters
                          </p>
                          {currentIndex > 0 && (
                            <Button
                              onClick={handleGoBack}
                              variant="outline"
                              className="border-[#d0d7de] dark:border-[#30363d] text-[#24292f] dark:text-[#e6edf3] hover:bg-[#f6f8fa] dark:hover:bg-[#21262d] mr-2"
                            >
                              ← Go Back
                            </Button>
                          )}
                          <Button
                            onClick={() => {
                              setSelectedSkills([]);
                              setCurrentIndex(0);
                            }}
                            variant="outline"
                            className="border-[#d0d7de] dark:border-[#30363d] text-[#24292f] dark:text-[#e6edf3] hover:bg-[#f6f8fa] dark:hover:bg-[#21262d]"
                          >
                            Clear filters to see more
                          </Button>
                        </>
                      )}
                    </div>
                  )}
                </>
              ) : (
                <div className="bg-white dark:bg-[#161b22] rounded-lg border border-[#d0d7de] dark:border-[#30363d] p-12 text-center transition-colors">
                  <Sparkles className="w-12 h-12 text-[#656d76] dark:text-[#8b949e] mx-auto mb-4" />
                  <h3 className="text-lg font-semibold text-[#24292f] dark:text-[#e6edf3] mb-2">
                    No issues found
                  </h3>
                  <p className="text-[#656d76] dark:text-[#8b949e] mb-4">
                    Try adjusting your skill filters to see more issues
                  </p>
                  <Button
                    onClick={() => {
                      setSelectedSkills([]);
                      setShowSkillSelector(true);
                    }}
                    className="bg-[#0969da] hover:bg-[#0860ca] dark:bg-[#1f6feb] dark:hover:bg-[#1a5cd7] text-white"
                  >
                    Adjust Filters
                  </Button>
                </div>
              )}
            </div>
          ) : (
            <div>
              <div className="mb-4">
                <h2 className="text-xl font-semibold text-[#24292f] dark:text-[#e6edf3]">Your Saved Issues</h2>
                <p className="text-sm text-[#656d76] dark:text-[#8b949e]">
                  Issues you're interested in contributing to
                </p>
              </div>
              <SavedIssues issues={savedIssues} onRemove={handleRemoveSaved} darkMode={darkMode} />
            </div>
          )}
        </main>

        {/* Footer */}
        <footer className="mt-16 border-t border-[#d0d7de] dark:border-[#30363d] py-8 transition-colors">
          <div className="max-w-7xl mx-auto px-4 text-center text-[#656d76] dark:text-[#8b949e] text-sm">
            <p>Find open source issues that match your skills and start contributing today</p>
          </div>
        </footer>
      </div>
    </div>
  );
}