<template>
  <div class="batch-table-wrap">
    <!-- Toolbar -->
    <div class="table-toolbar">
      <span class="item-count">{{ rows.length }} items</span>
      <div class="toolbar-actions">
        <!-- Upload whole folder at once -->
        <label class="toolbar-btn">
          📁 Upload Folder
          <input
            type="file"
            accept="image/*"
            multiple
            webkitdirectory
            style="display:none"
            @change="onFolderUpload"
          />
        </label>

        <!-- Upload individual files -->
        <label class="toolbar-btn">
          🖼 Add Images
          <input
            type="file"
            accept="image/*"
            multiple
            style="display:none"
            @change="onMultiFileUpload"
          />
        </label>

        <!-- Import metadata from CSV -->
        <label class="toolbar-btn">
          📄 Import CSV
          <input type="file" accept=".csv" style="display:none" @change="importCSV" />
        </label>

        <!-- Add empty row -->
        <button class="toolbar-btn" @click="addRow">+ Add Row</button>
      </div>
    </div>

    <!-- Table -->
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

            <!-- Image upload per row -->
            <td>
              <label class="img-upload">
                <img v-if="row.previewUrl" :src="row.previewUrl" class="img-preview" alt="" />
                <span v-else class="img-placeholder">+</span>
                <input
                  type="file"
                  accept="image/*"
                  style="display:none"
                  @change="(e: Event) => onSingleFileChange(e, i)"
                />
              </label>
            </td>

            <!-- Editable name -->
            <td>
              <input
                v-model="row.name"
                class="cell-input"
                placeholder="Enter name"
              />
            </td>

            <!-- Editable description -->
            <td>
              <input
                v-model="row.description"
                class="cell-input"
                placeholder="Enter description"
              />
            </td>

            <!-- Royalties -->
            <td>
              <input
                v-model.number="row.royalties"
                class="cell-input cell-input--narrow"
                type="number"
                min="0"
                max="100"
              />
            </td>

            <!-- Supply -->
            <td>
              <input
                v-model.number="row.supply"
                class="cell-input cell-input--narrow"
                type="number"
                min="1"
              />
            </td>

            <!-- Delete row -->
            <td>
              <button class="delete-btn" @click="removeRow(i)">✕</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Empty state -->
    <div v-if="rows.length === 0" class="empty-state">
      <p>No items yet. Upload a folder or add rows manually.</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

// ─── Types ────────────────────────────────────────────────────────────────────

export interface BatchRow {
  id: number
  name: string
  description: string
  royalties: number
  supply: number
  file: File | null
  previewUrl: string | null
}

// ─── State ────────────────────────────────────────────────────────────────────

const emit = defineEmits<{
  (e: 'change', rows: BatchRow[]): void
}>()

let nextId = 1
const rows = ref<BatchRow[]>([])

// ─── Helpers ──────────────────────────────────────────────────────────────────

function createRow(file?: File): BatchRow {
  const row: BatchRow = {
    id: nextId++,
    name: file ? fileNameWithoutExtension(file.name) : '',
    description: '',
    royalties: 5,
    supply: 1,
    file: file ?? null,
    previewUrl: null,
  }

  // Generate preview URL for image files
  if (file && file.type.startsWith('image/')) {
    const reader = new FileReader()
    reader.onload = (ev) => {
      row.previewUrl = ev.target?.result as string
    }
    reader.readAsDataURL(file)
  }

  return row
}

// Remove file extension and clean up the name
function fileNameWithoutExtension(filename: string): string {
  return filename.replace(/\.[^/.]+$/, '').replace(/[_-]/g, ' ')
}

function notifyChange() {
  emit('change', rows.value)
}

// ─── Actions ──────────────────────────────────────────────────────────────────

function addRow() {
  rows.value.push(createRow())
  notifyChange()
}

function removeRow(index: number) {
  rows.value.splice(index, 1)
  notifyChange()
}

// Upload entire folder — creates one row per image file
function onFolderUpload(e: Event) {
  const files = Array.from((e.target as HTMLInputElement).files ?? [])
  if (files.length === 0) return

  const imageFiles = files
    .filter((f) => f.type.startsWith('image/'))
    .sort((a, b) => a.name.localeCompare(b.name))

  imageFiles.forEach((file) => {
    rows.value.push(createRow(file))
  })

  notifyChange()
  ;(e.target as HTMLInputElement).value = ''
}

