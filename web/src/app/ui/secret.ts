import { ChangeDetectionStrategy, Component, inject, input, signal } from '@angular/core';

import { Toast } from './toast';

/** A value shown once — a node token or an invite link — with copy and, where
 *  the phone offers it, the system share sheet. */
@Component({
  selector: 'app-secret',
  template: `
    <div class="secret">
      <code class="value">{{ value() }}</code>
      <div class="actions">
        <button type="button" class="btn" (click)="copy()">{{ copied() ? 'Copied' : 'Copy' }}</button>
        @if (canShare) {
          <button type="button" class="btn" (click)="share()">Share…</button>
        }
      </div>
      <p class="hint">{{ hint() }}</p>
    </div>
  `,
  styles: `
    .secret {
      display: grid;
      gap: 10px;
    }

    .value {
      display: block;
      padding: 12px 14px;
      border-radius: 10px;
      background: var(--plane);
      border: 1px dashed var(--axis);
      font-family: ui-monospace, "SF Mono", Menlo, monospace;
      font-size: 0.8125rem;
      line-height: 1.5;
      word-break: break-all;
      user-select: all;
      -webkit-user-select: all;
    }

    .actions {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
      gap: 8px;
    }

    .hint {
      margin: 0;
      font-size: 0.8125rem;
      color: var(--ink-muted);
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Secret {
  readonly value = input.required<string>();
  /** Title for the share sheet. */
  readonly title = input('Humi');
  readonly hint = input('Shown once. Copy it now — it cannot be displayed again.');

  private readonly toast = inject(Toast);

  protected readonly canShare = typeof navigator !== 'undefined' && 'share' in navigator;
  protected readonly copied = signal(false);

  protected async copy(): Promise<void> {
    try {
      await navigator.clipboard.writeText(this.value());
      this.copied.set(true);
      setTimeout(() => this.copied.set(false), 2000);
    } catch {
      this.toast.error('Copy failed — select the text and copy it by hand');
    }
  }

  protected async share(): Promise<void> {
    const v = this.value();
    try {
      await navigator.share(v.startsWith('http') ? { title: this.title(), url: v } : { title: this.title(), text: v });
    } catch {
      // Closing the share sheet rejects; nothing to report.
    }
  }
}
