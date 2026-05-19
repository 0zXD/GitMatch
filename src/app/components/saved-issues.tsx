import { GitHubIssue, IssueAnalysis } from "./issue-card";
import { ExternalLink, GitFork, Star, MessageSquare, Trash2, Check, ArrowRight, Loader2 } from "lucide-react";
import { Button } from "./ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "./ui/dialog";
import { JSXElementConstructor, Key, ReactElement, ReactNode, ReactPortal, useState } from "react";
import { analyzeSavedIssue } from "../services/harvester";
import { Badge } from "./ui/badge";

interface SavedIssuesProps {
  issues: GitHubIssue[];
  onRemove: (id: number) => void;
  darkMode: boolean;
}

export function SavedIssues({ issues, onRemove, darkMode }: SavedIssuesProps) {
  const [analyzingIssue, setAnalyzingIssue] = useState<GitHubIssue | null>(null);
  const [analysis, setAnalysis] = useState<IssueAnalysis | null>(null);
  const [isAnalyzing, setIsAnalyzing] = useState(false);
  const [analysisError, setAnalysisError] = useState<string | null>(null);

  const handleAnalyze = async (issue: GitHubIssue) => {
    setAnalyzingIssue(issue);
    setAnalysis(null);
    setAnalysisError(null);
    setIsAnalyzing(true);
    
    try {
      const [owner, repo] = issue.repository.split("/");
      const issueNumber = issue.number || parseInt(issue.url.split("/").pop() || "0");
      if (owner && repo && issueNumber) {
        const result = await analyzeSavedIssue(owner, repo, issueNumber);
        setAnalysis(result);
      }
    } catch (error: any) {
      console.error(error);
      const msg = error?.message || "Failed to load analysis.";
      if (msg.includes("429") || msg.includes("quota")) {
        setAnalysisError("OpenAI API Quota Exceeded. Please check your billing details or use a different API key.");
      } else {
        setAnalysisError(msg);
      }
    } finally {
      setIsAnalyzing(false);
    }
  };

  const difficultyColors = {
    beginner: darkMode
      ? "bg-[#1a7f37] text-white border-[#1a7f37]"
      : "bg-[#1a7f37] text-white border-[#1a7f37]",
    intermediate: darkMode
      ? "bg-[#bf8700] text-white border-[#bf8700]"
      : "bg-[#bf8700] text-white border-[#bf8700]",
    advanced: darkMode
      ? "bg-[#cf222e] text-white border-[#cf222e]"
      : "bg-[#cf222e] text-white border-[#cf222e]",
  };

  const labelColors = [
    darkMode ? "bg-[#1f6feb] text-white" : "bg-[#0969da] text-white",
    darkMode ? "bg-[#2ea043] text-white" : "bg-[#1a7f37] text-white",
    darkMode ? "bg-[#a371f7] text-white" : "bg-[#8250df] text-white",
    darkMode ? "bg-[#d29922] text-black" : "bg-[#bf8700] text-white",
    darkMode ? "bg-[#f85149] text-white" : "bg-[#d1242f] text-white",
  ];

  if (issues.length === 0) {
    return (
      <div className={`rounded-lg border p-12 text-center transition-colors ${
        darkMode
          ? "bg-[#161b22] border-[#30363d]"
          : "bg-white border-[#d0d7de]"
      }`}>
        <p className={`text-lg mb-2 ${
          darkMode ? "text-[#8b949e]" : "text-[#656d76]"
        }`}>
          No saved issues yet
        </p>
        <p className={`text-sm ${
          darkMode ? "text-[#8b949e]" : "text-[#656d76]"
        }`}>
          Click "I'm Interested" on issues you want to contribute to
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {issues.map((issue) => (
        <div
          key={issue.id}
          className={`rounded-lg border transition-all ${
            darkMode
              ? "bg-[#161b22] border-[#30363d] hover:border-[#58a6ff]"
              : "bg-white border-[#d0d7de] hover:border-[#0969da]"
          }`}
        >
          <div className="p-4">
            <div className="flex items-start justify-between gap-3 mb-3">
              <div className="flex-1">
                <div className={`flex items-center gap-2 text-xs mb-1 ${
                  darkMode ? "text-[#8b949e]" : "text-[#656d76]"
                }`}>
                  <span className="font-semibold">{issue.repository}</span>
                </div>
                <h3 className={`text-base font-semibold leading-tight mb-2 ${
                  darkMode ? "text-[#e6edf3]" : "text-[#24292f]"
                }`}>
                  {issue.title}
                </h3>
                <p className={`text-sm mb-3 ${
                  darkMode ? "text-[#e6edf3]" : "text-[#24292f]"
                }`}>
                  {issue.description}
                </p>

                {/* Labels */}
                <div className="flex flex-wrap gap-2 mb-3">
                  <span className={`${difficultyColors[issue.difficulty]} text-xs px-2 py-0.5 rounded-full`}>
                    {issue.difficulty}
                  </span>
                  {issue.labels.map((label, idx) => (
                    <span
                      key={label}
                      className={`${labelColors[idx % labelColors.length]} text-xs px-2.5 py-0.5 rounded-full`}
                    >
                      {label}
                    </span>
                  ))}
                </div>

                {/* Stats */}
                <div className={`flex items-center gap-4 text-xs ${
                  darkMode ? "text-[#8b949e]" : "text-[#656d76]"
                }`}>
                  <div className="flex items-center gap-1">
                    <div className={`w-3 h-3 rounded-full ${
                      darkMode ? "bg-[#58a6ff]" : "bg-[#0969da]"
                    }`}></div>
                    <span className="font-medium">{issue.language}</span>
                  </div>
                  <div className="flex items-center gap-1">
                    <Star className="w-3.5 h-3.5" />
                    <span>{issue.stars.toLocaleString()}</span>
                  </div>
                  <div className="flex items-center gap-1">
                    <GitFork className="w-3.5 h-3.5" />
                    <span>{issue.forks.toLocaleString()}</span>
                  </div>
                  <div className="flex items-center gap-1">
                    <MessageSquare className="w-3.5 h-3.5" />
                    <span>{issue.comments}</span>
                  </div>
                </div>

                <div className="mt-4">
                  <Button 
                    onClick={() => handleAnalyze(issue)}
                    variant="outline" 
                    className={`w-full sm:w-auto ${darkMode ? "border-[#30363d] hover:bg-[#21262d]" : ""}`}
                  >
                    View More Details <ArrowRight className="w-4 h-4 ml-2" />
                  </Button>
                </div>
              </div>

              <div className="flex flex-col gap-2">
                <a
                  href={issue.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className={darkMode ? "text-[#58a6ff] hover:text-[#79c0ff]" : "text-[#0969da] hover:text-[#0860ca]"}
                >
                  <ExternalLink className="w-5 h-5" />
                </a>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => onRemove(issue.id)}
                  className={darkMode
                    ? "text-[#f85149] hover:text-[#f85149] hover:bg-[#f85149]/10 p-1"
                    : "text-[#cf222e] hover:text-[#cf222e] hover:bg-[#cf222e]/10 p-1"
                  }
                >
                  <Trash2 className="w-5 h-5" />
                </Button>
              </div>
            </div>
          </div>
        </div>
      ))}

      <Dialog open={!!analyzingIssue} onOpenChange={(open) => {
        if (!open) {
          setAnalyzingIssue(null);
          setAnalysisError(null);
        }
      }}>
        <DialogContent className={`max-w-2xl max-h-[85vh] overflow-y-auto ${darkMode ? "bg-[#0d1117] border-[#30363d] text-[#e6edf3]" : "bg-white"}`}>
          <DialogHeader>
            <DialogTitle className="text-xl flex items-center gap-2">
              Issue Breakdown
            </DialogTitle>
            <DialogDescription className={darkMode ? "text-[#8b949e]" : ""}>
              {analyzingIssue?.title}
            </DialogDescription>
          </DialogHeader>

          {isAnalyzing ? (
            <div className="flex flex-col items-center justify-center py-12 space-y-4">
              <Loader2 className="w-8 h-8 animate-spin text-[#0969da]" />
              <p className={darkMode ? "text-[#8b949e]" : "text-[#656d76]"}>
                Scanning README and Issue details with AI...
              </p>
            </div>
          ) : analysis ? (
            <div className="space-y-6 mt-4">
              <div>
                <h4 className="font-semibold mb-2">Debrief</h4>
                <p className={`text-sm ${darkMode ? "text-[#c9d1d9]" : "text-[#24292f]"}`}>
                  {analysis.issue_debrief}
                </p>
              </div>

              <div>
                <h4 className="font-semibold mb-2">Repository Insights</h4>
                <div className="flex flex-wrap gap-2 mb-3">
                  {analysis.setup_complexity <= 2 && (
                    <Badge variant="outline" className={darkMode ? "border-[#2ea043] text-[#2ea043]" : "border-[#1a7f37] text-[#1a7f37]"}>
                      <Check className="w-3 h-3 mr-1 inline" /> Easy Setup
                    </Badge>
                  )}
                  {analysis.contributing_friendliness >= 4 && (
                    <Badge variant="outline" className={darkMode ? "border-[#a371f7] text-[#a371f7]" : "border-[#8250df] text-[#8250df]"}>
                      Good Contributing Guide
                    </Badge>
                  )}
                  {analysis.mentorship_signals && (
                    <Badge variant="outline" className={darkMode ? "border-[#58a6ff] text-[#58a6ff]" : "border-[#0969da] text-[#0969da]"}>
                      Beginner Friendly
                    </Badge>
                  )}
                </div>
                <div className={`text-sm space-y-1 ${darkMode ? "text-[#8b949e]" : "text-[#656d76]"}`}>
                  <p><strong>Tech Stack:</strong> {analysis.tech_stack.join(", ") || "N/A"}</p>
                  <p><strong>Prerequisites:</strong> {analysis.prerequisites.join(", ") || "N/A"}</p>
                </div>
              </div>

              <div>
                <h4 className="font-semibold mb-2">How to Tackle</h4>
                <ul className="space-y-2">
                  {analysis.tackle_plan
                    .filter((step: string) => step.trim().length > 2)
                    .map((step: string, i: number) => (
                      <li key={i} className={`text-sm flex items-start gap-2 ${darkMode ? "text-[#c9d1d9]" : "text-[#24292f]"}`}>
                        <span className={`flex-shrink-0 flex items-center justify-center w-5 h-5 rounded-full text-xs font-bold ${darkMode ? "bg-[#30363d] text-[#8b949e]" : "bg-[#f6f8fa] text-[#656d76]"}`}>
                          {i + 1}
                        </span>
                        <span>{step}</span>
                      </li>
                  ))}
                </ul>
              </div>

              <div className="pt-4 border-t flex justify-end gap-2">
                <Button variant="outline" onClick={() => setAnalyzingIssue(null)}>Close</Button>
                <a href={analyzingIssue?.url} target="_blank" rel="noopener noreferrer">
                  <Button>Go to Issue <ExternalLink className="w-4 h-4 ml-2" /></Button>
                </a>
              </div>
            </div>
          ) : analysisError ? (
            <div className="py-8 text-center text-[#f85149]">
              <p className="font-semibold">{analysisError}</p>
            </div>
          ) : (
            <div className="py-8 text-center text-[#f85149]">
              Failed to load analysis.
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