// Upload multiple individual files
function onMultiFileUpload(e: Event) {
  const files = Array.from((e.target as HTMLInputElement).files ?? [])
  files.forEach((file) => {
    rows.value.push(createRow(file))
  })
  notifyChange()
  ;(e.target as HTMLInputElement).value = ''
}

// Change image for a single row
function onSingleFileChange(e: Event, index: number) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return

  // Guard against undefined row
  const row = rows.value[index]
  if (!row) return

  row.file = file
  row.name = row.name || fileNameWithoutExtension(file.name)

  if (file.type.startsWith('image/')) {
    const reader = new FileReader()
    reader.onload = (ev) => {
      row.previewUrl = ev.target?.result as string
    }
    reader.readAsDataURL(file)
  }

  notifyChange()
}

// Import metadata from CSV
// Expected columns: name, description, royalties, supply
// Matches rows by order — CSV row 1 = table row 1
function importCSV(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return

  const reader = new FileReader()
  reader.onload = (ev) => {
    const text = ev.target?.result as string
    const lines = text.split('\n').filter((l) => l.trim())
    if (lines.length < 2) return

    const headers = (lines[0] ?? '').split(',').map((h) => h.trim().toLowerCase())

    for (let i = 1; i < lines.length; i++) {
      // Guard against undefined line
      const line = lines[i] ?? ''
      const values = line.split(',').map((v) => v.trim().replace(/^"|"$/g, ''))
      const rowIndex = i - 1

      // If row doesn't exist yet, create one
      if (rowIndex >= rows.value.length) {
        rows.value.push(createRow())
      }

      const row = rows.value[rowIndex]
      if (!row) continue

      headers.forEach((header, j) => {
        const val = values[j] ?? ''
        if (header === 'name') row.name = val
        if (header === 'description') row.description = val
        if (header === 'royalties') row.royalties = parseFloat(val) || 5
        if (header === 'supply') row.supply = parseInt(val) || 1
      })
    }

    notifyChange()
  }
  reader.readAsText(file)
  ;(e.target as HTMLInputElement).value = ''
}

// Expose rows so parent can read them
defineExpose({ rows })
</script>

<style scoped>
.batch-table-wrap { width: 100%; }

.table-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  flex-wrap: wrap;
  gap: 8px;
}
.item-count { font-size: 13px; color: #666; }
.toolbar-actions { display: flex; gap: 8px; flex-wrap: wrap; }
.toolbar-btn {
  padding: 6px 12px;
  border: 1.5px solid #e0e0e0;
  border-radius: 8px;
  font-size: 12px;
  cursor: pointer;
  background: #fff;
  font-weight: 500;
  white-space: nowrap;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.toolbar-btn:hover { border-color: #534AB7; color: #534AB7; }

.table-scroll { overflow-x: auto; border: 1px solid #f0f0f0; border-radius: 10px; }
.batch-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.batch-table th {
  text-align: left;
  padding: 10px 12px;
  border-bottom: 1.5px solid #e0e0e0;
  font-size: 12px;
  color: #666;
  font-weight: 500;
  white-space: nowrap;
  background: #fafafa;
}
.batch-table td {
  padding: 6px 8px;
  border-bottom: 1px solid #f0f0f0;
  vertical-align: middle;
}
.batch-table tr:last-child td { border-bottom: none; }

.row-num { color: #bbb; font-size: 12px; text-align: center; }

.cell-input {
  width: 100%;
  padding: 6px 8px;
  border: 1px solid #e8e8e8;
  border-radius: 6px;
  font-size: 13px;
  outline: none;
  min-width: 120px;
  box-sizing: border-box;
}
.cell-input--narrow { min-width: 60px; width: 70px; }
.cell-input:focus { border-color: #534AB7; }

.img-upload {
  width: 40px;
  height: 40px;
  border: 1.5px dashed #ccc;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  overflow: hidden;
  flex-shrink: 0;
}
.img-upload:hover { border-color: #534AB7; }
.img-preview { width: 40px; height: 40px; object-fit: cover; }
.img-placeholder { font-size: 18px; color: #ccc; line-height: 1; }

.delete-btn {
  background: none;
  border: none;
  color: #ccc;
  cursor: pointer;
  font-size: 13px;
  padding: 4px 8px;
  border-radius: 4px;
}
.delete-btn:hover { color: #d32f2f; background: #fff0f0; }

.empty-state {
  text-align: center;
  padding: 40px;
  color: #aaa;
  font-size: 14px;
  border: 1.5px dashed #e0e0e0;
  border-radius: 10px;
  margin-top: 12px;
}
</style>