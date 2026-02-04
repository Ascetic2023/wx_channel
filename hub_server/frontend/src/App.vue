<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import MainLayout from './layouts/MainLayout.vue'
import AuthLayout from './layouts/AuthLayout.vue'
import SidebarLayout from './layouts/SidebarLayout.vue'

const route = useRoute()
const layout = computed(() => {
  if (route.meta.layout === 'Auth') return AuthLayout
  if (route.meta.layout === 'Sidebar') return SidebarLayout
  return MainLayout
})
</script>

<template>
  <component :is="layout">
    <router-view />
  </component>
</template>
