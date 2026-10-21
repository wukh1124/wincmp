import React from 'react';
import { t } from '../i18n';
import type { PhpLifecycleEntry } from '../phpLifecycle';

interface Props {
  entry?: PhpLifecycleEntry;
  /** compact 用於列表列，字級與內距更小 */
  compact?: boolean;
}

/**
 * PHP 分支支援狀態徽章：EOL（已停止支援）與「即將停止支援」預警。
 * supported 或未知分支不顯示任何標記。
 */
export default function PhpLifecycleBadge({ entry, compact = false }: Props) {
  if (!entry || (entry.State !== 'eol' && entry.State !== 'warning')) {
    return null;
  }

  const isEOL = entry.State === 'eol';
  const title = isEOL
    ? t("已於 %s 停止支援，上游不再提供安全修補", entry.EOLDate)
    : t("將於 %s 停止支援（剩餘 %d 天）", entry.EOLDate, Math.max(entry.DaysLeft, 0));

  return (
    <span
      className={`${compact ? 'text-[10px] px-1.5 py-0.5' : 'text-[10px] px-2 py-0.5'} rounded font-semibold shrink-0 select-none leading-none`}
      style={{
        border: `1px solid ${isEOL ? 'var(--status-error)' : 'var(--status-warn)'}`,
        color: isEOL ? 'var(--status-error)' : 'var(--status-warn)',
        background: isEOL ? 'var(--status-error-bg)' : 'var(--status-warn-bg)',
      }}
      title={title}
    >
      {isEOL ? t("EOL") : t("即將停止支援")}
    </span>
  );
}
