<template>
  <div class="min-h-screen bg-bg text-ink relative">
    <div class="grain"></div>
    <div class="pointer-events-none fixed inset-0 opacity-40" style="background:radial-gradient(1200px 500px at 10% -10%, #c9a22722, transparent 55%), radial-gradient(800px 400px at 110% 10%, #7ec8c418, transparent 50%);"></div>

    <div class="flex min-h-screen w-full">
      <aside class="hidden md:flex w-[252px] shrink-0 flex-col border-r border-line bg-elev/80 backdrop-blur-sm px-5 py-6 fade-enter">
        <div class="font-display text-2xl tracking-tight text-brass">GoGit</div>
        <p class="mt-1 text-sm text-muted font-serif">Carbon Archive</p>
        <nav class="mt-8 flex flex-col gap-1">
          <button v-for="(item, i) in nav" :key="item.id" class="text-left px-3 py-2 rounded-sm border border-transparent hover:border-line transition-colors" :class="tab===item.id ? 'bg-sunken border-brass/50 text-brass' : 'text-ink/80'" :style="{ animationDelay: i*60+'ms' }" @click="tab=item.id">
            <span class="font-mono text-[11px] text-muted mr-2">0{{ i+1 }}</span>{{ item.label }}
          </button>
        </nav>
        <div class="mt-auto pt-6 border-t border-line">
          <div class="text-[11px] uppercase tracking-widest text-muted">HEAD</div>
          <div class="mt-1 font-mono text-xs text-cyan break-all">{{ short(repo.head) || 'unborn' }}</div>
          <div class="mt-2 text-xs text-muted">{{ repo.current_branch }} · {{ repo.hash_algo }}</div>
        </div>
      </aside>

      <div class="flex-1 min-w-0 w-full">
        <header class="flex items-center gap-3 border-b border-line px-4 md:px-6 py-3 bg-elev/40">
          <div class="md:hidden font-display text-lg text-brass">GoGit</div>
          <div class="flex-1 overflow-x-auto md:hidden">
            <div class="flex gap-2">
              <button v-for="item in nav" :key="item.id" class="whitespace-nowrap px-3 py-1 text-sm border" :class="tab===item.id ? 'border-brass text-brass' : 'border-line text-muted'" @click="tab=item.id">{{ item.label }}</button>
            </div>
          </div>
          <div class="hidden md:flex items-center gap-3 ml-auto">
            <span class="px-2 py-1 border border-brass text-brass font-mono text-xs">{{ repo.current_branch || '—' }}</span>
            <span class="text-xs text-muted">对象 {{ repo.object_count ?? 0 }}</span>
            <span class="text-xs text-muted uppercase">{{ repo.hash_algo }}</span>
          </div>
        </header>

        <main class="w-full px-4 md:px-6 py-6 fade-enter" :key="tab">
          <section v-if="tab==='overview'">
            <h1 class="font-display text-3xl md:text-4xl">档案总览</h1>
            <p class="mt-2 text-muted">工作区位于 {{ repo.path }}</p>
            <div class="mt-6 grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-4">
              <article v-for="card in overviewCards" :key="card.k" class="border border-line bg-elev p-5">
                <div class="text-[11px] tracking-[0.2em] uppercase text-muted">{{ card.k }}</div>
                <div class="mt-3 font-mono text-xl text-cyan break-all">{{ card.v }}</div>
              </article>
            </div>
            <h2 class="mt-10 font-display text-xl">最近提交</h2>
            <ol class="mt-4 border-l border-brass/40 ml-2">
              <li v-for="c in commits.slice(0,5)" :key="c.hash" class="relative pl-6 py-3">
                <span class="absolute left-[-5px] top-5 h-2.5 w-2.5 rounded-full bg-brass"></span>
                <button class="text-left" @click="openCommit(c.hash)">
                  <div class="font-serif">{{ c.message }}</div>
                  <div class="mt-1 font-mono text-xs text-cyan">{{ short(c.hash) }} · {{ c.committed_at }} · {{ c.author }}</div>
                </button>
              </li>
              <li v-if="!commits.length" class="pl-6 py-3 text-muted">尚无提交</li>
            </ol>
          </section>

          <section v-else-if="tab==='files'">
            <div class="flex flex-wrap items-end justify-between gap-3">
              <h1 class="font-display text-3xl">工作区</h1>
              <button class="btn-ghost" @click="loadFiles">刷新</button>
            </div>
            <div class="mt-4 flex flex-wrap items-center gap-2 text-sm">
              <button class="text-cyan font-mono" @click="fileDir=''">/</button>
              <span v-for="(seg,i) in crumbs" :key="i" class="text-muted">
                / <button class="text-cyan" @click="fileDir=crumbs.slice(0,i+1).join('/')">{{ seg }}</button>
              </span>
            </div>
            <div class="mt-4 grid grid-cols-1 lg:grid-cols-2 gap-4">
              <div class="border border-line bg-elev min-h-[280px] overflow-x-auto">
                <div v-if="fileDir" class="px-4 py-2 border-b border-line font-mono text-sm text-muted cursor-pointer hover:text-ink" @click="upDir">../</div>
                <button v-for="e in files" :key="e.path" class="w-full flex items-center justify-between px-4 py-2 border-b border-line/60 hover:bg-sunken text-left" @click="e.type==='dir' ? (fileDir=e.path) : openFile(e.path)">
                  <span class="font-mono text-sm">{{ e.type==='dir' ? '▸' : '·' }} {{ e.name }}</span>
                  <span class="text-xs text-muted">{{ e.type==='dir' ? 'dir' : e.size+' B' }}</span>
                </button>
                <div v-if="!files.length" class="p-6 text-muted">此目录为空</div>
              </div>
              <div class="border border-line bg-sunken p-4">
                <div class="flex flex-wrap gap-2">
                  <button class="btn-primary" :disabled="!currentPath" @click="addPaths([currentPath])">Add 当前文件</button>
                  <button class="btn-ghost" @click="addPaths([fileDir || '.'])">Add 当前目录</button>
                  <button class="btn-danger" :disabled="!currentPath" @click="askDelete">删除</button>
                </div>
                <label class="block mt-4 text-xs text-muted">路径 *</label>
                <input v-model="editPath" class="inp" @blur="validatePath" />
                <p v-if="pathErr" class="err">{{ pathErr }}</p>
                <label class="block mt-3 text-xs text-muted">内容</label>
                <textarea v-model="editContent" class="inp min-h-[220px] font-mono text-xs" spellcheck="false"></textarea>
                <button class="btn-primary mt-3" @click="saveFile">保存到工作区</button>
              </div>
            </div>
          </section>

          <section v-else-if="tab==='stage'">
            <h1 class="font-display text-3xl">暂存区</h1>
            <div class="mt-6 grid grid-cols-1 xl:grid-cols-3 gap-4">
              <div class="border border-line bg-elev p-4">
                <h3 class="text-brass text-sm tracking-widest uppercase">Staged</h3>
                <ul class="mt-3 space-y-2">
                  <li v-for="s in status.staged" :key="'s'+s.path" class="flex justify-between gap-2 font-mono text-sm">
                    <button class="text-left hover:text-cyan" @click="showDiff(s.path, 'staged')">{{ s.path }}</button>
                    <span class="flex items-center gap-2">
                      <StatusBadge :status="s.status" />
                      <button class="text-xs text-muted" @click="doUnstage(s.path)">unstage</button>
                    </span>
                  </li>
                  <li v-if="!status.staged?.length" class="text-muted text-sm">空</li>
                </ul>
              </div>
              <div class="border border-line bg-elev p-4">
                <h3 class="text-danger text-sm tracking-widest uppercase">Unstaged</h3>
                <ul class="mt-3 space-y-2">
                  <li v-for="s in status.unstaged" :key="'u'+s.path" class="flex justify-between gap-2 font-mono text-sm">
                    <button class="text-left hover:text-cyan" @click="showDiff(s.path, 'unstaged')">{{ s.path }}</button>
                    <span class="flex gap-2">
                      <button class="text-cyan text-xs" @click="addPaths([s.path])">add</button>
                      <button class="text-muted text-xs" @click="doRestore(s.path)">restore</button>
                    </span>
                  </li>
                  <li v-if="!status.unstaged?.length" class="text-muted text-sm">空</li>
                </ul>
              </div>
              <div class="border border-line bg-elev p-4">
                <h3 class="text-cyan text-sm tracking-widest uppercase">Untracked</h3>
                <ul class="mt-3 space-y-2">
                  <li v-for="s in status.untracked" :key="'n'+s.path" class="flex justify-between gap-2 font-mono text-sm">
                    <span>{{ s.path }}</span>
                    <button class="text-cyan text-xs" @click="addPaths([s.path])">add</button>
                  </li>
                  <li v-if="!status.untracked?.length" class="text-muted text-sm">空</li>
                </ul>
              </div>
            </div>
            <form class="mt-8 border border-line bg-elev p-5 w-full" @submit.prevent="doCommit">
              <h3 class="font-display text-xl">生成快照</h3>
              <label class="block mt-4 text-xs text-muted">提交信息 *</label>
              <input v-model="commitMsg" class="inp" />
              <p v-if="commitErr" class="err">{{ commitErr }}</p>
              <label class="block mt-3 text-xs text-muted">作者</label>
              <input v-model="commitAuthor" class="inp" placeholder="Archivist <archivist@gogit.local>" />
              <button class="btn-primary mt-4" type="submit">Commit</button>
            </form>
            <pre v-if="diffPatch" class="mt-6 bg-sunken border border-line p-4 overflow-x-auto font-mono text-xs whitespace-pre-wrap">{{ diffPatch }}</pre>
            <div class="mt-6 overflow-x-auto border border-line">
              <table class="w-full text-sm">
                <thead class="bg-elev text-muted text-left">
                  <tr><th class="p-3">路径</th><th class="p-3">mode</th><th class="p-3">oid</th><th class="p-3">size</th></tr>
                </thead>
                <tbody>
                  <tr v-for="e in index.entries" :key="e.path" class="border-t border-line">
                    <td class="p-3 font-mono">{{ e.path }}</td>
                    <td class="p-3 font-mono text-muted">{{ e.mode }}</td>
                    <td class="p-3 font-mono text-cyan"><button @click="copy(e.oid)">{{ short(e.oid) }}</button></td>
                    <td class="p-3">{{ e.size }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </section>

          <section v-else-if="tab==='history'">
            <div class="flex flex-wrap items-end gap-3">
              <h1 class="font-display text-3xl">提交历史</h1>
              <select v-model="histBranch" class="inp w-auto" @change="loadCommits">
                <option v-for="b in branches" :key="b.name" :value="b.name">{{ b.name }}</option>
              </select>
            </div>
            <ol class="mt-6 border-l border-brass/40 ml-2">
              <li v-for="c in commits" :key="c.hash" class="relative pl-6 py-4">
                <span class="absolute left-[-5px] top-6 h-2.5 w-2.5 rounded-full bg-brass"></span>
                <div class="font-serif text-lg">{{ c.message }}</div>
                <div class="mt-1 font-mono text-xs text-muted">{{ c.committed_at }} · {{ c.author }}</div>
                <div class="mt-1 font-mono text-xs text-cyan">
                  <button @click="inspect(c.hash)">{{ c.hash }}</button>
                  <span v-if="c.parents?.length" class="text-muted"> ← {{ c.parents.map(short).join(', ') }}</span>
                </div>
                <button class="btn-ghost mt-2 text-xs" @click="openCommit(c.hash)">查看快照</button>
                <div v-if="snapshot.hash===c.hash" class="mt-3 bg-sunken border border-line p-3 font-mono text-xs overflow-x-auto">
                  <div v-for="e in snapshot.entries" :key="e.path" class="flex justify-between gap-4 py-1">
                    <span>{{ e.path }}</span><span class="text-cyan">{{ short(e.oid) }}</span>
                  </div>
                </div>
              </li>
            </ol>
          </section>

          <section v-else-if="tab==='branches'">
            <h1 class="font-display text-3xl">分支</h1>
            <ul class="mt-6 divide-y divide-line border border-line">
              <li v-for="b in branches" :key="b.name" class="flex flex-wrap items-center gap-3 px-4 py-3 bg-elev">
                <span class="font-mono" :class="b.current ? 'text-brass' : ''">{{ b.name }}</span>
                <span v-if="b.current" class="text-[10px] tracking-widest uppercase border border-brass text-brass px-1">HEAD</span>
                <span class="font-mono text-xs text-cyan">{{ short(b.hash) }}</span>
                <div class="ml-auto flex gap-2">
                  <button class="btn-ghost text-xs" :disabled="b.current" @click="askCheckout(b.name)">切换</button>
                  <button class="btn-ghost text-xs" :disabled="b.current" @click="askMerge(b.name)">Merge 入当前</button>
                </div>
              </li>
            </ul>
            <form class="mt-6 border border-line bg-elev p-5" @submit.prevent="doCreateBranch">
              <label class="text-xs text-muted">新分支名 *</label>
              <input v-model="newBranch" class="inp" placeholder="feature/topic" />
              <p v-if="branchErr" class="err">{{ branchErr }}</p>
              <button class="btn-primary mt-3" type="submit">创建分支</button>
            </form>
            <div v-if="conflicts.length" class="mt-6 border border-danger bg-sunken p-4">
              <h3 class="text-danger font-display">Merge 冲突</h3>
              <p class="text-sm text-muted mt-1">冲突文件已写入工作区，请编辑后 add + commit 完成合并。</p>
              <ul class="mt-2 font-mono text-sm text-danger">
                <li v-for="p in conflicts" :key="p">{{ p }}</li>
              </ul>
            </div>
          </section>

          <section v-else>
            <h1 class="font-display text-3xl">对象灯箱</h1>
            <form class="mt-4 flex flex-wrap gap-2" @submit.prevent="inspect(objHash)">
              <input v-model="objHash" class="inp flex-1 min-w-[220px] font-mono" placeholder="HEAD / 分支名 / 短哈希" />
              <button class="btn-primary" type="submit">解析</button>
            </form>
            <p v-if="objErr" class="err">{{ objErr }}</p>
            <pre v-if="objView" class="mt-4 bg-sunken border border-line p-4 overflow-x-auto font-mono text-xs whitespace-pre-wrap">{{ objView }}</pre>
          </section>
        </main>
      </div>
    </div>

    <div class="fixed bottom-4 right-4 z-[60] flex flex-col gap-2 w-[min(92vw,360px)]">
      <div v-for="t in toasts" :key="t.id" class="border px-4 py-3 bg-elev shadow-xl flex gap-3" :class="t.kind==='ok' ? 'border-ok' : 'border-danger'">
        <p class="flex-1 text-sm">{{ t.text }}</p>
        <button class="text-muted" @click="dismiss(t.id)">×</button>
      </div>
    </div>

    <div v-if="modal" class="fixed inset-0 z-[70] flex items-center justify-center bg-bg/70 backdrop-blur-sm px-4" @click.self="modal=null">
      <div class="w-full max-w-md border border-brass/40 bg-elev p-6">
        <h3 class="font-display text-xl">{{ modal.title }}</h3>
        <p class="mt-2 text-muted">{{ modal.body }}</p>
        <div class="mt-5 flex justify-end gap-2">
          <button class="btn-ghost" @click="modal=null">取消</button>
          <button :class="modal.danger ? 'btn-danger' : 'btn-primary'" @click="confirmModal">确认</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { api } from './api.js'
import StatusBadge from './components/StatusBadge.vue'

const nav = [
  { id: 'overview', label: '总览' },
  { id: 'files', label: '工作区' },
  { id: 'stage', label: '暂存区' },
  { id: 'history', label: '历史' },
  { id: 'branches', label: '分支' },
  { id: 'objects', label: '对象' },
]
const tab = ref('overview')
const repo = reactive({})
const files = ref([])
const fileDir = ref('')
const currentPath = ref('')
const editPath = ref('')
const editContent = ref('')
const pathErr = ref('')
const status = reactive({ staged: [], unstaged: [], untracked: [] })
const index = reactive({ entries: [] })
const commitMsg = ref('')
const commitAuthor = ref('Archivist <archivist@gogit.local>')
const commitErr = ref('')
const commits = ref([])
const branches = ref([])
const histBranch = ref('')
const snapshot = reactive({ hash: '', entries: [] })
const newBranch = ref('')
const branchErr = ref('')
const conflicts = ref([])
const objHash = ref('')
const objView = ref('')
const objErr = ref('')
const diffPatch = ref('')
const toasts = ref([])
const modal = ref(null)
let toastSeq = 0

const crumbs = computed(() => fileDir.value ? fileDir.value.split('/') : [])
const overviewCards = computed(() => [
  { k: '当前分支', v: repo.current_branch || '—' },
  { k: 'HEAD', v: short(repo.head) || 'unborn' },
  { k: '对象数', v: String(repo.object_count ?? 0) },
  { k: '算法', v: repo.hash_algo || 'sha1' },
])

function short(h) {
  return h ? String(h).slice(0, 7) : ''
}
function toast(text, kind = 'ok') {
  const id = ++toastSeq
  toasts.value.push({ id, text, kind })
  setTimeout(() => dismiss(id), 5000)
}
function dismiss(id) {
  toasts.value = toasts.value.filter((t) => t.id !== id)
}
async function copy(s) {
  await navigator.clipboard.writeText(s)
  toast('已复制哈希')
}
function validatePath() {
  pathErr.value = ''
  const p = editPath.value.trim()
  if (!p) {
    pathErr.value = '路径不能为空'
    return false
  }
  if (p.includes('..') || p.startsWith('/') || p.startsWith('.gogit')) {
    pathErr.value = '路径不合法（禁止 ..、绝对路径、.gogit）'
    return false
  }
  return true
}
function validateCommit() {
  commitErr.value = ''
  if (!commitMsg.value.trim()) {
    commitErr.value = '提交信息不能为空'
    return false
  }
  return true
}
function validateBranch() {
  branchErr.value = ''
  const n = newBranch.value.trim()
  if (!n) {
    branchErr.value = '分支名不能为空'
    return false
  }
  if (!/^[A-Za-z0-9._/\-]+$/.test(n) || n.includes('..') || n.startsWith('/') || n.endsWith('/')) {
    branchErr.value = '仅允许字母数字 . _ - /，且不能以 / 开头结尾'
    return false
  }
  return true
}

async function refreshAll() {
  try {
    Object.assign(repo, await api.repo())
    histBranch.value = histBranch.value || repo.current_branch
    branches.value = await api.branches()
    await loadCommits()
    await loadFiles()
    Object.assign(status, await api.status())
    Object.assign(index, await api.index())
  } catch (e) {
    toast(e.message, 'err')
  }
}
async function loadFiles() {
  const data = await api.files(fileDir.value)
  files.value = data.entries || []
}
async function loadCommits() {
  commits.value = await api.commits(histBranch.value || repo.current_branch)
}
function upDir() {
  const parts = fileDir.value.split('/')
  parts.pop()
  fileDir.value = parts.join('/')
}
async function openFile(path) {
  try {
    currentPath.value = path
    editPath.value = path
    const data = await api.fileContent(path)
    editContent.value = data.binary ? '[binary]' : data.content
  } catch (e) {
    toast(e.message, 'err')
  }
}
async function saveFile() {
  if (!validatePath()) {
    toast('请修正路径后再保存', 'err')
    return
  }
  try {
    await api.putFile(editPath.value.trim(), editContent.value)
    toast('已写入工作区')
    currentPath.value = editPath.value.trim()
    await refreshAll()
  } catch (e) {
    toast(e.message, 'err')
  }
}
function askDelete() {
  if (!currentPath.value) return
  modal.value = { title: '删除文件', body: `删除工作区文件 ${currentPath.value}？`, danger: true, run: async () => {
    await api.deleteFile(currentPath.value)
    currentPath.value = ''
    editContent.value = ''
    toast('已删除')
    await refreshAll()
  }}
}
async function showDiff(path, side) {
  try {
    const d = await api.diff(path, side)
    diffPatch.value = d.patch || '(no changes)'
  } catch (e) {
    toast(e.message, 'err')
  }
}
async function doUnstage(path) {
  try {
    await api.unstage([path])
    toast('已移出暂存区')
    await refreshAll()
  } catch (e) {
    toast(e.message, 'err')
  }
}
async function doRestore(path) {
  modal.value = {
    title: '恢复工作区文件',
    body: `用 HEAD 快照覆盖 ${path}？未暂存修改会丢失。`,
    danger: true,
    run: async () => {
      await api.restore([path])
      toast('已从 HEAD 恢复')
      await refreshAll()
    },
  }
}
async function addPaths(paths) {
  try {
    const p = paths.map((x) => x || '.')
    await api.add(p)
    toast('已写入暂存区')
    await refreshAll()
  } catch (e) {
    toast(e.message, 'err')
  }
}
async function doCommit() {
  if (!validateCommit()) {
    toast('请填写提交信息', 'err')
    return
  }
  try {
    const c = await api.createCommit(commitMsg.value.trim(), commitAuthor.value.trim())
    commitMsg.value = ''
    toast('提交 ' + short(c.hash))
    await refreshAll()
  } catch (e) {
    toast(e.message, 'err')
  }
}
async function openCommit(hash) {
  const data = await api.commitTree(hash)
  snapshot.hash = hash
  snapshot.entries = data.entries || []
  tab.value = 'history'
}
async function inspect(hash) {
  objErr.value = ''
  objHash.value = hash
  try {
    let oid = hash
    if (hash && (hash === 'HEAD' || hash.includes('/') || hash.length < 40)) {
      const parsed = await api.revParse(hash)
      oid = parsed.oid
    }
    const o = await api.object(oid)
    objView.value = JSON.stringify(o, null, 2)
    tab.value = 'objects'
  } catch (e) {
    objErr.value = e.message
    toast(e.message, 'err')
  }
}
function askCheckout(name) {
  modal.value = { title: '切换分支', body: `将 HEAD 指向 ${name} 并恢复工作区快照。未提交且会被覆盖的改动会被拒绝。`, run: async () => {
    await api.checkout(name)
    toast('已切换到 ' + name)
    await refreshAll()
  }}
}
function askMerge(name) {
  modal.value = { title: '三方合并', body: `将 ${name} 合并进当前分支 ${repo.current_branch}。`, run: async () => {
    try {
      const res = await api.merge(name)
      conflicts.value = []
      toast(res.fast_forward ? '快进合并完成' : '合并提交已生成')
      await refreshAll()
    } catch (e) {
      conflicts.value = e.details || []
      toast(e.message, 'err')
      await refreshAll()
    }
  }}
}
async function doCreateBranch() {
  if (!validateBranch()) {
    toast('请修正分支名', 'err')
    return
  }
  try {
    await api.createBranch(newBranch.value.trim())
    toast('分支已创建')
    newBranch.value = ''
    await refreshAll()
  } catch (e) {
    toast(e.message, 'err')
  }
}
async function confirmModal() {
  const m = modal.value
  modal.value = null
  try {
    await m.run()
  } catch (e) {
    toast(e.message, 'err')
  }
}

watch(fileDir, () => loadFiles().catch((e) => toast(e.message, 'err')))
watch(tab, () => refreshAll())
onMounted(refreshAll)
</script>

<style scoped>
.inp {
  @apply mt-1 w-full bg-sunken border border-line px-3 py-2 outline-none focus:border-brass;
}
.btn-primary {
  @apply px-4 py-2 bg-brass text-bg border border-brass font-display tracking-wide hover:-translate-y-px transition-transform disabled:opacity-40;
}
.btn-ghost {
  @apply px-4 py-2 border border-brass text-brass hover:bg-brass/10 disabled:opacity-40;
}
.btn-danger {
  @apply px-4 py-2 border border-danger text-danger hover:bg-danger/10 disabled:opacity-40;
}
.err {
  @apply mt-1 text-sm text-danger;
}
</style>
