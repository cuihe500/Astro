export interface App {
  id: number;
  name: string;
  image: string;
  replicas: number;
  status: string;
  user_id: number;
  namespace: string;
  created_at: string;
  updated_at: string;
}

export interface CreateAppInput {
  name: string;
  image: string;
  replicas: number;
  port: number;
}

export type LifecycleAction = "start" | "stop" | "restart";
