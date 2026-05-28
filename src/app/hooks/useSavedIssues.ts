import { useState, useCallback } from "react";
import { GitHubIssue } from "../components/issue-card";
import type { UserProfile } from "../types/user-profile";

export function useSavedIssues(userProfile: UserProfile | null) {
  const [savedIssues, setSavedIssues] = useState<GitHubIssue[]>([]);

  const loadSavedIssues = useCallback(async (username: string) => {
    try {
      const res = await fetch(`http://localhost:8084/saved_issues?username=${encodeURIComponent(username)}`);
      if (res.ok) {
        const issues = await res.json();
        setSavedIssues(issues);
      }
    } catch (err) {
      console.error("Failed to fetch saved issues", err);
    }
  }, []);

  const handleInterested = async (issue: GitHubIssue) => {
    if (!userProfile) return;
    setSavedIssues((prev) => [...prev, issue]);

    try {
      const res = await fetch("http://localhost:8084/saved_issues", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          username: userProfile.username,
          issue_id: issue.id,
          issue_data: issue,
        }),
      });
      if (!res.ok) {
        setSavedIssues((prev) => prev.filter((i) => i.id !== issue.id));
      }
    } catch (err) {
      console.error("Failed to save issue to db", err);
      setSavedIssues((prev) => prev.filter((i) => i.id !== issue.id));
    }
  };

  const handleRemoveSaved = async (id: number) => {
    if (!userProfile) return;
    const removedIssue = savedIssues.find((i) => i.id === id);
    setSavedIssues((prev) => prev.filter((issue) => issue.id !== id));

    try {
      const res = await fetch(
        `http://localhost:8084/saved_issues?username=${encodeURIComponent(userProfile.username)}&issue_id=${id}`,
        { method: "DELETE" }
      );
      if (!res.ok && removedIssue) {
        setSavedIssues((prev) => [...prev, removedIssue]);
      }
    } catch (err) {
      console.error("Failed to delete issue from db", err);
      if (removedIssue) {
        setSavedIssues((prev) => [...prev, removedIssue]);
      }
    }
  };

  return { savedIssues, setSavedIssues, loadSavedIssues, handleInterested, handleRemoveSaved };
}
