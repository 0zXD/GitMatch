import { GitHubIssue } from "./issue-card";
import { ExternalLink, GitFork, Star, MessageSquare, Trash2 } from "lucide-react";
import { Button } from "./ui/button";

interface SavedIssuesProps {
  issues: GitHubIssue[];
  onRemove: (id: number) => void;
  darkMode: boolean;
}

export function SavedIssues({ issues, onRemove, darkMode }: SavedIssuesProps) {
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
                <h3 className={`text-base font-semibold hover:underline cursor-pointer leading-tight mb-2 ${
                  darkMode ? "text-[#58a6ff]" : "text-[#0969da]"
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
    </div>
  );
}
