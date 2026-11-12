<template>
  <div class="w-full h-full flex flex-col bg-dark-900 overflow-hidden font-sans text-xs select-none">
    <!-- Subheader Toolbar: Result Set Info, Quick Filter, Warnings, Actions -->
    <div
      v-if="!hideToolbar"
      class="h-8 bg-dark-850 border-b border-dark-700 flex items-center justify-between px-2 flex-shrink-0 space-x-2"
    >
      <!-- Left: Result Set Label & Quick Filter -->
      <div class="flex items-center space-x-2 min-w-0">
        <!-- Multiple Result Set Index Badge -->
        <SqChip
          v-if="totalSets > 1"
          tone="neutral"
          :label="`Result #${setIndex + 1} (${resultSet.rowCount ?? resultSet.rows.length})`"
        >
          <template #icon>
            <Table2 class="w-3 h-3" />
          </template>
        </SqChip>

        <!-- Quick Filter Input: prefix/suffix slots replace PrimeVue's IconField/InputIcon pair -->
        <SqInput
          v-model="quickFilterInput"
          type="text"
          :placeholder="hasRows ? $t('results.filterPlaceholder') : $t('results.noDataToSearch')"
          :disabled="!hasRows"
          size="sm"
          class="w-44 sm:w-56"
        >
          <template #prefix>
            <Search class="w-3.5 h-3.5 text-dark-500" />
          </template>
          <template #suffix>
            <button
              v-if="quickFilterInput"
              type="button"
              class="flex items-center justify-center text-dark-500 hover:text-dark-100 transition-colors"
              :title="$t('results.clearFilter')"
              @click="clearQuickFilter"
            >
              <X class="w-3 h-3" />
            </button>
          </template>
        </SqInput>

        <!-- Truncation Warning Badge (when max rows limit reached) -->
        <SqChip
          v-if="resultSet.isTruncated"
          tone="warn"
          :label="$t('results.truncatedWarning', { count: resultSet.rowCount.toLocaleString(), total: (resultSet.totalCount ?? resultSet.rowCount).toLocaleString() })"
          class="hidden md:inline-flex"
        >
          <template #icon>
            <TriangleAlert class="w-3 h-3" />
          </template>
        </SqChip>

        <!-- Estimated Plan Badge -->
        <SqChip
          v-if="queryStore.activeResultTab?.isShowplan"
          tone="plan"
          :label="$t('results.estimatedPlan')"
          class="hidden md:inline-flex"
        >
          <template #icon>
            <Network class="w-3 h-3" />
          </template>
        </SqChip>

        <div class="h-3.5 w-px bg-dark-750 mx-1 flex-shrink-0"></div>

        <!-- Inline Editing Toolbar: Add / Delete / Revert / Commit -->
        <div class="flex items-center space-x-1 flex-shrink-0">
          <template v-if="editability.canEdit">
            <!-- Add Row Button -->
            <SqButton
              type="button"
              :aria-label="$t('results.addRow')"
              size="sm"
              tone="ok"
              variant="outline"
              :disabled="!canAddRow"
              @click="handleAddRow"
              :v-tooltip.top="$t('results.addRowTooltip')"
            >
              <template #icon><Plus class="w-3.5 h-3.5" /></template>
            </SqButton>

            <!-- Delete Row Button -->
            <SqButton
              type="button"
              :aria-label="$t('results.deleteRows')"
              size="sm"
              tone="danger"
              variant="outline"
              :disabled="!canDeleteRows"
              @click="handleDeleteRows"
              :v-tooltip.top="$t('results.deleteRowsTooltip')"
            >
              <template #icon><Trash2 class="w-3.5 h-3.5" /></template>
            </SqButton>

            <!-- Revert Button -->
            <SqButton
              type="button"
              :label="$t('common.revert')"
              size="sm"
              :tone="pendingRowCount > 0 ? 'warn' : 'neutral'"
              variant="outline"
              :disabled="pendingRowCount === 0"
              @click="handleRevertChanges"
              :v-tooltip.top="$t('results.revertAllTooltip')"
            >
              <template #icon><Undo2 class="w-3.5 h-3.5" /></template>
            </SqButton>

            <!-- Commit Button -->
            <SqButton
              type="button"
              :label="pendingRowCount > 0 ? $t('results.commitWithCount', { count: pendingRowCount }) : $t('common.commit')"
              size="sm"
              :tone="pendingRowCount > 0 ? 'ok' : 'neutral'"
              :variant="pendingRowCount > 0 ? 'solid' : 'outline'"
              :disabled="pendingRowCount === 0"
              :loading="commitModal.isExecuting"
              @click="handleCommitAction"
              :v-tooltip.top="$t('results.commitChangesTooltip')"
              class="font-semibold"
            >
              <template #icon><Check class="w-3.5 h-3.5" /></template>
            </SqButton>
          </template>

          <!-- Read-only Indicator when editing is not supported -->
          <template v-else>
            <SqChip
              tone="neutral"
              :label="editability.shortReason || $t('results.readOnly')"
              v-tooltip.top="editability.reason"
              class="select-none"
            >
              <template #icon>
                <Lock class="w-3 h-3" />
              </template>
            </SqChip>
          </template>
        </div>
      </div>

      <!-- Right: Copy Tools, Row Stats & Maximize Toggle -->
      <div class="flex items-center space-x-1.5 flex-shrink-0">
        <!-- Refresh Button -->
        <SqButton
          type="button"
          :label="isRefreshing ? $t('common.refreshing') : $t('common.refresh')"
          size="sm"
          variant="outline"
          :loading="isRefreshing"
          :disabled="isRefreshing"
          @click="handleRefresh"
          :v-tooltip.top="isRefreshing ? $t('common.refreshing') : $t('results.refreshTooltip')"
        >
          <template #icon><RefreshCw class="w-3.5 h-3.5" /></template>
        </SqButton>

        <CopyFormatMenu :copied="copyJustCompleted" @copy="handleCopyFormat" />


        <!-- Maximize / Restore Toggle (when multiple result sets) -->
        <SqButton
          v-if="totalSets > 1"
          type="button"
          size="sm"
          class="ml-1"
          :aria-label="isMaximized ? $t('results.restoreGrid') : $t('results.maximizeGrid')"
          @click="$emit('toggle-maximize')"
          :v-tooltip.top="isMaximized ? $t('results.restoreGrid') : $t('results.maximizeGrid')"
        >
          <template #icon>
            <SquareMinus v-if="isMaximized" class="w-3.5 h-3.5" />
            <SquarePlus v-else class="w-3.5 h-3.5" />
          </template>
        </SqButton>
      </div>
    </div>

    <!-- Empty State: idle (nothing run yet) vs a statement that matched zero rows -->
    <ResultsEmptyState
      v-if="!resultSet || !hasRows"
      :variant="resultSetColumnCount > 0 ? 'no-rows' : 'idle'"
      :column-count="resultSetColumnCount"
    >
      <!-- A zero-row result keeps its columns, so an editable single table can still accept a
           first row even though Tabulator is not mounted yet. -->
      <SqButton
        v-if="editability.canEdit"
        type="button"
        :label="$t('results.addFirstRow')"
        size="sm"
        tone="ok"
        variant="outline"
        class="mt-3 font-semibold"
        :disabled="!canAddRow"
        @click="handleAddRow"
        :v-tooltip.top="$t('results.addRowTooltip')"
      >
        <template #icon><Plus class="w-3.5 h-3.5" /></template>
      </SqButton>
    </ResultsEmptyState>

    <!-- Tabulator grid -->
    <div
      v-else
      ref="gridContainerRef"
      class="sqlight-grid flex-1 w-full overflow-hidden relative"
      :class="{ 'is-h-scrolling': isHorizontalScrolling }"
      :style="{
        '--sqlight-grid-font': settingsStore.gridFontFamily,
        '--sqlight-grid-font-size': `${settingsStore.gridFontSize}px`,
      }"
      @contextmenu.prevent
      @keydown="handleGridKeydown"
      @scroll.capture.passive="handleGridScroll"
    >
      <div ref="gridTableRef" class="w-full h-full"></div>
    </div>

    <!-- Excel-Grade Live Aggregate Bar -->
    <div
      v-if="!hideToolbar"
      class="h-6 bg-dark-850 border-t border-dark-700 flex items-center justify-between px-3 text-xxs font-sans tabular-nums text-dark-300 flex-shrink-0 select-none"
    >
      <!-- Left: Statistics or Default Summary -->
      <div class="flex items-center space-x-2.5 overflow-x-auto min-w-0">
        <template v-if="selectionStats">
          <div class="flex items-center space-x-1 font-semibold text-accent flex-shrink-0">
            <span>{{ $t('results.selected') }}:</span>
            <span v-if="selectedColumnsCount > 1" class="text-warn font-mono">
              {{ selectedColumnsCount }} {{ $t('results.columnsUnit') }}
            </span>
            <span class="font-mono text-dark-100">
              {{ selectedColumnsCount > 1 ? `(${selectionStats.totalCells.toLocaleString()} ${$t('results.cellsUnit')})` : `${selectionStats.totalCells.toLocaleString()} ${$t('results.cellsUnit')}` }}
            </span>
            <span v-if="selectionStats.numericCount > 0" class="text-dark-400 font-mono text-xxs font-medium">
              [{{ selectionStats.numericCount.toLocaleString() }} {{ $t('results.numericUnit') }}]
            </span>
          </div>

          <template v-if="selectionStats.numericCount > 0">
            <span class="text-dark-600 flex-shrink-0">|</span>
            <div class="flex-shrink-0">
              {{ $t('results.sum') }}: <strong class="font-mono text-ok">{{ formatAggregateNumber(selectionStats.sum) }}</strong>
            </div>
            <span class="text-dark-600 flex-shrink-0">|</span>
            <div class="flex-shrink-0">
              {{ $t('results.avg') }}: <strong class="font-mono text-info">{{ formatAggregateNumber(selectionStats.avg) }}</strong>
            </div>
            <span class="text-dark-600 flex-shrink-0">|</span>
            <div class="flex-shrink-0">
              {{ $t('results.min') }}: <strong class="font-mono text-warn">{{ formatAggregateNumber(selectionStats.min) }}</strong>
            </div>
            <span class="text-dark-600 flex-shrink-0">|</span>
            <div class="flex-shrink-0">
              {{ $t('results.max') }}: <strong class="font-mono text-plan">{{ formatAggregateNumber(selectionStats.max) }}</strong>
            </div>
          </template>

          <span class="text-dark-600 flex-shrink-0">|</span>
          <div class="flex-shrink-0">
            {{ $t('results.distinct') }}: <strong class="font-mono text-dark-100">{{ selectionStats.distinctCount.toLocaleString() }}</strong>
          </div>

          <SqButton
            type="button"
            :label="$t('common.clear')"
            size="sm"
            @click="clearCellSelection"
            :v-tooltip.top="$t('results.clearSelectionTooltip')"
            class="ml-1 font-medium underline"
          />
        </template>

        <template v-else>
          <div class="flex items-center space-x-2 text-dark-400">
            <span>{{ $t('results.totalRows', { count: resultSet.rows.length.toLocaleString() }) }}</span>
            <span class="text-dark-600">|</span>
            <span>{{ $t('results.totalColumns', { count: resultSet.columns.length }) }}</span>
            <template v-if="displayDuration !== undefined">
              <span class="text-dark-600">|</span>
              <span>Duration: <strong class="text-accent font-mono">{{ displayDuration }}ms</strong></span>
            </template>
            <template v-if="pendingRowCount > 0">
              <span class="text-dark-600">|</span>
              <span class="text-warn font-medium">
                {{ $t('results.pendingChanges', { inserted: pendingInsertCount, updated: modifiedRowCount, deleted: pendingDeleteCount }) }}
              </span>
            </template>
          </div>
        </template>
      </div>

    </div>

    <!-- Custom Context Menu for Cells & Column Pinning -->
    <div
      v-if="contextMenu.visible"
      :style="{ top: `${contextMenu.y}px`, left: `${contextMenu.x}px` }"
      class="fixed z-50 bg-dark-800 border border-dark-700 rounded shadow-overlay py-1 w-52 text-xs font-sans text-dark-200 select-none"
      @click="contextMenu.visible = false"
    >
      <div class="px-2.5 py-1 text-xxs text-dark-400 border-b border-dark-750 font-mono truncate">
        {{ contextMenu.colName }}: {{ String(contextMenu.cellValue ?? 'NULL') }}
      </div>

      <button
        @click="copyCellValue"
        class="w-full text-left px-2.5 py-1.5 hover:bg-dark-750 hover:text-dark-100 flex items-center space-x-2 transition-colors"
      >
        <Copy class="w-3.5 h-3.5 text-accent" />
        <span>{{ $t('results.copyCell') }}</span>
      </button>

      <button
        @click="copyColumnName"
        class="w-full text-left px-2.5 py-1.5 hover:bg-dark-750 hover:text-dark-100 flex items-center space-x-2 transition-colors"
      >
        <Heading class="w-3.5 h-3.5 text-plan" />
        <span>{{ $t('results.copyColumnName') }}</span>
      </button>

      <button
        @click="copyCurrentRow"
        class="w-full text-left px-2.5 py-1.5 hover:bg-dark-750 hover:text-dark-100 flex items-center space-x-2 transition-colors"
      >
        <FileText class="w-3.5 h-3.5 text-ok" />
        <span>{{ $t('results.copyRow') }}</span>
      </button>

      <button
        @click="copyCurrentRowAsJson"
        class="w-full text-left px-2.5 py-1.5 hover:bg-dark-750 hover:text-dark-100 flex items-center space-x-2 transition-colors"
      >
        <Braces class="w-3.5 h-3.5 text-ok" />
        <span>{{ $t('results.copyRowJson') }}</span>
      </button>

      <button
        @click="openDataView"
        class="w-full text-left px-2.5 py-1.5 hover:bg-dark-750 hover:text-dark-100 flex items-center space-x-2 transition-colors"
      >
        <Eye class="w-3.5 h-3.5 text-info" />
        <span>{{ $t('results.dataView') }}</span>
      </button>

      <!-- Row / Cell Editing Quick Actions in Context Menu -->
      <template v-if="editability.canEdit">
        <div class="my-1 border-t border-dark-750"></div>

        <button
          v-if="canAddRow"
          @click="handleAddRow"
          class="w-full text-left px-2.5 py-1.5 hover:bg-dark-750 hover:text-dark-100 flex items-center space-x-2 transition-colors"
        >
          <Plus class="w-3.5 h-3.5 text-ok" />
          <span>{{ $t('results.addRow') }}</span>
        </button>

        <button
          v-if="contextMenu.rowData"
          @click="duplicateContextRow"
          class="w-full text-left px-2.5 py-1.5 hover:bg-dark-750 hover:text-dark-100 flex items-center space-x-2 transition-colors"
        >
          <CopyPlus class="w-3.5 h-3.5 text-info" />
          <span>{{ $t('results.duplicateRow') }}</span>
        </button>

        <button
          v-if="contextMenu.rowData"
          @click="toggleContextRowDeletion"
          class="w-full text-left px-2.5 py-1.5 hover:bg-dark-750 flex items-center space-x-2 transition-colors"
          :class="isContextRowDeleted ? 'text-warn' : 'text-danger'"
        >
          <Undo2 v-if="isContextRowDeleted" class="w-3.5 h-3.5" />
          <Trash2 v-else class="w-3.5 h-3.5" />
          <span v-if="isContextRowDeleted">{{ $t('results.undoDeleteRow') }}</span>
          <span v-else-if="isContextRowInserted">{{ $t('results.removeInsertedRow') }}</span>
          <span v-else>{{ $t('results.deleteRow') }}</span>
        </button>

        <button
          v-if="isCurrentCellEditable && isCurrentCellNullable && contextMenu.cellValue !== null"
          @click="setCellNull"
          class="w-full text-left px-2.5 py-1.5 hover:bg-dark-750 hover:text-warn flex items-center space-x-2 transition-colors text-warn"
        >
          <Slash class="w-3.5 h-3.5" />
          <span>{{ $t('results.setNull') }}</span>
        </button>
        <button
          v-if="isCurrentCellEditable && isCurrentCellModified"
          @click="revertSingleCell"
          class="w-full text-left px-2.5 py-1.5 hover:bg-dark-750 hover:text-warn flex items-center space-x-2 transition-colors text-warn"
        >
          <Undo2 class="w-3.5 h-3.5" />
          <span>{{ $t('results.revertCell') }}</span>
        </button>
      </template>

      <div class="my-1 border-t border-dark-750"></div>

      <!-- Selection Copy (if selection active) -->
      <button
        v-if="hasSelection"
        @click="copySelectedCells"
        class="w-full text-left px-2.5 py-1.5 hover:bg-dark-750 hover:text-dark-100 flex items-center space-x-2 transition-colors"
      >
        <Copy class="w-3.5 h-3.5 text-accent" />
        <span>{{ $t('results.copySelection') }} ({{ selectionStats?.totalCells }} {{ $t('results.cellsUnit') }})</span>
      </button>

      <button
        v-if="hasSelection"
        @click="copySelectedAsJson"
        class="w-full text-left px-2.5 py-1.5 hover:bg-dark-750 hover:text-dark-100 flex items-center space-x-2 transition-colors"
      >
        <Braces class="w-3.5 h-3.5 text-er" />
        <span>{{ $t('results.copySelectionAsJson') }}</span>
      </button>

      <!-- DML SQL Generation Options -->
      <button
        @click="handleGenerateDml('INSERT')"
        class="w-full text-left px-2.5 py-1.5 hover:bg-dark-750 hover:text-dark-100 flex items-center space-x-2 transition-colors"
      >
        <PlusCircle class="w-3.5 h-3.5 text-info" />
        <span>{{ $t('results.generateInsert') }}</span>
      </button>

      <button
        @click="handleGenerateDml('UPDATE')"
        class="w-full text-left px-2.5 py-1.5 hover:bg-dark-750 hover:text-dark-100 flex items-center space-x-2 transition-colors"
      >
        <Edit3 class="w-3.5 h-3.5 text-warn" />
        <span>{{ $t('results.generateUpdate') }}</span>
      </button>

      <button
        @click="handleGenerateDml('DELETE')"
        class="w-full text-left px-2.5 py-1.5 hover:bg-dark-750 hover:text-dark-100 flex items-center space-x-2 transition-colors"
      >
        <Trash2 class="w-3.5 h-3.5 text-danger" />
        <span>{{ $t('results.generateDelete') }}</span>
      </button>
    </div>

    <!-- Commit Confirmation Modal -->
    <SqDialog
      :visible="commitModal.visible"
      @update:visible="(val) => !val && closeCommitModal()"
      :closable="false"
      :dismissableMask="true"
      :showHeader="false"
      class="w-[92vw] max-w-5xl h-[88vh] max-h-[850px]"
      :class="[
        dialogWindow.marker,
        requiresModificationPrompt
          ? (commitModal.confirmStep === 2 ? 'sq-dialog-accent-danger' : 'sq-dialog-accent-warn')
          : ''
      ]"
      contentClass="!p-0 h-full flex flex-col"
      @show="dialogWindow.onShow"
      @hide="dialogWindow.onHide"
    >
      <!-- Modal Header -->
      <div
        class="sq-dialog-drag-handle px-5 py-3.5 border-b flex items-center justify-between flex-shrink-0"
        :class="requiresModificationPrompt && commitModal.confirmStep === 2 ? 'bg-rose-50 dark:bg-rose-950/30 border-rose-200 dark:border-rose-900/50' : 'bg-dark-800 border-dark-750'"
      >
        <div class="flex items-center space-x-2.5">
          <div
            class="w-7 h-7 rounded-md flex items-center justify-center"
            :class="requiresModificationPrompt ? (commitModal.confirmStep === 2 ? 'bg-rose-100 dark:bg-rose-950 border border-rose-200 dark:border-rose-700/60' : 'bg-amber-500/20 border border-amber-500/40') : 'bg-emerald-100 dark:bg-emerald-950 border border-emerald-200 dark:border-emerald-700/60'"
          >
            <TriangleAlert
              v-if="requiresModificationPrompt && commitModal.confirmStep === 2"
              class="w-3.5 h-3.5 text-danger animate-pulse"
            />
            <TriangleAlert v-else-if="requiresModificationPrompt" class="w-3.5 h-3.5 text-warn" />
            <Check v-else class="w-3.5 h-3.5 text-ok" />
          </div>
          <div>
            <h3 class="font-semibold text-base text-dark-100 leading-tight">
              {{ requiresModificationPrompt ? (commitModal.confirmStep === 2 ? '確認提交資料變更 (高危最終確認 2/2)' : '確認提交資料變更 (修改提示 1/2)') : '確認提交資料變更 (Commit Changes)' }}
            </h3>
            <p class="text-xxs text-dark-400 mt-0.5">
              {{ requiresModificationPrompt ? (commitModal.confirmStep === 2 ? '注意：此操作將直接更動目標資料庫！資料修改後可能無法復原' : '連線已啟用修改提示保護，請核實即將寫入資料庫的交易語法與異動範圍') : '請確認以下即將寫入資料庫的交易 T-SQL 語法與異動範圍' }}
            </p>
          </div>
        </div>
        <SqButton
          type="button"
          tone="danger"
          aria-label="關閉 (Esc)"
          @click="closeCommitModal"
          v-tooltip.top="'關閉 (Esc)'"
        >
          <template #icon><X class="w-3.5 h-3.5" /></template>
        </SqButton>
      </div>

      <!-- Modal Body -->
      <div class="p-5 flex-1 flex flex-col min-h-0 space-y-3.5 text-xs bg-dark-900">
        <!-- Summary Info Strip -->
        <div class="bg-dark-850 border border-dark-750 rounded-md px-4 py-2.5 flex items-center justify-between font-mono flex-shrink-0">
          <div class="flex items-center space-x-2">
            <span class="text-dark-400 text-xxs">目標資料表: </span>
            <strong class="text-accent font-semibold text-xs">[{{ editability.targetTable?.schema }}].[{{ editability.targetTable?.tableName }}]</strong>
          </div>
          <div class="flex items-center space-x-3 text-dark-300 text-xxs">
            <span>新增 <strong class="text-ok font-semibold text-xs">{{ commitModal.insertCount }}</strong> 列</span>
            <span class="text-dark-600">|</span>
            <span>更新 <strong class="text-warn font-semibold text-xs">{{ commitModal.updateCount }}</strong> 列</span>
            <span class="text-dark-600">|</span>
            <span>刪除 <strong class="text-danger font-semibold text-xs">{{ commitModal.deleteCount }}</strong> 列</span>
            <span class="text-dark-600">|</span>
            <span>異動格數: <strong class="text-warn font-semibold text-xs">{{ commitModal.cellCount }}</strong> 格</span>
          </div>
        </div>

        <!-- Error Alert if any -->
        <div v-if="commitModal.error" class="bg-rose-100 dark:bg-rose-950/50 border border-rose-200 dark:border-rose-800 text-danger p-3 rounded-md text-xs flex items-start space-x-2 flex-shrink-0">
          <TriangleAlert class="w-3.5 h-3.5 text-danger flex-shrink-0 mt-0.5" />
          <div class="flex-1 font-mono break-all whitespace-pre-wrap">{{ commitModal.error }}</div>
        </div>

        <!-- Insert coverage warning: a projected subset may omit required table columns. -->
        <div
          v-if="commitModal.insertCount > 0 && missingRequiredInsertColumns.length > 0"
          class="bg-amber-500/10 border border-amber-500/30 text-warn p-2.5 rounded-md text-xxs flex items-start space-x-2 flex-shrink-0"
        >
          <TriangleAlert class="w-3.5 h-3.5 text-warn flex-shrink-0 mt-0.5" />
          <span class="flex-1 leading-relaxed">
            {{ $t('results.insertMissingRequiredColumns', { columns: missingRequiredInsertColumns.join(', ') }) }}
          </span>
        </div>

        <!-- SQL Preview with Monaco Syntax Highlighting -->
        <div class="flex-1 flex flex-col min-h-0">
          <div class="text-xxs text-dark-400 mb-1.5 flex items-center justify-between flex-shrink-0">
            <span class="flex items-center space-x-1.5">
              <span class="font-medium text-dark-200">即將執行的安全交易 T-SQL 語法</span>
              <span class="text-dark-500">(含 @@ROWCOUNT 防護，任一列失敗自動完整 ROLLBACK)</span>
            </span>
            <SqButton
              type="button"
              label="複製語法"
              size="sm"
              variant="outline"
              @click="copyCommitSql"
              v-tooltip.top="'複製語法至剪貼簿'"
            >
              <template #icon><Copy class="w-3.5 h-3.5" /></template>
            </SqButton>
          </div>

          <!-- Monaco SQL Code Viewer Container -->
          <div class="flex-1 min-h-[350px] border border-dark-750 rounded-md overflow-hidden relative">
            <SqlCodeViewer :code="commitModal.sql" language="sql" class="w-full h-full" />
          </div>
        </div>
      </div>

      <!-- Modal Footer -->
      <div class="px-5 py-3 border-t border-dark-750 flex items-center justify-between bg-dark-800 flex-shrink-0">
        <!-- Left: High Risk Warning / Standard Note -->
        <div v-if="requiresModificationPrompt" class="flex-1 min-w-0 mr-4">
          <div
            v-if="commitModal.confirmStep === 1"
            class="flex items-center space-x-2 px-3 py-1.5 rounded bg-amber-500/15 border border-amber-500/30 text-warn text-xxs leading-normal"
          >
            <TriangleAlert class="w-3.5 h-3.5 text-warn shrink-0" />
            <span>
              <strong>高危提醒 (1/2)：</strong>連線「{{ currentConnection?.name }}」已啟用修改提示防護。此操作將直接更動資料庫，需進行 <strong>2 次重複確認</strong> 才可提交！
            </span>
          </div>
          <div
            v-else
            class="flex items-center space-x-2 px-3 py-1.5 rounded bg-rose-100 dark:bg-rose-950/60 border border-rose-200 dark:border-rose-700/80 text-danger text-xxs leading-normal animate-pulse"
          >
            <CircleAlert class="w-3.5 h-3.5 text-danger shrink-0" />
            <span>
              <strong class="text-danger">高危提醒 (2/2 最終確認)：</strong>即將對目標資料表實施實體資料更動！資料修改後可能無法復原，請再次核實無誤後點擊執行。
            </span>
          </div>
        </div>
        <div v-else class="text-xxs text-dark-400">
          提示：所有異動包含在同一交易 (BEGIN TRAN) 中，任何錯誤均完整復原
        </div>

        <!-- Right: Buttons -->
        <div class="flex items-center space-x-2 shrink-0">
          <SqButton
            type="button"
            :label="requiresModificationPrompt && commitModal.confirmStep === 2 ? '放棄提交 (Esc)' : '取消'"
            variant="outline"
            @click="closeCommitModal"
          />

          <!-- Guarded Step 1 Button -->
          <SqButton
            v-if="requiresModificationPrompt && commitModal.confirmStep === 1"
            type="button"
            label="初次確認提交 (1/2)"
            tone="warn"
            variant="outline"
            class="font-semibold"
            @click="commitModal.confirmStep = 2"
          >
            <template #icon><ArrowRight class="w-3.5 h-3.5" /></template>
          </SqButton>

          <!-- Guarded Step 2 Button -->
          <SqButton
            v-else-if="requiresModificationPrompt && commitModal.confirmStep === 2"
            type="button"
            :label="commitModal.isExecuting ? '提交執行中...' : '確定立即提交 (最終確認 2/2)'"
            tone="danger"
            variant="solid"
            :loading="commitModal.isExecuting"
            :disabled="commitModal.isExecuting"
            class="font-semibold"
            @click="executeCommit()"
          >
            <template #icon><TriangleAlert class="w-3.5 h-3.5" /></template>
          </SqButton>

          <!-- Standard Commit Button -->
          <SqButton
            v-else
            type="button"
            :label="commitModal.isExecuting ? '提交執行中...' : `確認提交 (${commitModal.rowCount} 列 / ${commitModal.cellCount} 格)`"
            tone="accent"
            variant="solid"
            :loading="commitModal.isExecuting"
            :disabled="commitModal.isExecuting"
            @click="executeCommit()"
          >
            <template #icon><Check class="w-3.5 h-3.5" /></template>
          </SqButton>
        </div>
      </div>
    </SqDialog>
  </div>
