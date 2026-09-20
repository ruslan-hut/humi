import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { DecimalPipe } from '@angular/common';
import { RouterLink } from '@angular/router';

import { BAND_LABEL, NodeState, rhBand } from '../../core/models';

@Component({
  selector: 'app-node-card',
  imports: [RouterLink, DecimalPipe],
  templateUrl: './node-card.html',
  styleUrl: './node-card.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class NodeCard {
  readonly node = input.required<NodeState>();

  /** State is shown as icon + word + color, never colour alone. */
  readonly state = computed(() => {
    const n = this.node();
    if (!n.online || !n.last) {
      return { key: 'offline', label: 'Offline', glyph: '—' };
    }
    const band = rhBand(n.last.rh);
    const glyph = band === 'normal' ? '✓' : band === 'damp' ? '▲' : '●';
    return { key: band, label: BAND_LABEL[band], glyph };
  });

  readonly lowBattery = computed(() => {
    const v = this.node().last?.vbat;
    return v !== undefined && v < 3.4;
  });

  readonly seen = computed(() => {
    const at = this.node().last_seen;
    if (!at) {
      return 'never';
    }
    const mins = Math.round((Date.now() / 1000 - at) / 60);
    if (mins < 1) return 'just now';
    if (mins < 60) return `${mins} min ago`;
    const hours = Math.round(mins / 60);
    if (hours < 24) return `${hours} h ago`;
    return `${Math.round(hours / 24)} d ago`;
  });
}
