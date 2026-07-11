<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { BookOpen, ChartNoAxesCombined, ChevronLeft, ChevronRight, FileUp, Library, LogOut, NotebookPen } from '@lucide/vue'
import { clearAuth, currentUser } from '@/services/pocketbase'
import Button from '@/components/ui/Button.vue'

const router = useRouter()
const sidebarCollapsed = ref(false)
const userName = computed(() => currentUser()?.name || currentUser()?.email || 'Reader')
function logout() { clearAuth(); router.push({ name: 'login' }) }
</script>

<template>
  <div class="shell" :class="{ 'shell--sidebar-collapsed': sidebarCollapsed }">
    <aside class="sidebar flex flex-col justify-between">
      <div>
        <div class="sidebar-brand flex items-start justify-between gap-3">
          <div class="sidebar-brand__copy min-w-0">
            <span class="block text-[11px] font-extrabold uppercase tracking-widest text-[#705c21]">Ebook Reader</span>
            <h1 class="mt-0.5 text-base font-extrabold leading-snug tracking-tight text-[#142217]">青简书房</h1>
          </div>
          <Button variant="outline" size="sm" class="sidebar-collapse-button shrink-0 px-2" :title="sidebarCollapsed ? '展开侧边栏' : '收起侧边栏'" @click="sidebarCollapsed = !sidebarCollapsed">
            <ChevronRight v-if="sidebarCollapsed" data-icon="inline-start" />
            <ChevronLeft v-else data-icon="inline-start" />
          </Button>
        </div>

        <nav class="nav-links mt-6 flex flex-col gap-1.5">
          <RouterLink to="/books" class="nav-link" title="书籍管理"><Library class="nav-link__icon" /><span class="nav-link__label">书籍管理</span></RouterLink>
          <RouterLink to="/books/upload" class="nav-link" title="上传书籍"><FileUp class="nav-link__icon" /><span class="nav-link__label">上传书籍</span></RouterLink>
          <RouterLink to="/notes" class="nav-link" title="笔记管理"><NotebookPen class="nav-link__icon" /><span class="nav-link__label">笔记管理</span></RouterLink>
          <RouterLink to="/summary" class="nav-link" title="阅读汇总"><ChartNoAxesCombined class="nav-link__icon" /><span class="nav-link__label">阅读汇总</span></RouterLink>
        </nav>
      </div>

      <div class="sidebar-footer flex flex-col gap-4 border-t border-[#cbe0bf]/80 pt-4">
        <div class="rounded-lg border border-[#cbe0bf]/80 bg-white p-3 shadow-sm">
          <span class="block text-xs font-bold uppercase tracking-wider text-[#384c3d]">当前读者</span>
          <strong class="mt-0.5 block truncate text-sm font-bold text-[#142217]" :title="userName">{{ userName }}</strong>
        </div>
        <Button variant="outline" class="w-full text-red-700 hover:bg-red-50 hover:text-red-800" @click="logout"><LogOut data-icon="inline-start" />退出登录</Button>
      </div>
    </aside>

    <main class="content-area">
      <RouterView />
    </main>
  </div>
</template>