</template>


<script setup lang="ts">
import { ref, shallowRef, computed, reactive, watch, toRaw, nextTick, onMounted, onBeforeUnmount } from 'vue';
import { useI18n } from 'vue-i18n';
import SqButton from '@/components/ui/SqButton.vue';
import CopyFormatMenu from '@/components/common/CopyFormatMenu.vue';
import type { CopyFormat } from '@/types/copyFormat';
import SqInput from '@/components/ui/SqInput.vue';
import SqChip from '@/components/ui/SqChip.vue';
import SqDialog from '@/components/ui/SqDialog.vue';
import {
  Copy,
  CopyPlus,
  FileText,
  Plus,
  PlusCircle,
  Edit3,
  Trash2,
  Braces,
  CircleAlert,
  Slash,
  Undo2,
  Eye,
  Heading,
  X,
  Table2,
  TriangleAlert,
  Network,
  Lock,
  Search,
  Check,
  RefreshCw,
  ArrowRight,
  SquareMinus,
  SquarePlus,
} from '@/components/icons';
import type {
  TabulatorCellComponent,
  TabulatorColumnComponent,
  TabulatorColumnDefinition,
  TabulatorRowComponent,
  TabulatorRowData,
} from 'tabulator-tables';
import { useSettingsStore } from '@/stores/settingsStore';
import { useQueryStore } from '@/stores/queryStore';
import { useWorkspaceStore } from '@/stores/workspaceStore';
import { useConnectionStore } from '@/stores/connectionStore';
import { useSchemaStore } from '@/stores/schemaStore';
import { useDataViewStore } from '@/stores/dataViewStore';
import { useGridLayoutStore } from '@/stores/gridLayoutStore';
import { useDialogWindow } from '@/composables/useDialogWindow';
import { queryService } from '@/services/queryService';
import { checkTableEditability } from '@/utils/tableEditability';
import type { RowModification } from '@/utils/batchUpdateGenerator';
import {
  generateBatchChangesScript,
  type DeleteRowChange,
  type InsertColumnValue,
  type InsertRowChange,
} from '@/utils/batchChangeGenerator';
import SqlCodeViewer from '@/components/common/SqlCodeViewer.vue';
import ResultsEmptyState from '@/components/results/ResultsEmptyState.vue';
import {
  calculateColumnWidth,
  formatCellForExport,
  formatValueForDisplay,
} from '@/composables/useColumnAutoWidth';
import { useTabulatorTable } from '@/composables/useTabulatorTable';
import { useGridSelection } from '@/composables/useGridSelection';
import { useGridExport } from '@/composables/useGridExport';
import { startGridPerfDiag } from '@/composables/useGridPerfDiag';
import {
  ROW_INDEX_FIELD,
  applyCellValueClasses,
  buildDataColumn,
  buildRowIndexColumn,
  isBinaryCellValue,
} from '@/utils/tabulatorColumns';
import { activeRangeCell, editCell } from '@/utils/tabulatorGrid';
import {
  generateInsertStatement,
  generateUpdateStatement,
  generateDeleteStatement,
  parseTargetTableFromSql,
  extractAllTableNamesFromSql,
  type ColumnInfo,
  type GenerateDmlParams,
} from '@/utils/sqlGenerator';
import type { ResultSet, CellValue, QueryResultTab } from '@/types/query';
import type { TableSchema } from '@/types/schema';

