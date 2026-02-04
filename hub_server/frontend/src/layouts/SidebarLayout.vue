<template>
  <div class="min-h-screen w-full bg-bg flex font-sans">
    <!-- 侧边栏 -->
    <Sidebar />
    
    <!-- 主内容区域 -->
    <div class="flex-1 flex flex-col overflow-hidden">
      <!-- 顶部栏（可选，用于显示用户信息等） -->
      <header class="bg-bg shadow-neu-sm px-8 py-4 flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold text-slate-800">{{ pageTitle }}</h1>
          <p class="text-sm text-slate-500">{{ pageDescription }}</p>
        </div>
        
        <!-- 用户信息 -->
        <div class="flex items-center gap-4">
          <div class="text-right">
            <p class="text-sm font-medium text-slate-800">{{ user?.username || 'Guest' }}</p>
            <p class="text-xs text-slate-500">{{ user?.email || '' }}</p>
          </div>
          <div class="w-10 h-10 rounded-xl bg-bg shadow-neu flex items-center justify-center text-primary font-bold">
            {{ userInitial }}
          </div>
        </div>
      </header>

      <!-- 主内容 -->
      <main class="flex-1 overflow-y-auto">
        <slot></slot>
      </main>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import Sidebar from '../components/Sidebar.vue'
import { useUserStore } from '../store/user'

const route = useRoute()
const userStore = useUserStore()

const user = computed(() => userStore.user)

const userInitial = computed(() => {
  if (!user.value?.username) return 'G'
  return user.value.username.charAt(0).toUpperCase()
})

const pageTitle = computed(() => {
  const titles = {
    '/dashboard': '在线终端',
    '/search': '穿透搜索',
    '/subscriptions': '订阅管理',
    '/devices': '设备管理',
    '/tasks': '任务追踪',
    '/monitoring': '系统监控',
    '/profile': '个人资料',
    '/settings': '系统设置',
    '/admin': '系统管理'
  }
  // 处理动态路由
  if (route.path.includes('/subscriptions/') && route.path.includes('/videos')) {
    return '订阅视频'
  }
  return titles[route.path] || '控制面板'
})

const pageDescription = computed(() => {
  const descriptions = {
    '/dashboard': '查看所有在线的客户端终端',
    '/search': '远程搜索视频号内容',
    '/subscriptions': '管理您的视频号订阅',
    '/devices': '管理您绑定的所有设备',
    '/tasks': '查看和管理任务执行状态',
    '/monitoring': '实时监控系统运行状态',
    '/profile': '查看和编辑个人信息',
    '/settings': '配置系统参数和选项',
    '/admin': '管理用户和系统资源'
  }
  // 处理动态路由
  if (route.path.includes('/subscriptions/') && route.path.includes('/videos')) {
    return '查看订阅的视频内容'
  }
  return descriptions[route.path] || '欢迎使用 Hub Control'
})
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
