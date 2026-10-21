import { phpver } from '../wailsjs/go/models';

export type PhpLifecycleState = 'supported' | 'warning' | 'eol';

export interface PhpLifecycleEntry {
  MajorMin: string;
  EOLDate: string;
  DaysLeft: number;
  State: PhpLifecycleState | string;
}

/**
 * 將後端 PHPLifecycle 清單轉為以「主.次」版本為 key 的查表，
 * 供依賴庫與服務卡片依分支查詢支援狀態。
 */
export function buildPhpLifecycleIndex(list?: phpver.Branch[] | null): Record<string, PhpLifecycleEntry> {
  const index: Record<string, PhpLifecycleEntry> = {};
  (list || []).forEach(branch => {
    if (branch && branch.MajorMin) {
      index[branch.MajorMin] = branch as unknown as PhpLifecycleEntry;
    }
  });
  return index;
}

/**
 * 由依賴 key（如 php85）或完整版本（如 8.5.12）取出「主.次」版本（8.5）。
 * 無法解析時回傳空字串，呼叫端據此略過標示。
 */
export function toMajorMin(value: string): string {
  const digits = (value || '').replace(/^php_?/, '');
  if (digits.includes('.')) {
    return digits.split('.').slice(0, 2).join('.');
  }
  if (digits.length >= 2) {
    return `${digits[0]}.${digits[1]}`;
  }
  return '';
}

/** 判斷某個主次版本是否已停止支援或進入預警期（供條件渲染使用） */
export function isPhpLifecycleAlert(entry?: PhpLifecycleEntry): boolean {
  return !!entry && (entry.State === 'eol' || entry.State === 'warning');
}
