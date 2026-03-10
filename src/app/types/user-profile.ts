export interface UserProfile {
  name: string;
  username: string;
  bio: string;
  location: string;
  company: string;
  twitter: string;
  blog: string;
  public_repos: number;
  followers: number;
  following: number;
  created_at: string;
  languages: Record<string, number>;
  topics: Record<string, number>;
}
