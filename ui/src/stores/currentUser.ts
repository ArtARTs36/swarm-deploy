import { defineStore } from "pinia";

import { fetchCurrentUser } from "../api/users";

interface CurrentUserState {
  displayName: string;
  loading: boolean;
  loaded: boolean;
}

export const useCurrentUserStore = defineStore("currentUser", {
  state: (): CurrentUserState => ({
    displayName: "",
    loading: false,
    loaded: false,
  }),
  actions: {
    clearCurrentUser() {
      this.displayName = "";
      this.loaded = false;
      this.loading = false;
    },
    async loadCurrentUser(force = false): Promise<boolean> {
      if (this.loading || (this.loaded && !force)) {
        return this.displayName.trim().length > 0;
      }

      this.loading = true;
      try {
        const response = await fetchCurrentUser();
        this.displayName = response.name.trim();
        return this.displayName.length > 0;
      } catch {
        this.displayName = "";
        return false;
      } finally {
        this.loading = false;
        this.loaded = true;
      }
    },
  },
});
