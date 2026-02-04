<template>
  <aside class="w-64 bg-bg flex flex-col shrink-0 h-full shadow-neu z-50">
    <div class="px-6 py-8 flex items-center gap-3">
      <div class="w-10 h-10 rounded-xl bg-bg shadow-neu-sm flex items-center justify-center text-primary">
        <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
          <path d="M12 2L4.5 20.29l.71.71L12 18l6.79 3 .71-.71z" />
        </svg>
      </div>
      <h2 class="font-serif text-xl font-bold text-slate-800 tracking-tight">Hub Control</h2>
    </div>

    <nav class="flex-1 px-4 py-2 overflow-y-auto">
      <div class="text-xs font-bold text-slate-400 uppercase tracking-widest px-4 mb-2 mt-4 font-sans">Core</div>
      <router-link to="/dashboard" active-class="bg-bg shadow-neu-pressed text-primary !text-primary" class="flex items-center gap-3 px-4 py-3 mb-2 rounded-xl text-slate-500 font-medium transition-all hover:bg-bg hover:shadow-neu-sm hover:text-primary active:shadow-neu-pressed">
        <component :is="LayoutDashboard" class="w-5 h-5" />
        <span>在线终端</span>
      </router-link>
      <router-link to="/search" active-class="bg-bg shadow-neu-pressed text-primary !text-primary" class="flex items-center gap-3 px-4 py-3 mb-2 rounded-xl text-slate-500 font-medium transition-all hover:bg-bg hover:shadow-neu-sm hover:text-primary active:shadow-neu-pressed">
        <component :is="Globe" class="w-5 h-5" />
        <span>穿透搜索</span>
      </router-link>
      
      <div class="text-xs font-bold text-slate-400 uppercase tracking-widest px-4 mb-2 mt-6 font-sans">Content</div>
      <router-link to="/subscriptions" active-class="bg-bg shadow-neu-pressed text-primary !text-primary" class="flex items-center gap-3 px-4 py-3 mb-2 rounded-xl text-slate-500 font-medium transition-all hover:bg-bg hover:shadow-neu-sm hover:text-primary active:shadow-neu-pressed">
        <component :is="Rss" class="w-5 h-5" />
        <span>订阅管理</span>
      </router-link>
      
      <div class="text-xs font-bold text-slate-400 uppercase tracking-widest px-4 mb-2 mt-6 font-sans">Management</div>
      <router-link to="/devices" active-class="bg-bg shadow-neu-pressed text-primary !text-primary" class="flex items-center gap-3 px-4 py-3 mb-2 rounded-xl text-slate-500 font-medium transition-all hover:bg-bg hover:shadow-neu-sm hover:text-primary active:shadow-neu-pressed">
        <component :is="Monitor" class="w-5 h-5" />
        <span>设备管理</span>
      </router-link>
      <router-link to="/tasks" active-class="bg-bg shadow-neu-pressed text-primary !text-primary" class="flex items-center gap-3 px-4 py-3 mb-2 rounded-xl text-slate-500 font-medium transition-all hover:bg-bg hover:shadow-neu-sm hover:text-primary active:shadow-neu-pressed">
        <component :is="ListTodo" class="w-5 h-5" />
        <span>任务追踪</span>
      </router-link>
      <router-link to="/monitoring" active-class="bg-bg shadow-neu-pressed text-primary !text-primary" class="flex items-center gap-3 px-4 py-3 mb-2 rounded-xl text-slate-500 font-medium transition-all hover:bg-bg hover:shadow-neu-sm hover:text-primary active:shadow-neu-pressed">
        <component :is="Activity" class="w-5 h-5" />
        <span>系统监控</span>
      </router-link>
      
      <div class="text-xs font-bold text-slate-400 uppercase tracking-widest px-4 mb-2 mt-6 font-sans">Settings</div>
      <router-link to="/profile" active-class="bg-bg shadow-neu-pressed text-primary !text-primary" class="flex items-center gap-3 px-4 py-3 mb-2 rounded-xl text-slate-500 font-medium transition-all hover:bg-bg hover:shadow-neu-sm hover:text-primary active:shadow-neu-pressed">
        <component :is="User" class="w-5 h-5" />
        <span>个人资料</span>
      </router-link>
      <router-link to="/settings" active-class="bg-bg shadow-neu-pressed text-primary !text-primary" class="flex items-center gap-3 px-4 py-3 mb-2 rounded-xl text-slate-500 font-medium transition-all hover:bg-bg hover:shadow-neu-sm hover:text-primary active:shadow-neu-pressed">
        <component :is="Settings" class="w-5 h-5" />
        <span>系统设置</span>
      </router-link>
    </nav>

    <div class="p-6 border-t border-slate-200">
      <div class="flex justify-between items-center bg-bg shadow-neu-pressed rounded-xl p-4">
        <span class="text-xs font-bold text-slate-400 uppercase">Status</span>
        <span class="text-green-500 font-bold text-xs flex items-center gap-1">
            <span class="w-2 h-2 bg-green-500 rounded-full animate-pulse"></span>
            Online
        </span>
      </div>
    </div>
  </aside>
</template>

<script setup>
import { LayoutDashboard, Globe, ListTodo, Activity, Settings, Monitor, Rss, User } from 'lucide-vue-next'
</script>
