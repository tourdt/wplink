import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import AdminLayout from '../layouts/AdminLayout.vue'

const LoginView = () => import('../views/LoginView.vue')
const DashboardView = () => import('../views/DashboardView.vue')
const ResourceReviewView = () => import('../views/ResourceReviewView.vue')
const ResourceReportView = () => import('../views/ResourceReportView.vue')
const MerchantView = () => import('../views/MerchantView.vue')
const VerificationView = () => import('../views/VerificationView.vue')
const EntitlementView = () => import('../views/EntitlementView.vue')
const OperationLogView = () => import('../views/OperationLogView.vue')
const SearchLogView = () => import('../views/SearchLogView.vue')
const ResourceTypeConfigView = () => import('../views/ResourceTypeConfigView.vue')
const BannerTopicView = () => import('../views/BannerTopicView.vue')
const HotSearchKeywordView = () => import('../views/HotSearchKeywordView.vue')
const SourcingMapView = () => import('../views/SourcingMapView.vue')
const VIPConfigView = () => import('../views/VIPConfigView.vue')
const GrowthCampaignView = () => import('../views/GrowthCampaignView.vue')
const AdminPermissionView = () => import('../views/AdminPermissionView.vue')

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
      { path: 'dashboard', name: 'dashboard', component: DashboardView, meta: { moduleCode: 'dashboard' } },
      { path: 'resources/pending', name: 'resourceReview', component: ResourceReviewView, meta: { moduleCode: 'resource_review' } },
      { path: 'resource-reports', name: 'resourceReports', component: ResourceReportView, meta: { moduleCode: 'resource_reports' } },
      { path: 'merchants', name: 'merchants', component: MerchantView, meta: { moduleCode: 'merchants' } },
      { path: 'verifications', name: 'verifications', component: VerificationView, meta: { moduleCode: 'verification_review' } },
      { path: 'entitlements', name: 'entitlements', component: EntitlementView, meta: { moduleCode: 'entitlements' } },
      { path: 'banner-topics', name: 'bannerTopics', component: BannerTopicView, meta: { moduleCode: 'banner_topics' } },
      { path: 'hot-search-keywords', name: 'hotSearchKeywords', component: HotSearchKeywordView, meta: { moduleCode: 'hot_search_keywords' } },
      { path: 'vip-configs', name: 'vipConfigs', component: VIPConfigView, meta: { moduleCode: 'vip_configs' } },
      { path: 'growth-campaigns', name: 'growthCampaigns', component: GrowthCampaignView, meta: { moduleCode: 'growth_campaigns' } },
      { path: 'sourcing-map', name: 'sourcingMap', component: SourcingMapView, meta: { moduleCode: 'sourcing_map' } },
      { path: 'resource-type-configs', name: 'resourceTypeConfigs', component: ResourceTypeConfigView, meta: { moduleCode: 'resource_type_configs' } },
      { path: 'admin-permissions', name: 'adminPermissions', component: AdminPermissionView, meta: { moduleCode: 'admin_permissions', requiresSuperAdmin: true } },
      { path: 'operation-logs', name: 'operationLogs', component: OperationLogView, meta: { moduleCode: 'operation_logs' } },
      { path: 'search-logs', name: 'searchLogs', component: SearchLogView, meta: { moduleCode: 'search_logs' } },
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
  if ((to.meta.requiresSuperAdmin && !auth.isSuperAdmin) || !auth.canAccessModule(to.meta.moduleCode)) {
    return { name: firstAccessibleRouteName(auth) }
  }
  if (to.name === 'login' && auth.isLoggedIn) {
    return { name: firstAccessibleRouteName(auth) }
  }
  return true
})

function firstAccessibleRouteName(auth) {
  const adminChildren = routes.find((route) => route.path === '/')?.children || []
  const route = adminChildren.find((item) => !item.meta?.requiresSuperAdmin && auth.canAccessModule(item.meta?.moduleCode))
  return route?.name || 'dashboard'
}

export default router