const props = defineProps<{
  resultSet: ResultSet;
  setIndex: number;
  totalSets: number;
  queryTab?: QueryResultTab | null;
  isMaximized?: boolean;
  tabId?: string | null;
  /** Hides the grid toolbar and the info bar, leaving only the data area (multi result set tabs). */
  hideToolbar?: boolean;
  durationMs?: number;
}>();

defineEmits<{
  (e: 'toggle-maximize'): void;
}>();

const { t } = useI18n();
const settingsStore = useSettingsStore();
const queryStore = useQueryStore();
const workspaceStore = useWorkspaceStore();
const connectionStore = useConnectionStore();
const schemaStore = useSchemaStore();
const dataViewStore = useDataViewStore();
const gridLayoutStore = useGridLayoutStore();
const dialogWindow = useDialogWindow();

const displayDuration = computed<number | undefined>(() => {
  if (props.durationMs !== undefined) return props.durationMs;
  if (props.queryTab?.durationMs !== undefined) return props.queryTab.durationMs;
  if (props.tabId) {
    const tab = queryStore.resultTabs.find((t) => t.id === props.tabId);
    if (tab) return tab.durationMs;
  }
  return queryStore.activeResultTab?.durationMs ?? queryStore.activeResult?.executionTimeMs;
});

// Quick filter input is debounced once the result set is big enough for a single filter pass to
// be felt. Measured on a 150 column result set: ~21ms per pass at 1k rows, ~497ms at 50k rows,
// so ~10k rows (also the app's default row cap) is where one pass reaches the ~100ms budget.
// Below the threshold the grid updates as you type, which is the nicer behaviour when it's free.
const QUICK_FILTER_DEBOUNCE_MS = 250;
const QUICK_FILTER_DEBOUNCE_ROW_THRESHOLD = 10000;

const quickFilterInput = ref('');
const quickFilter = ref('');
let quickFilterTimer: ReturnType<typeof setTimeout> | null = null;

watch(quickFilterInput, (value) => {
  if (quickFilterTimer) {
    clearTimeout(quickFilterTimer);
    quickFilterTimer = null;
  }

  if ((props.resultSet?.rows.length ?? 0) < QUICK_FILTER_DEBOUNCE_ROW_THRESHOLD) {
    quickFilter.value = value;
    return;
  }

  quickFilterTimer = setTimeout(() => {
    quickFilterTimer = null;
    quickFilter.value = value;
  }, QUICK_FILTER_DEBOUNCE_MS);
});

/**
 * Clear the quick filter immediately, bypassing the debounce so the grid resets without a lag
 * even on large result sets.
 */
function clearQuickFilter() {
  quickFilterInput.value = '';
  quickFilter.value = '';
  if (quickFilterTimer) {
    clearTimeout(quickFilterTimer);
    quickFilterTimer = null;
  }
}

const gridContainerRef = ref<HTMLDivElement | null>(null);

/**
 * Result rows are `markRaw` positional arrays, so pushing/splicing them is invisible to Vue and to
 * Tabulator's data watcher. Every structural change bumps this revision and syncs the grid
 * explicitly. It also lets the empty state react when the first row is inserted into a zero-row
 * result (Tabulator is not even mounted in that state).
 */
const rowRevision = ref(0);
/** Rows created with "insert row" and not committed yet. */
const insertedRows = shallowRef<Set<CellValue[]>>(new Set());
/** Existing rows marked for deletion; they stay visible (struck through) until commit. */
const deletedRows = shallowRef<Set<CellValue[]>>(new Set());
/** Row last clicked in the grid, used as the delete target when no range is selected. */
const activeRowData = shallowRef<CellValue[] | null>(null);
/**
 * Shallow snapshot of the row order captured before the first insert/delete. Restoring it on
 * "revert all" puts deleted rows back in their exact place without a database round trip.
 */
let baselineRows: CellValue[][] | null = null;

/**
 * Pending-state marker stored on the row payload itself. Tabulator may rebuild its internal Row
 * component around the same data array, and the component also receives row data through several
 * callbacks; the marker keeps an inserted/deleted row recognisable even if a Set lookup misses.
 */
