import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '../store/user'

const router = createRouter({
    history: createWebHistory(),
    routes: [
        {
            path: '/login',
            name: 'Login',
            component: () => import('../views/Login.vue'),
            meta: { layout: 'Auth' }
        },
        {
            path: '/register',
            name: 'Register',
            component: () => import('../views/Register.vue'),
            meta: { layout: 'Auth' }
        },
        {
            path: '/',
            redirect: '/dashboard'
        },
        {
            path: '/dashboard',
            name: 'Dashboard',
            component: () => import('../views/Dashboard.vue'),
            meta: { requiresAuth: true }
        },
        {
            path: '/search',
            name: 'Search',
            component: () => import('../views/Search.vue'),
            meta: { requiresAuth: true }
        },
        {
            path: '/profile',
            name: 'UserProfile',
            component: () => import('../views/UserProfile.vue'),
            meta: { requiresAuth: true }
        },
        {
            path: '/subscriptions',
            name: 'Subscriptions',
            component: () => import('../views/Subscriptions.vue'),
            meta: { requiresAuth: true }
        },
        {
            path: '/subscriptions/:id/videos',
            name: 'SubscriptionVideos',
            component: () => import('../views/SubscriptionVideos.vue'),
            meta: { requiresAuth: true }
        },
        {
            path: '/tasks',
            name: 'Tasks',
            component: () => import('../views/Tasks.vue'),
            meta: { requiresAuth: true }
        },
        {
            path: '/settings',
            component: () => import('../views/Settings.vue'),
            meta: { layout: 'MainLayout', requiresAuth: true }
        },
        {
            path: '/monitoring',
            name: 'Monitoring',
            component: () => import('../views/Monitoring.vue'),
            meta: { requiresAuth: true }
        },
        {
            path: '/admin',
            component: () => import('../views/Admin.vue'),
            meta: {
                layout: 'MainLayout',
                requiresAuth: true,
                requiresAdmin: true
            }
        }
    ]
})

router.beforeEach(async (to, from, next) => {
    const userStore = useUserStore()

    // Init auth if needed
    if (!userStore.token && localStorage.getItem('token')) {
        await userStore.initAuth()
    }

    if (to.meta.requiresAuth && !userStore.isAuthenticated) {
        next('/login')
        return
    }

    if (to.meta.requiresAdmin && userStore.user?.role !== 'admin') {
        next('/dashboard') // Redirect non-admins
        return
    }

    if (!to.meta.requiresAuth && userStore.isAuthenticated) {
        if (to.path === '/login' || to.path === '/register') {
            next('/dashboard')
            return
        }
    }

    next()
})

export default router
