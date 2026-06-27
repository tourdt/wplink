import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import AdminLayout from '../layouts/AdminLayout.vue'
import LoginView from '../views/LoginView.vue'
import DashboardView from '../views/DashboardView.vue'
import ResourceReviewView from '../views/ResourceReviewView.vue'
import MerchantView from '../views/MerchantView.vue'
import DemandView from '../views/DemandView.vue'
import VerificationView from '../views/VerificationView.vue'
import EntitlementView from '../views/EntitlementView.vue'
import OperationLogView from '../views/OperationLogView.vue'

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
      { path: 'demands', name: 'demands', component: DemandView },
      { path: 'verifications', name: 'verifications', component: VerificationView },
      { path: 'entitlements', name: 'entitlements', component: EntitlementView },
      { path: 'operation-logs', name: 'operationLogs', component: OperationLogView },
    ],
  },
]

const router = createRouter({
  history: createWebHistory(),
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
