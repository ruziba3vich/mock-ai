export type User = {
  id: string;
  display_name: string;
  avatar_url?: string;
  email?: string;
  email_verified: boolean;
  roles: string[];
  created_at: string;
};

type UserResponse = { user: User };
type ErrorResponse = { error?: { code?: string; message?: string } };

export class APIError extends Error {
  constructor(public status: number, message: string) {
    super(message);
  }
}

export async function apiRequest<T>(path: string, options: RequestInit = {}): Promise<T> {
  const response = await fetch(path, {
    ...options,
    credentials: "include",
    headers: {
      ...(options.body ? { "Content-Type": "application/json" } : {}),
      ...options.headers
    }
  });
  if (!response.ok) {
    let message = "Something went wrong. Please try again.";
    try {
      const body = await response.json() as ErrorResponse;
      message = body.error?.message || message;
    } catch {
      // The status still gives callers enough information to recover.
    }
    throw new APIError(response.status, message);
  }
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}

export const authAPI = {
  me: async () => (await apiRequest<UserResponse>("/api/v1/me")).user,
  register: async (displayName: string, email: string, password: string) =>
    (await apiRequest<UserResponse>("/api/v1/auth/register", {
      method: "POST",
      body: JSON.stringify({ display_name: displayName, email, password })
    })).user,
  login: async (email: string, password: string) =>
    (await apiRequest<UserResponse>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password })
    })).user,
  telegram: async (initData: string) =>
    (await apiRequest<UserResponse>("/api/v1/auth/telegram", {
      method: "POST",
      body: JSON.stringify({ init_data: initData })
    })).user,
  logout: () => apiRequest<void>("/api/v1/auth/logout", { method: "POST" })
};

declare global {
  interface Window {
    Telegram?: {
      WebApp?: {
        initData: string;
        ready(): void;
        expand(): void;
      };
    };
  }
}
