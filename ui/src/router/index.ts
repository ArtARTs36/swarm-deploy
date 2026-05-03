import { createRouter, createWebHistory } from "vue-router";
import type { Pinia } from "pinia";

import AppShell from "../components/layout/AppShell.vue";
import { useAuthStore } from "../stores/auth";
import ServicesView from "../views/ApplicationsView.vue";
import ClusterView from "../views/ClusterView.vue";
import LoginView from "../views/LoginView.vue";
import NetworksView from "../views/NetworksView.vue";
import OverviewView from "../views/OverviewView.vue";
import SecretsView from "../views/SecretsView.vue";
import ServiceView from "../views/ServiceView.vue";

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: "/",
      component: AppShell,
      children: [
        {
          path: "",
          redirect: "/overview",
        },
        {
          path: "/overview",
          name: "overview",
          component: OverviewView,
        },
        {
          path: "/services",
          name: "services",
          component: ServicesView,
        },
        {
          path: "/services/:stack/:service",
          name: "service-details",
          component: ServiceView,
        },
        {
          path: "/cluster",
          name: "cluster",
          component: ClusterView,
        },
        {
          path: "/networks",
          name: "networks",
          component: NetworksView,
        },
        {
          path: "/secrets",
          name: "secrets",
          component: SecretsView,
        },
      ],
    },
    {
      path: "/login",
      name: "login",
      component: LoginView,
    },
  ],
});

export function installRouterGuards(pinia: Pinia) {
  router.beforeEach(async (to) => {
    const authStore = useAuthStore(pinia);
    await authStore.loadMethods();

    if (!authStore.hasAnyAuthenticator) {
      if (to.path === "/login") {
        return { path: "/overview", replace: true };
      }
      return true;
    }

    if (to.path === "/login") {
      const authenticated = await authStore.resolveSession();
      if (authenticated) {
        return { path: "/overview", replace: true };
      }

      return true;
    }

    const authenticated = await authStore.resolveSession();
    if (authenticated) {
      return true;
    }

    return { path: "/login", query: { redirect: to.fullPath } };
  });
}