const ROW_STATE_FIELD = '__sqlightRowState';
type RowState = 'insert' | 'delete' | null;

function rowStateOf(row: CellValue[] | null | undefined): RowState {
  if (!row) return null;
  const state = (row as unknown as Record<string, unknown>)[ROW_STATE_FIELD];
  return state === 'insert' || state === 'delete' ? state : null;
}

function setRowState(row: CellValue[], state: RowState) {
  const record = row as unknown as Record<string, unknown>;
  if (state) {
    Object.defineProperty(record, ROW_STATE_FIELD, {
      value: state,
      writable: true,
      configurable: true,
      enumerable: false,
    });
  } else {
    delete record[ROW_STATE_FIELD];
  }
}

function isPendingInsert(row: CellValue[] | null | undefined): boolean {
  return !!row && (insertedRows.value.has(row) || rowStateOf(row) === 'insert');
}

function isPendingDelete(row: CellValue[] | null | undefined): boolean {
  return !!row && (deletedRows.value.has(row) || rowStateOf(row) === 'delete');
}

/** Number of rows in the given pending state, counting both the Set and the row markers. */
function countPendingRows(kind: 'insert' | 'delete'): number {
  const set = kind === 'insert' ? insertedRows.value : deletedRows.value;
  const seen = new Set<CellValue[]>(set);
  for (const row of props.resultSet?.rows ?? []) {
    if (kind === 'insert' ? isPendingInsert(row) : isPendingDelete(row)) {
      seen.add(row);
    }
  }
  return seen.size;
}

// Tabulator must not hold a Vue reactive proxy for the row data: with wide result sets the proxy
// cost is paid on every cell read. Raw arrays keep rendering on plain objects while all
// modification tracking stays in `modifiedCells`.
const gridRowData = computed<CellValue[][]>(() => {
  // The revision is only a dependency: a structural mutation must invalidate this computed so the
  // explicit `grid.sync()` that follows reads the new row array.
  rowRevision.value;
  const rows = props.resultSet?.rows;
  return rows ? (toRaw(rows) as CellValue[][]) : [];
});

// Horizontal-scroll mode: while the user drags the horizontal scrollbar, decorative
// transitions inside the grid are switched off so each frame stays paint-cheap.
const isHorizontalScrolling = ref(false);
let horizontalScrollTimer: ReturnType<typeof setTimeout> | null = null;
let lastScrollLeft = 0;

function markHorizontalScrolling() {
  if (!isHorizontalScrolling.value) {
    isHorizontalScrolling.value = true;
  }
  if (horizontalScrollTimer) {
    clearTimeout(horizontalScrollTimer);
  }
  horizontalScrollTimer = setTimeout(() => {
    horizontalScrollTimer = null;
    isHorizontalScrolling.value = false;
  }, 150);
}

function handleGridScroll(event: Event) {
  const target = event.target as HTMLElement | null;
  if (!target || typeof target.scrollLeft !== 'number') return;
  if (target.scrollLeft !== lastScrollLeft) {
    lastScrollLeft = target.scrollLeft;
    markHorizontalScrolling();
  }
}

const currentTab = computed(() => props.queryTab ?? queryStore.activeResultTab);

// Stable key identifying this result set's remembered column layout (session only).
const layoutKey = computed(() =>
  gridLayoutStore.layoutKey(props.tabId ?? currentTab.value?.id ?? null, props.setIndex)
);

// Layout snapshots are debounced: a resize drag fires many events and every capture copies
// the whole column state array.
let captureTimer: ReturnType<typeof setTimeout> | null = null;

function captureLayoutNow() {
  if (captureTimer) {
    clearTimeout(captureTimer);
    captureTimer = null;
  }
  gridLayoutStore.capture(layoutKey.value, grid.table.value);
}

function scheduleLayoutCapture() {
  if (captureTimer) clearTimeout(captureTimer);
  captureTimer = setTimeout(() => {
    captureTimer = null;
    gridLayoutStore.capture(layoutKey.value, grid.table.value);
  }, 200);
}

const currentConnection = computed(() => {
  const connId = currentTab.value?.connectionId || connectionStore.activeConnectionId;
  return connectionStore.getConnectionById(connId) || connectionStore.activeConnection;
});

const requiresModificationPrompt = computed(() => {
  return currentConnection.value?.modificationPrompt ?? false;
});

const isRefreshing = ref(false);

/**
 * Refresh drops the visible view state: sorting and filtering are reset so the reloaded result
 * shows the raw data again. Column widths/order (the remembered layout) are deliberately kept.
 */
function resetGridState() {
  clearQuickFilter();

  const table = grid.table.value;
  if (table) {
    table.clearSort();
    table.clearFilter();
  }

  // Overwrite the remembered layout so a later rebuild does not restore the dropped sorters.
  gridLayoutStore.capture(layoutKey.value, table);
}

async function handleRefresh() {
  if (isRefreshing.value) return;
  const tab = currentTab.value;
  if (!tab) {
    workspaceStore.showToast('無法取得查詢分頁資訊', 'error', 2000);
    return;
  }

  isRefreshing.value = true;
  try {
    const res = await queryStore.refreshTabResultSet(tab.id, props.setIndex);
    if (res.success) {
      clearPendingState();
      resetGridState();
      grid.redraw(true);
      applyRowStateClasses();
      workspaceStore.showToast(`資料已重新整理（共 ${res.rowCount.toLocaleString()} 筆）`, 'success', 2000);
    } else {
      workspaceStore.showToast(`重新整理失敗: ${res.error || '未知錯誤'}`, 'error', 3500);
    }
  } catch (err: any) {
    workspaceStore.showToast(`重新整理失敗: ${err?.message || err}`, 'error', 3500);
  } finally {
    isRefreshing.value = false;
  }
}

// Check editability for this result set
const editability = computed(() => {
  return checkTableEditability({
    tab: currentTab.value,
    columns: props.resultSet.columns ?? [],
    getTableSchema: (tbl, cId, db) => {
      const conn = cId || currentTab.value?.connectionId || connectionStore.activeConnectionId || undefined;
      const dbase = db || currentTab.value?.database || connectionStore.activeDatabase || undefined;
      return schemaStore.getTable(tbl, conn, dbase);
    },
  });
});

/**
 * Full schema of the target table. The editability check only reports the key shape; inserting a
 * row also needs identity / computed / rowversion flags and default definitions.
 */
const targetTableSchema = computed<TableSchema | null>(() => {
  const tab = currentTab.value;
  const target = editability.value.targetTable;
  if (!target) return null;

  const connId = tab?.connectionId || connectionStore.activeConnectionId || undefined;
  const db = tab?.database || connectionStore.activeDatabase || undefined;
  if (!connId || !db) return null;

  return (
    schemaStore.getTable(target.tableName, connId, db) ||
    schemaStore.getTable(`${target.schema}.${target.tableName}`, connId, db) ||
    null
  );
});

function tableColumnFor(name: string) {
  const lower = name.toLowerCase();
  return targetTableSchema.value?.columns.find((column) => column.name.toLowerCase() === lower);
}

// Set of lowercased identity column names
const identityColumnNames = computed<Set<string>>(() => {
  return new Set(
    (targetTableSchema.value?.columns ?? [])
      .filter((c) => c.isIdentity)
      .map((c) => c.name.toLowerCase())
  );
});

/** Identity, computed and rowversion values are produced by SQL Server; never write them. */
const generatedColumnNames = computed<Set<string>>(() => {
  return new Set(
    (targetTableSchema.value?.columns ?? [])
      .filter((c) => c.isIdentity || c.isComputed || c.isRowVersion)
      .map((c) => c.name.toLowerCase())
  );
});

/**
 * Required table columns the query does not project. The commit preview warns about them because
 * the INSERT can only write columns the result set knows about.
 */
const missingRequiredInsertColumns = computed<string[]>(() => {
  if (!editability.value.canEdit) return [];

  const projected = new Set((props.resultSet?.columns ?? []).map((c) => c.name.toLowerCase()));
  return (targetTableSchema.value?.columns ?? [])
    .filter(
      (c) =>
        !c.isNullable &&
        !c.isIdentity &&
        !c.isComputed &&
        !c.isRowVersion &&
        !c.defaultDefinition
    )
    .filter((c) => !projected.has(c.name.toLowerCase()))
    .map((c) => c.name);
});

// Set of lowercased primary key column names
const primaryKeyColumnNames = computed<Set<string>>(() => {
  if (editability.value.pkColumns.length > 0) {
    return new Set(editability.value.pkColumns.map((c) => c.toLowerCase()));
  }

  const tab = currentTab.value;
  const connId = tab?.connectionId || connectionStore.activeConnectionId || undefined;
  const db = tab?.database || connectionStore.activeDatabase || undefined;
  if (!connId || !db) return new Set();

  const sql = tab?.sql || '';
  const parsedTables = extractAllTableNamesFromSql(sql);

  if (parsedTables.length === 0 && tab?.tableName) {
    parsedTables.push({ schema: tab.schema, tableName: tab.tableName });
  }

  const pkNames = new Set<string>();
  for (const t of parsedTables) {
    const tableSchema =
      schemaStore.getTable(t.tableName, connId, db) ||
      (t.schema ? schemaStore.getTable(`${t.schema}.${t.tableName}`, connId, db) : undefined);
    if (tableSchema) {
      for (const col of tableSchema.columns) {
        if (col.isPrimaryKey) {
          pkNames.add(col.name.toLowerCase());
        }
      }
    }
  }

  return pkNames;
});

/**
 * Read-only columns: generated values (identity / computed / rowversion) are never editable, and
 * existing primary keys stay immutable so the commit WHERE clause cannot lose its target. A newly
 * inserted row is the exception: a natural PK has to be typed by the user before it can be written.
 */
function isColumnEditable(colIdx: number, rowData?: CellValue[] | null): boolean {
  const column = props.resultSet?.columns[colIdx];
  if (!column || !editability.value.canEdit) return false;

  const name = column.name.toLowerCase();
  if (generatedColumnNames.value.has(name)) return false;
  // Expression / alias columns in the projection are not writable table columns.
  if (!tableColumnFor(column.name)) return false;
  if (isPendingDelete(rowData)) return false;
  if (isPendingInsert(rowData)) return true;

  return !primaryKeyColumnNames.value.has(name);
}

// Cell modification tracking
interface CellModification {
  rowKey: string;
  originalRowIndex: number;
  colIdx: number;
  colName: string;
  originalValue: CellValue;
  currentValue: CellValue;
  rowRef: CellValue[];
}

const modifiedCells = ref<Record<string, CellModification>>({});
const modifiedCount = computed(() => Object.keys(modifiedCells.value).length);
/** Cell edits that will actually become UPDATE statements (not on inserted / deleted rows). */
const activeModifiedCells = computed(() =>
  Object.values(modifiedCells.value).filter(
    (mod) => !isPendingInsert(mod.rowRef) && !isPendingDelete(mod.rowRef)
  )
);
const activeModifiedCellCount = computed(() => activeModifiedCells.value.length);
/** Distinct rows with at least one pending UPDATE. */
const modifiedRowCount = computed(
  () => new Set(activeModifiedCells.value.map((mod) => mod.rowKey)).size
);
const pendingInsertCount = computed(() => {
  // Structural changes are invisible to Vue unless the revision is read.
  rowRevision.value;
  return countPendingRows('insert');
});
const pendingDeleteCount = computed(() => {
  rowRevision.value;
  return countPendingRows('delete');
});
/** Total affected rows across inserts, updates and deletes (drives the commit button). */
const pendingRowCount = computed(
  () => modifiedRowCount.value + pendingInsertCount.value + pendingDeleteCount.value
);
const hasRows = computed(() => {
  // Structural row mutations do not trigger Vue on their own (the row array is markRaw).
  rowRevision.value;
  return (props.resultSet?.rows?.length ?? 0) > 0;
});
/**
 * Separates "this tab has never run anything" from "the statement matched zero rows". A query
 * that returns no rows still ships its column metadata, so an empty grid with columns is a
 * finished statement, not an untouched one.
 */
