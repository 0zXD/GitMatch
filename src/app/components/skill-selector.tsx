import { X, User, Globe } from "lucide-react";

interface SkillSelectorProps {
  selectedSkills: string[];
  onSkillToggle: (skill: string) => void;
  darkMode: boolean;
  userLanguages?: string[];
}

const availableSkills = [
  "JavaScript", "TypeScript", "React", "Vue", "Angular",
  "Python", "Django", "FastAPI", "Java", "Spring Boot",
  "Go", "Rust", "C++", "C#", ".NET",
  "Ruby", "Rails", "PHP", "Laravel", "Node.js",
  "Docker", "Kubernetes", "AWS", "GraphQL", "REST API",
  "MongoDB", "PostgreSQL", "MySQL", "Redis", "HTML/CSS"
];

function SkillButton({
  skill,
  isSelected,
  darkMode,
  onToggle,
}: {
  skill: string;
  isSelected: boolean;
  darkMode: boolean;
  onToggle: () => void;
}) {
  return (
    <button
      onClick={onToggle}
      className={`px-3 py-1.5 text-sm font-medium rounded-full border transition-all ${
        isSelected
          ? darkMode
            ? "bg-[#1f6feb] text-white border-[#1f6feb] hover:bg-[#1a5cd7]"
            : "bg-[#0969da] text-white border-[#0969da] hover:bg-[#0860ca]"
          : darkMode
          ? "bg-[#161b22] text-[#e6edf3] border-[#30363d] hover:bg-[#21262d] hover:border-[#8b949e]"
          : "bg-white text-gray-700 border-gray-300 hover:bg-gray-50 hover:border-gray-400"
      }`}
    >
      {skill}
      {isSelected && <X className="inline ml-1.5 h-3 w-3" />}
    </button>
  );
}

export function SkillSelector({ selectedSkills, onSkillToggle, darkMode, userLanguages = [] }: SkillSelectorProps) {
  // Languages the user actually uses (from their GitHub repos)
  const userLangs = userLanguages.length > 0 ? userLanguages : [];

  // Other skills: everything in availableSkills that isn't already in the user's languages
  const userLangsLower = new Set(userLangs.map((l) => l.toLowerCase()));
  const otherSkills = availableSkills.filter(
    (skill) => !userLangsLower.has(skill.toLowerCase())
  );

  return (
    <div className="space-y-5">
      {/* User's Languages Section */}
      {userLangs.length > 0 && (
        <div className="space-y-2.5">
          <div className="flex items-center gap-2">
            <User className={`w-3.5 h-3.5 ${darkMode ? "text-[#58a6ff]" : "text-[#0969da]"}`} />
            <span className={`text-xs font-semibold uppercase tracking-wide ${
              darkMode ? "text-[#8b949e]" : "text-[#656d76]"
            }`}>
              Your Languages
            </span>
          </div>
          <div className="flex flex-wrap gap-2">
            {userLangs.map((skill) => (
              <SkillButton
                key={skill}
                skill={skill}
                isSelected={selectedSkills.includes(skill)}
                darkMode={darkMode}
                onToggle={() => onSkillToggle(skill)}
              />
            ))}
          </div>
        </div>
      )}

      {/* Other Languages & Fields Section */}
      <div className="space-y-2.5">
        <div className="flex items-center gap-2">
          <Globe className={`w-3.5 h-3.5 ${darkMode ? "text-[#8b949e]" : "text-[#656d76]"}`} />
          <span className={`text-xs font-semibold uppercase tracking-wide ${
            darkMode ? "text-[#8b949e]" : "text-[#656d76]"
          }`}>
            Other Languages & Fields
          </span>
        </div>
        <div className="flex flex-wrap gap-2">
          {otherSkills.map((skill) => (
            <SkillButton
              key={skill}
              skill={skill}
              isSelected={selectedSkills.includes(skill)}
              darkMode={darkMode}
              onToggle={() => onSkillToggle(skill)}
            />
          ))}
        </div>
      </div>
    </div>
  );
}