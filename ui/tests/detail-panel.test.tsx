import assert from 'node:assert/strict'
import { renderToStaticMarkup } from 'react-dom/server'
import { DetailPanel } from '../src/components/DetailPanel'
import type { Column, DiffResult, SelectionState } from '../src/types'

const column = (comment: string): Column => ({
  Name: 'image_url',
  OrdinalPos: 1,
  DataType: 'text',
  RawType: 'text',
  Nullable: false,
  Default: null,
  Comment: comment,
  CharMaxLen: null,
  NumPrecision: null,
  NumScale: null,
})

const result: DiffResult = {
  SourceName: 'QA Database',
  TargetName: 'PROD Database',
  Tables: [{
    Name: 'ai_avatar_valid',
    Change: 'modified',
    Columns: [{
      Name: 'image_url',
      Change: 'modified',
      Source: column(''),
      Target: column('AI avatar image URL'),
    }],
    Indexes: [],
    Constraints: [],
  }],
  Views: [],
  Identical: false,
}

const selection: SelectionState = {
  tables: new Set(['ai_avatar_valid']),
  columns: { ai_avatar_valid: new Set(['image_url']) },
  indexes: {},
  constraints: {},
  views: new Set(),
}

const markup = renderToStaticMarkup(
  <DetailPanel
    result={result}
    activeItem={{ type: 'table', name: 'ai_avatar_valid' }}
    selection={selection}
    onToggleColumn={() => {}}
    onToggleIndex={() => {}}
    onToggleConstraint={() => {}}
    onToggleAllColumns={() => {}}
    onToggleAllIndexes={() => {}}
    onToggleAllConstraints={() => {}}
  />,
)

assert.ok(
  markup.includes('Source (QA Database)'),
  'modified columns should label the source database',
)
assert.ok(
  markup.includes('Target (PROD Database)'),
  'modified columns should label the target database',
)