const resultSetColumnCount = computed(() => props.resultSet?.columns?.length ?? 0);

const pkColumnIndices = computed<number[]>(() => {
  if (!props.resultSet || editability.value.pkColumns.length === 0) return [];
  const map = new Map<string, number>();
  props.resultSet.columns.forEach((c, idx) => map.set(c.name.toLowerCase(), idx));
  return editability.value.pkColumns
    .map((pk) => map.get(pk.toLowerCase()))
    .filter((idx): idx is number => idx !== undefined);
});

function getRowPkKey(row: CellValue[] | undefined): string {
  if (!row || pkColumnIndices.value.length === 0) return '';
  const indices = pkColumnIndices.value;
  let key = '';
  for (let i = 0; i < indices.length; i++) {
    if (i > 0) key += ':::';
    key += String(row[indices[i]!] ?? '');
  }
  return key;
}

function isCellModified(rowData: CellValue[] | undefined, colIdx: number): boolean {
  if (modifiedCount.value === 0 || !rowData) return false;
  // A row pending deletion shows its own red styling; its tracked edits stay hidden until the
  // deletion is cancelled or reverted.
  if (isPendingDelete(rowData)) return false;
  const rowKey = getRowPkKey(rowData);
  if (!rowKey) return false;
  return `${rowKey}___col_${colIdx}` in modifiedCells.value;
}

/**
 * Records (or clears) the pending modification for one cell. Shared by the inline editor and the
 * "set NULL" context menu action so both paths produce identical commit statements.
 */
function recordModification(
  rowData: CellValue[],
  colIdx: number,
  previousValue: CellValue,
  nextValue: CellValue
): void {
  // A pending insert has no primary key yet, and its whole row is read at commit time, so there
  // is nothing useful to track per cell.
  if (isPendingInsert(rowData)) return;

  const rowKey = getRowPkKey(rowData);
  if (!rowKey || !props.resultSet) return;

  const cellKey = `${rowKey}___col_${colIdx}`;
  const existing = modifiedCells.value[cellKey];
  const originalValue = existing ? existing.originalValue : previousValue;

  if (nextValue === originalValue) {
    const copy = { ...modifiedCells.value };
    delete copy[cellKey];
    modifiedCells.value = copy;
    return;
  }

  modifiedCells.value = {
    ...modifiedCells.value,
    [cellKey]: {
      rowKey,
      originalRowIndex: props.resultSet.rows.indexOf(rowData),
      colIdx,
      colName: props.resultSet.columns[colIdx]?.name ?? '',
      originalValue,
      currentValue: nextValue,
      rowRef: rowData,
    },
  };
}

/** Drops every pending cell edit and structural change. */
function clearPendingState() {
  // Clear the markers on the rows we tracked, whichever result set is currently displayed.
  for (const row of insertedRows.value) setRowState(row, null);
  for (const row of deletedRows.value) setRowState(row, null);
  modifiedCells.value = {};
  insertedRows.value = new Set();
  deletedRows.value = new Set();
  baselineRows = null;
  activeRowData.value = null;
  rowRevision.value += 1;
}

/**
 * Re-applies the pending insert/delete classes to every rendered row. `rowFormatter` normally does
 * this, but a cell editor, a rejected replaceData or a virtual re-render can skip it; the row is
 * still recognisable from its marker, so painting here makes the state deterministic.
 */
function applyRowStateClasses() {
  const table = grid.table.value;
  if (!table) return;

  for (const row of table.getRows()) {
    // Do not call getElement(): it would materialise an element for every off-screen row.
    const element = (row as unknown as { element?: HTMLElement }).element;
    if (!element) continue;
    const data = row.getData() as unknown as CellValue[];
    element.classList.toggle('sqlight-row-inserted', isPendingInsert(data));
    element.classList.toggle('sqlight-row-pending-delete', isPendingDelete(data));
  }
}

/** Pushes a structural row change to both Vue (revision) and Tabulator (replaceData). */
async function syncGridRows() {
  rowRevision.value += 1;
  clearCellSelection();
  await nextTick();
  await grid.sync();
  applyRowStateClasses();
}

/** Captures the untouched row order before the first insert/delete so revert can restore it. */
function captureBaselineRows() {
  if (baselineRows || !props.resultSet) return;
  baselineRows = props.resultSet.rows.slice();
}

function dropModificationsForRow(row: CellValue[]) {
  const entries = Object.entries(modifiedCells.value).filter(([, mod]) => mod.rowRef !== row);
  if (entries.length !== Object.keys(modifiedCells.value).length) {
    modifiedCells.value = Object.fromEntries(entries);
  }
}

// Clear pending work when switching result sets or tabs
watch(
  () => [props.resultSet, queryStore.activeResultTabId],
  () => {
    clearPendingState();
  }
);

// Commit confirmation modal state
const commitModal = reactive({
  visible: false,
  isExecuting: false,
  error: '',
  sql: '',
  rowCount: 0,
  cellCount: 0,
  insertCount: 0,
  updateCount: 0,
  deleteCount: 0,
  confirmStep: 1 as 1 | 2,
});

function closeCommitModal() {
  commitModal.visible = false;
  commitModal.confirmStep = 1;
  commitModal.error = '';
}

async function handleRevertChanges() {
  if (!props.resultSet || pendingRowCount.value === 0) {
    workspaceStore.showToast('目前無未提交的異動可供退回', 'info', 2000);
    return;
  }

  const count = pendingRowCount.value;
  const rows = props.resultSet.rows;
  const hadStructuralChanges =
    pendingInsertCount.value > 0 || pendingDeleteCount.value > 0;

  // Restore cell values first; deleted rows keep their pending cell edits dropped at delete time.
  for (const mod of Object.values(modifiedCells.value)) {
    mod.rowRef[mod.colIdx] = mod.originalValue;
  }

  // Put deleted rows back exactly where they were and drop inserted rows.
  if (baselineRows) {
    rows.splice(0, rows.length, ...baselineRows);
  }

  clearPendingState();
  await nextTick();

  if (hadStructuralChanges) {
    await grid.sync();
  } else {
    grid.redraw(true);
  }
  applyRowStateClasses();

  workspaceStore.showToast(`已退回 ${count} 筆未提交的異動，資料已還原`, 'info', 2200);
}

/** Primary key values of one row, as the WHERE condition for UPDATE / DELETE. */
function buildPkWhere(row: CellValue[]): Record<string, CellValue> {
  const where: Record<string, CellValue> = {};
  if (!props.resultSet) return where;

  for (const pkCol of editability.value.pkColumns) {
    const index = props.resultSet.columns.findIndex(
      (column) => column.name.toLowerCase() === pkCol.toLowerCase()
    );
    if (index >= 0) {
      where[pkCol] = row[index] ?? null;
    }
  }

  return where;
}

/**
 * Writable columns of an inserted row. Identity / computed / rowversion values never leave the
 * server, aliases are not real table columns, and an untouched column with a server default is
 * omitted so the default applies instead of an explicit NULL.
 */
function buildInsertValues(row: CellValue[]): InsertColumnValue[] {
  const values: InsertColumnValue[] = [];
  if (!props.resultSet) return values;

  props.resultSet.columns.forEach((column, index) => {
    const schemaColumn = tableColumnFor(column.name);
    if (!schemaColumn) return;
    if (schemaColumn.isIdentity || schemaColumn.isComputed || schemaColumn.isRowVersion) return;

    const value = row[index] ?? null;
    if (value === null && schemaColumn.defaultDefinition) return;

    // Use the table's column name, not the result alias, so `SELECT UserName AS Name` still
    // writes the real column.
    values.push({ column: schemaColumn.name, value });
  });

  return values;
}

/** Blocks commits that the database would reject for a predictable reason. */
function validateInsertRows(rows: CellValue[][]): string | null {
  if (!props.resultSet) return null;

  const seen = new Set<string>();
  for (const row of rows) {
    const keyValues: CellValue[] = [];
    const missingKeys: string[] = [];

    for (const pkCol of editability.value.pkColumns) {
      const index = props.resultSet.columns.findIndex(
        (column) => column.name.toLowerCase() === pkCol.toLowerCase()
      );
      if (index < 0) continue;

      const schemaColumn = tableColumnFor(pkCol);
      if (schemaColumn?.isIdentity || schemaColumn?.isComputed || schemaColumn?.isRowVersion) {
        continue;
      }

      const value = row[index] ?? null;
      if (value === null) {
        missingKeys.push(pkCol);
      } else {
        keyValues.push(value);
      }
    }

    if (missingKeys.length > 0) {
      return t('results.insertPkRequired', { columns: missingKeys.join(', ') });
    }

    const missingRequired: string[] = [];
    props.resultSet.columns.forEach((column, index) => {
      const schemaColumn = tableColumnFor(column.name);
      if (!schemaColumn) return;
      if (
        schemaColumn.isNullable ||
        schemaColumn.isIdentity ||
        schemaColumn.isComputed ||
        schemaColumn.isRowVersion ||
        schemaColumn.defaultDefinition
      ) {
        return;
      }
      if ((row[index] ?? null) === null) {
        missingRequired.push(column.name);
      }
    });

    if (missingRequired.length > 0) {
      return t('results.insertRequiredMissing', { columns: missingRequired.join(', ') });
    }

    if (keyValues.length === 0) continue;

    const key = JSON.stringify(keyValues);
    if (seen.has(key)) {
      return t('results.insertPkDuplicate', {
        value: keyValues.map((value) => String(value)).join(', '),
      });
    }
    seen.add(key);
  }

  return null;
}

/**
 * Builds and validates the change plan without opening the preview modal. Returns false when the
 * plan cannot be committed (the reason has already been surfaced to the user).
 */
function buildCommitPlan(): boolean {
  if (!editability.value.canEdit || !editability.value.targetTable || isRefreshing.value) {
    return false;
  }
  if (pendingRowCount.value === 0) {
    workspaceStore.showToast('目前無未提交的異動可供提交', 'info', 2000);
    return false;
  }

  const rowsMap = new Map<string, RowModification>();
  for (const mod of Object.values(modifiedCells.value)) {
    // Defensive: modifications on rows that were deleted or are pending inserts never become UPDATEs.
    if (isPendingDelete(mod.rowRef) || isPendingInsert(mod.rowRef)) continue;

    let rm = rowsMap.get(mod.rowKey);
    if (!rm) {
      rm = {
        rowIndex: mod.originalRowIndex,
        updates: {},
        pkWhere: buildPkWhere(mod.rowRef),
      };
      rowsMap.set(mod.rowKey, rm);
    }
    // Resolve result aliases back to the real table column before generating the UPDATE.
    rm.updates[tableColumnFor(mod.colName)?.name ?? mod.colName] = mod.currentValue;
  }

  const updates = Array.from(rowsMap.values());
  const insertRows = collectPendingRows('insert');
  const deleteRows = collectPendingRows('delete');
  const inserts: InsertRowChange[] = insertRows.map((row) => ({
    values: buildInsertValues(row),
  }));
  const deletes: DeleteRowChange[] = deleteRows.map((row) => ({
    pkWhere: buildPkWhere(row),
  }));

  const insertError = validateInsertRows(insertRows);
  if (insertError) {
    workspaceStore.showToast(insertError, 'error', 4000);
    return false;
  }

  if (updates.length + inserts.length + deletes.length === 0) {
    workspaceStore.showToast('目前無未提交的異動可供提交', 'info', 2000);
    return false;
  }

  const tab = currentTab.value;
  const db = tab?.database || connectionStore.activeDatabase || undefined;

  try {
    const sql = generateBatchChangesScript({
      tableName: editability.value.targetTable.tableName,
      schema: editability.value.targetTable.schema,
      database: db,
      updates,
      inserts,
      deletes,
    });

    commitModal.sql = sql;
    commitModal.rowCount = updates.length + inserts.length + deletes.length;
    commitModal.updateCount = updates.length;
    commitModal.insertCount = inserts.length;
    commitModal.deleteCount = deletes.length;
    commitModal.cellCount = activeModifiedCellCount.value;
    commitModal.error = '';
    commitModal.isExecuting = false;
    return true;
  } catch (err: any) {
    workspaceStore.showToast(`產生異動語法失敗: ${err?.message || err}`, 'error', 3500);
    return false;
  }
}

