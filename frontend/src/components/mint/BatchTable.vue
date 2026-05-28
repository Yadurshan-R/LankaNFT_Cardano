<template>
  <div class="batch-table-wrap">

    <!-- ── Drop zone — click OR drag to add files ── -->
    <div
      class="drop-zone"
      :class="{ 'drop-zone--active': isDragging }"
      @dragover.prevent="isDragging = true"
      @dragleave.prevent="isDragging = false"
      @drop.prevent="onDrop"
      @click="triggerFolderInput"
    >
      <div class="drop-zone-content">
        <UploadCloud :size="28" color="#534AB7" style="margin-bottom:8px" />
        <p class="drop-title">Drag & drop images or a folder here</p>
        <p class="drop-sub">Or click to browse files</p>
      </div>
      <!-- Hidden file input triggered on click -->
      <input
        ref="folderInputRef"
        type="file"
        accept="image/*"
        multiple
        style="display:none"
        @change="onMultiFileUpload"
      />
    </div>

    <!-- ── Toolbar ── -->
    <div class="table-toolbar">
      <span class="item-count">{{ rows.length }} item{{ rows.length !== 1 ? 's' : '' }}</span>
      <div class="toolbar-actions">
        <label class="toolbar-btn">
          <ImageIcon :size="13" /> Add Images
          <input type="file" accept="image/*" multiple style="display:none" @change="onMultiFileUpload" />
        </label>
        <label class="toolbar-btn">
          <FileText :size="13" /> Import CSV
          <input type="file" accept=".csv" style="display:none" @change="importCSV" />
        </label>
        <button class="toolbar-btn" @click="addRow">+ Add Row</button>
        <button v-if="rows.length > 0" class="toolbar-btn toolbar-btn--danger" @click="clearAll">
          <Trash2 :size="13" /> Clear All
        </button>
      </div>
    </div>

    <!-- ── Table ── -->
    <div class="table-scroll">
      <table class="batch-table">
        <thead>
          <tr>
            <th style="width:36px">#</th>
            <th style="width:52px">Image</th>
            <th>NFT Name *</th>
            <th>Description</th>
            <th style="width:100px">Royalties (%)</th>
            <th style="width:80px">Supply</th>
            <th style="width:36px"></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(row, i) in rows" :key="row.id">
            <td class="row-num">{{ i + 1 }}</td>
            <td>
              <label class="img-upload" title="Click to change image">
                <img v-if="row.previewUrl" :src="row.previewUrl" class="img-preview" alt="" />
                <span v-else class="img-placeholder"><ImageIcon :size="16" color="#ccc" /></span>
                <input
                  type="file"
                  accept="image/*"
                  style="display:none"
                  @change="(e: Event) => onSingleFileChange(e, i)"
                />
              </label>
            </td>
            <td>
              <input
                v-model="row.name"
                class="cell-input"
                placeholder="Enter name"
                maxlength="28"
              />
              <span class="name-count" :class="{ 'name-count--warn': row.name.length > 24 }">
                {{ row.name.length }}/28
              </span>
            </td>
            <td>
              <input
                v-model="row.description"
                class="cell-input"
                placeholder="Enter description"
                maxlength="200"
              />
              <span class="name-count" :class="{ 'name-count--warn': row.description.length > 180 }">
                {{ row.description.length }}/200
              </span>
            </td>
            <td>
              <input
                v-model.number="row.royalties"
                class="cell-input cell-input--narrow"
                type="number" min="0" max="100"
              />
            </td>
            <td>
              <input
                v-model.number="row.supply"
                class="cell-input cell-input--narrow"
                type="number" min="1"
              />
            </td>
            <td>
              <button class="delete-btn" @click="removeRow(i)" title="Remove row">
                <X :size="13" />
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- ── Empty state ── -->
    <div v-if="rows.length === 0" class="empty-state">
      <p>No items yet. Drag images above or click Add Images.</p>
    </div>

  </div>
</template>

<script setup lang="ts">
// ─────────────────────────────────────────────────────────────────────────────
// BatchTable.vue
//
// Editable table for batch NFT minting. Supports:
//   - Drag & drop images or folders onto the drop zone
//   - Click drop zone to browse and select files
//   - Per-row image upload via clicking the image cell
//   - CSV import for bulk metadata entry
//   - Manual row addition and deletion
//
// Emits 'change' whenever rows are mutated so the parent can react.
// ─────────────────────────────────────────────────────────────────────────────

import { ref } from 'vue'
import { FileText, Image as ImageIcon, Trash2, UploadCloud, X } from 'lucide-vue-next'

export interface BatchRow {
  id:          number
  name:        string
  description: string
  royalties:   number
  supply:      number
  file:        File | null
  previewUrl:  string | null
}

