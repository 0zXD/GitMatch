import { useState, useMemo, useEffect } from "react";
import { IssueCard } from "./components/issue-card";
import { SkillSelector } from "./components/skill-selector";
import { SavedIssues } from "./components/saved-issues";
import { GitHubLogin } from "./components/github-login";
import { UserProfileOverview } from "./components/user-profile-overview";
import { Button } from "./components/ui/button";
import { Github, Sparkles, BookMarked, Search, Filter, Moon, Sun, LogOut, Loader2 } from "lucide-react";
import { Input } from "./components/ui/input";
import type { UserProfile } from "./types/user-profile";

import { useGitHubIssues } from "./hooks/useGitHubIssues";
import { useSavedIssues } from "./hooks/useSavedIssues";
import { useUserExperience } from "./hooks/useUserExperience";
import { useDarkMode } from "./hooks/useDarkMode";

export default function App() {
  const [userProfile, setUserProfile] = useState<UserProfile | null>(() => {
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
  const [activeTab, setActiveTab] = useState<"discover" | "saved">("discover");
  const [showSkillSelector, setShowSkillSelector] = useState(false);

  const { darkMode, toggleDarkMode } = useDarkMode();
  const { userLanguages, userExperienceData } = useUserExperience(userProfile, selectedSkills);
  
  const { savedIssues, loadSavedIssues, handleInterested, handleRemoveSaved, setSavedIssues } = useSavedIssues(userProfile);
  
  const { 
    apiIssues, loading, loadingMore, apiError, hasMorePages, loadMoreIssues, setApiIssues 
  } = useGitHubIssues({
    selectedSkills,
    level: userExperienceData.level,
    count: userExperienceData.count
  });

  const githubUser = userProfile?.username ?? null;

  useEffect(() => {
    // Catch OAuth Redirects via URL params
    const params = new URLSearchParams(window.location.search);
    const oauthUser = params.get("username");
    if (oauthUser && !userProfile) {
      // Clear url parameter without page reload
      window.history.replaceState({}, document.title, window.location.pathname);
      
      // Fetch profile automatically
      fetch(`http://localhost:8084/user?username=${encodeURIComponent(oauthUser)}`)
        .then(res => res.json())
        .then(profile => {
          handleLogin(profile);
        })
        .catch(console.error);
    } else if (userProfile) {
      loadSavedIssues(userProfile.username);
    }
  }, [userProfile, loadSavedIssues]);

  const handleLogin = (profile: UserProfile) => {
    localStorage.setItem("userProfile", JSON.stringify(profile));
    setUserProfile(profile);
    loadSavedIssues(profile.username);
  };

  const handleLogout = () => {
    localStorage.removeItem("userProfile");
    setUserProfile(null);
    setSelectedSkills([]);
    setSavedIssues([]);
    setApiIssues([]);
  };

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

  const handleGoBack = () => {
    setCurrentIndex((prev) => Math.max(0, prev - 6));
  };

  const handleSkip = () => {};

  if (!userProfile) {
    return <GitHubLogin onSubmit={handleLogin} />;
  }

  return (
    <div className={darkMode ? "dark" : ""}>
      <div className="min-h-screen bg-[#f6f8fa] dark:bg-[#0d1117] transition-colors">
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
                >
                  {darkMode ? <Sun className="w-5 h-5 text-[#f0883e]" /> : <Moon className="w-5 h-5 text-[#8b949e]" />}
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
                <button onClick={handleLogout} className="p-2 rounded-lg hover:bg-[#30363d] transition-colors">
                  <LogOut className="w-5 h-5 text-[#8b949e]" />
                </button>
              </div>
            </div>
          </div>
        </header>

        <div className="bg-white dark:bg-[#0d1117] border-b border-[#d0d7de] dark:border-[#30363d] transition-colors">
          <div className="max-w-7xl mx-auto px-4">
            <div className="flex items-center gap-4">
              <button
                onClick={() => setActiveTab("discover")}
                className={`px-4 py-3 text-sm font-medium border-b-2 transition-colors ${
                  activeTab === "discover" ? "border-[#fd8c73] text-[#24292f] dark:text-[#e6edf3]" : "border-transparent text-[#656d76] dark:text-[#8b949e] hover:text-[#24292f] dark:hover:text-[#e6edf3]"
                }`}
              >
                <Sparkles className="inline w-4 h-4 mr-2" />
                Discover Issues
              </button>
              <button
                onClick={() => setActiveTab("saved")}
                className={`px-4 py-3 text-sm font-medium border-b-2 transition-colors ${
                  activeTab === "saved" ? "border-[#fd8c73] text-[#24292f] dark:text-[#e6edf3]" : "border-transparent text-[#656d76] dark:text-[#8b949e] hover:text-[#24292f] dark:hover:text-[#e6edf3]"
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
              <div className="bg-white dark:bg-[#161b22] rounded-lg border border-[#d0d7de] dark:border-[#30363d] p-4 mb-6 transition-colors">
                <div className="flex items-center justify-between mb-3">
                  <div className="flex items-center gap-2">
                    <Filter className="w-4 h-4 text-[#656d76] dark:text-[#8b949e]" />
                    <span className="text-sm font-semibold text-[#24292f] dark:text-[#e6edf3]">
                      Filter by your skills
                    </span>
                    {selectedSkills.length > 0 && <span className="text-xs text-[#656d76] dark:text-[#8b949e]">({selectedSkills.length} selected)</span>}
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
                    <SkillSelector selectedSkills={selectedSkills} onSkillToggle={handleSkillToggle} darkMode={darkMode} userLanguages={userLanguages} />
                    {selectedSkills.length > 0 && (
                      <div className="mt-3 flex items-center justify-between">
                        <span className="text-sm text-[#656d76] dark:text-[#8b949e]">
                          {loading ? (
                            <span className="flex items-center gap-2"><Loader2 className="w-3.5 h-3.5 animate-spin" /> Searching GitHub...</span>
                          ) : apiError ? (
                            <span className="text-[#cf222e] dark:text-[#f85149]">API unavailable — local results ({filteredIssues.length})</span>
                          ) : (
                            `Showing ${filteredIssues.length} matching issues`
                          )}
                        </span>
                        <Button onClick={() => setSelectedSkills([])} variant="link" size="sm" className="text-[#0969da] dark:text-[#58a6ff]">Clear all</Button>
                      </div>
                    )}
                  </div>
                )}
              </div>

              {selectedSkills.length === 0 && userProfile ? (
                <UserProfileOverview profile={userProfile} darkMode={darkMode} />
              ) : loading ? (
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
                  {Array.from({ length: 4 }).map((_, i) => (
                    <div key={i} className="rounded-lg border bg-white dark:bg-[#161b22] border-[#d0d7de] dark:border-[#30363d] p-6 animate-pulse">
                      <div className="h-4 bg-[#d0d7de] dark:bg-[#30363d] rounded w-1/3 mb-3"></div>
                    </div>
                  ))}
                </div>
              ) : currentIssues.length > 0 ? (
                <>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
                    {currentIssues.map((issue) => (
                      <IssueCard key={issue.id} issue={issue} onInterested={() => handleInterested(issue)} onSkip={handleSkip} darkMode={darkMode} />
                    ))}
                  </div>

                  <div className="flex items-center justify-center gap-3 mt-2">
                    {currentIndex > 0 && (
                      <Button onClick={handleGoBack} variant="outline" className="border-[#d0d7de] dark:border-[#30363d] text-[#24292f] dark:text-[#e6edf3]">← Previous</Button>
                    )}
                    {filteredIssues.length > 0 && <span className="text-xs text-[#656d76] dark:text-[#8b949e]">{currentIndex + 1}–{Math.min(currentIndex + 6, filteredIssues.length)} of {filteredIssues.length}</span>}
                    {currentIndex + 6 < filteredIssues.length && <Button onClick={() => setCurrentIndex(c => c + 6)} className="bg-[#0969da] hover:bg-[#0860ca] text-white">Load More →</Button>}
                  </div>

                  {currentIndex + 6 >= filteredIssues.length && filteredIssues.length > 0 && (
                    <div className="bg-white dark:bg-[#161b22] rounded-lg border p-8 text-center mt-4">
                      {hasMorePages ? (
                        <>
                          <p className="text-[#656d76] dark:text-[#8b949e] mb-3">Seen all loaded results. Find more?</p>
                          <Button onClick={loadMoreIssues} disabled={loadingMore} className="bg-[#0969da] text-white">
                            {loadingMore ? <><Loader2 className="w-4 h-4 mr-2 animate-spin" /> Fetching…</> : "Load More Issues"}
                          </Button>
                        </>
                      ) : (
                        <>
                          <p className="text-[#656d76] dark:text-[#8b949e] mb-2">Seen all available issues</p>
                          <Button onClick={() => { setSelectedSkills([]); setCurrentIndex(0); }} variant="outline">Clear filters</Button>
                        </>
                      )}
                    </div>
                  )}
                </>
              ) : (
                <div className="bg-white dark:bg-[#161b22] rounded-lg border border-[#d0d7de] dark:border-[#30363d] p-12 text-center transition-colors">
                  <Sparkles className="w-12 h-12 text-[#656d76] dark:text-[#8b949e] mx-auto mb-4" />
                  <h3 className="text-lg font-semibold text-[#24292f] dark:text-[#e6edf3] mb-2">No issues found</h3>
                  <Button onClick={() => { setSelectedSkills([]); setShowSkillSelector(true); }} className="bg-[#0969da] text-white">Adjust Filters</Button>
                </div>
              )}
            </div>
          ) : (
            <div>
              <div className="mb-4">
                <h2 className="text-xl font-semibold text-[#24292f] dark:text-[#e6edf3]">Your Saved Issues</h2>
              </div>
              <SavedIssues issues={savedIssues} onRemove={handleRemoveSaved} darkMode={darkMode} />
            </div>
          )}
        </main>
      </div>
    </div>
  );
}