/** Opens the preview + confirmation modal (the existing commit flow). */
function openCommitModal() {
  if (!buildCommitPlan()) return;
  commitModal.confirmStep = 1;
  commitModal.visible = true;
}

/**
 * Commit button entry point. The SQL preview is opt-in; by default the transaction executes
 * immediately. A connection that enables its own modification prompt always keeps the review
 * modal, because that setting is an explicit safety opt-in.
 */
async function handleCommitAction() {
  if (settingsStore.showCommitSqlPreview || requiresModificationPrompt.value) {
    openCommitModal();
    return;
  }

  if (!buildCommitPlan()) return;
  await executeCommit(true);
}

/** Reloads the result set after an INSERT / DELETE so identity and default values are accurate. */
async function reloadResultAfterCommit(): Promise<boolean> {
  const tab = currentTab.value;
  if (!tab) return false;

  try {
    const res = await queryStore.refreshTabResultSet(tab.id, props.setIndex);
    if (!res.success) return false;

    resetGridState();
    await nextTick();
    await grid.sync();
    return true;
  } catch {
    return false;
  }
}

/** A direct commit has no visible modal, so failures have to surface as a toast. */
function reportCommitError(message: string, direct: boolean) {
  if (!direct) {
    commitModal.error = message;
    return;
  }
  workspaceStore.showToast(t('results.commitFailed', { error: message }), 'error', 6000);
}

async function executeCommit(direct = false) {
  const tab = currentTab.value;
  const connId = tab?.connectionId || connectionStore.activeConnectionId || undefined;
  const db = tab?.database || connectionStore.activeDatabase || undefined;

  if (!connId || !db) {
    reportCommitError('無法取得當前連線或資料庫資訊', direct);
    return;
  }

  commitModal.isExecuting = true;
  commitModal.error = '';

  const insertCount = commitModal.insertCount;
  const updateCount = commitModal.updateCount;
  const deleteCount = commitModal.deleteCount;
  const hadStructuralChanges = insertCount + deleteCount > 0;

  try {
    const result = await queryService.executeQuery(connId, db, commitModal.sql);

    // The Go query facade reports SQL Server failures as result messages with a nil Go error, so a
    // resolved promise is not proof that the transaction committed.
    const sqlError = result.messages?.find((message) => message.level === 'error');
    if (sqlError) {
      reportCommitError(sqlError.message, direct);
      return;
    }

    const cancelled = result.messages?.some(
      (message) =>
        message.level === 'warning' && message.message.toLowerCase().includes('cancel')
    );
    if (cancelled) {
      reportCommitError(t('results.commitCancelled'), direct);
      return;
    }

    clearPendingState();
    closeCommitModal();

    const reloaded = hadStructuralChanges ? await reloadResultAfterCommit() : true;
    if (!hadStructuralChanges) {
      grid.redraw(true);
    }

    if (reloaded) {
      workspaceStore.showToast(
        t('results.commitMixedSuccess', {
          inserted: insertCount,
          updated: updateCount,
          deleted: deleteCount,
        }),
        'success',
        3200
      );
    } else {
      workspaceStore.showToast(t('results.commitRefreshFailed'), 'warning', 5000);
    }
  } catch (err: any) {
    reportCommitError(err?.message || String(err), direct);
  } finally {
    commitModal.isExecuting = false;
  }
}

function copyCommitSql() {
  if (!commitModal.sql) return;
  try {
    navigator.clipboard?.writeText(commitModal.sql);
    workspaceStore.showToast('已複製批次更新語法至剪貼簿', 'success', 2000);
  } catch (err) {
    // Ignore clipboard error
  }
}

// Cell context menu quick action computed and methods
const isCurrentCellEditable = computed(() => {
  if (!editability.value.canEdit || !contextMenu.visible) return false;
  const colIdx = getColIndex(contextMenu.colId);
  if (colIdx === undefined || !props.resultSet) return false;
  return isColumnEditable(colIdx, contextMenu.rowData);
});

const isCurrentCellNullable = computed(() => {
  if (!isCurrentCellEditable.value) return false;
  const colIdx = getColIndex(contextMenu.colId);
  if (colIdx === undefined || !props.resultSet) return false;
  return props.resultSet.columns[colIdx]?.nullable ?? true;
});

const isCurrentCellModified = computed(() => {
  if (!contextMenu.visible || !contextMenu.rowData) return false;
  const colIdx = getColIndex(contextMenu.colId);
  if (colIdx === undefined) return false;
  return isCellModified(contextMenu.rowData, colIdx);
});

function setCellNull() {
  if (!isCurrentCellEditable.value || !contextMenu.rowData || !props.resultSet) return;
  const colIdx = getColIndex(contextMenu.colId);
  if (colIdx === undefined) return;

  const oldVal: CellValue = contextMenu.rowData[colIdx] ?? null;
  if (oldVal === null) {
    contextMenu.visible = false;
    return;
  }

  contextMenu.rowData[colIdx] = null;
  recordModification(contextMenu.rowData, colIdx, oldVal, null);
  grid.redraw(true);
  contextMenu.visible = false;
}

function revertSingleCell() {
  if (!contextMenu.rowData || !props.resultSet) return;
  const colIdx = getColIndex(contextMenu.colId);
  if (colIdx === undefined) return;
  const rowKey = getRowPkKey(contextMenu.rowData);
  const cellKey = `${rowKey}___col_${colIdx}`;
  const mod = modifiedCells.value[cellKey];
  if (!mod) {
    contextMenu.visible = false;
    return;
  }

  mod.rowRef[colIdx] = mod.originalValue;
  const copy = { ...modifiedCells.value };
  delete copy[cellKey];
  modifiedCells.value = copy;

  grid.redraw(true);
  contextMenu.visible = false;
  workspaceStore.showToast(`已還原欄位 [${mod.colName}]`, 'info', 1500);
}

// Auto-fetch database schema if not yet loaded
watch(
  () => [currentTab.value?.connectionId, currentTab.value?.database] as const,
  ([cId, db]) => {
    const connId = cId || connectionStore.activeConnectionId;
    const database = db || connectionStore.activeDatabase;
    if (connId && database && !schemaStore.isDatabaseLoaded(connId, database)) {
      schemaStore.loadDatabaseSchema(connId, database).catch(() => {});
    }
  },
  { immediate: true }
);

// Selection composable (Tabulator range based)
const selection = useGridSelection({
  getTable: () => grid.table.value,
  getContainer: () => gridContainerRef.value,
  getColumns: () => props.resultSet.columns ?? [],
  onCopySelected: () => gridExport.copySelectedCells(),
});

const {
  selectionStats,
  hasSelection,
  selectedColumnsCount,
  getColIndex,
  formatAggregateNumber,
  clearCellSelection,
} = selection;

// --------------------------------------------------------------------------
// Row insert / delete
// --------------------------------------------------------------------------

const canAddRow = computed(
  () => editability.value.canEdit && resultSetColumnCount.value > 0 && !isRefreshing.value
);

/** Deleting targets the active range when one exists, otherwise the row last clicked. */
const canDeleteRows = computed(
  () =>
    editability.value.canEdit &&
    !isRefreshing.value &&
    (hasSelection.value || activeRowData.value !== null)
);

const isContextRowDeleted = computed(() => isPendingDelete(contextMenu.rowData));

const isContextRowInserted = computed(() => isPendingInsert(contextMenu.rowData));

/**
 * All pending rows of one kind. Live grid rows are preferred because they carry the latest edits;
 * Set members cover rows that are filtered out of the display or no longer mounted.
 */
function collectPendingRows(kind: 'insert' | 'delete'): CellValue[][] {
  const rows: CellValue[][] = [];
  const seen = new Set<CellValue[]>();
  const matches = (row: CellValue[]) =>
    kind === 'insert' ? isPendingInsert(row) : isPendingDelete(row);
  const add = (row: CellValue[]) => {
    if (!seen.has(row) && matches(row)) {
      seen.add(row);
      rows.push(row);
    }
  };

  for (const row of grid.table.value?.getRows() ?? []) {
    add(row.getData() as unknown as CellValue[]);
  }
  for (const row of kind === 'insert' ? insertedRows.value : deletedRows.value) {
    add(row);
  }

  return rows;
}

/** Rows covered by the current range selection, falling back to the last clicked row. */
function collectTargetRows(): CellValue[][] {
  const table = grid.table.value;
  const resultRows = new Set(props.resultSet?.rows ?? []);
  const rows: CellValue[][] = [];
  const seen = new Set<CellValue[]>();
  const add = (row: CellValue[]) => {
    // A range may outlive a row that was just removed (e.g. an inserted row deleted again).
    if (!resultRows.has(row) || seen.has(row)) return;
    seen.add(row);
    rows.push(row);
  };

  if (table) {
    for (const range of table.getRanges()) {
      for (const row of range.getRows()) {
        add(row.getData() as unknown as CellValue[]);
      }
    }
  }

  if (rows.length === 0 && activeRowData.value) {
    add(activeRowData.value);
  }

  return rows;
}

/** Marks an existing row for deletion, or drops an inserted row that was never committed. */
function markRowDeleted(row: CellValue[]) {
  if (!props.resultSet) return;

  if (isPendingInsert(row)) {
    removeInsertedRow(row);
    return;
  }

  if (isPendingDelete(row)) return;

  captureBaselineRows();
  // Cell edits stay tracked (but hidden) so cancelling the deletion restores them; a committed
  // DELETE supersedes them.
  setRowState(row, 'delete');
  deletedRows.value = new Set(deletedRows.value).add(row);
}

/** Puts a deleted row back at its original place in the baseline order. */
function restoreDeletedRow(row: CellValue[]) {
  const rows = props.resultSet?.rows;
  if (!rows || !isPendingDelete(row)) return;

  const base = baselineRows ?? rows;
  const baseIndex = base.indexOf(row);
  let insertAt = rows.length;

  if (baseIndex >= 0) {
    insertAt = 0;
    for (let i = baseIndex - 1; i >= 0; i--) {
      const previousIndex = rows.indexOf(base[i]!);
      if (previousIndex >= 0) {
        insertAt = previousIndex + 1;
        break;
      }
    }
  }

  rows.splice(insertAt, 0, row);
  setRowState(row, null);
  const next = new Set(deletedRows.value);
  next.delete(row);
  deletedRows.value = next;
}

function removeInsertedRow(row: CellValue[]) {
  const rows = props.resultSet?.rows;
  if (!rows) return;

  const index = rows.indexOf(row);
  if (index >= 0) {
    rows.splice(index, 1);
  }

  setRowState(row, null);
  const next = new Set(insertedRows.value);
  next.delete(row);
  insertedRows.value = next;
  dropModificationsForRow(row);
  if (activeRowData.value === row) {
    activeRowData.value = null;
  }
}

/** Scrolls to a pending row and opens the first empty writable cell (fallback: first writable). */
async function focusRowEditor(row: CellValue[]) {
  const table = grid.table.value;
  const tableRow = table
    ?.getRows()
    .find((candidate) => (candidate.getData() as unknown) === row);
  if (!table || !tableRow || !props.resultSet) return;

  table.scrollToRow(tableRow, 'bottom', false);

  const columns = props.resultSet.columns;
  const isEmptyEditable = (index: number) =>
    isColumnEditable(index, row) && (row[index] ?? null) === null;

  // A new primary key is the one value the row cannot be committed without, so prefer it.
  let target = columns.findIndex(
    (column, index) =>
      isEmptyEditable(index) && primaryKeyColumnNames.value.has(column.name.toLowerCase())
  );
  if (target < 0) {
    target = columns.findIndex((_, index) => isEmptyEditable(index));
  }
  if (target < 0) {
    target = columns.findIndex((_, index) => isColumnEditable(index, row));
  }
  if (target < 0) return;

  try {
    const cell = tableRow.getCell(String(target));
    if (cell) editCell(table, cell);
  } catch {
    // A column outside the rendered viewport may refuse the editor; the row is still added.
  }
}

