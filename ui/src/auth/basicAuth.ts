const basicAuthStorageKey = "swarm_deploy_basic_auth_header";

let basicAuthHeader = loadInitialBasicAuthHeader();

function loadInitialBasicAuthHeader(): string {
  if (typeof window === "undefined") {
    return "";
  }

  return window.sessionStorage.getItem(basicAuthStorageKey) || "";
}

export function getBasicAuthHeader(): string {
  return basicAuthHeader;
}

export function setBasicAuthHeader(header: string): void {
  basicAuthHeader = header.trim();

  if (typeof window === "undefined") {
    return;
  }

  if (basicAuthHeader.length === 0) {
    window.sessionStorage.removeItem(basicAuthStorageKey);
    return;
  }

  window.sessionStorage.setItem(basicAuthStorageKey, basicAuthHeader);
}

export function clearBasicAuthHeader(): void {
  setBasicAuthHeader("");
}
