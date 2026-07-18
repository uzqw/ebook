<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { BookOpen, ChartNoAxesCombined, ChevronLeft, ChevronRight, FileUp, Library, LogOut, NotebookPen } from '@lucide/vue'
import { clearAuth, currentUser } from '@/services/pocketbase'
import Button from '@/components/ui/Button.vue'

const router = useRouter()
const route = useRoute()
const sidebarCollapsed = ref(false)
const mainEl = ref<HTMLElement | null>(null)
const userName = computed(() => currentUser()?.name || currentUser()?.email || 'Reader')
function logout() { clearAuth(); router.push({ name: 'login' }) }

watch(() => route.fullPath, async () => {
  await nextTick()
  mainEl.value?.focus({ preventScroll: true })
})
</script>

<template>
  <div class="shell" :class="{ 'shell--sidebar-collapsed': sidebarCollapsed }">
    <a href="#main-content" class="skip-link">跳到主要内容</a>
    <aside class="sidebar flex flex-col justify-between">
      <div class="sidebar-top">
        <div class="sidebar-brand flex items-start justify-between gap-3">
          <div class="sidebar-brand__copy min-w-0">
            <span class="block text-[10px] font-extrabold uppercase tracking-widest text-[#0f7643]/80">Ebook Reader</span>
            <h1 class="mt-0.5 text-lg font-black leading-snug tracking-tight text-[#0f1e14]">青简书房</h1>
          </div>
          <Button variant="outline" size="sm" class="sidebar-collapse-button shrink-0 px-2" :aria-label="sidebarCollapsed ? '展开侧边栏' : '收起侧边栏'" :title="sidebarCollapsed ? '展开侧边栏' : '收起侧边栏'" @click="sidebarCollapsed = !sidebarCollapsed">
            <ChevronRight v-if="sidebarCollapsed" data-icon="inline-start" />
            <ChevronLeft v-else data-icon="inline-start" />
          </Button>
        </div>

        <nav class="nav-links mt-6 flex flex-col gap-1.5" aria-label="主导航">
          <RouterLink to="/books" class="nav-link" title="书籍管理"><Library class="nav-link__icon" /><span class="nav-link__label">书籍管理</span></RouterLink>
          <RouterLink to="/books/upload" class="nav-link" title="上传书籍"><FileUp class="nav-link__icon" /><span class="nav-link__label">上传书籍</span></RouterLink>
          <RouterLink to="/notes" class="nav-link" title="笔记管理"><NotebookPen class="nav-link__icon" /><span class="nav-link__label">笔记管理</span></RouterLink>
          <RouterLink to="/summary" class="nav-link" title="阅读汇总"><ChartNoAxesCombined class="nav-link__icon" /><span class="nav-link__label">阅读汇总</span></RouterLink>
        </nav>
      </div>

      <div class="sidebar-footer flex flex-col gap-4 border-t border-emerald-500/10 pt-4">
        <div class="sidebar-user-card rounded-xl border border-emerald-500/10 bg-white p-3.5 shadow-sm">
          <span class="block text-[10px] font-extrabold uppercase tracking-wider text-[#4a5c50]">当前读者</span>
          <strong class="mt-0.5 block truncate text-sm font-bold text-[#0f1e14]" :title="userName">{{ userName }}</strong>
        </div>
        <Button variant="outline" class="sidebar-logout w-full text-red-700 hover:bg-red-50 hover:text-red-800" aria-label="退出登录" @click="logout"><LogOut data-icon="inline-start" /><span class="sidebar-logout-label">退出登录</span></Button>
      </div>
    </aside>

    <main id="main-content" ref="mainEl" class="content-area" tabindex="-1">
      <RouterView />
    </main>
  </div>
</template>
