import * as Turbo from '@hotwired/turbo';
import { morphElements } from '@hotwired/turbo';

export { Turbo };

export function streamActionReload() {
  document.querySelectorAll(`turbo-frame`).forEach((frame) => {
    if (
      frame.dataset.turboReload === undefined ||
      (frame.dataset.turboReload !== '' &&
        frame.dataset.turboReload.toLowerCase !== 'turbo-reload')
    ) {
      return;
    }

    const newFrame = this.templateContent.getElementById(frame.id);
    if (frame.getAttribute('refresh') === 'morph') {
      morphElements(frame, newFrame);
      return;
    }
    frame.replaceWith(newFrame);
  });
}