const emit = defineEmits<{ (e: 'change', rows: BatchRow[]): void }>()

let nextId = 1
const rows          = ref<BatchRow[]>([])
const isDragging    = ref(false)
const folderInputRef = ref<HTMLInputElement | null>(null)

// Triggers the hidden file input when user clicks the drop zone
function triggerFolderInput(e: MouseEvent) {
  // Don't trigger if user clicked a label inside (e.g. toolbar buttons)
  if ((e.target as HTMLElement).closest('label')) return
  folderInputRef.value?.click()
}

function createRow(file?: File): BatchRow {
  const rowId = nextId++
  const row: BatchRow = {
    id:          rowId,
    name:        file ? fileNameWithoutExtension(file.name) : '',
    description: '',
    royalties:   5,
    supply:      1,
    file:        file ?? null,
    previewUrl:  null,
  }
  if (file && file.type.startsWith('image/')) {
    const reader = new FileReader()
    reader.onload = (ev) => {
      const reactiveRow = rows.value.find((r) => r.id === rowId)
      if (reactiveRow) reactiveRow.previewUrl = ev.target?.result as string
    }
    reader.readAsDataURL(file)
  }
  return row
}

// Strips extension and cleans up filename for use as default NFT name
function fileNameWithoutExtension(filename: string): string {
  return filename
    .replace(/\.[^/.]+$/, '')
    .replace(/[_-]/g, ' ')
    .slice(0, 28)
}

function notifyChange() { emit('change', rows.value) }

function addFiles(files: File[]) {
  const imageFiles = files
    .filter((f) => f.type.startsWith('image/'))
    .sort((a, b) => a.name.localeCompare(b.name))
  imageFiles.forEach((file) => rows.value.push(createRow(file)))
  if (imageFiles.length > 0) notifyChange()
}

function addRow()              { rows.value.push(createRow()); notifyChange() }
function removeRow(i: number)  { rows.value.splice(i, 1);      notifyChange() }
function clearAll()            { rows.value = [];               notifyChange() }

// ── Drop handling ─────────────────────────────────────────────────────────────

async function onDrop(e: DragEvent) {
  isDragging.value = false
  const items = Array.from(e.dataTransfer?.items ?? [])
  const allFiles: File[] = []

  await Promise.all(items.map(async (item) => {
    const entry = item.webkitGetAsEntry?.()
    if (!entry) return
    if (entry.isFile) {
      const file = await getFileFromEntry(entry as FileSystemFileEntry)
      if (file) allFiles.push(file)
    } else if (entry.isDirectory) {
      const files = await readDirectoryFiles(entry as FileSystemDirectoryEntry)
      allFiles.push(...files)
    }
  }))

  addFiles(allFiles)
}

function getFileFromEntry(entry: FileSystemFileEntry): Promise<File | null> {
  return new Promise((resolve) => {
    entry.file((file) => resolve(file), () => resolve(null))
  })
}

async function readDirectoryFiles(dirEntry: FileSystemDirectoryEntry): Promise<File[]> {
  return new Promise((resolve) => {
    const reader   = dirEntry.createReader()
    const allFiles: File[] = []

    function readBatch() {
      reader.readEntries(async (entries) => {
        if (entries.length === 0) { resolve(allFiles); return }
        for (const entry of entries) {
          if (entry.isFile) {
            const file = await getFileFromEntry(entry as FileSystemFileEntry)
            if (file) allFiles.push(file)
          } else if (entry.isDirectory) {
            const nested = await readDirectoryFiles(entry as FileSystemDirectoryEntry)
            allFiles.push(...nested)
          }
        }
        readBatch()
      })
    }
    readBatch()
  })
}

// ── File input handlers ───────────────────────────────────────────────────────

function onMultiFileUpload(e: Event) {
  const files = Array.from((e.target as HTMLInputElement).files ?? [])
  addFiles(files)
  ;(e.target as HTMLInputElement).value = ''
}

function onSingleFileChange(e: Event, index: number) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  const row = rows.value[index]
  if (!row) return
  row.file = file
  row.name = row.name || fileNameWithoutExtension(file.name)
  if (file.type.startsWith('image/')) {
    const reader = new FileReader()
    reader.onload = (ev) => { row.previewUrl = ev.target?.result as string }
    reader.readAsDataURL(file)
  }
  notifyChange()
}

// ── CSV import ────────────────────────────────────────────────────────────────

