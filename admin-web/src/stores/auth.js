import { defineStore } from 'pinia'
import { loginAdmin } from '../api/auth'

const TOKEN_KEY = 'wplink_admin_token'
const USER_KEY = 'wplink_admin_user'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: localStorage.getItem(TOKEN_KEY) || '',
    user: readStoredUser(),
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
      localStorage.setItem(TOKEN_KEY, this.token)
      localStorage.setItem(USER_KEY, JSON.stringify(this.user))
    },
    logout() {
      this.token = ''
      this.user = null
      localStorage.removeItem(TOKEN_KEY)
      localStorage.removeItem(USER_KEY)
    },
  },
})

function readStoredUser() {
  try {
    return JSON.parse(localStorage.getItem(USER_KEY) || 'null')
  } catch {
    return null
  }
}
