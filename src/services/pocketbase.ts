import PocketBase from 'pocketbase'
import type { UserRecord } from '@/types/models'

export const pb = new PocketBase(import.meta.env.VITE_POCKETBASE_URL || window.location.origin)
pb.autoCancellation(false)

export const isAuthenticated = () => pb.authStore.isValid
export const currentUser = () => pb.authStore.model as UserRecord | null
export const clearAuth = () => pb.authStore.clear()
