import { createRouter, createWebHashHistory } from 'vue-router'
import { clearAuth, isAuthenticated, pb } from '@/services/pocketbase'

const AppLayout = () => import('@/layouts/AppLayout.vue')
const LoginView = () => import('@/views/auth/LoginView.vue')
const RegisterView = () => import('@/views/auth/RegisterView.vue')
const LibraryView = () => import('@/views/library/LibraryView.vue')
const UploadBookView = () => import('@/views/books/UploadBookView.vue')
const ReaderView = () => import('@/views/books/ReaderView.vue')
const BookInfoView = () => import('@/views/books/BookInfoView.vue')
const NotesView = () => import('@/views/notes/NotesView.vue')
const SummaryView = () => import('@/views/summary/SummaryView.vue')

export const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/login', name: 'login', component: LoginView, meta: { guestOnly: true } },
    { path: '/register', name: 'register', component: RegisterView, meta: { guestOnly: true } },
    { path: '/books/:id/read', name: 'book-reader', component: ReaderView, props: true, meta: { requiresAuth: true } },
    {
      path: '/', component: AppLayout, meta: { requiresAuth: true }, children: [
        { path: '', redirect: '/books' },
        { path: 'books', name: 'books', component: LibraryView },
        { path: 'books/upload', name: 'book-upload', component: UploadBookView },
        { path: 'books/:id/info', name: 'book-info', component: BookInfoView, props: true },
        { path: 'notes', name: 'notes', component: NotesView },
        { path: 'summary', name: 'summary', component: SummaryView },
      ],
    },
  ],
})

router.beforeEach((to) => {
  const authed = isAuthenticated()
  if (to.meta.requiresAuth && !authed) return { name: 'login', query: { redirect: to.fullPath } }
  if (to.meta.guestOnly && authed) return { name: 'books' }
  return true
})

pb.authStore.onChange((token) => {
  if (!token) {
    clearAuth()
    if (router.currentRoute.value.meta.requiresAuth) router.push({ name: 'login' })
  }
})
