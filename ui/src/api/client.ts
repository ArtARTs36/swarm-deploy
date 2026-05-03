interface ApiErrorPayload {
  error_message?: string;
}

import { getBasicAuthHeader } from "../auth/basicAuth";

export class ApiError extends Error {
  readonly status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

async function readErrorMessage(response: Response): Promise<string> {
  try {
    const payload = (await response.json()) as ApiErrorPayload;
    if (payload.error_message) {
      return payload.error_message;
    }
  } catch {
    // Keep default fallback.
  }

  return `HTTP ${response.status}`;
}

export async function apiRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers);
  const basicAuthHeader = getBasicAuthHeader();
  if (basicAuthHeader && !headers.has("Authorization")) {
    headers.set("Authorization", basicAuthHeader);
  }

  const response = await fetch(path, {
    ...init,
    headers,
  });
  if (!response.ok) {
    const errorMessage = await readErrorMessage(response);
    throw new ApiError(errorMessage, response.status);
  }

  return (await response.json()) as T;
}
