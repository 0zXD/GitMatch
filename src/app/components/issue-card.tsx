import { Badge } from "./ui/badge";
import { ExternalLink, GitFork, Star, MessageSquare, AlertCircle, Check, X } from "lucide-react";
import { Button } from "./ui/button";

export interface RepoAnalysis {
  setup_complexity: number;
  contributing_friendliness: number;
  tech_stack: string[];
  prerequisites: string[];
  mentorship_signals: boolean;
}

export interface IssueAnalysis {
  setup_complexity: number;
  contributing_friendliness: number;
  tech_stack: string[];
  prerequisites: string[];
  mentorship_signals: boolean;
  issue_debrief: string;
  recommendation?: string;
  tackle_plan: string[];
}

export interface GitHubIssue {
  number: number;
  id: number;
  title: string;
  repository: string;
  description: string;
  labels: string[];
  language: string;
  stars: number;
  forks: number;
  comments: number;
  difficulty: "beginner" | "intermediate" | "advanced";
  url: string;
  openIssues?: number;
  languageTags?: string[];
  repoAnalysis?: RepoAnalysis;
}

interface IssueCardProps {
  issue: GitHubIssue;
  onInterested: () => void;
  onSkip: () => void;
  darkMode: boolean;
}

export function IssueCard({ issue, onInterested, onSkip, darkMode }: IssueCardProps) {
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

  return (
    <div className={`rounded-lg border transition-all ${
      darkMode
        ? "bg-[#161b22] border-[#30363d] hover:border-[#58a6ff]"
        : "bg-white border-[#d0d7de] hover:border-[#0969da]"
    }`}>
      {/* Issue Header */}
      <div className={`p-4 border-b ${
        darkMode ? "border-[#30363d]" : "border-[#d0d7de]"
      }`}>
        <div className="flex items-start justify-between gap-3 mb-2">
          <div className="flex-1">
            <div className={`flex items-center gap-2 text-xs mb-1 ${
              darkMode ? "text-[#8b949e]" : "text-[#656d76]"
            }`}>
              <span className="font-semibold">{issue.repository}</span>
            </div>
            <h3 className={`text-lg font-semibold hover:underline cursor-pointer leading-tight ${
              darkMode ? "text-[#58a6ff]" : "text-[#0969da]"
            }`}>
              {issue.title}
            </h3>
          </div>
          <Badge className={`${difficultyColors[issue.difficulty]} text-xs px-2 py-0.5`}>
            {issue.difficulty}
          </Badge>
        </div>
      </div>

      {/* Issue Body */}
      <div className="p-4 space-y-4">
        {/* Repo Analysis Highlights */}
        {issue.repoAnalysis && (
          <div className="flex flex-wrap gap-2 mb-2">
            {issue.repoAnalysis.setup_complexity <= 2 && (
              <Badge variant="outline" className={`text-xs ${darkMode ? "border-[#2ea043] text-[#2ea043]" : "border-[#1a7f37] text-[#1a7f37]"}`}>
                <Check className="w-3 h-3 mr-1 inline" /> Easy Setup
              </Badge>
            )}
            {issue.repoAnalysis.contributing_friendliness >= 4 && (
              <Badge variant="outline" className={`text-xs ${darkMode ? "border-[#a371f7] text-[#a371f7]" : "border-[#8250df] text-[#8250df]"}`}>
                Has Contributing Guide
              </Badge>
            )}
            {issue.repoAnalysis.mentorship_signals && (
              <Badge variant="outline" className={`text-xs ${darkMode ? "border-[#58a6ff] text-[#58a6ff]" : "border-[#0969da] text-[#0969da]"}`}>
                Beginner Friendly Repo
              </Badge>
            )}
          </div>
        )}

        <p className={`text-sm leading-relaxed ${
          darkMode ? "text-[#e6edf3]" : "text-[#24292f]"
        }`}>
          {issue.description}
        </p>

        {/* Labels */}
        <div className="flex flex-wrap gap-2">
          {issue.labels.map((label, idx) => (
            <span
              key={label}
              className={`${labelColors[idx % labelColors.length]} text-xs px-2.5 py-0.5 rounded-full font-medium`}
            >
              {label}
            </span>
          ))}
        </div>

        {/* Language Combination Tags */}
        {issue.languageTags && issue.languageTags.length > 1 && (
          <div className="flex flex-wrap items-center gap-1.5">
            {issue.languageTags.map((tag) => (
              <span
                key={tag}
                className={`text-xs px-2 py-0.5 rounded border font-medium ${
                  darkMode
                    ? "bg-[#0d1117] text-[#e6edf3] border-[#30363d]"
                    : "bg-[#f6f8fa] text-[#24292f] border-[#d0d7de]"
                }`}
              >
                {tag}
              </span>
            ))}
          </div>
        )}

        {/* Repository Stats */}
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
          {issue.forks > 0 && (
            <div className="flex items-center gap-1">
              <GitFork className="w-3.5 h-3.5" />
              <span>{issue.forks.toLocaleString()}</span>
            </div>
          )}
          {issue.comments > 0 && (
            <div className="flex items-center gap-1">
              <MessageSquare className="w-3.5 h-3.5" />
              <span>{issue.comments}</span>
            </div>
          )}
          {issue.openIssues !== undefined && issue.openIssues > 0 && (
            <div className="flex items-center gap-1">
              <AlertCircle className="w-3.5 h-3.5" />
              <span>{issue.openIssues} open</span>
            </div>
          )}
        </div>

        {/* Action Buttons */}
        <div className="flex gap-2 pt-2">
          <Button
            onClick={onInterested}
            className={darkMode
              ? "flex-1 bg-[#238636] hover:bg-[#2ea043] text-white"
              : "flex-1 bg-[#1a7f37] hover:bg-[#1a7f37]/90 text-white"
            }
          >
            <Check className="w-4 h-4 mr-2" />
            I'm Interested
          </Button>
          <Button
            onClick={onSkip}
            variant="outline"
            className={darkMode
              ? "flex-1 border-[#30363d] text-[#e6edf3] hover:bg-[#21262d]"
              : "flex-1 border-[#d0d7de] text-[#24292f] hover:bg-[#f6f8fa]"
            }
          >
            <X className="w-4 h-4 mr-2" />
            Skip
          </Button>
        </div>

        {/* View on GitHub */}
        <a
          href={issue.url}
          target="_blank"
          rel="noopener noreferrer"
          className={`flex items-center justify-center gap-2 hover:underline text-sm font-medium ${
            darkMode ? "text-[#58a6ff]" : "text-[#0969da]"
          }`}
          onClick={(e) => e.stopPropagation()}
        >
          View on GitHub <ExternalLink className="w-4 h-4" />
        </a>
      </div>
    </div>
  );
}