async function handleAddRow() {
  if (!canAddRow.value || !props.resultSet) return;

  contextMenu.visible = false;
  // A new row must be visible: the quick filter could hide its all-NULL values.
  clearQuickFilter();

  const row: CellValue[] = props.resultSet.columns.map(() => null);
  captureBaselineRows();
  setRowState(row, 'insert');
  props.resultSet.rows.push(row);
  insertedRows.value = new Set(insertedRows.value).add(row);

  await syncGridRows();
  await focusRowEditor(row);

  workspaceStore.showToast(t('results.rowAdded'), 'info', 2200);
}

/**
 * Duplicates the values of an existing row directly below it. Generated columns and every primary
 * key column are cleared so the copy can receive its own key; binary placeholders cannot be
 * written back, so they are cleared as well.
 */
async function handleDuplicateRow(sourceRow: CellValue[]) {
  if (!canAddRow.value || !props.resultSet) return;

  contextMenu.visible = false;
  clearQuickFilter();

  const generated = generatedColumnNames.value;
  const primaryKeys = primaryKeyColumnNames.value;
  const row: CellValue[] = props.resultSet.columns.map((column, index) => {
    const name = column.name.toLowerCase();
    if (generated.has(name) || primaryKeys.has(name)) return null;

    const value = sourceRow[index] ?? null;
    return isBinaryCellValue(value) ? null : value;
  });

  captureBaselineRows();
  setRowState(row, 'insert');
  const sourceIndex = props.resultSet.rows.indexOf(sourceRow);
  const insertAt = sourceIndex >= 0 ? sourceIndex + 1 : props.resultSet.rows.length;
  props.resultSet.rows.splice(insertAt, 0, row);
  insertedRows.value = new Set(insertedRows.value).add(row);

  await syncGridRows();
  await focusRowEditor(row);

  workspaceStore.showToast(t('results.rowDuplicated'), 'info', 2400);
}

async function duplicateContextRow() {
  const row = contextMenu.rowData;
  if (row) await handleDuplicateRow(row);
}

async function handleDeleteRows() {
  if (!canDeleteRows.value) {
    if (editability.value.canEdit) {
      workspaceStore.showToast(t('results.noRowSelected'), 'info', 2000);
    }
    return;
  }

  const targets = collectTargetRows();
  if (targets.length === 0) {
    workspaceStore.showToast(t('results.noRowSelected'), 'info', 2000);
    return;
  }

  // Pressing delete on an already marked selection restores it instead of doing nothing.
  const allDeleted = targets.every((row) => isPendingDelete(row));
  if (allDeleted) {
    targets.forEach((row) => restoreDeletedRow(row));
    await syncGridRows();
    workspaceStore.showToast(t('results.rowDeletesRestored', { count: targets.length }), 'info', 2200);
    return;
  }

  targets.forEach((row) => markRowDeleted(row));
  await syncGridRows();
  workspaceStore.showToast(t('results.rowsDeleted', { count: targets.length }), 'info', 2400);
}

async function toggleContextRowDeletion() {
  const row = contextMenu.rowData;
  contextMenu.visible = false;
  if (!row || !editability.value.canEdit) return;

  if (isPendingInsert(row)) {
    removeInsertedRow(row);
    await syncGridRows();
    workspaceStore.showToast(t('results.insertedRowRemoved'), 'info', 1800);
    return;
  }

  if (isPendingDelete(row)) {
    restoreDeletedRow(row);
    await syncGridRows();
    workspaceStore.showToast(t('results.rowDeletesRestored', { count: 1 }), 'info', 1800);
    return;
  }

  markRowDeleted(row);
  await syncGridRows();
  workspaceStore.showToast(t('results.rowsDeleted', { count: 1 }), 'info', 2000);
}

// Double-click copies the cell value on read-only cells (read-only result sets, PK/Identity
// columns). Editable cells keep their double-click-to-edit behaviour: Tabulator opens its editor
// on the same gesture, so the copy handler has to yield to it.
function handleCellDoubleClick(_event: MouseEvent, cell: TabulatorCellComponent) {
  const colIdx = getColIndex(cell.getField());
  if (colIdx === undefined || !props.resultSet) return;

  const rowData = cell.getRow().getData() as unknown as CellValue[];
  if (isColumnEditable(colIdx, rowData)) return;

  const column = props.resultSet.columns[colIdx];
  const colName = column?.name ?? '';
  const generated = generatedColumnNames.value.has(colName.toLowerCase());
  if (generated && (cell.getValue() === null || cell.getValue() === undefined)) {
    // A pending row's identity/computed cell is still NULL; explain where the value will come from
    // instead of copying the literal "NULL".
    workspaceStore.showToast(t('results.generatedCellHint'), 'info', 3200);
    return;
  }

  const text = formatCellForExport(cell.getValue() as CellValue);

  navigator.clipboard.writeText(text).then(
    () => workspaceStore.showToast(`已複製「${colName}」的值至剪貼簿`, 'success', 1800),
    () => workspaceStore.showToast('複製失敗：無法寫入剪貼簿', 'warning', 2500)
  );
}

// Clear selection when result set changes
watch(
  () => props.resultSet,
  () => {
    clearCellSelection();
  }
);

// Custom cell context menu state
const contextMenu = reactive<{
  visible: boolean;
  x: number;
  y: number;
  colId: string;
  colName: string;
  cellValue: unknown;
  rowIndex: number;
  rowData: CellValue[] | null;
}>({
  visible: false,
  x: 0,
  y: 0,
  colId: '',
  colName: '',
  cellValue: null,
  rowIndex: -1,
  rowData: null,
});

// Export composable
const gridExport = useGridExport({
  getRows: () => props.resultSet.rows ?? [],
  getColumns: () => props.resultSet.columns ?? [],
  selection,
  showToast: (msg, type, duration) => workspaceStore.showToast(msg, type, duration),
  onMenuClose: () => {
    contextMenu.visible = false;
  },
});

const {
  copiedTsv,
  copiedCsv,
  copiedJson,
  copiedMd,
  copyCellValue: exportCopyCellValue,
  copyColumnName: exportCopyColumnName,
  copyCurrentRow: exportCopyCurrentRow,
  copyCurrentRowAsJson: exportCopyCurrentRowAsJson,
  copySelectedCells,
  copySelectedAsJson,
  copyAsTsv,
  copyAsCsv,
  copyAsJson,
  copyAsMarkdown,
} = gridExport;

const copyJustCompleted = computed(
  () => copiedTsv.value || copiedCsv.value || copiedJson.value || copiedMd.value
);

function handleCopyFormat(format: CopyFormat) {
  if (format === 'tsv') copyAsTsv();
  else if (format === 'csv') copyAsCsv();
  else if (format === 'json') copyAsJson();
  else copyAsMarkdown();
}

function copyCellValue() {
  exportCopyCellValue(contextMenu.cellValue);
}

function copyColumnName() {
  exportCopyColumnName(contextMenu.colName);
}

function copyCurrentRow() {
  const row = contextMenu.rowData || (contextMenu.rowIndex >= 0 && props.resultSet ? props.resultSet.rows[contextMenu.rowIndex] : null);
  exportCopyCurrentRow(row);
}

function copyCurrentRowAsJson() {
  const row = contextMenu.rowData || (contextMenu.rowIndex >= 0 && props.resultSet ? props.resultSet.rows[contextMenu.rowIndex] : null);
  if (row && props.resultSet) {
    exportCopyCurrentRowAsJson(props.resultSet.columns, row);
  }
}

function openDataView() {
  if (!hasRows.value) {
    workspaceStore.showToast('查無資料列可供檢視 (0 筆)', 'info', 2000);
    contextMenu.visible = false;
    return;
  }
  const row = contextMenu.rowData || (contextMenu.rowIndex >= 0 && props.resultSet ? props.resultSet.rows[contextMenu.rowIndex] : null);
  if (!row || !props.resultSet) {
    contextMenu.visible = false;
    return;
  }
  dataViewStore.openDataView({
    columns: props.resultSet.columns,
    row,
    rowIndex: contextMenu.rowIndex,
    totalRows: props.resultSet.rows.length,
    allRows: props.resultSet.rows,
    tableName: currentTab.value?.title || currentTab.value?.tableName,
  });
  contextMenu.visible = false;
}

// --------------------------------------------------------------------------
// Tabulator grid
// --------------------------------------------------------------------------

const gridIsActive = computed(() => !!props.resultSet && hasRows.value);

// Row count is deliberately not part of the signature: adding a row only changes the data, so the
// table must be updated with `replaceData`, never torn down and rebuilt (which would lose scroll).
const gridColumnSignature = computed(() =>
  [
    props.resultSet?.columns.map((c) => `${c.name}|${c.dataType}|${c.nullable ? 1 : 0}`).join('\u0001') ?? '',
    [...primaryKeyColumnNames.value].sort().join(','),
    [...identityColumnNames.value].sort().join(','),
    editability.value.canEdit ? '1' : '0',
  ].join('\u0002')
);

function buildColumnDefinitions(): TabulatorColumnDefinition[] {
  if (!props.resultSet) return [];

  const firstRow = props.resultSet.rows[0];
  const dataColumns = props.resultSet.columns.map((col, colIdx) => {
    const isPk = primaryKeyColumnNames.value.has(col.name.toLowerCase());
    const isIdentity = identityColumnNames.value.has(col.name.toLowerCase());
    const isStmtText = col.name === 'StmtText';
    const firstVal = firstRow ? firstRow[colIdx] : undefined;
    const baseWidth = calculateColumnWidth(col.name, firstVal, isPk);
    const colWidth = isStmtText ? Math.max(baseWidth, 360) : baseWidth;

    return buildDataColumn({
      column: col,
      columnIndex: colIdx,
      width: colWidth,
      minWidth: isStmtText ? 200 : 70,
      headerTooltip: isPk
        ? `🔑 [主鍵 / Primary Key (唯讀)] 型別 (Type): ${col.dataType}${col.nullable ? ' | 可為 NULL' : ' | NOT NULL'} (拖曳表頭調整順序，點選表頭選取整欄)`
        : isIdentity
          ? `🔒 [識別欄位 / Identity (唯讀)] 型別 (Type): ${col.dataType} (拖曳表頭調整順序)`
          : `型別 (Type): ${col.dataType}${col.nullable ? ' | 可為 NULL' : ' | NOT NULL'}${editability.value.canEdit ? ' (雙擊可編輯)' : ''} (拖曳表頭調整順序)`,
      isPrimaryKey: isPk,
      isIdentity,
      isEditable: (cell) =>
        isColumnEditable(colIdx, cell?.getRow().getData() as CellValue[] | undefined),
      isModified: isCellModified,
      extraClass: isStmtText ? 'sqlight-stmt-text' : undefined,
    });
  });

  return [buildRowIndexColumn({ rowCount: props.resultSet.rows.length }), ...dataColumns];
}

/** Visible data field names in display order; the quick filter scans exactly these. */
function dataFieldNames(): string[] {
  const table = grid.table.value;
  if (!table) return (props.resultSet?.columns ?? []).map((_, index) => String(index));
  return table
    .getColumns()
    .filter((column) => column.isVisible() && column.getField() !== ROW_INDEX_FIELD)
    .map((column) => column.getField());
}

/** Quick filter semantics: every whitespace separated term must match somewhere in the row. */
function buildQuickFilter(term: string): (data: TabulatorRowData) => boolean {
  const terms = term
    .trim()
    .toLowerCase()
    .split(/\s+/)
    .filter(Boolean);
  const fields = dataFieldNames();
  return (data) => {
    for (const needle of terms) {
      let matched = false;
      for (const field of fields) {
        const raw = (data as unknown as Record<string, CellValue>)[field];
        if (raw === null || raw === undefined) continue;
        if (formatValueForDisplay(raw).toLowerCase().includes(needle)) {
          matched = true;
          break;
        }
      }
      if (!matched) return false;
    }
    return true;
  };
}

