import { useMemo } from "react";
import type { UserProfile } from "../types/user-profile";

export function useUserExperience(userProfile: UserProfile | null, selectedSkills: string[]) {
  // Derive sorted user languages from profile (most-used first)
  const userLanguages = useMemo(() => {
    if (!userProfile?.languages) return [];
    return Object.entries(userProfile.languages)
      .sort(([, a], [, b]) => b - a)
      .map(([lang]) => lang);
  }, [userProfile]);

  const userExperienceData = useMemo(() => {
    if (!userProfile?.languages || selectedSkills.length === 0) return { level: "beginner" as const, count: 0 };
    
    // Find the max repo count among the relevant languages
    let maxRepos = 0;
    
    const profileLookup = new Map<string, number>();
    Object.entries(userProfile.languages).forEach(([lang, count]) => {
      profileLookup.set(lang.toLowerCase(), count);
    });

    for (const skill of selectedSkills) {
      const lowerSkill = skill.toLowerCase();
      const count = profileLookup.get(lowerSkill) || 0;
      if (count > maxRepos) {
        maxRepos = count;
      }
    }

    if (maxRepos <= 10) return { level: "beginner" as const, count: maxRepos };
    if (maxRepos >= 10) return { level: "intermediate" as const, count: maxRepos };
    return { level: "advanced" as const, count: maxRepos };
  }, [userProfile, selectedSkills]);

  return { userLanguages, userExperienceData };
}
