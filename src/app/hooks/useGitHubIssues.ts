import { useState, useCallback, useRef, useEffect } from "react";
import { GitHubIssue } from "../components/issue-card";
import { fetchIssuesForSkills } from "../services/harvester";

interface UseGitHubIssuesProps {
  selectedSkills: string[];
  level: "beginner" | "intermediate" | "advanced";
  count: number;
}

export function useGitHubIssues({ selectedSkills, level, count }: UseGitHubIssuesProps) {
  const [apiIssues, setApiIssues] = useState<GitHubIssue[]>([]);
  const [loading, setLoading] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [apiError, setApiError] = useState<string | null>(null);
  const [apiPage, setApiPage] = useState(1);
  const [hasMorePages, setHasMorePages] = useState(false);
  const [apiCursor, setApiCursor] = useState<string | undefined>();
  const abortRef = useRef<AbortController | null>(null);

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

    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;

    const debounce = setTimeout(async () => {
      setLoading(true);
      setApiError(null);
      try {
        const result = await fetchIssuesForSkills(selectedSkills, level, count, controller.signal, 1);
        if (!controller.signal.aborted) {
          setApiIssues(result.issues);
          setApiPage(1);
          setHasMorePages(result.hasMore);
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
  }, [selectedSkills, level, count]);

  const loadMoreIssues = useCallback(async () => {
    if (loadingMore || !hasMorePages) return;
    const nextPage = apiPage + 1;
    setLoadingMore(true);
    try {
      const result = await fetchIssuesForSkills(selectedSkills, level, count, undefined, nextPage, apiCursor);
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
  }, [loadingMore, hasMorePages, apiPage, selectedSkills, apiIssues, apiCursor, level, count]);

  return {
    apiIssues,
    loading,
    loadingMore,
    apiError,
    hasMorePages,
    loadMoreIssues,
    setApiIssues
  };
}
