<template>
  <nav class="w-full bg-white shadow-neu-sm z-50 sticky top-0 px-6 md:px-12 py-4 flex items-center justify-between shrink-0">
    <!-- Brand -->
    <div class="flex items-center gap-3">
      <div class="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center text-primary shrink-0">
        <svg viewBox="0 0 24 24" width="22" height="22" fill="currentColor">
          <path d="M12 2L4.5 20.29l.71.71L12 18l6.79 3 .71-.71z" />
        </svg>
      </div>
      <span class="font-serif text-xl font-bold text-slate-800 tracking-tight">Hub Control</span>
    </div>

    <!-- Navigation Links -->
    <div class="flex items-center gap-1 md:gap-4">
      <router-link to="/dashboard" active-class="!text-primary !bg-primary/5" class="px-4 py-2 rounded-lg text-slate-600 font-medium text-sm transition-all hover:text-primary hover:bg-slate-50">
        概览
      </router-link>
      <router-link to="/search" active-class="!text-primary !bg-primary/5" class="px-4 py-2 rounded-lg text-slate-600 font-medium text-sm transition-all hover:text-primary hover:bg-slate-50">
        搜索
      </router-link>
      <router-link to="/subscriptions" active-class="!text-primary !bg-primary/5" class="px-4 py-2 rounded-lg text-slate-600 font-medium text-sm transition-all hover:text-primary hover:bg-slate-50">
        订阅
      </router-link>
      <router-link to="/tasks" active-class="!text-primary !bg-primary/5" class="px-4 py-2 rounded-lg text-slate-600 font-medium text-sm transition-all hover:text-primary hover:bg-slate-50">
        任务
      </router-link>
      <router-link to="/monitoring" active-class="!text-primary !bg-primary/5" class="px-4 py-2 rounded-lg text-slate-600 font-medium text-sm transition-all hover:text-primary hover:bg-slate-50">
        监控
      </router-link>
      <router-link to="/settings" active-class="!text-primary !bg-primary/5" class="px-4 py-2 rounded-lg text-slate-600 font-medium text-sm transition-all hover:text-primary hover:bg-slate-50">
        设置
      </router-link>
      <router-link v-if="userStore.user?.role === 'admin'" to="/admin" active-class="!text-primary !bg-primary/5" class="px-4 py-2 rounded-lg text-slate-600 font-medium text-sm transition-all hover:text-primary hover:bg-slate-50">
        管理
      </router-link>
    </div>

    <!-- User Info & Credits -->
    <div class="flex items-center gap-4">
        <!-- Credits -->
        <div class="hidden md:flex items-center gap-2 px-3 py-1.5 rounded-xl bg-amber-50 border border-amber-100 text-amber-700">
             <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
             <span class="font-bold font-mono">{{ userStore.user?.credits || 0 }}</span>
             <span class="text-xs font-medium opacity-80">积分</span>
        </div>

        <!-- User Dropdown (Simplified) -->
        <div class="flex items-center gap-3 pl-3 border-l border-slate-200">
             <div class="hidden md:block text-right">
                 <div class="text-xs font-bold text-slate-700">{{ userStore.user?.email }}</div>
                 <div class="text-[10px] text-slate-400 uppercase tracking-wider">{{ userStore.user?.role || 'User' }}</div>
             </div>
             <button @click="handleLogout" class="p-2 rounded-lg hover:bg-red-50 text-slate-400 hover:text-red-500 transition-colors" title="退出登录">
                 <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" /></svg>
             </button>
        </div>
    </div>
  </nav>
</template>

<script setup>
import { useUserStore } from '../store/user'
import { useRouter } from 'vue-router'

const userStore = useUserStore()
const router = useRouter()

const handleLogout = () => {
    userStore.logout()
    router.push('/login')
}
</script>

<style scoped>
/* Hide scrollbar for cleaner look */
.no-scrollbar::-webkit-scrollbar {
  display: none;
}
.no-scrollbar {
  -ms-overflow-style: none;
  scrollbar-width: none;
}
</style>