function applyQuickFilterTerm(term: string) {
  const table = grid.table.value;
  if (!table) return;
  if (!term.trim()) {
    table.clearFilter();
    return;
  }
  table.setFilter(buildQuickFilter(term));
}

let disposeGridPerfDiag: (() => void) | null = null;
let movingRowIndexColumn = false;
let editSnapshot: { rowData: CellValue[]; colIdx: number; previousValue: CellValue } | null = null;
/** Last clicked cell, used as the F2 target when the range module has no active cell. */
let lastClickedCell: TabulatorCellComponent | null = null;

function handleColumnMoved(moved: TabulatorColumnComponent) {
  const table = grid.table.value;
  scheduleLayoutCapture();
  if (!table || movingRowIndexColumn) return;

  const indexColumn = table.getColumn(ROW_INDEX_FIELD);
  const firstVisible = table.getColumns().find((column) => column.isVisible());
  if (indexColumn && firstVisible && firstVisible !== indexColumn) {
    movingRowIndexColumn = true;
    // `move` takes a target column/field, not an index: drop the frozen column back in front of
    // whatever now leads the visible order.
    indexColumn.move(firstVisible);
    movingRowIndexColumn = false;
    workspaceStore.showToast('「#」欄為固定欄，無法移動', 'info', 1600);
  }
  void moved;
}

function handleCellEditing(cell: TabulatorCellComponent) {
  const colIdx = getColIndex(cell.getField());
  if (colIdx === undefined) return;
  editSnapshot = {
    rowData: cell.getRow().getData() as unknown as CellValue[],
    colIdx,
    previousValue: cell.getValue() as CellValue,
  };
}

function handleCellEdited(cell: TabulatorCellComponent) {
  const colIdx = getColIndex(cell.getField());
  const rowData = cell.getRow().getData() as unknown as CellValue[];
  if (colIdx === undefined || !props.resultSet) return;

  const snapshot = editSnapshot;
  editSnapshot = null;
  if (!snapshot || snapshot.rowData !== rowData || snapshot.colIdx !== colIdx) return;

  // Tabulator turns an empty editor result into `undefined`; the column semantics are `NULL`.
  const rawNext = cell.getValue() as CellValue;
  const nextValue = rawNext === undefined ? null : rawNext;

  recordModification(rowData, colIdx, snapshot.previousValue, nextValue);
  applyCellValueClasses(cell.getElement(), nextValue, isCellModified(rowData, colIdx));
}

/** Any click remembers the row for the toolbar delete button; `#` also selects the whole row. */
function handleCellClick(_event: MouseEvent, cell: TabulatorCellComponent) {
  activeRowData.value = cell.getRow().getData() as unknown as CellValue[];
  lastClickedCell = cell;
  if (cell.getField() !== ROW_INDEX_FIELD) return;
  selection.selectRow(cell.getRow());
}

/**
 * F2 mirrors Tabulator's own Enter-to-edit shortcut for the active range cell. Read-only cells
 * (PK, identity, generated columns) simply keep the selection unchanged.
 */
function handleGridKeydown(event: KeyboardEvent) {
  if (event.key !== 'F2') return;

  const target = event.target as HTMLElement | null;
  if (
    target &&
    (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable)
  ) {
    return;
  }

  const cell = activeRangeCell(grid.table.value) ?? lastClickedCell;
  if (!cell || !props.resultSet) return;

  const colIdx = getColIndex(cell.getField());
  if (colIdx === undefined) return;

  event.preventDefault();
  event.stopPropagation();

  if (cell.getElement().classList.contains('tabulator-editing')) return;
  if (!isColumnEditable(colIdx, cell.getRow().getData() as unknown as CellValue[])) return;

  try {
    editCell(grid.table.value, cell);
  } catch {
    // A cell from a table that was rebuilt under the pointer is simply no longer editable.
  }
}

/** Clicking the `#` header selects the whole table. */
function handleHeaderClick(_event: MouseEvent, column: TabulatorColumnComponent) {
  if (column.getField() !== ROW_INDEX_FIELD) return;
  selection.selectAll();
}

function handleCellContext(event: MouseEvent, cell: TabulatorCellComponent) {
  event.preventDefault();
  event.stopPropagation();

  const menuWidth = 220;
  const menuHeight = 300;
  const x = Math.min(event.clientX, Math.max(0, window.innerWidth - menuWidth - 8));
  const y = Math.min(event.clientY, Math.max(0, window.innerHeight - menuHeight - 8));

  const field = cell.getField();
  const colIdx = getColIndex(field);
  const realColName =
    colIdx !== undefined && props.resultSet ? props.resultSet.columns[colIdx]?.name ?? field : '#';

  contextMenu.visible = true;
  contextMenu.x = x;
  contextMenu.y = y;
  contextMenu.colId = field;
  contextMenu.colName = realColName || '';
  contextMenu.cellValue = cell.getValue();
  contextMenu.rowIndex = cell.getRow().getPosition() - 1;
  contextMenu.rowData = cell.getRow().getData() as unknown as CellValue[];
  activeRowData.value = contextMenu.rowData;

  function closeMenu() {
    contextMenu.visible = false;
    document.removeEventListener('click', closeMenu);
  }
  setTimeout(() => {
    document.addEventListener('click', closeMenu);
  }, 0);
}

const grid = useTabulatorTable({
  isActive: () => gridIsActive.value,
  getRows: () => gridRowData.value,
  getColumnSignature: () => gridColumnSignature.value,
  buildOptions: () => ({
    height: '100%',
    layout: 'fitData',
    // Per the migration decision the grid ships with Tabulator's default all-columns renderer;
    // `renderHorizontal: "virtual"` is the documented follow-up if 150 column scrolling suffers.
    renderHorizontal: 'basic',
    movableColumns: true,
    selectableRows: false,
    selectableRange: true,
    selectableRangeColumns: true,
    selectableRangeRows: false,
    selectableRangeInitializeDefault: false,
    selectableRangeAutoFocus: false,
    editTriggerEvent: 'dblclick',
    headerSortClickElement: 'icon',
    tooltipDelay: 150,
    index: '__sqlightRowId',
    rowHeight: 24,
    columns: buildColumnDefinitions(),
    rowFormatter: (row: TabulatorRowComponent) => {
      const data = row.getData() as unknown as CellValue[];
      const element = row.getElement();
      element.classList.toggle('sqlight-row-inserted', isPendingInsert(data));
      element.classList.toggle('sqlight-row-pending-delete', isPendingDelete(data));
    },
  }),
  onReady: (table) => {
    lastClickedCell = null;
    selection.attach(table);
    table.on('cellDblClick', handleCellDoubleClick);
    table.on('cellContext', handleCellContext);
    table.on('cellClick', handleCellClick);
    table.on('headerClick', handleHeaderClick);
    table.on('cellEditing', handleCellEditing);
    table.on('cellEdited', handleCellEdited);
    table.on('columnMoved', handleColumnMoved);
    table.on('columnResized', scheduleLayoutCapture);
    table.on('columnVisibilityChanged', scheduleLayoutCapture);
    table.on('dataSorted', scheduleLayoutCapture);

    // Restore the remembered layout when returning to a previously viewed result set.
    gridLayoutStore.restore(layoutKey.value, table);
    applyQuickFilterTerm(quickFilter.value);
    applyRowStateClasses();

    const container = gridContainerRef.value;
    if (container) {
      disposeGridPerfDiag?.();
      disposeGridPerfDiag = startGridPerfDiag({
        table,
        container,
        label: currentTab.value?.title,
        applyFilter: applyQuickFilterTerm,
        getFilter: () => quickFilter.value,
      });
    }
  },
});

watch([gridIsActive, gridColumnSignature], async () => {
  await nextTick();
  await grid.sync();
});

// `useTabulatorTable` builds the table into its own container element; the wrapper keeps the
// theme/scroll responsibilities.
const gridTableRef = grid.containerRef;

// The result set is usually already available when this component mounts, so the first build has
// to be triggered explicitly (the watchers only cover later changes).
onMounted(async () => {
  await nextTick();
  await grid.sync();
});

watch(
  () => gridRowData.value,
  async () => {
    await nextTick();
    await grid.sync();
  }
);

watch(quickFilter, (value) => {
  applyQuickFilterTerm(value);
});

function handleGenerateDml(type: 'INSERT' | 'UPDATE' | 'DELETE') {
  if (!hasRows.value) {
    workspaceStore.showToast('查無資料列可供產生 DML (0 筆)', 'info', 2000);
    contextMenu.visible = false;
    return;
  }
  const row = contextMenu.rowData || (contextMenu.rowIndex >= 0 && props.resultSet ? props.resultSet.rows[contextMenu.rowIndex] : null);
  if (!row || !props.resultSet) {
    contextMenu.visible = false;
    return;
  }

  const tab = currentTab.value;
  const sql = tab?.sql || '';
  const parsedTarget = parseTargetTableFromSql(sql);

  const tableName = parsedTarget?.tableName || tab?.tableName || tab?.title || 'TargetTable';
  let schema = parsedTarget?.schema || tab?.schema;

  const connId = tab?.connectionId || connectionStore.activeConnectionId || undefined;
  const db = tab?.database || connectionStore.activeDatabase || undefined;

  const tableSchema = connId && db
    ? (schemaStore.getTable(tableName, connId, db) || (schema ? schemaStore.getTable(`${schema}.${tableName}`, connId, db) : undefined))
    : undefined;

  if (!schema && tableSchema?.schema) {
    schema = tableSchema.schema;
  }
  if (!schema) {
    schema = 'dbo';
  }

  const primaryKeyColumns = tableSchema?.columns
    .filter((c) => c.isPrimaryKey)
    .map((c) => c.name);

  const pkColNames = new Set(
    primaryKeyColumns?.map((c) => c.toLowerCase()) ?? []
  );

  const identityColNames = new Set(
    tableSchema?.columns
      .filter((c) => c.isIdentity)
      .map((c) => c.name.toLowerCase()) ?? []
  );

  const columns: ColumnInfo[] = props.resultSet.columns.map((col) => ({
    name: col.name,
    dataType: col.dataType,
    isPrimaryKey: pkColNames.has(col.name.toLowerCase()),
    isIdentity: identityColNames.has(col.name.toLowerCase()),
  }));

  const dmlParams: GenerateDmlParams = {
    tableName,
    schema,
    database: db,
    columns,
    row,
    primaryKeyColumns,
  };

  let generated = '';
  try {
    if (type === 'INSERT') {
      generated = generateInsertStatement(dmlParams);
    } else if (type === 'UPDATE') {
      generated = generateUpdateStatement(dmlParams);
    } else if (type === 'DELETE') {
      generated = generateDeleteStatement(dmlParams);
    }

    try {
      navigator.clipboard?.writeText(generated);
    } catch (err) {
      // Ignore clipboard error
    }

    workspaceStore.addSqlTab(generated, `${type}: [${schema}].[${tableName}]`, connId, db);
    workspaceStore.showToast(`已建立 ${type} 語法並開啟新分頁（已複製至剪貼簿）`, 'success', 2500);
  } catch (err: any) {
    workspaceStore.showToast(`產生 ${type} 語法失敗: ${err?.message || err}`, 'error', 3500);
  }

  contextMenu.visible = false;
}

onBeforeUnmount(() => {
  captureLayoutNow();
  disposeGridPerfDiag?.();
  disposeGridPerfDiag = null;
  if (horizontalScrollTimer) {
    clearTimeout(horizontalScrollTimer);
    horizontalScrollTimer = null;
  }
  if (quickFilterTimer) {
    clearTimeout(quickFilterTimer);
    quickFilterTimer = null;
  }
});
</script>
