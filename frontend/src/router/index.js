import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const routes = [
  { path: '/', component: () => import('../views/FeedView.vue') },
  { path: '/login', component: () => import('../views/LoginView.vue') },
  { path: '/register', component: () => import('../views/RegisterView.vue') },
  { path: '/video/:id', component: () => import('../views/VideoDetailView.vue') },
  { path: '/publish', component: () => import('../views/PublishView.vue'), meta: { requiresAuth: true } },
  { path: '/messages', component: () => import('../views/MessageView.vue'), meta: { requiresAuth: true } },
  { path: '/profile', component: () => import('../views/ProfileView.vue'), meta: { requiresAuth: true } },
  { path: '/profile/:id', component: () => import('../views/ProfileView.vue'), props: true },
  { path: '/settings', component: () => import('../views/SettingsView.vue'), meta: { requiresAuth: true } },
  { path: '/:pathMatch(.*)*', redirect: '/' },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior: () => ({ top: 0 }),
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (to.meta.requiresAuth && !auth.isLoggedIn) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }
  if ((to.path === '/login' || to.path === '/register') && auth.isLoggedIn) {
    return { path: '/' }
  }
  return true
})

export default router
