(function initAssistantAvailability() {
  const cacheKey = "assistant-enabled-cache-v1";
  const updatedAtKey = "assistant-enabled-cache-updated-at-v1";
  let inFlightRequest = null;

  function readCache(maxAgeMs) {
    try {
      const rawValue = localStorage.getItem(cacheKey);
      const rawUpdatedAt = localStorage.getItem(updatedAtKey);
      if (rawValue !== "true" && rawValue !== "false") {
        return null;
      }

      const updatedAt = Number(rawUpdatedAt);
      if (!Number.isFinite(updatedAt)) {
        return null;
      }

      if (Date.now() - updatedAt > maxAgeMs) {
        return null;
      }

      return rawValue === "true";
    } catch (error) {
      return null;
    }
  }

  function writeCache(enabled) {
    try {
      localStorage.setItem(cacheKey, enabled ? "true" : "false");
      localStorage.setItem(updatedAtKey, String(Date.now()));
    } catch (error) {
      // Ignore storage quota/privacy mode errors.
    }
  }

  async function fetchFromAPI() {
    if (inFlightRequest) {
      return inFlightRequest;
    }

    inFlightRequest = (async () => {
      const response = await fetch("/api/v1/stacks");
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }

      const data = await response.json();
      const enabled = Boolean(data.sync && data.sync.assistant_enabled === "true");
      writeCache(enabled);
      return enabled;
    })();

    try {
      return await inFlightRequest;
    } finally {
      inFlightRequest = null;
    }
  }

  async function getAssistantEnabled(options = {}) {
    const maxAgeMs = Number(options.maxAgeMs) > 0 ? Number(options.maxAgeMs) : 30000;
    const forceRefresh = Boolean(options.forceRefresh);
    if (!forceRefresh) {
      const cached = readCache(maxAgeMs);
      if (cached !== null) {
        return cached;
      }
    }

    return fetchFromAPI();
  }

  function setAssistantEnabledCache(enabled) {
    writeCache(Boolean(enabled));
  }

  window.getAssistantEnabled = getAssistantEnabled;
  window.setAssistantEnabledCache = setAssistantEnabledCache;
}());
