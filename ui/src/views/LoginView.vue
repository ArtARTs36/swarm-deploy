<script setup lang="ts">
import { computed, ref } from "vue";
import { useRoute, useRouter } from "vue-router";

import { useAuthStore } from "../stores/auth";

const authStore = useAuthStore();
const route = useRoute();
const router = useRouter();

const username = ref("");
const password = ref("");
const basicPending = ref(false);
const passkeyPending = ref(false);
const errorMessage = ref("");

const hasBasic = computed(() => authStore.basicEnabled);
const hasPasskey = computed(() => authStore.passkeyEnabled);
const usernameRequired = computed(() => hasBasic.value || hasPasskey.value);

const redirectPath = computed(() => {
  const redirect = String(route.query.redirect || "").trim();
  if (!redirect.startsWith("/") || redirect.startsWith("//")) {
    return "/overview";
  }

  return redirect;
});

async function redirectAfterSuccess() {
  await router.replace(redirectPath.value);
}

async function submitBasicAuth() {
  if (!hasBasic.value || basicPending.value) {
    return;
  }

  const loginValue = username.value.trim();
  if (!loginValue || !password.value) {
    errorMessage.value = "Fill login and password.";
    return;
  }

  errorMessage.value = "";
  basicPending.value = true;

  try {
    const authenticated = await authStore.loginWithBasic(loginValue, password.value);
    if (!authenticated) {
      errorMessage.value = "Invalid login or password.";
      return;
    }

    await redirectAfterSuccess();
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : "Sign in failed.";
  } finally {
    basicPending.value = false;
  }
}

async function submitPasskeyAuth() {
  if (!hasPasskey.value || passkeyPending.value) {
    return;
  }

  const loginValue = username.value.trim();
  if (!loginValue) {
    errorMessage.value = "Fill login.";
    return;
  }

  errorMessage.value = "";
  passkeyPending.value = true;

  try {
    const authenticated = await authStore.loginWithPasskey(loginValue);
    if (!authenticated) {
      errorMessage.value = "Passkey authentication failed.";
      return;
    }

    await redirectAfterSuccess();
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : "Passkey authentication failed.";
  } finally {
    passkeyPending.value = false;
  }
}

function onSubmit(event: Event) {
  event.preventDefault();
  void submitBasicAuth();
}
</script>

<template>
  <div class="login-page">
    <section class="login-panel">
      <h1 class="login-title">Swarm Deploy</h1>
      <p class="login-subtitle">Sign in</p>

      <form class="login-form" @submit="onSubmit">
        <label v-if="usernameRequired" class="login-field">
          <span>Login</span>
          <input v-model="username" type="text" autocomplete="username webauthn" />
        </label>

        <label v-if="hasBasic" class="login-field">
          <span>Password</span>
          <input v-model="password" type="password" autocomplete="current-password" />
        </label>

        <p v-if="errorMessage" class="login-error">{{ errorMessage }}</p>

        <button v-if="hasBasic" type="submit" :disabled="basicPending || passkeyPending">
          {{ basicPending ? "Signing in..." : "Sign in" }}
        </button>

        <button v-if="hasPasskey" type="button" :disabled="passkeyPending || basicPending" @click="submitPasskeyAuth">
          {{ passkeyPending ? "Checking..." : "Passkey" }}
        </button>
      </form>
    </section>
  </div>
</template>
