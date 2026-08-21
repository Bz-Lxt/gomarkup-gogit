const BASE = ''

async function request(path, options = {}) {
  const res = await fetch(BASE + path, {
    headers: { 'Content-Type': 'application/json', ...(options.headers || {}) },
    ...options,
  })
  const text = await res.text()
  let body = null
  if (text) {
    try {
      body = JSON.parse(text)
    } catch {
      throw new Error('服务器返回了无法解析的响应')
    }
  }
  if (!res.ok) {
    const err = new Error(body?.error?.message || `HTTP ${res.status}`)
    err.code = body?.error?.code
    err.details = body?.error?.details || []
    err.status = res.status
    throw err
  }
  return body?.data
}

export const api = {
  health: () => request('/api/v1/health'),
  repo: () => request('/api/v1/repo'),
  files: (path = '') => request('/api/v1/files?path=' + encodeURIComponent(path)),
  fileContent: (path) => request('/api/v1/files/content?path=' + encodeURIComponent(path)),
  putFile: (path, content) => request('/api/v1/files', { method: 'PUT', body: JSON.stringify({ path, content }) }),
  deleteFile: (path) => request('/api/v1/files?path=' + encodeURIComponent(path), { method: 'DELETE' }),
  add: (paths) => request('/api/v1/index/add', { method: 'POST', body: JSON.stringify({ paths }) }),
  index: () => request('/api/v1/index'),
  status: () => request('/api/v1/status'),
  createCommit: (message, author) => request('/api/v1/commits', { method: 'POST', body: JSON.stringify({ message, author }) }),
  commits: (branch) => request('/api/v1/commits' + (branch ? '?branch=' + encodeURIComponent(branch) : '')),
  getCommit: (hash) => request('/api/v1/commits/' + hash),
  commitTree: (hash) => request('/api/v1/commits/' + hash + '/tree'),
  branches: () => request('/api/v1/branches'),
  createBranch: (name) => request('/api/v1/branches', { method: 'POST', body: JSON.stringify({ name }) }),
  checkout: (name) => request('/api/v1/checkout', { method: 'POST', body: JSON.stringify({ name }) }),
  merge: (branch) => request('/api/v1/merge', { method: 'POST', body: JSON.stringify({ branch }) }),
  object: (hash) => request('/api/v1/objects/' + hash),
  diff: (path, side) => request('/api/v1/diff?path=' + encodeURIComponent(path) + '&side=' + encodeURIComponent(side || 'unstaged')),
  unstage: (paths) => request('/api/v1/index/unstage', { method: 'POST', body: JSON.stringify({ paths }) }),
  restore: (paths) => request('/api/v1/files/restore', { method: 'POST', body: JSON.stringify({ paths }) }),
  revParse: (q) => request('/api/v1/rev-parse?q=' + encodeURIComponent(q)),
  fsck: () => request('/api/v1/fsck'),
}
