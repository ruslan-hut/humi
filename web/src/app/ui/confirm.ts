import {
  ChangeDetectionStrategy,
  Component,
  ElementRef,
  Injectable,
  effect,
  inject,
  signal,
  viewChild,
} from '@angular/core';

export interface ConfirmRequest {
  title: string;
  message: string;
  confirm: string;
  danger?: boolean;
}

interface Pending extends ConfirmRequest {
  resolve: (ok: boolean) => void;
}

/** A modal yes/no question. On a phone it is a bottom sheet within thumb reach. */
@Injectable({ providedIn: 'root' })
export class Confirm {
  readonly pending = signal<Pending | null>(null);

  ask(req: ConfirmRequest): Promise<boolean> {
    this.pending()?.resolve(false);
    return new Promise(resolve => this.pending.set({ ...req, resolve }));
  }

  answer(ok: boolean): void {
    this.pending()?.resolve(ok);
    this.pending.set(null);
  }
}

@Component({
  selector: 'app-confirm',
  template: `
    <dialog #dialog aria-labelledby="confirm-title" (cancel)="confirm.answer(false)" (click)="backdrop($event)">
      @if (confirm.pending(); as p) {
        <div class="body">
          <h2 id="confirm-title">{{ p.title }}</h2>
          <p>{{ p.message }}</p>
          <div class="actions">
            <button type="button" class="btn" (click)="confirm.answer(false)">Cancel</button>
            <button type="button" class="btn" [class.danger]="p.danger" [class.primary]="!p.danger"
                    (click)="confirm.answer(true)">{{ p.confirm }}</button>
          </div>
        </div>
      }
    </dialog>
  `,
  styles: `
    dialog {
      width: min(420px, 100vw - 32px);
      padding: 0;
      border: 1px solid var(--hairline);
      border-radius: 18px;
      background: var(--surface);
      color: var(--ink);

      &::backdrop { background: rgba(0, 0, 0, 0.4); }
    }

    @media (max-width: 559px) {
      dialog[open] {
        width: 100%;
        max-width: none;
        margin: auto 0 0;
        padding-bottom: env(safe-area-inset-bottom);
        border-radius: 18px 18px 0 0;
        border-width: 1px 0 0;
        animation: sheet 0.2s ease-out;
      }
    }

    @keyframes sheet {
      from { transform: translateY(24px); opacity: 0; }
    }

    .body { padding: 20px; }

    h2 { margin: 0 0 8px; font-size: 1.0625rem; font-weight: 600; }
    p { margin: 0 0 20px; font-size: 0.9375rem; line-height: 1.45; color: var(--ink-secondary); }

    .actions {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 10px;
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ConfirmHost {
  protected readonly confirm = inject(Confirm);
  private readonly dialog = viewChild.required<ElementRef<HTMLDialogElement>>('dialog');

  constructor() {
    effect(() => {
      const el = this.dialog().nativeElement;
      if (this.confirm.pending() && !el.open) {
        el.showModal();
      } else if (!this.confirm.pending() && el.open) {
        el.close();
      }
    });
  }

  /** A tap on the dimmed area cancels; the box itself is covered by .body. */
  protected backdrop(event: MouseEvent): void {
    if (event.target === this.dialog().nativeElement) {
      this.confirm.answer(false);
    }
  }
}
