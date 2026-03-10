import { useState } from "react";
import { Github, ArrowRight, Loader2 } from "lucide-react";
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import type { UserProfile } from "../types/user-profile";

interface GitHubLoginProps {
  onSubmit: (profile: UserProfile) => void;
}

export function GitHubLogin({ onSubmit }: GitHubLoginProps) {
  const [username, setUsername] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const trimmed = username.trim();
    if (!trimmed) {
      setError("Please enter your GitHub username");
      return;
    }
    if (!/^[a-zA-Z0-9](?:[a-zA-Z0-9]|-(?=[a-zA-Z0-9])){0,38}$/.test(trimmed)) {
      setError("Invalid GitHub username format");
      return;
    }
    setError("");
    setLoading(true);

    try {
      const res = await fetch(
        `http://localhost:8084/user?username=${encodeURIComponent(trimmed)}`
      );
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || `Failed to fetch user (${res.status})`);
      }
      const profile: UserProfile = await res.json();
      onSubmit(profile);
    } catch (err: any) {
      setError(err.message ?? "Could not reach the server. Is the backend running?");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-[#0d1117] flex items-center justify-center px-4">
      <div className="w-full max-w-md space-y-8">
        {/* Logo */}
        <div className="flex flex-col items-center gap-4">
          <Github className="w-16 h-16 text-white" />
          <h1 className="text-3xl font-bold text-white tracking-tight">
            Welcome to GitMatch
          </h1>
          <p className="text-[#8b949e] text-center text-sm leading-relaxed max-w-sm">
            Find open source issues that match your skills. Enter your GitHub
            username to get started.
          </p>
        </div>

        {/* Form Card */}
        <div className="bg-[#161b22] border border-[#30363d] rounded-xl p-6">
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <label
                htmlFor="github-username"
                className="block text-sm font-medium text-[#e6edf3]"
              >
                GitHub Username
              </label>
              <div className="relative">
                <span className="absolute left-3 top-1/2 -translate-y-1/2 text-[#656d76] text-sm select-none">
                  github.com/
                </span>
                <Input
                  id="github-username"
                  type="text"
                  value={username}
                  onChange={(e) => {
                    setUsername(e.target.value);
                    if (error) setError("");
                  }}
                  placeholder="username"
                  autoFocus
                  autoComplete="off"
                  spellCheck={false}
                  disabled={loading}
                  className="pl-[6.5rem] bg-[#0d1117] border-[#30363d] text-white placeholder:text-[#484f58] focus:border-[#58a6ff] focus:ring-[#58a6ff]/40 h-11"
                />
              </div>
              {error && (
                <p className="text-sm text-[#f85149]">{error}</p>
              )}
            </div>

            <Button
              type="submit"
              disabled={loading}
              className="w-full bg-[#238636] hover:bg-[#2ea043] text-white h-11 text-sm font-semibold"
            >
              {loading ? (
                <>
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  Fetching profile…
                </>
              ) : (
                <>
                  Continue
                  <ArrowRight className="w-4 h-4 ml-2" />
                </>
              )}
            </Button>
          </form>
        </div>

        <p className="text-center text-xs text-[#484f58]">
          Your username is stored locally and never shared.
        </p>
      </div>
    </div>
  );
}
