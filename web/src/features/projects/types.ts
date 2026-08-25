export interface Project {
  id: number;
  name: string;
  user_id: number;
  namespace: string;
  created_at: string;
  updated_at: string;
}

export interface CreateProjectInput {
  name: string;
}
