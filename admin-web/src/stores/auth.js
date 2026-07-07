import { defineStore } from 'pinia'
import { loginAdmin } from '../api/auth'
import { clearAdminSession, readAdminToken, readAdminUser, writeAdminSession } from './adminSession'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: readAdminToken(),
    user: readAdminUser(),
  }),
  getters: {
    isLoggedIn: (state) => Boolean(state.token),
    displayName: (state) => state.user?.loginName || state.user?.userId || '运营人员',
  },
  actions: {
    async login(form) {
      const data = await loginAdmin(form)
      this.token = data.token
      this.user = {
        userId: data.userId,
        roles: data.roles || [],
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
