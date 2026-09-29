import { ChangeDetectionStrategy, Component, Injectable, inject, signal } from '@angular/core';

const SHOW_MS = 2600;

/** One short status line at the bottom of the screen, e.g. "Saved". */
@Injectable({ providedIn: 'root' })
export class Toast {
  readonly message = signal<{ text: string; error: boolean } | null>(null);
  private timer: ReturnType<typeof setTimeout> | undefined;

  show(text: string): void {
    this.set(text, false);
  }

  error(text: string): void {
    this.set(text, true);
  }

  private set(text: string, error: boolean): void {
    clearTimeout(this.timer);
    this.message.set({ text, error });
    this.timer = setTimeout(() => this.message.set(null), SHOW_MS);
  }
}

@Component({
  selector: 'app-toast',
  template: `
    <div class="toast" role="status" aria-live="polite">
      @if (toast.message(); as m) {
        <span class="msg" [class.error]="m.error">{{ m.text }}</span>
      }
    </div>
  `,
  styles: `
    .toast {
      position: fixed;
      left: 0;
      right: 0;
      bottom: calc(20px + env(safe-area-inset-bottom));
      z-index: 20;
      display: flex;
      justify-content: center;
      padding: 0 16px;
      pointer-events: none;
    }

    .msg {
      max-width: 420px;
      padding: 11px 16px;
      border-radius: 12px;
      background: var(--ink);
      color: var(--surface);
      font-size: 0.875rem;
      font-weight: 600;
      box-shadow: 0 6px 24px rgba(0, 0, 0, 0.18);
      animation: rise 0.18s ease-out;

      &.error { background: var(--critical); color: #fff; }
    }

    @keyframes rise {
      from { transform: translateY(8px); opacity: 0; }
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ToastHost {
  protected readonly toast = inject(Toast);
}
