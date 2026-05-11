<template>
  <div id="app">
    <!-- Navbar shown on all pages except auth -->
    <AppNavbar v-if="!isAuthPage" />

    <!-- Page content -->
    <main :class="{ 'with-navbar': !isAuthPage }">
      <router-view />
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import AppNavbar from '@/components/ui/AppNavbar.vue'

const route = useRoute()

// Hide navbar on the auth page
const isAuthPage = computed(() => route.name === 'auth')
</script>

<style>
/* Global styles */
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
  background: #ffffff;
  color: #111111;
}

#app {
  min-height: 100vh;
}

.with-navbar {
  padding-top: 0;
}

a {
  text-decoration: none;
  color: inherit;
}
</style>