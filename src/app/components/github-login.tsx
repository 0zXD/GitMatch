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

  const handleOAuthLogin = () => {
    setLoading(true);
    // Redirects the browser to the backend OAuth initialization route
    window.location.href = "http://localhost:8084/auth/github/login";
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
            Find open source issues that match your skills. Please sign in with GitHub
            to get started.
          </p>
        </div>

        {/* OAuth Form Card */}
        <div className="bg-[#161b22] border border-[#30363d] rounded-xl p-6 flex flex-col items-center gap-4">
          <Button
            onClick={handleOAuthLogin}
            disabled={loading}
            className="w-full bg-[#238636] hover:bg-[#2ea043] text-white h-11 text-sm font-semibold"
          >
            {loading ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                Connecting to GitHub…
              </>
            ) : (
              <>
                <Github className="w-5 h-5 mr-2" />
                Sign in with GitHub
              </>
            )}
          </Button>
        </div>

        <p className="text-center text-xs text-[#484f58]">
          This authorizes GitMatch to find issues using your account rate limits.
        </p>
      </div>
    </div>
  );
}