function importCSV(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return

  const reader = new FileReader()
  reader.onload = (ev) => {
    const text    = ev.target?.result as string
    const lines   = text.split('\n').filter((l) => l.trim())
    if (lines.length < 2) return

    const headers = (lines[0] ?? '').split(',').map((h) => h.trim().toLowerCase())

    for (let i = 1; i < lines.length; i++) {
      const values   = (lines[i] ?? '').split(',').map((v) => v.trim().replace(/^"|"$/g, ''))
      const rowIndex = i - 1
      if (rowIndex >= rows.value.length) rows.value.push(createRow())
      const row = rows.value[rowIndex]
      if (!row) continue
      headers.forEach((header, j) => {
        const val = values[j] ?? ''
        if (header === 'name')        row.name        = val
        if (header === 'description') row.description = val
        if (header === 'royalties')   row.royalties   = parseFloat(val) || 5
        if (header === 'supply')      row.supply      = parseInt(val)   || 1
      })
    }
    notifyChange()
  }
  reader.readAsText(file)
  ;(e.target as HTMLInputElement).value = ''
}

defineExpose({ rows })
</script>

<style scoped>
.batch-table-wrap { width: 100%; }

/* ── Drop zone — consistent with UploadAsset.vue ── */
.drop-zone {
  border: 2px dashed #d8d8d8;
  border-radius: 12px;
  padding: 28px 20px;
  text-align: center;
  margin-bottom: 16px;
  cursor: pointer;
  transition: all 0.2s;
}
.drop-zone:hover,
.drop-zone--active {
  border-color: #534AB7;
  background: #EEEDFE;
}
.drop-zone-content {
  pointer-events: none;
  display: flex; flex-direction: column; align-items: center;
}
.drop-title {
  font-size: 13px; font-weight: 500; color: #444;
  margin: 0 0 4px;
}
.drop-sub { font-size: 12px; color: #888; margin: 0; }

/* ── Toolbar ── */
.table-toolbar {
  display: flex; justify-content: space-between; align-items: center;
  margin-bottom: 12px; flex-wrap: wrap; gap: 8px;
}
.item-count { font-size: 13px; color: #666; }
.toolbar-actions { display: flex; gap: 8px; flex-wrap: wrap; }

.toolbar-btn {
  padding: 6px 12px;
  border: 1.5px solid #e0e0e0; border-radius: 8px;
  font-size: 12px; cursor: pointer; background: #fff;
  font-weight: 500; white-space: nowrap;
  display: inline-flex; align-items: center; gap: 4px;
  transition: all 0.15s;
}
.toolbar-btn:hover          { border-color: #534AB7; color: #534AB7; }
.toolbar-btn--danger:hover  { border-color: #d32f2f; color: #d32f2f; }

/* ── Table ── */
.table-scroll {
  overflow-x: auto;
  border: 1px solid #f0f0f0;
  border-radius: 10px;
}
.batch-table {
  width: 100%; border-collapse: collapse; font-size: 13px;
}
.batch-table th {
  text-align: left; padding: 10px 12px;
  border-bottom: 1.5px solid #e0e0e0;
  font-size: 12px; color: #666; font-weight: 500;
  white-space: nowrap; background: #fafafa;
}
.batch-table td {
  padding: 6px 8px;
  border-bottom: 1px solid #f0f0f0;
  vertical-align: middle;
}
.batch-table tr:last-child td { border-bottom: none; }
.batch-table tr:hover td { background: #fafafa; }

.row-num { color: #bbb; font-size: 12px; text-align: center; }

.cell-input {
  width: 100%; padding: 6px 8px;
  border: 1px solid #e8e8e8; border-radius: 6px;
  font-size: 13px; outline: none;
  min-width: 120px; box-sizing: border-box;
  transition: border-color 0.15s;
}
.cell-input--narrow { min-width: 60px; width: 70px; }
.cell-input:focus   { border-color: #534AB7; }

/* Image upload cell */
.img-upload {
  width: 40px; height: 40px;
  border: 1.5px dashed #ccc; border-radius: 6px;
  display: flex; align-items: center; justify-content: center;
  cursor: pointer; overflow: hidden; flex-shrink: 0;
  transition: border-color 0.15s;
}
.img-upload:hover  { border-color: #534AB7; }
.img-preview       { width: 40px; height: 40px; object-fit: cover; }
.img-placeholder   { display: flex; align-items: center; justify-content: center; }

.delete-btn {
  background: none; border: none; color: #ccc;
  cursor: pointer; padding: 4px 8px; border-radius: 4px;
  display: flex; align-items: center; transition: all 0.15s;
}
.delete-btn:hover { color: #d32f2f; background: #fff0f0; }

/* Name/desc character count */
.name-count          { font-size: 10px; color: #bbb; display: block; margin-top: 2px; }
.name-count--warn    { color: #d32f2f; }

/* Empty state */
.empty-state {
  text-align: center; padding: 32px; color: #aaa;
  font-size: 13px; border: 1.5px dashed #e0e0e0;
  border-radius: 10px; margin-top: 12px;
}
</style>