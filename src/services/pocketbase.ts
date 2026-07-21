import PocketBase from 'pocketbase'
import { installResilientFetch } from '@/services/http'
import type { UserRecord } from '@/types/models'

export const pb = new PocketBase(import.meta.env.VITE_POCKETBASE_URL || window.location.origin)
pb.autoCancellation(false)
installResilientFetch(pb.baseUrl)

export const isAuthenticated = () => pb.authStore.isValid
export const currentUser = () => pb.authStore.model as UserRecord | null
export const clearAuth = () => pb.authStore.clear()
