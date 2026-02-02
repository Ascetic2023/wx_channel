<template>
  <div class="min-h-screen bg-bg p-8 lg:p-12 font-sans text-text">
    <header class="flex justify-between items-center mb-12">
      <h1 class="font-serif font-bold text-4xl text-text">我的订阅</h1>
      <button 
          @click="updateAllSubscriptions" 
          :disabled="updatingAll"
          class="px-6 py-3 rounded-xl bg-primary text-white font-semibold shadow-neu-btn hover:bg-primary-dark active:shadow-neu-btn-active transition-all disabled:opacity-50">
        {{ updatingAll ? '更新中...' : '一键更新全部' }}
      </button>
    </header>

    <!-- Loading State -->
    <div v-if="loading" class="flex justify-center p-12">
      <div class="text-text-muted">加载中...</div>
    </div>

    <!-- Empty State -->
    <div v-else-if="subscriptions.length === 0" class="text-center p-12">
      <p class="text-text-muted text-lg mb-4">暂无订阅</p>
      <button @click="$router.push('/search')" class="px-6 py-3 rounded-xl bg-primary text-white font-semibold shadow-neu-btn hover:bg-primary-dark">
        去搜索用户
      </button>
    </div>

    <!-- Subscription List -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      <div 
          v-for="sub in subscriptions" 
          :key="sub.id"
          class="bg-bg rounded-2xl p-6 shadow-neu border border-white/40 hover:-translate-y-1 hover:shadow-neu-sm transition-all">
        
        <!-- User Info -->
        <div class="flex items-center gap-4 mb-4">
          <div class="w-16 h-16 rounded-full bg-bg shadow-neu-sm p-1">
            <img :src="sub.wx_head_url || placeholderImg" class="w-full h-full rounded-full object-cover" @error="onImgError">
          </div>
          <div class="flex-1">
            <h3 class="font-bold text-lg text-text mb-1">{{ sub.wx_nickname }}</h3>
            <p class="text-xs text-text-muted line-clamp-2">{{ sub.wx_signature || '暂无签名' }}</p>
          </div>
        </div>

        <!-- Stats -->
        <div class="flex items-center justify-between mb-4 text-sm">
          <div class="text-text-muted">
            <span class="font-semibold text-text">{{ sub.video_count }}</span> 个视频
          </div>
          <div class="text-text-muted text-xs">
            {{ formatDate(sub.last_fetched_at) }}
          </div>
        </div>

        <!-- Actions -->
        <div class="flex gap-2">
          <button 
              @click="viewVideos(sub)" 
              class="flex-1 px-4 py-2 rounded-xl bg-primary text-white font-semibold shadow-neu-btn hover:bg-primary-dark active:shadow-neu-btn-active transition-all">
            查看视频
          </button>
          <button 
              @click="fetchVideos(sub)" 
              :disabled="updating[sub.id]"
              class="px-4 py-2 rounded-xl bg-bg text-text shadow-neu-btn hover:text-primary active:shadow-neu-btn-active transition-all disabled:opacity-50">
            {{ updating[sub.id] ? '更新中' : '更新' }}
          </button>
          <button 
              @click="unsubscribe(sub)" 
              class="px-4 py-2 rounded-xl bg-bg text-red-500 shadow-neu-btn hover:bg-red-50 active:shadow-neu-btn-active transition-all">
            取消
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const subscriptions = ref([])
const loading = ref(false)
const updatingAll = ref(false)
const updating = ref({})
const placeholderImg = 'https://via.placeholder.com/100'

onMounted(() => {
  loadSubscriptions()
})

const loadSubscriptions = async () => {
  loading.value = true
  try {
    const token = localStorage.getItem('token')
    const res = await fetch('/api/subscriptions', {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    const data = await res.json()
    if (data.code === 0) {
      subscriptions.value = data.data || []
    }
  } catch (e) {
    console.error('Failed to load subscriptions:', e)
    alert('加载订阅失败')
  } finally {
    loading.value = false
  }
}

const fetchVideos = async (sub) => {
  updating.value[sub.id] = true
  try {
    const token = localStorage.getItem('token')
    const res = await fetch(`/api/subscriptions/${sub.id}/fetch`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${token}` }
    })
    const data = await res.json()
    if (data.code === 0) {
      alert(`成功获取 ${data.data.new_videos} 个新视频！\n总视频数: ${data.data.total_videos}`)
      loadSubscriptions() // Reload to update counts
    } else {
      alert('更新失败')
    }
  } catch (e) {
    console.error('Failed to fetch videos:', e)
    alert('更新失败: ' + e.message)
  } finally {
    updating.value[sub.id] = false
  }
}

const updateAllSubscriptions = async () => {
  if (!confirm(`确定要更新所有 ${subscriptions.value.length} 个订阅吗？`)) return
  
  updatingAll.value = true
  let totalNew = 0
  
  for (const sub of subscriptions.value) {
    try {
      const token = localStorage.getItem('token')
      const res = await fetch(`/api/subscriptions/${sub.id}/fetch`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` }
      })
      const data = await res.json()
      if (data.code === 0) {
        totalNew += data.data.new_videos
      }
    } catch (e) {
      console.error(`Failed to update ${sub.wx_nickname}:`, e)
    }
  }
  
  updatingAll.value = false
  alert(`全部更新完成！共获取 ${totalNew} 个新视频`)
  loadSubscriptions()
}

const viewVideos = (sub) => {
  router.push({
    name: 'SubscriptionVideos',
    params: { id: sub.id },
    query: {
      nickname: sub.wx_nickname,
      headUrl: sub.wx_head_url
    }
  })
}

const unsubscribe = async (sub) => {
  if (!confirm(`确定要取消订阅 ${sub.wx_nickname} 吗？`)) return
  
  try {
    const token = localStorage.getItem('token')
    const res = await fetch(`/api/subscriptions/${sub.id}`, {
      method: 'DELETE',
      headers: { 'Authorization': `Bearer ${token}` }
    })
    
    if (res.ok) {
      subscriptions.value = subscriptions.value.filter(s => s.id !== sub.id)
    } else {
      alert('取消订阅失败')
    }
  } catch (e) {
    console.error('Failed to unsubscribe:', e)
    alert('操作失败: ' + e.message)
  }
}

const formatDate = (dateStr) => {
  if (!dateStr || dateStr === '0001-01-01T00:00:00Z') return '未更新'
  const date = new Date(dateStr)
  const now = new Date()
  const diff = now - date
  const minutes = Math.floor(diff / 60000)
  const hours = Math.floor(diff / 3600000)
  const days = Math.floor(diff / 86400000)
  
  if (minutes < 1) return '刚刚更新'
  if (minutes < 60) return `${minutes}分钟前`
  if (hours < 24) return `${hours}小时前`
  if (days < 7) return `${days}天前`
  return date.toLocaleDateString('zh-CN')
}

const onImgError = (e) => {
  e.target.src = placeholderImg
}
</script>
