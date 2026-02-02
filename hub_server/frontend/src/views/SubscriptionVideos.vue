<template>
  <div class="min-h-screen bg-bg p-8 lg:p-12 font-sans text-text">
    <header class="flex justify-between items-start mb-12">
      <div class="flex items-center gap-6">
          <button class="w-12 h-12 rounded-full bg-bg shadow-neu-btn flex items-center justify-center text-text hover:text-primary active:shadow-neu-btn-active transition-all" @click="goBack">
            ←
          </button>
          <div class="flex items-center gap-4" v-if="subscription">
             <div class="w-16 h-16 rounded-full bg-bg shadow-neu-sm p-1">
                <img :src="subscription.headUrl || placeholderImg" class="w-full h-full rounded-full object-cover" @error="onImgError">
             </div>
             <div>
                <h2 class="font-serif font-bold text-2xl text-text mb-1">{{ subscription.nickname }} 的订阅视频</h2>
                <p class="text-text-muted text-sm">共 {{ totalVideos }} 个视频</p>
             </div>
          </div>
      </div>
    </header>

    <div class="max-w-6xl mx-auto">
        <!-- Loading State -->
        <div v-if="loading && videos.length === 0" class="flex justify-center p-12">
          <div class="text-text-muted">加载中...</div>
        </div>

        <!-- Empty State -->
        <div v-else-if="videos.length === 0" class="text-center p-12">
          <p class="text-text-muted text-lg">暂无视频</p>
        </div>

        <!-- Video Grid -->
        <div v-else class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-6">
            <div 
                v-for="video in videos" 
                :key="video.id"
                @click="playVideo(video)"
                class="bg-bg rounded-2xl overflow-hidden shadow-neu border border-white/40 cursor-pointer transition-all hover:-translate-y-1 hover:shadow-neu-sm hover:border-primary/30 group">
                
                <!-- Cover -->
                <div class="relative aspect-[9/16] w-full">
                    <img :src="video.cover_url || placeholderImg" class="w-full h-full object-cover" @error="onImgError">
                    <!-- Play Icon Overlay -->
                    <div class="absolute inset-0 bg-black/20 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center">
                        <div class="w-12 h-12 rounded-full bg-white/90 flex items-center justify-center">
                            <svg class="w-6 h-6 text-primary ml-1" fill="currentColor" viewBox="0 0 24 24">
                                <path d="M8 5v14l11-7z"/>
                            </svg>
                        </div>
                    </div>
                    <!-- Duration Badge -->
                    <div v-if="video.duration" class="absolute bottom-2 right-2 px-2 py-1 rounded-lg bg-black/70 text-white text-xs">
                        {{ formatDuration(video.duration) }}
                    </div>
                </div>

                <!-- Info -->
                <div class="p-4">
                    <h3 class="font-semibold text-sm text-text line-clamp-2 mb-2 group-hover:text-primary transition-colors">
                        {{ video.title || '无标题' }}
                    </h3>
                    <div class="flex items-center gap-3 text-xs text-text-muted">
                        <span class="flex items-center gap-1">
                            <svg class="w-3 h-3" fill="currentColor" viewBox="0 0 24 24">
                                <path d="M12 21.35l-1.45-1.32C5.4 15.36 2 12.28 2 8.5 2 5.42 4.42 3 7.5 3c1.74 0 3.41.81 4.5 2.09C13.09 3.81 14.76 3 16.5 3 19.58 3 22 5.42 22 8.5c0 3.78-3.4 6.86-8.55 11.54L12 21.35z"/>
                            </svg>
                            {{ formatCount(video.like_count) }}
                        </span>
                        <span>{{ formatDate(video.published_at) }}</span>
                    </div>
                </div>
            </div>
        </div>

        <!-- Load More Button -->
        <div v-if="hasMore && !loading" class="text-center mt-12 mb-8">
            <button 
                @click="loadMoreVideos"
                class="px-8 py-3 rounded-xl bg-bg shadow-neu-btn text-text font-semibold hover:text-primary active:shadow-neu-btn-active transition-all">
                加载更多
            </button>
        </div>

        <div v-if="loading && videos.length > 0" class="flex justify-center p-8">
          <div class="text-text-muted">加载中...</div>
        </div>
    </div>

    <!-- Video Player Modal -->
    <div v-if="playerUrl" class="fixed inset-0 z-50 flex justify-center items-center bg-black/60 backdrop-blur-md p-4" @click="closePlayer">
      <div class="w-full max-w-5xl bg-bg rounded-3xl shadow-neu border border-white/50 p-4" @click.stop>
        <div class="flex justify-between items-center mb-4 px-2">
          <h3 class="font-serif font-bold text-xl text-text">{{ currentVideoTitle }}</h3>
          <button class="w-10 h-10 rounded-full bg-bg shadow-neu-btn flex items-center justify-center text-text hover:text-red-500 active:shadow-neu-btn-active transition-all" @click="closePlayer">×</button>
        </div>
        <div class="rounded-2xl overflow-hidden shadow-inner bg-black aspect-video">
           <video :src="playerUrl" controls autoplay class="w-full h-full"></video>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'

const router = useRouter()
const route = useRoute()

const subscription = ref(null)
const videos = ref([])
const loading = ref(false)
const currentPage = ref(1)
const totalVideos = ref(0)
const hasMore = ref(false)

const playerUrl = ref('')
const currentVideoTitle = ref('')
const placeholderImg = 'https://via.placeholder.com/100'

