import { defineStore } from 'pinia'
import { loginAdmin } from '../api/auth'
import { canAccessAdminModule, userModules } from '../common/adminModulePermissions'
import { clearAdminSession, readAdminToken, readAdminUser, writeAdminSession } from './adminSession'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: readAdminToken(),
    user: readAdminUser(),
  }),
  getters: {
    isLoggedIn: (state) => Boolean(state.token),
    roles: (state) => state.user?.roles || [],
    modules: (state) => userModules(state.user),
    isSuperAdmin: (state) => (state.user?.roles || []).includes('super_admin'),
    canAccessModule: (state) => (moduleCode) => canAccessAdminModule(state.user, moduleCode),
    displayName: (state) => state.user?.loginName || state.user?.operatorId || '运营人员',
  },
  actions: {
    async login(form) {
      const data = await loginAdmin(form)
      this.token = data.token
      this.user = {
        operatorId: data.operatorId,
        roles: data.roles || [],
        modules: data.modules || [],
        loginName: form.loginName,
      }
      writeAdminSession(this.token, this.user)
    },
    logout() {
      this.token = ''
      this.user = null
      clearAdminSession()
    },
  },
})
