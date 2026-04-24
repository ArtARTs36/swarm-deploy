import { apiRequest } from "./client";
import type {
  EventCategory,
  EventHistoryResponse,
  EventSeverity,
  QueueResponse,
  ServiceDeploymentsResponse,
  ServiceStatusResponse,
  StacksResponse,
} from "./types";

export function fetchStacks(): Promise<StacksResponse> {
  return apiRequest<StacksResponse>("/api/v1/stacks");
}

export function triggerSync(): Promise<QueueResponse> {
  return apiRequest<QueueResponse>("/api/v1/sync", {
    method: "POST",
  });
}

export interface FetchEventsParams {
  severities?: EventSeverity[];
  categories?: EventCategory[];
}

export function fetchEvents(params?: FetchEventsParams): Promise<EventHistoryResponse> {
  const searchParams = new URLSearchParams();
  for (const severity of params?.severities ?? []) {
    searchParams.append("severities", severity);
  }
  for (const category of params?.categories ?? []) {
    searchParams.append("categories", category);
  }

  const query = searchParams.toString();
  const path = query === "" ? "/api/v1/events" : `/api/v1/events?${query}`;
  return apiRequest<EventHistoryResponse>(path);
}

export function fetchServiceStatus(stackName: string, serviceName: string): Promise<ServiceStatusResponse> {
  const encodedStack = encodeURIComponent(stackName);
  const encodedService = encodeURIComponent(serviceName);
  return apiRequest<ServiceStatusResponse>(`/api/v1/stacks/${encodedStack}/services/${encodedService}/status`);
}

export function fetchServiceDeployments(
  stackName: string,
  serviceName: string,
  limit?: number,
): Promise<ServiceDeploymentsResponse> {
  const encodedStack = encodeURIComponent(stackName);
  const encodedService = encodeURIComponent(serviceName);
  const query = typeof limit === "number" ? `?limit=${encodeURIComponent(String(limit))}` : "";
  return apiRequest<ServiceDeploymentsResponse>(
    `/api/v1/stacks/${encodedStack}/services/${encodedService}/deployments${query}`,
  );
}