onMounted(() => {
  const subscriptionId = route.params.id
  subscription.value = {
    id: subscriptionId,
    nickname: route.query.nickname || '未知用户',
    headUrl: route.query.headUrl || ''
  }
  loadVideos()
})

const loadVideos = async () => {
  loading.value = true
  try {
    const token = localStorage.getItem('token')
    const res = await fetch(`/api/subscriptions/${subscription.value.id}/videos?page=${currentPage.value}`, {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    const data = await res.json()
    if (data.code === 0) {
      videos.value.push(...data.data.videos)
      totalVideos.value = data.data.total
      const totalPages = Math.ceil(data.data.total / 20) // pageSize = 20
      hasMore.value = currentPage.value < totalPages
    }
  } catch (e) {
    console.error('Failed to load videos:', e)
    alert('加载视频失败')
  } finally {
    loading.value = false
  }
}

const loadMoreVideos = () => {
  currentPage.value++
  loadVideos()
}

const playVideo = async (video) => {
  try {
    currentVideoTitle.value = video.title || '无标题'
    
    console.log('[PlayVideo] Video object:', video)
    
    // Call feed_profile to get full video details with decrypt key (like UserProfile.vue)
    const token = localStorage.getItem('token')
    const profileRes = await fetch('/api/remoteCall', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify({
        action: 'api_call',
        data: {
          key: 'key:channels:feed_profile',
          body: {
            object_id: video.object_id,
            nonce_id: video.object_nonce_id
          }
        }
      })
    })
    
    const profileData = await profileRes.json()
    console.log('[PlayVideo] Profile response:', profileData)
    
    // Extract actual video object from response - handle ResponsePayload.Data
    let actualVideo = {}
    
    // ResponsePayload has Data field which is json.RawMessage, need to parse it
    if (profileData.Data) {
      try {
        const parsedData = typeof profileData.Data === 'string' 
          ? JSON.parse(profileData.Data) 
          : profileData.Data
        console.log('[PlayVideo] Parsed Data:', parsedData)
        
        if (parsedData.data && parsedData.data.object) {
          actualVideo = parsedData.data.object
        } else if (parsedData.object) {
          actualVideo = parsedData.object
        }
      } catch (e) {
        console.error('[PlayVideo] Failed to parse Data:', e)
      }
    }
    
    // Fallback to old logic if above didn't work
    if (!actualVideo.objectDesc && !actualVideo.desc) {
      if (profileData.data && profileData.data.object) {
        actualVideo = profileData.data.object
      } else if (profileData.data && profileData.data.data && profileData.data.data.object) {
        actualVideo = profileData.data.data.object
      } else {
        actualVideo = profileData.data || {}
      }
    }
    
    console.log('[PlayVideo] Actual video:', actualVideo)
    
    // Extract media list
    const desc = actualVideo.objectDesc || actualVideo.desc || {}
    const mediaList = desc.media || []
    
    console.log('[PlayVideo] Media list:', mediaList)
    
    if (mediaList.length === 0) {
      alert('无法获取视频播放地址')
      return
    }
    
    // Use lowest quality (first item) for faster loading
    const media = mediaList[0]
    
    // Build complete video URL with urlToken (contains token, sign, etc.)
    let videoUrl = media.url + (media.urlToken || '')
    const decryptKey = media.decodeKey || media.decryptKey || ''
    
    console.log('[PlayVideo] Base URL:', media.url)
    console.log('[PlayVideo] URL Token:', media.urlToken)
    
    // Add video spec if available (for format selection)
    if (media.spec && media.spec.length > 0) {
      const lowestSpec = media.spec.reduce((prev, curr) => {
        return (curr.bitRate || 99999) < (prev.bitRate || 99999) ? curr : prev
      })
      if (lowestSpec.fileFormat) {
        videoUrl += `&X-snsvideoflag=${lowestSpec.fileFormat}`
        console.log('[PlayVideo] Added file format:', lowestSpec.fileFormat)
      }
    }
    
    console.log('[PlayVideo] Complete Video URL:', videoUrl)
    console.log('[PlayVideo] Decrypt key:', decryptKey)
    
    if (!videoUrl) {
      alert('视频地址无效')
      return
    }
    
    // Build proxy URL with decrypt key
    let finalUrl = `/api/video/play?url=${encodeURIComponent(videoUrl)}`
    if (decryptKey) {
      finalUrl += `&key=${decryptKey}`
    }
    
    console.log('[PlayVideo] Final URL:', finalUrl)
    playerUrl.value = finalUrl
  } catch (e) {
    alert('播放失败: ' + e.message)
    console.error('Video playback error:', e)
  }
}

const closePlayer = () => {
  playerUrl.value = ''
  currentVideoTitle.value = ''
}

const goBack = () => {
  router.push('/subscriptions')
}

const formatDuration = (seconds) => {
  const mins = Math.floor(seconds / 60)
  const secs = seconds % 60
  return `${mins}:${secs.toString().padStart(2, '0')}`
}

const formatCount = (count) => {
  if (count >= 10000) {
    return (count / 10000).toFixed(1) + '万'
  }
  return count || 0
}

const formatDate = (dateStr) => {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  const now = new Date()
  const diff = now - date
  const days = Math.floor(diff / 86400000)
  
  if (days === 0) return '今天'
  if (days === 1) return '昨天'
  if (days < 7) return `${days}天前`
  if (days < 30) return `${Math.floor(days / 7)}周前`
  if (days < 365) return `${Math.floor(days / 30)}个月前`
  return date.toLocaleDateString('zh-CN')
}

const onImgError = (e) => {
  e.target.src = placeholderImg
}
</script>
