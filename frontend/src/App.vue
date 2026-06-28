<script setup lang="ts">
import { RouterLink, RouterView } from 'vue-router'
</script>

<template>
  <a-layout style="min-height: 100vh">
    <a-layout-header style="display: flex; align-items: center">
      <router-link to="/" style="color: #fff; font-size: 18px; font-weight: 600; text-decoration: none">
        HTTP Bench
      </router-link>
    </a-layout-header>
    <a-layout-content style="padding: 24px">
      <div style="max-width: 1200px; margin: 0 auto; background: #fff; padding: 24px; border-radius: 8px">
        <RouterView />
      </div>
    </a-layout-content>
  </a-layout>
</template>
