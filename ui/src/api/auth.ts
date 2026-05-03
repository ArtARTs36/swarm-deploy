import { apiRequest } from "./client";
import type { AuthMethodsResponse } from "./types";

export interface PasskeyLoginBeginResponse {
  publicKey?: unknown;
}

export function fetchAuthMethods(): Promise<AuthMethodsResponse> {
  return apiRequest<AuthMethodsResponse>("/api/v1/auth/methods");
}

export function beginPasskeyLogin(username: string): Promise<PasskeyLoginBeginResponse> {
  return apiRequest<PasskeyLoginBeginResponse>("/api/passkey/loginBegin", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ username }),
  });
}

export function finishPasskeyLogin(payload: unknown): Promise<unknown> {
  return apiRequest<unknown>("/api/passkey/loginFinish", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });
}
