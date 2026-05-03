import { startAuthentication } from "@simplewebauthn/browser";
import { defineStore } from "pinia";

import { beginPasskeyLogin, fetchAuthMethods, finishPasskeyLogin } from "../api/auth";
import { clearBasicAuthHeader, setBasicAuthHeader } from "../auth/basicAuth";
import { useCurrentUserStore } from "./currentUser";

interface AuthState {
  methodsLoaded: boolean;
  basicEnabled: boolean;
  passkeyEnabled: boolean;
  checkingSession: boolean;
  sessionChecked: boolean;
  authenticated: boolean;
}

function encodeBase64(value: string): string {
  const bytes = new TextEncoder().encode(value);
  let binary = "";
  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }

  return window.btoa(binary);
}

export const useAuthStore = defineStore("auth", {
  state: (): AuthState => ({
    methodsLoaded: false,
    basicEnabled: false,
    passkeyEnabled: false,
    checkingSession: false,
    sessionChecked: false,
    authenticated: false,
  }),
  getters: {
    hasAnyAuthenticator: (state): boolean => state.basicEnabled || state.passkeyEnabled,
  },
  actions: {
    async loadMethods(force = false) {
      if (this.methodsLoaded && !force) {
        return;
      }

      const methods = await fetchAuthMethods();
      this.basicEnabled = methods.basic_enabled;
      this.passkeyEnabled = methods.passkey_enabled;
      this.methodsLoaded = true;
    },
    async resolveSession(force = false): Promise<boolean> {
      if (this.checkingSession || (this.sessionChecked && !force)) {
        return this.authenticated;
      }

      this.checkingSession = true;
      const currentUserStore = useCurrentUserStore();

      try {
        const authenticated = await currentUserStore.loadCurrentUser(force);
        this.authenticated = authenticated;
        this.sessionChecked = true;
        if (!authenticated) {
          currentUserStore.clearCurrentUser();
        }
        return authenticated;
      } finally {
        this.checkingSession = false;
      }
    },
    async loginWithBasic(username: string, password: string): Promise<boolean> {
      const value = `${username}:${password}`;
      setBasicAuthHeader(`Basic ${encodeBase64(value)}`);

      const authenticated = await this.resolveSession(true);
      if (!authenticated) {
        clearBasicAuthHeader();
      }

      return authenticated;
    },
    async loginWithPasskey(username: string): Promise<boolean> {
      const optionsResponse = await beginPasskeyLogin(username);
      const options = optionsResponse.publicKey ?? optionsResponse;
      const authentication = await startAuthentication({
        optionsJSON: options as Parameters<typeof startAuthentication>[0]["optionsJSON"],
      });

      await finishPasskeyLogin(authentication);
      return this.resolveSession(true);
    },
    logoutLocal() {
      clearBasicAuthHeader();
      this.authenticated = false;
      this.sessionChecked = false;
      useCurrentUserStore().clearCurrentUser();
    },
  },
});
