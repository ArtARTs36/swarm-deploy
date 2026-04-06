const servicesStatusEl = document.getElementById("services-status");
const servicesListEl = document.getElementById("services-list");
const assistantChat = window.createAssistantChat();

function escapeHtml(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll("\"", "&quot;")
    .replaceAll("'", "&#39;");
}

function normalizeRepositoryUrl(value) {
  const raw = String(value || "").trim();
  if (!raw) {
    return "";
  }

  const candidates = [raw];
  if (!raw.includes("://")) {
    candidates.push(`https://${raw}`);
  }

  for (const candidate of candidates) {
    try {
      const parsed = new URL(candidate);
      if (parsed.protocol === "https:" || parsed.protocol === "http:") {
        return parsed.toString();
      }
    } catch (error) {
      // Ignore malformed candidate and continue fallback attempts.
    }
  }

  return "";
}

function resolveRepositoryProvider(repositoryUrl) {
  if (!repositoryUrl) {
    return "";
  }

  try {
    const host = new URL(repositoryUrl).hostname.toLowerCase();
    if (host.includes("github")) {
      return "github";
    }
    if (host.includes("gitlab")) {
      return "gitlab";
    }
    if (host.includes("bitbucket")) {
      return "bitbucket";
    }
  } catch (error) {
    return "";
  }

  return "";
}

function renderRepositoryIcon(provider) {
  if (provider === "github") {
    return `<span class="repo-icon repo-icon-github" aria-hidden="true">GH</span>`;
  }
  if (provider === "gitlab") {
    return `<span class="repo-icon repo-icon-gitlab" aria-hidden="true">GL</span>`;
  }
  if (provider === "bitbucket") {
    return `<span class="repo-icon repo-icon-bitbucket" aria-hidden="true">BB</span>`;
  }
  return `<span class="repo-icon repo-icon-generic" aria-hidden="true">REPO</span>`;
}

function renderImageWithRepository(image, repositoryUrl) {
  const safeImage = escapeHtml(image || "n/a");
  const normalizedRepositoryUrl = normalizeRepositoryUrl(repositoryUrl);
  if (!normalizedRepositoryUrl) {
    return safeImage;
  }

  const provider = resolveRepositoryProvider(normalizedRepositoryUrl);
  const icon = renderRepositoryIcon(provider);

  return `${safeImage} <a class="repo-link" href="${escapeHtml(normalizedRepositoryUrl)}" target="_blank" rel="noopener noreferrer" title="Open repository">${icon}</a>`;
}

function renderStatus(message) {
  servicesStatusEl.textContent = message;
}

function renderWebRoutes(webRoutes) {
  if (!Array.isArray(webRoutes) || webRoutes.length === 0) {
    return `<p class="meta"><strong>urls:</strong> n/a</p>`;
  }

  const items = webRoutes
    .map((route) => {
      const address = route?.address || "n/a";
      const port = route?.port ? ` (port: ${escapeHtml(route.port)})` : "";
      return `<li>${escapeHtml(address)}${port}</li>`;
    })
    .join("");

  return `
    <div class="meta">
      <strong>urls:</strong>
      <ul class="route-list">${items}</ul>
    </div>
  `;
}

function renderServices(services) {
  if (!Array.isArray(services) || services.length === 0) {
    servicesListEl.innerHTML = `
      <article class="service-card">
        <p class="meta">No services captured yet. Trigger a deploy to collect metadata.</p>
      </article>
    `;
    return;
  }

  servicesListEl.innerHTML = services
    .map((service) => {
      const serviceType = service.type || "application";
      return `
        <article class="service-card">
          <div class="service-card-header">
            <h3 class="service-name">${escapeHtml(service.name || "unknown")}</h3>
            <span class="service-type ${escapeHtml(serviceType)}">${escapeHtml(serviceType)}</span>
          </div>
          <p class="meta"><strong>stack:</strong> ${escapeHtml(service.stack || "n/a")}</p>
          <p class="meta"><strong>image:</strong> ${renderImageWithRepository(service.image, service.repository_url)}</p>
          <p class="meta"><strong>description:</strong> ${escapeHtml(service.description || "n/a")}</p>
          ${renderWebRoutes(service.web_routes)}
        </article>
      `;
    })
    .join("");
}

async function refreshServices() {
  renderStatus("Loading services...");
  try {
    const response = await fetch("/api/v1/services");
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }

    const data = await response.json();
    const services = Array.isArray(data.services) ? data.services : [];
    renderStatus(`Total services: ${services.length}`);
    renderServices(services);
  } catch (err) {
    renderStatus(`Failed to load services: ${err.message}`);
    servicesListEl.innerHTML = "";
  }
}

assistantChat.setEnabled(true);

async function refreshAll() {
  await Promise.all([refreshServices()]);
}

refreshAll();
setInterval(refreshAll, 10000);
