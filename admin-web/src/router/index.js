import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import AdminLayout from '../layouts/AdminLayout.vue'

const LoginView = () => import('../views/LoginView.vue')
const DashboardView = () => import('../views/DashboardView.vue')
const ResourceReviewView = () => import('../views/ResourceReviewView.vue')
const MerchantView = () => import('../views/MerchantView.vue')
const VerificationView = () => import('../views/VerificationView.vue')
const EntitlementView = () => import('../views/EntitlementView.vue')
const OperationLogView = () => import('../views/OperationLogView.vue')
const SearchLogView = () => import('../views/SearchLogView.vue')
const ResourceTypeConfigView = () => import('../views/ResourceTypeConfigView.vue')
const BannerTopicView = () => import('../views/BannerTopicView.vue')
const HotSearchKeywordView = () => import('../views/HotSearchKeywordView.vue')
const SourcingMapView = () => import('../views/SourcingMapView.vue')

const routes = [
  {
    path: '/login',
    name: 'login',
    component: LoginView,
    meta: { public: true },
  },
  {
    path: '/',
    component: AdminLayout,
    redirect: '/dashboard',
    children: [
      { path: 'dashboard', name: 'dashboard', component: DashboardView },
      { path: 'resources/pending', name: 'resourceReview', component: ResourceReviewView },
      { path: 'merchants', name: 'merchants', component: MerchantView },
      { path: 'verifications', name: 'verifications', component: VerificationView },
      { path: 'entitlements', name: 'entitlements', component: EntitlementView },
      { path: 'banner-topics', name: 'bannerTopics', component: BannerTopicView },
      { path: 'hot-search-keywords', name: 'hotSearchKeywords', component: HotSearchKeywordView },
      { path: 'sourcing-map', name: 'sourcingMap', component: SourcingMapView },
      { path: 'resource-type-configs', name: 'resourceTypeConfigs', component: ResourceTypeConfigView },
      { path: 'operation-logs', name: 'operationLogs', component: OperationLogView },
      { path: 'search-logs', name: 'searchLogs', component: SearchLogView },
    ],
  },
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (!to.meta.public && !auth.isLoggedIn) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.name === 'login' && auth.isLoggedIn) {
    return { name: 'dashboard' }
  }
  return true
})

export default router
