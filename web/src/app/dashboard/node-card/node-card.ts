import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { DecimalPipe } from '@angular/common';
import { RouterLink } from '@angular/router';

import { BAND_LABEL, NodeState, rhBand } from '../../core/models';
import { ago } from '../../core/time';

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
    const band = rhBand(n.last.rh, n);
    const glyph = band === 'normal' ? '✓' : band === 'damp' ? '▲' : '●';
    return { key: band, label: BAND_LABEL[band], glyph };
  });

  readonly lowBattery = computed(() => {
    const v = this.node().last?.vbat;
    return v !== undefined && v < 3.4;
  });

  readonly seen = computed(() => ago(this.node().last_seen));
}
