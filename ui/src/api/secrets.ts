import { apiRequest } from "./client";
import type { SecretsResponse } from "./types";

export function fetchSecrets(): Promise<SecretsResponse> {
  return apiRequest<SecretsResponse>("/api/v1/secrets");
}
