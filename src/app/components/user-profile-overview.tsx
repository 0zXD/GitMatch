import {
  MapPin, Building2, Link2, Calendar, Users, BookOpen, Code2,
  Hash, Twitter
} from "lucide-react";
import type { UserProfile } from "../types/user-profile";

interface UserProfileOverviewProps {
  profile: UserProfile;
  darkMode: boolean;
}

export function UserProfileOverview({ profile, darkMode }: UserProfileOverviewProps) {
  const sortedLanguages = Object.entries(profile.languages)
    .sort(([, a], [, b]) => b - a);
  const totalLangRepos = sortedLanguages.reduce((sum, [, count]) => sum + count, 0);

  const sortedTopics = Object.entries(profile.topics)
    .sort(([, a], [, b]) => b - a)
    .slice(0, 12);

  const joinedDate = profile.created_at
    ? new Date(profile.created_at).toLocaleDateString("en-US", {
        month: "short",
        year: "numeric",
      })
    : null;

  const languageColors: Record<string, string> = {
    JavaScript: "#f1e05a", TypeScript: "#3178c6", Python: "#3572A5",
    Java: "#b07219", Go: "#00ADD8", Rust: "#dea584", "C++": "#f34b7d",
    "C#": "#178600", C: "#555555", Ruby: "#701516", PHP: "#4F5D95",
    Lua: "#000080", Shell: "#89e051", Nix: "#7e7eff", TeX: "#3D6117",
    "Jupyter Notebook": "#DA5B0B", HTML: "#e34c26", CSS: "#563d7c",
    Haskell: "#5e5086", Scala: "#c22d40", Kotlin: "#A97BFF",
    Swift: "#F05138", Dart: "#00B4AB", Zig: "#ec915c",
  };

  return (
    <div className="space-y-4">
      {/* Profile Card */}
      <div className={`rounded-lg border p-6 ${
        darkMode
          ? "bg-[#161b22] border-[#30363d]"
          : "bg-white border-[#d0d7de]"
      }`}>
        <div className="flex items-start gap-5">
          <img
            src={`https://github.com/${profile.username}.png?size=96`}
            alt={profile.username}
            className="w-20 h-20 rounded-full border-2 border-[#30363d]"
            onError={(e) => { (e.target as HTMLImageElement).style.display = "none"; }}
          />
          <div className="flex-1 min-w-0">
            <h2 className={`text-xl font-bold leading-tight ${
              darkMode ? "text-[#e6edf3]" : "text-[#24292f]"
            }`}>
              {profile.name || profile.username}
            </h2>
            <p className={`text-sm ${
              darkMode ? "text-[#8b949e]" : "text-[#656d76]"
            }`}>
              {profile.username}
            </p>
            {profile.bio && (
              <p className={`mt-2 text-sm leading-relaxed ${
                darkMode ? "text-[#e6edf3]" : "text-[#24292f]"
              }`}>
                {profile.bio}
              </p>
            )}
            {/* Meta row */}
            <div className={`flex flex-wrap items-center gap-x-4 gap-y-1 mt-3 text-xs ${
              darkMode ? "text-[#8b949e]" : "text-[#656d76]"
            }`}>
              {profile.company && (
                <span className="flex items-center gap-1">
                  <Building2 className="w-3.5 h-3.5" /> {profile.company}
                </span>
              )}
              {profile.location && (
                <span className="flex items-center gap-1">
                  <MapPin className="w-3.5 h-3.5" /> {profile.location}
                </span>
              )}
              {profile.blog && (
                <a
                  href={profile.blog.startsWith("http") ? profile.blog : `https://${profile.blog}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className={`flex items-center gap-1 hover:underline ${
                    darkMode ? "text-[#58a6ff]" : "text-[#0969da]"
                  }`}
                >
                  <Link2 className="w-3.5 h-3.5" /> {profile.blog}
                </a>
              )}
              {profile.twitter && (
                <span className="flex items-center gap-1">
                  <Twitter className="w-3.5 h-3.5" /> @{profile.twitter}
                </span>
              )}
              {joinedDate && (
                <span className="flex items-center gap-1">
                  <Calendar className="w-3.5 h-3.5" /> Joined {joinedDate}
                </span>
              )}
            </div>
          </div>
        </div>

        {/* Stats counters */}
        <div className={`flex items-center gap-6 mt-5 pt-4 border-t text-sm ${
          darkMode ? "border-[#30363d] text-[#e6edf3]" : "border-[#d0d7de] text-[#24292f]"
        }`}>
          <div className="flex items-center gap-1.5">
            <Users className={`w-4 h-4 ${darkMode ? "text-[#8b949e]" : "text-[#656d76]"}`} />
            <span className="font-semibold">{profile.followers}</span>
            <span className={darkMode ? "text-[#8b949e]" : "text-[#656d76]"}>followers</span>
          </div>
          <div className="flex items-center gap-1.5">
            <span className="font-semibold">{profile.following}</span>
            <span className={darkMode ? "text-[#8b949e]" : "text-[#656d76]"}>following</span>
          </div>
          <div className="flex items-center gap-1.5">
            <BookOpen className={`w-4 h-4 ${darkMode ? "text-[#8b949e]" : "text-[#656d76]"}`} />
            <span className="font-semibold">{profile.public_repos}</span>
            <span className={darkMode ? "text-[#8b949e]" : "text-[#656d76]"}>repositories</span>
          </div>
        </div>
      </div>

      {/* Languages Card */}
      {sortedLanguages.length > 0 && (
        <div className={`rounded-lg border p-5 ${
          darkMode
            ? "bg-[#161b22] border-[#30363d]"
            : "bg-white border-[#d0d7de]"
        }`}>
          <h3 className={`flex items-center gap-2 text-sm font-semibold mb-4 ${
            darkMode ? "text-[#e6edf3]" : "text-[#24292f]"
          }`}>
            <Code2 className="w-4 h-4" /> Languages
          </h3>

          {/* Language bar */}
          <div className="flex h-2 rounded-full overflow-hidden mb-3">
            {sortedLanguages.map(([lang, count]) => (
              <div
                key={lang}
                style={{
                  width: `${(count / totalLangRepos) * 100}%`,
                  backgroundColor: languageColors[lang] || (darkMode ? "#8b949e" : "#656d76"),
                }}
                title={`${lang}: ${count} repos`}
              />
            ))}
          </div>

          {/* Legend */}
          <div className="flex flex-wrap gap-x-4 gap-y-1.5 text-xs">
            {sortedLanguages.map(([lang, count]) => (
              <span key={lang} className="flex items-center gap-1.5">
                <span
                  className="w-2.5 h-2.5 rounded-full inline-block"
                  style={{ backgroundColor: languageColors[lang] || (darkMode ? "#8b949e" : "#656d76") }}
                />
                <span className={darkMode ? "text-[#e6edf3]" : "text-[#24292f]"}>
                  {lang}
                </span>
                <span className={darkMode ? "text-[#8b949e]" : "text-[#656d76]"}>
                  {((count / totalLangRepos) * 100).toFixed(1)}%
                </span>
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Topics Card */}
      {sortedTopics.length > 0 && (
        <div className={`rounded-lg border p-5 ${
          darkMode
            ? "bg-[#161b22] border-[#30363d]"
            : "bg-white border-[#d0d7de]"
        }`}>
          <h3 className={`flex items-center gap-2 text-sm font-semibold mb-3 ${
            darkMode ? "text-[#e6edf3]" : "text-[#24292f]"
          }`}>
            <Hash className="w-4 h-4" /> Top Topics
          </h3>
          <div className="flex flex-wrap gap-2">
            {sortedTopics.map(([topic, count]) => (
              <span
                key={topic}
                className={`text-xs px-2.5 py-1 rounded-full font-medium ${
                  darkMode
                    ? "bg-[#388bfd1a] text-[#58a6ff] border border-[#30363d]"
                    : "bg-[#ddf4ff] text-[#0969da] border border-[#54aeff66]"
                }`}
              >
                {topic}
                <span className={`ml-1 ${darkMode ? "text-[#8b949e]" : "text-[#656d76]"}`}>
                  {count}
                </span>
              </span>
            ))}
          </div>
        </div>
      )}

      {/* CTA */}
      <div className={`rounded-lg border p-6 text-center ${
        darkMode
          ? "bg-[#161b22] border-[#30363d]"
          : "bg-white border-[#d0d7de]"
      }`}>
        <p className={`text-sm ${darkMode ? "text-[#8b949e]" : "text-[#656d76]"}`}>
          Select languages or skills above to discover open source projects that match your expertise
        </p>
      </div>
    </div>
  );
}
